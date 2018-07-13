// Copyright 2018 PingCAP, Inc.
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

package tidbcluster

import (
	"fmt"

	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/new-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/new-operator/pkg/client/listers/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/new-operator/pkg/controller/tidbcluster/membermanager"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1.SchemeGroupVersion.WithKind("TidbCluster")

// Controller controls tidbclusters.
type Controller struct {
	// kubernetes client interface
	kubeClient kubernetes.Interface
	// operator client interface
	cli versioned.Interface
	// control returns an interface capable of syncing a tidb cluster.
	// Abstracted out for testing.
	control ControlInterface
	// tcLister is able to list/get tidbclusters from a shared informer's store
	tcLister listers.TidbClusterLister
	// tcListerSynced returns true if the tidbcluster shared informer has synced at least once
	tcListerSynced cache.InformerSynced
	// setLister is able to list/get stateful sets from a shared informer's store
	setLister appslisters.StatefulSetLister
	// setListerSynced returns true if the statefulset shared informer has synced at least once
	setListerSynced cache.InformerSynced
	// tidbclusters that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a tidbcluster controller.
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
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "tidbcluster"})

	tcInformer := informerFactory.Pingcap().V1().TidbClusters()
	setInformer := kubeInformerFactory.Apps().V1beta1().StatefulSets()

	tcc := &Controller{
		kubeClient: kubeCli,
		cli:        cli,
		control: NewDefaultTidbClusterControl(
			NewRealTidbClusterStatusUpdater(cli, tcInformer.Lister()),
			mm.NewPDMemberManager(
				controller.NewRealStatefuSetControl(kubeCli, setInformer.Lister(), recorder),
				setInformer.Lister(),
			),
			recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"tidbcluster",
		),
	}

	tcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: tcc.enqueueTidbCluster,
		UpdateFunc: func(old, cur interface{}) {
			tcc.enqueueTidbCluster(cur)
		},
		DeleteFunc: tcc.enqueueTidbCluster,
	})
	tcc.tcLister = tcInformer.Lister()
	tcc.tcListerSynced = tcInformer.Informer().HasSynced

	setInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: tcc.addStatefulSet,
		UpdateFunc: func(old, cur interface{}) {
			tcc.updateStatefuSet(old, cur)
		},
		DeleteFunc: tcc.deleteStatefulSet,
	})
	tcc.setLister = setInformer.Lister()
	tcc.setListerSynced = setInformer.Informer().HasSynced

	return tcc
}

// Run runs the tidbcluster controller.
func (tcc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tcc.queue.ShutDown()

	glog.Infof("Starting tidbcluster controller")
	defer glog.Infof("Shutting down tidbcluster controller")

	if !cache.WaitForCacheSync(stopCh, tcc.tcListerSynced, tcc.setListerSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(tcc.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (tcc *Controller) worker() {
	for tcc.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (tcc *Controller) processNextWorkItem() bool {
	key, quit := tcc.queue.Get()
	if quit {
		return false
	}
	defer tcc.queue.Done(key)
	if err := tcc.sync(key.(string)); err != nil {
		utilruntime.HandleError(fmt.Errorf("Error syncing TidbCluster %v, requeuing: %v", key.(string), err))
		tcc.queue.AddRateLimited(key)
	} else {
		tcc.queue.Forget(key)
	}
	return true
}

// sync syncs the given tidbcluster.
func (tcc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(4).Infof("Finished syncing TidbCluster %q (%v)", key, time.Now().Sub(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	tc, err := tcc.tcLister.TidbClusters(ns).Get(name)
	if errors.IsNotFound(err) {
		glog.Infof("TidbCluster has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tcc.syncTidbCluster(tc.DeepCopy())
}

func (tcc *Controller) syncTidbCluster(tc *v1.TidbCluster) error {
	return tcc.control.UpdateTidbCluster(tc)
}

// enqueueTidbCluster enqueues the given tidbcluster in the work queue.
func (tcc *Controller) enqueueTidbCluster(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
		return
	}
	tcc.queue.Add(key)
}

// addStatefulSet adds the tidbcluster for the statefulset to the sync queue
func (tcc *Controller) addStatefulSet(obj interface{}) {
	set := obj.(*apps.StatefulSet)
	ns := set.GetNamespace()
	setName := set.GetName()

	if set.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new statefulset shows up in a state that
		// is already pending deletion. Prevent the statefulset from being a creation observation.
		tcc.deleteStatefulSet(set)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	tc := tcc.resolveControllerRef(ns, set)
	if tc == nil {
		return
	}
	glog.V(4).Infof("StatefuSet %s/%s created, TidbCluster: %s/%s", ns, setName, ns, tc.Name)
	tcc.enqueueTidbCluster(tc)
}

// updateStatefuSet adds the tidbcluster for the current and old statefulsets to the sync queue.
func (tcc *Controller) updateStatefuSet(old, cur interface{}) {
	curSet := cur.(*apps.StatefulSet)
	oldSet := old.(*apps.StatefulSet)
	ns := curSet.GetNamespace()
	setName := curSet.GetName()
	if curSet.ResourceVersion == oldSet.ResourceVersion {
		// Periodic resync will send update events for all known statefulsets.
		// Two different versions of the same statefulset will always have different RVs.
		return
	}

	// If it has a ControllerRef, that's all that matters.
	tc := tcc.resolveControllerRef(ns, curSet)
	if tc == nil {
		return
	}
	glog.V(4).Infof("StatefulSet %s/%s updated, %+v -> %+v.", ns, setName, oldSet.Spec, curSet.Spec)
	tcc.enqueueTidbCluster(tc)
}

// deleteStatefulSet enqueues the tidbcluster for the statefulset accounting for deletion tombstones.
func (tcc *Controller) deleteStatefulSet(obj interface{}) {
	set, ok := obj.(*apps.StatefulSet)
	ns := set.GetNamespace()
	setName := set.GetName()

	// When a delete is dropped, the relist will notice a statefuset in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		set, ok = tombstone.Obj.(*apps.StatefulSet)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a statefuset %+v", obj))
			return
		}
	}

	// If it has a ControllerRef, that's all that matters.
	tc := tcc.resolveControllerRef(ns, set)
	if tc == nil {
		return
	}
	glog.V(4).Infof("StatefulSet %s/%s deleted through %v.", ns, setName, utilruntime.GetCaller())
	tcc.enqueueTidbCluster(tc)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (tcc *Controller) resolveControllerRef(namespace string, set *apps.StatefulSet) *v1.TidbCluster {
	controllerRef := metav1.GetControllerOf(set)
	if controllerRef == nil {
		return nil
	}

	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != controllerKind.Kind {
		return nil
	}
	tc, err := tcc.tcLister.TidbClusters(namespace).Get(controllerRef.Name)
	if err != nil {
		return nil
	}
	if tc.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return tc
}
