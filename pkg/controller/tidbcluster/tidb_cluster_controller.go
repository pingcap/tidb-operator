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

// Controller controls tidbclusters.
type Controller struct {
	deps *controller.Dependencies
	// control returns an interface capable of syncing a tidb cluster.
	// Abstracted out for testing.
	control ControlInterface
	// tidbclusters that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a tidbcluster controller.
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		control: NewDefaultTidbClusterControl(
			deps.TiDBClusterControl,
			mm.NewPDMemberManager(deps, mm.NewPDScaler(deps), mm.NewPDUpgrader(deps), mm.NewPDFailover(deps)),
			mm.NewTiKVMemberManager(deps, mm.NewTiKVFailover(deps), mm.NewTiKVScaler(deps), mm.NewTiKVUpgrader(deps)),
			mm.NewTiDBMemberManager(deps, mm.NewTiDBUpgrader(deps), mm.NewTiDBFailover(deps)),
			meta.NewReclaimPolicyManager(deps),
			meta.NewMetaManager(deps),
			mm.NewOrphanPodsCleaner(deps),
			mm.NewRealPVCCleaner(deps),
			mm.NewPVCResizer(deps),
			mm.NewPumpMemberManager(deps),
			mm.NewTiFlashMemberManager(deps, mm.NewTiFlashFailover(deps), mm.NewTiFlashScaler(deps), mm.NewTiFlashUpgrader(deps)),
			mm.NewTiCDCMemberManager(deps),
			mm.NewTidbDiscoveryManager(deps),
			mm.NewTidbClusterStatusManager(deps),
			&tidbClusterConditionUpdater{},
			deps.Recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "tidbcluster"),
	}

	deps.TiDBClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueTidbCluster,
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueTidbCluster(cur)
		},
		DeleteFunc: c.enqueueTidbCluster,
	})
	deps.StatefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.addStatefulSet,
		UpdateFunc: func(old, cur interface{}) {
			c.updateStatefulSet(old, cur)
		},
		DeleteFunc: c.deleteStatefulSet,
	})

	return c
}

// Run runs the tidbcluster controller.
func (tcc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tcc.queue.ShutDown()

	klog.Info("Starting tidbcluster controller")
	defer klog.Info("Shutting down tidbcluster controller")

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
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("TidbCluster: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TidbCluster: %v, sync failed %v, requeuing", key.(string), err))
		}
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
		klog.V(4).Infof("Finished syncing TidbCluster %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	tc, err := tcc.deps.TiDBClusterLister.TidbClusters(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TidbCluster has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tcc.syncTidbCluster(tc.DeepCopy())
}

func (tcc *Controller) syncTidbCluster(tc *v1alpha1.TidbCluster) error {
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
	tc := tcc.resolveTidbClusterFromSet(ns, set)
	if tc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s created, TidbCluster: %s/%s", ns, setName, ns, tc.Name)
	tcc.enqueueTidbCluster(tc)
}

// updateStatefulSet adds the tidbcluster for the current and old statefulsets to the sync queue.
func (tcc *Controller) updateStatefulSet(old, cur interface{}) {
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
	tc := tcc.resolveTidbClusterFromSet(ns, curSet)
	if tc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s updated, TidbCluster: %s/%s", ns, setName, ns, tc.Name)
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

	// If it has a TidbCluster, that's all that matters.
	tc := tcc.resolveTidbClusterFromSet(ns, set)
	if tc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s deleted through %v.", ns, setName, utilruntime.GetCaller())
	tcc.enqueueTidbCluster(tc)
}

// resolveTidbClusterFromSet returns the TidbCluster by a StatefulSet,
// or nil if the StatefulSet could not be resolved to a matching TidbCluster
// of the correct Kind.
func (tcc *Controller) resolveTidbClusterFromSet(namespace string, set *apps.StatefulSet) *v1alpha1.TidbCluster {
	controllerRef := metav1.GetControllerOf(set)
	if controllerRef == nil {
		return nil
	}

	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != controller.ControllerKind.Kind {
		return nil
	}
	tc, err := tcc.deps.TiDBClusterLister.TidbClusters(namespace).Get(controllerRef.Name)
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
