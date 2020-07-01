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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
)

// Controller syncs TidbInitializer
type Controller struct {
	cli      versioned.Interface
	control  ControlInterface
	tiLister listers.TidbInitializerLister
	queue    workqueue.RateLimitingInterface
}

// NewController creates a backup controller.
func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidbinitializer"})

	tidbInitializerInformer := informerFactory.Pingcap().V1alpha1().TidbInitializers()
	tidbClusterInformer := informerFactory.Pingcap().V1alpha1().TidbClusters()
	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	typedControl := controller.NewTypedControl(controller.NewRealGenericControl(genericCli, recorder))

	tic := &Controller{
		cli: cli,
		control: NewDefaultTidbInitializerControl(
			recorder,
			member.NewTiDBInitManager(
				jobInformer.Lister(),
				genericCli,
				tidbInitializerInformer.Lister(),
				tidbClusterInformer.Lister(),
				typedControl,
			),
		),
		tiLister: tidbInitializerInformer.Lister(),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"tidbinitializer",
		),
	}

	controller.WatchForObject(tidbInitializerInformer.Informer(), tic.queue)
	m := make(map[string]string)
	m[label.ComponentLabelKey] = label.InitJobLabelVal
	controller.WatchForController(jobInformer.Informer(), tic.queue, func(ns, name string) (runtime.Object, error) {
		return tic.tiLister.TidbInitializers(ns).Get(name)
	}, m)

	return tic
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
		// revive:disable:empty-block
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
			utilruntime.HandleError(perrors.Errorf("TiDBInitializer: %v, sync failed, err: %v, requeuing", key.(string), err))
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
	ti, err := tic.tiLister.TidbInitializers(ns).Get(name)
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
