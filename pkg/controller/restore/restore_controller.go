// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package restore

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/restore"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller controls restore.
type Controller struct {
	deps *controller.Dependencies
	// control returns an interface capable of syncing a restore.
	// Abstracted out for testing.
	control ControlInterface
	// restores that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a restore controller.
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultRestoreControl(restore.NewRestoreManager(deps)),
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"restore",
		),
	}

	restoreInformer := deps.InformerFactory.Pingcap().V1alpha1().Restores()
	restoreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updateRestore,
		UpdateFunc: func(old, cur interface{}) {
			c.updateRestore(cur)
		},
		DeleteFunc: c.enqueueRestore,
	})
	return c
}

// Name returns the name of the restore controller
func (c *Controller) Name() string {
	return "restore"
}

// Run runs the restore controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting restore controller")
	defer klog.Info("Shutting down restore controller")

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (c *Controller) processNextWorkItem() bool {
	metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(1)
	defer metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(-1)

	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("Restore: %v, still need sync: %v, requeuing", key.(string), err)
			c.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.V(4).Infof("Restore: %v, ignore err: %v", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("Restore: %v, sync failed, err: %v, requeuing", key.(string), err))
			c.queue.AddRateLimited(key)
		}
	} else {
		c.queue.Forget(key)
	}
	return true
}

// sync syncs the given restore.
func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing Restore %q (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	restore, err := c.deps.RestoreLister.Restores(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("Restore has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.syncRestore(restore.DeepCopy())
}

func (c *Controller) syncRestore(restore *v1alpha1.Restore) error {
	return c.control.UpdateRestore(restore)
}

func (c *Controller) updateRestore(cur interface{}) {
	newRestore := cur.(*v1alpha1.Restore)
	klog.V(4).Infof("restore-manager update %v", newRestore)

	ns := newRestore.GetNamespace()
	name := newRestore.GetName()

	if v1alpha1.IsRestoreInvalid(newRestore) {
		klog.V(4).Infof("restore %s/%s is Invalid, skipping.", ns, name)
		return
	}

	if v1alpha1.IsRestoreComplete(newRestore) {
		if newRestore.Spec.Warmup == v1alpha1.RestoreWarmupModeASync {
			if !v1alpha1.IsRestoreWarmUpComplete(newRestore) {
				c.enqueueRestore(newRestore)
				return
			}
		}

		klog.V(4).Infof("restore %s/%s is Complete, skipping.", ns, name)
		return
	}

	if v1alpha1.IsRestoreFailed(newRestore) {
		klog.V(4).Infof("restore %s/%s is Failed, skipping.", ns, name)
		return
	}

	// check if job failed, but restore status not failed
	jobFailed, reason, err := c.isRestoreJobFailed(newRestore)
	if err != nil {
		klog.Warningf("restore %s/%s check is job failed error: %s", ns, name, err.Error())
		return
	}
	if jobFailed {
		klog.Errorf("restore %s/%s job failed: %s", ns, name, reason)
		err = c.control.UpdateCondition(newRestore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "AlreadyFailed",
			Message: reason,
		})
		if err != nil {
			klog.Errorf("Fail to update the condition of restore %s/%s, %v", ns, name, err)
		}
		return
	}

	if v1alpha1.IsRestoreDataComplete(newRestore) {
		tc, err := c.getTC(newRestore)
		if err != nil {
			klog.Errorf("Fail to get tidbcluster for restore %s/%s, %v", ns, name, err)
			return
		}
		if tc.IsRecoveryMode() && newRestore.Spec.FederalVolumeRestorePhase == v1alpha1.FederalVolumeRestoreFinish {
			c.enqueueRestore(newRestore)
			return
		}

		klog.V(4).Infof("restore %s/%s is already DataComplete, skipping.", ns, name)
		return
	}

	if v1alpha1.IsRestoreTiKVComplete(newRestore) {
		if newRestore.Spec.FederalVolumeRestorePhase == v1alpha1.FederalVolumeRestoreData ||
			newRestore.Spec.FederalVolumeRestorePhase == v1alpha1.FederalVolumeRestoreFinish {
			c.enqueueRestore(newRestore)
			return
		}

		klog.V(4).Infof("restore %s/%s is already TikvComplete, skipping.", ns, name)
		return
	}

	if v1alpha1.IsRestoreVolumeComplete(newRestore) {
		c.enqueueRestore(newRestore)
		return
	}

	if v1alpha1.IsRestoreScheduled(newRestore) || v1alpha1.IsRestoreRunning(newRestore) {
		klog.V(4).Infof("restore %s/%s is already Scheduled, Running or Failed, skipping.", ns, name)
		return
	}

	klog.V(4).Infof("restore object %s/%s enqueue", ns, name)
	c.enqueueRestore(newRestore)
}

// enqueueRestore enqueues the given restore in the work queue.
func (c *Controller) enqueueRestore(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}

func (c *Controller) getTC(restore *v1alpha1.Restore) (*v1alpha1.TidbCluster, error) {
	restoreNamespace := restore.GetNamespace()
	if restore.Spec.BR.ClusterNamespace != "" {
		restoreNamespace = restore.Spec.BR.ClusterNamespace
	}

	return c.deps.TiDBClusterLister.TidbClusters(restoreNamespace).Get(restore.Spec.BR.Cluster)
}

func (c *Controller) isRestoreJobFailed(restore *v1alpha1.Restore) (jobFailed bool, reason string, err error) {
	ns := restore.GetNamespace()
	name := restore.GetName()
	jobName := restore.GetRestoreJobName()

	job, err := c.deps.JobLister.Jobs(ns).Get(jobName)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Fail to get job %s for restore %s/%s, error %v ", jobName, ns, name, err)
		return false, "", err
	}
	if job != nil {
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				reason = fmt.Sprintf("Job %s has failed: %s", jobName, condition.Reason)
				return true, reason, nil
			}
		}
	}
	return false, "", nil
}
