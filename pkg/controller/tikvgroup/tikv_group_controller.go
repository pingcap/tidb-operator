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

package tikvgroup

import (
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller struct {
	control  ControlInterface
	tgLister listers.TiKVGroupLister
	queue    workqueue.RateLimitingInterface
}

func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tikvgroup-controller-manager"})

	pdControl := pdapi.NewDefaultPDControl(kubeCli)
	tikvGroupInformer := informerFactory.Pingcap().V1alpha1().TiKVGroups()
	setInformer := kubeInformerFactory.Apps().V1().StatefulSets()
	svcInformer := kubeInformerFactory.Core().V1().Services()
	tcInformer := informerFactory.Pingcap().V1alpha1().TidbClusters()
	podInformer := kubeInformerFactory.Core().V1().Pods()

	tgControl := controller.NewRealTiKVGroupControl(cli, tikvGroupInformer.Lister(), recorder)
	setControl := controller.NewRealStatefuSetControl(kubeCli, setInformer.Lister(), recorder)
	svcControl := controller.NewRealServiceControl(kubeCli, svcInformer.Lister(), recorder)
	typedControl := controller.NewTypedControl(controller.NewRealGenericControl(genericCli, recorder))

	tikvManager := member.NewTiKVGroupMemberManager(genericCli,
		svcInformer.Lister(),
		setInformer.Lister(),
		podInformer.Lister(),
		tcInformer.Lister(),
		svcControl,
		setControl,
		typedControl,
		pdControl)

	tg := &Controller{
		control:  NewDefaultTikvGroupControl(tgControl, tikvManager),
		tgLister: tikvGroupInformer.Lister(),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"tikvgroup"),
	}
	controller.WatchForObject(tikvGroupInformer.Informer(), tg.queue)
	return tg
}

func (tgc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tgc.queue.ShutDown()

	klog.Info("Starting TiKVGroup controller")
	defer klog.Info("Shutting down TiKVGroup controller")
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
			klog.Infof("TiKVGroup: %v, still need sync: %v, requeue", key.(string), err)
		} else {
			utilruntime.HandleError(perrors.Errorf("TiKVGroup: %v, sync failed, err: %v", key.(string), err))
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
		klog.V(4).Infof("Finished syncing TiKVGroup %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	ta, err := tgc.tgLister.TiKVGroups(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TiKVGroup has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tgc.control.ReconcileTiKVGroup(ta)
}
