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

package tidbgroup

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type Controller struct {
	control  ControlInterface
	tgLister listers.TiDBGroupLister
	queue    workqueue.RateLimitingInterface
}

func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory) *Controller {
	tidbGroupInformer := informerFactory.Pingcap().V1alpha1().TiDBGroups()
	tg := &Controller{
		control:  NewDefaultTiDBGroupControl(),
		tgLister: tidbGroupInformer.Lister(),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"tidbgroup"),
	}
	controller.WatchForObject(tidbGroupInformer.Informer(), tg.queue)
	return tg
}

func (tgc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tgc.queue.ShutDown()

	klog.Info("Starting TiDBGroup controller")
	defer klog.Info("Shutting down TiDBGroup controller")
	for i := 0; i < workers; i++ {
		go wait.Until(tgc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (tgc *Controller) worker() {
	for tgc.processNestWorkItem() {
		// revive:disable:empty-block
	}
}

func (tgc *Controller) processNestWorkItem() bool {
	key, quit := tgc.queue.Get()
	if quit {
		return false
	}
	defer tgc.queue.Done(key)
	if err := tgc.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("TiDBGroup: %v, still need sync: %v, requeue", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TiDBGroup: %v, sync failed, err: %v", key.(string), err))
		}
		tgc.queue.AddRateLimited(key)
	} else {
		tgc.queue.Forget(key)
	}
	return true
}

func (tgc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing TiDBGroup %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	ta, err := tgc.tgLister.TiDBGroups(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TiDBGroup has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tgc.control.ReconcileTiDBGroup(ta)
}
