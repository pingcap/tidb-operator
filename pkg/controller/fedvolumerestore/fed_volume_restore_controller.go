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

package fedvolumerestore

import (
	"fmt"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup/restore"
	"github.com/pingcap/tidb-operator/pkg/metrics"
)

// Controller controls VolumeRestore.
type Controller struct {
	deps *controller.BrFedDependencies
	// control returns an interface capable of syncing a VolumeRestore.
	// Abstracted out for testing.
	control ControlInterface
	// VolumeRestores that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a VolumeRestore controller.
func NewController(deps *controller.BrFedDependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultVolumeRestoreControl(deps.Clientset, restore.NewRestoreManager(deps)),
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"volumeRestore",
		),
	}

	volumeRestoreInformer := deps.InformerFactory.Federation().V1alpha1().VolumeRestores()
	volumeRestoreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updateRestore,
		UpdateFunc: func(old, cur interface{}) {
			c.updateRestore(cur)
		},
		DeleteFunc: c.updateRestore,
	})

	return c
}

// Name returns VolumeRestore controller name.
func (c *Controller) Name() string {
	return "volumeRestore"
}

// Run runs the VolumeRestore controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting volumeRestore controller")
	defer klog.Info("Shutting down volumeRestore controller")

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
			klog.Infof("VolumeRestore: %v, still need sync: %v, requeuing", key.(string), err)
			c.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.V(4).Infof("VolumeRestore: %v, ignore err: %v", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("VolumeRestore: %v, sync failed, err: %v, requeuing", key.(string), err))
			c.queue.AddRateLimited(key)
		}
	} else {
		c.queue.Forget(key)
	}
	return true
}

// sync syncs the given VolumeRestore.
func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing VolumeRestore %q (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	volumeRestore, err := c.deps.VolumeRestoreLister.VolumeRestores(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("VolumeRestore has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.syncRestore(volumeRestore.DeepCopy())
}

func (c *Controller) syncRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	return c.control.UpdateRestore(volumeRestore)
}

func (c *Controller) updateRestore(cur interface{}) {
	newVolumeRestore := cur.(*v1alpha1.VolumeRestore)
	ns := newVolumeRestore.GetNamespace()
	name := newVolumeRestore.GetName()

	if newVolumeRestore.DeletionTimestamp != nil {
		// the restore is being deleted, we need to do some cleanup work, enqueue backup.
		klog.Infof("VolumeRestore %s/%s is being deleted", ns, name)
		c.enqueueRestore(newVolumeRestore)
		return
	}

	if v1alpha1.IsVolumeRestoreFailed(newVolumeRestore) {
		klog.V(4).Infof("volume restore %s/%s is failed, skipping.", ns, name)
		return
	}

	if newVolumeRestore.Spec.Template.Warmup == pingcapv1alpha1.RestoreWarmupModeASync {
		if !v1alpha1.IsVolumeRestoreWarmUpComplete(newVolumeRestore) {
			c.enqueueRestore(newVolumeRestore)
		}
	}

	if v1alpha1.IsVolumeRestoreComplete(newVolumeRestore) {
		klog.V(4).Infof("volume restore %s/%s is complete, skipping.", ns, name)
		return
	}

	klog.V(4).Infof("VolumeRestore object %s/%s enqueue", ns, name)
	c.enqueueRestore(newVolumeRestore)
}

// enqueueRestore enqueues the given VolumeRestore in the work queue.
func (c *Controller) enqueueRestore(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}
