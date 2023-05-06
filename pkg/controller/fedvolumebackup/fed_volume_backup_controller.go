// Copyright 2023 PingCAP, Inc.
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

package fedvolumebackup

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup/backup"
	"github.com/pingcap/tidb-operator/pkg/metrics"
)

// Controller controls VolumeBackup.
type Controller struct {
	deps *controller.BrFedDependencies
	// control returns an interface capable of syncing a VolumeBackup.
	// Abstracted out for testing.
	control ControlInterface
	// VolumeBackups that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a VolumeBackup controller.
func NewController(deps *controller.BrFedDependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultVolumeBackupControl(deps.Clientset, backup.NewBackupManager(deps)),
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"volumeBackup",
		),
	}

	volumeBackupInformer := deps.InformerFactory.Federation().V1alpha1().VolumeBackups()
	volumeBackupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updateBackup,
		UpdateFunc: func(old, cur interface{}) {
			c.updateBackup(cur)
		},
		DeleteFunc: c.updateBackup,
	})

	return c
}

// Name returns VolumeBackup controller name.
func (c *Controller) Name() string {
	return "volumeBackup"
}

// Run runs the VolumeBackup controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting volumeBackup controller")
	defer klog.Info("Shutting down volumeBackup controller")

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
			klog.Infof("VolumeBackup: %v, still need sync: %v, requeuing", key.(string), err)
			c.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.V(4).Infof("VolumeBackup: %v, ignore err: %v", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("VolumeBackup: %v, sync failed, err: %v, requeuing", key.(string), err))
			c.queue.AddRateLimited(key)
		}
	} else {
		c.queue.Forget(key)
	}
	return true
}

// sync syncs the given VolumeBackup.
func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing VolumeBackup %q (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	volumeBackup, err := c.deps.VolumeBackupLister.VolumeBackups(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("VolumeBackup has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.syncBackup(volumeBackup.DeepCopy())
}

func (c *Controller) syncBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	return c.control.UpdateBackup(volumeBackup)
}

func (c *Controller) updateBackup(cur interface{}) {
	newVolumeBackup := cur.(*v1alpha1.VolumeBackup)
	ns := newVolumeBackup.GetNamespace()
	name := newVolumeBackup.GetName()

	if newVolumeBackup.DeletionTimestamp != nil {
		// the backup is being deleted, we need to do some cleanup work, enqueue backup.
		klog.Infof("VolumeBackup %s/%s is being deleted", ns, name)
		c.enqueueBackup(newVolumeBackup)
		return
	}

	// TODO(federation): check something like non-federation's
	// `IsBackupInvalid`, `IsBackupComplete`, `IsBackupFailed`, `IsBackupScheduled`, `IsBackupRunning`, `IsBackupPrepared`, `IsLogBackupStopped

	klog.V(4).Infof("VolumeBackup object %s/%s enqueue", ns, name)
	c.enqueueBackup(newVolumeBackup)
}

// enqueueBackup enqueues the given VolumeBackup in the work queue.
func (c *Controller) enqueueBackup(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}
