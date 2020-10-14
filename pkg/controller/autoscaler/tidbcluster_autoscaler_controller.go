// Copyright 2020 PingCAP, Inc.
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

package autoscaler

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type Controller struct {
	deps    *controller.Dependencies
	control ControlInterface
	queue   workqueue.RateLimitingInterface
}

func NewController(deps *controller.Dependencies) *Controller {
	t := &Controller{
		deps:    deps,
		control: NewDefaultAutoScalerControl(autoscaler.NewAutoScalerManager(deps)),
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "tidbclusterautoscaler"),
	}
	tidbAutoScalerInformer := deps.InformerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers()
	controller.WatchForObject(tidbAutoScalerInformer.Informer(), t.queue)
	return t
}

func (tac *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tac.queue.ShutDown()

	klog.Info("Starting TidbClusterAutoScaler controller")
	defer klog.Info("Shutting down tidbclusterAutoScaler controller")
	for i := 0; i < workers; i++ {
		go wait.Until(tac.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (tac *Controller) worker() {
	for tac.processNextWorkItem() {
	}
}

func (tac *Controller) processNextWorkItem() bool {
	key, quit := tac.queue.Get()
	if quit {
		return false
	}
	defer tac.queue.Done(key)
	if err := tac.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("TidbClusterAutoScaler: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TidbClusterAutoScaler: %v, sync failed, err: %v", key.(string), err))
		}
		tac.queue.AddRateLimited(key)
	} else {
		tac.queue.Forget(key)
	}
	return true
}

func (tac *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing TidbClusterAutoScaler %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	ta, err := tac.deps.TiDBClusterAutoScalerLister.TidbClusterAutoScalers(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TidbClusterAutoScaler has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tac.control.ResconcileAutoScaler(ta)
}
