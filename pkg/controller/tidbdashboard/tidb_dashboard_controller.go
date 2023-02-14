// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/manager/tidbdashboard"
	"github.com/pingcap/tidb-operator/pkg/metrics"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller composes informer, queue and worker to a single object.
// It acts as a high-level manager of async event processing for TiDBDashboard crd.
type Controller struct {
	deps    *controller.Dependencies
	control ControlInterface
	queue   workqueue.RateLimitingInterface
}

func NewController(deps *controller.Dependencies) *Controller {
	control := NewTiDBDashboardControl(
		deps,
		tidbdashboard.NewManager(deps),
		tidbdashboard.NewTcTlsManager(deps),
		meta.NewReclaimPolicyManager(deps),
		deps.Recorder,
	)

	c := &Controller{
		deps:    deps,
		control: control,
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"tidb-dashboard",
		),
	}

	tdInformer := deps.InformerFactory.Pingcap().V1alpha1().TidbDashboards()
	stsInformer := deps.KubeInformerFactory.Apps().V1().StatefulSets()
	controller.WatchForObject(tdInformer.Informer(), c.queue)
	controller.WatchForController(
		stsInformer.Informer(),
		c.queue,
		func(ns, name string) (runtime.Object, error) {
			return c.deps.TiDBDashboardLister.TidbDashboards(ns).Get(name)
		},
		nil,
	)

	return c
}

// Name returns the name of the controller.
func (c *Controller) Name() string {
	return "tidb-dashboard"
}

func (c *Controller) Run(numOfWorkers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting tidb-dashboard controller")
	defer klog.Info("Shutting down tidb-dashboard controller")

	for i := 0; i < numOfWorkers; i++ {
		go wait.Until(c.doWork, time.Second, stopCh)
	}

	<-stopCh
}

func (c *Controller) doWork() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(1)
	defer metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(-1)

	keyIface, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(keyIface)

	key := keyIface.(string)
	err := c.sync(key)
	if err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("TidbDashboard %v still need sync: %v, re-queuing", key, err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TidbDashboard %v sync failed, err: %v", key, err))
		}
		c.queue.AddRateLimited(key)
	} else {
		c.queue.Forget(err)
	}

	return true
}

func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing TidbDashboard %s (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	td, err := c.deps.TiDBDashboardLister.TidbDashboards(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TidbDashboard %s has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.control.Reconcile(td)
}
