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

package backupschedule

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/backupschedule"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// Controller controls restore.
type Controller struct {
	deps *controller.Dependencies
	// control returns an interface capable of syncing a restore.
	// Abstracted out for testing.
	control ControlInterface
	// backupSchedules that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a backup schedule controller.
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultBackupScheduleControl(controller.NewRealBackupScheduleStatusUpdater(deps), backupschedule.NewBackupScheduleManager(deps)),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"backupSchedule",
		),
	}

	deps.BackupScheduleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueBackupSchedule,
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueBackupSchedule(cur)
		},
		DeleteFunc: c.enqueueBackupSchedule,
	})

	return c
}

// Run runs the backup schedule controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting backup schedule controller")
	defer klog.Info("Shutting down backup schedule controller")

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
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("BackupSchedule: %v, still need sync: %v, requeuing", key.(string), err)
			c.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.V(4).Infof("BackupSchedule: %v, ignore err: %v, waiting for the next sync", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("BackupSchedule: %v, sync failed, err: %v, requeuing", key.(string), err))
			c.queue.AddRateLimited(key)
		}
	} else {
		c.queue.Forget(key)
	}
	return true
}

// sync syncs the given backupSchedule.
func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing BackupSchedule %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	bs, err := c.deps.BackupScheduleLister.BackupSchedules(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("BackupSchedule has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.syncBackupSchedule(bs.DeepCopy())
}

func (c *Controller) syncBackupSchedule(bs *v1alpha1.BackupSchedule) error {
	return c.control.UpdateBackupSchedule(bs)
}

// enqueueBackupSchedule enqueues the given restore in the work queue.
func (c *Controller) enqueueBackupSchedule(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}
