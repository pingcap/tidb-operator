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

package restore

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/restore"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
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
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1alpha1.SchemeGroupVersion.WithKind("Restore")

// Controller controls restore.
type Controller struct {
	// kubernetes client interface
	kubeClient kubernetes.Interface
	// operator client interface
	cli versioned.Interface
	// control returns an interface capable of syncing a restore.
	// Abstracted out for testing.
	control ControlInterface
	// restoreLister is able to list/get restore from a shared informer's store
	restoreLister listers.RestoreLister
	// restoreListerSynced returns true if the restore shared informer has synced at least once
	restoreListerSynced cache.InformerSynced
	// restores that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a restore controller.
func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "restore"})

	restoreInformer := informerFactory.Pingcap().V1alpha1().Restores()
	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	statusUpdater := controller.NewRealRestoreConditionUpdater(cli, restoreInformer.Lister(), recorder)
	jobControl := controller.NewRealJobControl(kubeCli, jobInformer.Lister(), recorder)

	rsc := &Controller{
		kubeClient: kubeCli,
		cli:        cli,
		control: NewDefaultRestoreControl(
			statusUpdater,
			restore.NewRestoreManager(
				restoreInformer.Lister(),
				jobControl,
			),
			recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"restore",
		),
	}

	restoreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: rsc.enqueueRestore,
		UpdateFunc: func(old, cur interface{}) {
			rsc.enqueueRestore(cur)
		},
		DeleteFunc: rsc.enqueueRestore,
	})
	rsc.restoreLister = restoreInformer.Lister()
	rsc.restoreListerSynced = restoreInformer.Informer().HasSynced

	return rsc
}

// Run runs the restore controller.
func (rsc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer rsc.queue.ShutDown()

	glog.Info("Starting restore controller")
	defer glog.Info("Shutting down restore controller")

	if !cache.WaitForCacheSync(stopCh, rsc.restoreListerSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(rsc.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (rsc *Controller) worker() {
	for rsc.processNextWorkItem() {
		// revive:disable:empty-block
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (rsc *Controller) processNextWorkItem() bool {
	key, quit := rsc.queue.Get()
	if quit {
		return false
	}
	defer rsc.queue.Done(key)
	if err := rsc.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			glog.Infof("Restore: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("restore: %v, sync failed %v, requeuing", key.(string), err))
		}
		rsc.queue.AddRateLimited(key)
	} else {
		rsc.queue.Forget(key)
	}
	return true
}

// sync syncs the given restore.
func (rsc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(4).Infof("Finished syncing Restore %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	restore, err := rsc.restoreLister.Restores(ns).Get(name)
	if errors.IsNotFound(err) {
		glog.Infof("Restore has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return rsc.syncRestore(restore.DeepCopy())
}

func (rsc *Controller) syncRestore(tc *v1alpha1.Restore) error {
	return rsc.control.UpdateRestore(tc)
}

// enqueueRestore enqueues the given restore in the work queue.
func (rsc *Controller) enqueueRestore(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
		return
	}
	rsc.queue.Add(key)
}
