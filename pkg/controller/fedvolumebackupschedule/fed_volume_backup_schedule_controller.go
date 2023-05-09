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

package fedvolumebackupschedule

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
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup/backupschedule"
	"github.com/pingcap/tidb-operator/pkg/metrics"
)

// Controller controls VolumeBackupSchedule.
type Controller struct {
	deps *controller.BrFedDependencies
	// control returns an interface capable of syncing a VolumeBackupSchedule.
	// Abstracted out for testing.
	control ControlInterface
	// VolumeBackupSchedules that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a VolumeBackupSchedule controller.
func NewController(deps *controller.BrFedDependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultVolumeBackupScheduleControl(deps.Clientset, backupschedule.NewBackupScheduleManager(deps)),
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"volumeBackupSchedule",
		),
	}

	volumeBackupScheduleInformer := deps.InformerFactory.Federation().V1alpha1().VolumeBackupSchedules()
	volumeBackupScheduleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updateBackupSchedule,
		UpdateFunc: func(old, cur interface{}) {
			c.updateBackupSchedule(cur)
		},
		DeleteFunc: c.updateBackupSchedule,
	})

	return c
}

// Name returns VolumeBackupSchedule controller name.
func (c *Controller) Name() string {
	return "volumeBackupSchedule"
}

// Run runs the VolumeBackupSchedule controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting volumeBackupSchedule controller")
	defer klog.Info("Shutting down volumeBackupSchedule controller")

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
			klog.Infof("VolumeBackupSchedule: %v, still need sync: %v, requeuing", key.(string), err)
			c.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.V(4).Infof("VolumeBackupSchedule: %v, ignore err: %v", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("VolumeBackupSchedule: %v, sync failed, err: %v, requeuing", key.(string), err))
			c.queue.AddRateLimited(key)
		}
	} else {
		c.queue.Forget(key)
	}
	return true
}

// sync syncs the given VolumeBackupSchedule.
func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing VolumeBackupSchedule %q (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	volumeBackupSchedule, err := c.deps.VolumeBackupScheduleLister.VolumeBackupSchedules(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("VolumeBackupSchedule has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.syncBackupSchedule(volumeBackupSchedule.DeepCopy())
}

func (c *Controller) syncBackupSchedule(volumeBackupSchedule *v1alpha1.VolumeBackupSchedule) error {
	return c.control.UpdateBackupSchedule(volumeBackupSchedule)
}

func (c *Controller) updateBackupSchedule(cur interface{}) {
	newVolumeBackupSchedule := cur.(*v1alpha1.VolumeBackupSchedule)
	ns := newVolumeBackupSchedule.GetNamespace()
	name := newVolumeBackupSchedule.GetName()
	klog.V(4).Infof("VolumeBackupSchedule object %s/%s enqueue", ns, name)

	c.enqueueBackupSchedule(newVolumeBackupSchedule)
}

// enqueueBackupSchedule enqueues the given VolumeBackupSchedule in the work queue.
func (c *Controller) enqueueBackupSchedule(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}
