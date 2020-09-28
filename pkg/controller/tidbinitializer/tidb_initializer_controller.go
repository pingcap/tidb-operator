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

package tidbinitializer

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
)

// Controller syncs TidbInitializer
type Controller struct {
	deps    *controller.Dependencies
	control ControlInterface
	queue   workqueue.RateLimitingInterface
}

// NewController creates a backup controller.
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultTidbInitializerControl(member.NewTiDBInitManager(deps)),
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "tidbinitializer"),
	}

	controller.WatchForObject(deps.TiDBInitializerInformer.Informer(), c.queue)
	m := make(map[string]string)
	m[label.ComponentLabelKey] = label.InitJobLabelVal
	controller.WatchForController(deps.JobInformer.Informer(), c.queue, func(ns, name string) (runtime.Object, error) {
		return c.deps.TiDBInitializerLister.TidbInitializers(ns).Get(name)
	}, m)

	return c
}

// Run run workers
func (tic *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tic.queue.ShutDown()

	klog.Info("Starting tidbinitializer controller")
	defer klog.Info("Shutting down tidbinitializer controller")

	for i := 0; i < workers; i++ {
		go wait.Until(tic.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (tic *Controller) worker() {
	for tic.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (tic *Controller) processNextWorkItem() bool {
	key, quit := tic.queue.Get()
	if quit {
		return false
	}
	defer tic.queue.Done(key)
	if err := tic.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("TiDBInitializer: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TiDBInitializer: %v, sync failed, err: %v, requeuing", key.(string), err))
		}
		tic.queue.AddRateLimited(key)
	} else {
		tic.queue.Forget(key)
	}
	return true
}

func (tic *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing TiDBInitializer %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	ti, err := tic.deps.TiDBInitializerLister.TidbInitializers(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TiDBInitializer %v has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}
	if ti.DeletionTimestamp != nil {
		return nil
	}
	return tic.control.ReconcileTidbInitializer(ti)
}
