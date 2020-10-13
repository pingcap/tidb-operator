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

package dmcluster

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"

	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// Controller controls dmclusters.
type Controller struct {
	deps *controller.Dependencies
	// Abstracted out for testing.
	control ControlInterface
	// dmclusters that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a dm controller.
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps: deps,
		control: NewDefaultDMClusterControl(
			deps.DMClusterControl,
			mm.NewMasterMemberManager(deps, mm.NewMasterScaler(deps), mm.NewMasterUpgrader(deps), mm.NewMasterFailover(deps)),
			mm.NewWorkerMemberManager(deps, mm.NewWorkerScaler(deps), mm.NewWorkerFailover(deps)),
			meta.NewReclaimPolicyManager(deps),
			mm.NewOrphanPodsCleaner(deps),
			mm.NewRealPVCCleaner(deps),
			mm.NewPVCResizer(deps),
			&dmClusterConditionUpdater{},
			deps.Recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"dmcluster",
		),
	}

	dmClusterInformer := deps.InformerFactory.Pingcap().V1alpha1().DMClusters()
	statefulsetInformer := deps.KubeInformerFactory.Apps().V1().StatefulSets()
	dmClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueDMCluster,
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueDMCluster(cur)
		},
		DeleteFunc: c.enqueueDMCluster,
	})
	statefulsetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.addStatefulSet,
		UpdateFunc: func(old, cur interface{}) {
			c.updateStatefulSet(old, cur)
		},
		DeleteFunc: c.deleteStatefulSet,
	})
	return c
}

// Run runs the dmcluster controller.
func (dcc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer dcc.queue.ShutDown()

	klog.Info("Starting dmcluster controller")
	defer klog.Info("Shutting down dmcluster controller")

	for i := 0; i < workers; i++ {
		go wait.Until(dcc.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (dcc *Controller) worker() {
	for dcc.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (dcc *Controller) processNextWorkItem() bool {
	key, quit := dcc.queue.Get()
	if quit {
		return false
	}
	defer dcc.queue.Done(key)
	if err := dcc.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("DMCluster: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("DMCluster: %v, sync failed %v, requeuing", key.(string), err))
		}
		dcc.queue.AddRateLimited(key)
	} else {
		dcc.queue.Forget(key)
	}
	return true
}

// sync syncs the given dmcluster.
func (dcc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing DMCluster %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	dc, err := dcc.deps.DMClusterLister.DMClusters(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("DMCluster has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return dcc.syncDMCluster(dc.DeepCopy())
}

func (dcc *Controller) syncDMCluster(dc *v1alpha1.DMCluster) error {
	return dcc.control.UpdateDMCluster(dc)
}

// enqueueDMCluster enqueues the given dmcluster in the work queue.
func (dcc *Controller) enqueueDMCluster(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
		return
	}
	dcc.queue.Add(key)
}

// addStatefulSet adds the dmcluster for the statefulset to the sync queue
func (dcc *Controller) addStatefulSet(obj interface{}) {
	set := obj.(*apps.StatefulSet)
	ns := set.GetNamespace()
	setName := set.GetName()

	if set.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new statefulset shows up in a state that
		// is already pending deletion. Prevent the statefulset from being a creation observation.
		dcc.deleteStatefulSet(set)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	dc := dcc.resolveDMClusterFromSet(ns, set)
	if dc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s created, DMCluster: %s/%s", ns, setName, ns, dc.Name)
	dcc.enqueueDMCluster(dc)
}

// updateStatefulSet adds the dmcluster for the current and old statefulsets to the sync queue.
func (dcc *Controller) updateStatefulSet(old, cur interface{}) {
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
	dc := dcc.resolveDMClusterFromSet(ns, curSet)
	if dc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s updated, DMCluster: %s/%s", ns, setName, ns, dc.Name)
	dcc.enqueueDMCluster(dc)
}

// deleteStatefulSet enqueues the dmcluster for the statefulset accounting for deletion tombstones.
func (dcc *Controller) deleteStatefulSet(obj interface{}) {
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

	// If it has a DMCluster, that's all that matters.
	dc := dcc.resolveDMClusterFromSet(ns, set)
	if dc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s deleted through %v.", ns, setName, utilruntime.GetCaller())
	dcc.enqueueDMCluster(dc)
}

// resolveDMClusterFromSet returns the DMCluster by a StatefulSet,
// or nil if the StatefulSet could not be resolved to a matching DMCluster
// of the correct Kind.
func (dcc *Controller) resolveDMClusterFromSet(namespace string, set *apps.StatefulSet) *v1alpha1.DMCluster {
	controllerRef := metav1.GetControllerOf(set)
	if controllerRef == nil {
		return nil
	}

	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != controller.DMControllerKind.Kind {
		return nil
	}
	dc, err := dcc.deps.DMClusterLister.DMClusters(namespace).Get(controllerRef.Name)
	if err != nil {
		return nil
	}
	if dc.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return dc
}
