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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type Controller struct {
	control  ControlInterface
	taLister listers.TidbClusterAutoScalerLister
	queue    workqueue.RateLimitingInterface
}

func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidbclusterautoscaler"})
	autoScalerInformer := informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers()
	asm := autoscaler.NewAutoScalerManager(kubeCli, cli, informerFactory, kubeInformerFactory, recorder)

	tac := &Controller{
		control:  NewDefaultAutoScalerControl(recorder, asm),
		taLister: autoScalerInformer.Lister(),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"tidbclusterautoscaler"),
	}
	controller.WatchForObject(autoScalerInformer.Informer(), tac.queue)
	return tac
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
		// revive:disable:empty-block
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
	ta, err := tac.taLister.TidbClusterAutoScalers(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TidbClusterAutoScaler has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tac.control.ResconcileAutoScaler(ta)
}
