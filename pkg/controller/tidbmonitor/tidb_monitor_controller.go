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

package tidbmonitor

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/monitor/monitor"
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
)

// Controller syncs TidbMonitor
type Controller struct {
	cli      client.Client
	control  ControlInterface
	tmLister listers.TidbMonitorLister
	// tidbMonitor that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a backup controller.
func NewController(
	kubeCli kubernetes.Interface,
	genericCli client.Client,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidbmonitor"})

	tidbMonitorInformer := informerFactory.Pingcap().V1alpha1().TidbMonitors()
	deploymentInformer := kubeInformerFactory.Apps().V1().Deployments()
	typedControl := controller.NewTypedControl(controller.NewRealGenericControl(genericCli, recorder))
	monitorManager := monitor.NewMonitorManager(kubeCli, cli, informerFactory, kubeInformerFactory, typedControl, recorder)

	tmc := &Controller{
		cli:      genericCli,
		control:  NewDefaultTidbMonitorControl(recorder, typedControl, monitorManager),
		tmLister: tidbMonitorInformer.Lister(),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"tidbmonitor",
		),
	}

	controller.WatchForObject(tidbMonitorInformer.Informer(), tmc.queue)
	controller.WatchForController(deploymentInformer.Informer(), tmc.queue, func(ns, name string) (runtime.Object, error) {
		return tmc.tmLister.TidbMonitors(ns).Get(name)
	}, nil)

	return tmc
}

func (tmc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tmc.queue.ShutDown()

	klog.Info("Starting tidbmonitor controller")
	defer klog.Info("Shutting down tidbmonitor controller")

	for i := 0; i < workers; i++ {
		go wait.Until(tmc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (tmc *Controller) worker() {
	for tmc.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (tmc *Controller) processNextWorkItem() bool {
	key, quit := tmc.queue.Get()
	if quit {
		return false
	}
	defer tmc.queue.Done(key)
	if err := tmc.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("TidbMonitor: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TidbMonitor: %v, sync failed, err: %v", key.(string), err))
		}
		tmc.queue.AddRateLimited(key)
	} else {
		tmc.queue.Forget(key)
	}
	return true
}

func (tmc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing TidbMonitor %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	tm, err := tmc.tmLister.TidbMonitors(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TidbMonitor has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tmc.control.ReconcileTidbMonitor(tm)
}
