// Copyright 2021 PingCAP, Inc.
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

package tidbngmonitoring

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/manager/tidbngmonitoring"
	"github.com/pingcap/tidb-operator/pkg/metrics"

	perrors "github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller sync TidbNGMonitoring
type Controller struct {
	deps    *controller.Dependencies
	control ControlInterface
	queue   workqueue.RateLimitingInterface
}

func NewController(deps *controller.Dependencies) *Controller {
	control := NewDefaultTiDBNGMonitoringControl(
		deps,
		tidbngmonitoring.NewNGMonitorManager(deps),
		tidbngmonitoring.NewTCAssetManager(deps),
		meta.NewReclaimPolicyManager(deps),
		deps.Recorder,
	)

	c := &Controller{
		deps:    deps,
		control: control,
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"tidb-ng-monitoring",
		),
	}

	tnmInformer := deps.InformerFactory.Pingcap().V1alpha1().TidbNGMonitorings()
	stsInformer := deps.KubeInformerFactory.Apps().V1().StatefulSets()
	controller.WatchForObject(tnmInformer.Informer(), c.queue)
	controller.WatchForController(
		stsInformer.Informer(),
		c.queue,
		func(ns, name string) (runtime.Object, error) {
			return c.deps.TiDBNGMonitoringLister.TidbNGMonitorings(ns).Get(name)
		},
		nil,
	)

	return c
}

// Name returns the name of the controller
func (c *Controller) Name() string {
	return "tidb-ng-monitoring"
}

func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting tidbngmonitor controller")
	defer klog.Info("Shutting down tidbngmonitor controller")

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *Controller) worker() {
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
			klog.Infof("TidbNGMonitoring %v still need sync: %v, requeuing", key, err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TidbNGMonitoring %v sync failed, err: %v", key, err))
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
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing TidbNGMonitoring %s (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	tngm, err := c.deps.TiDBNGMonitoringLister.TidbNGMonitorings(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TidbNGMonitoring %s has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.control.Reconcile(tngm)
}
