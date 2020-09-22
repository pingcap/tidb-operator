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

	"github.com/pingcap/tidb-operator/pkg/deps"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Controller controls tidbclusters.
type Controller struct {
	deps *deps.Dependencies
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
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
) *Controller {
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{QPS: 1})
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidb-controller-manager"})

	tcInformer := informerFactory.Pingcap().V1alpha1().TidbClusters()
	setInformer := kubeInformerFactory.Apps().V1().StatefulSets()
	svcInformer := kubeInformerFactory.Core().V1().Services()
	epsInformer := kubeInformerFactory.Core().V1().Endpoints()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()
	scInformer := kubeInformerFactory.Storage().V1().StorageClasses()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	scalerInformer := informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers()

	tcControl := controller.NewRealTidbClusterControl(cli, tcInformer.Lister(), recorder)
	pdControl := pdapi.NewDefaultPDControl(kubeCli)
	cdcControl := controller.NewDefaultTiCDCControl(kubeCli)
	tidbControl := controller.NewDefaultTiDBControl(kubeCli)
	cmControl := controller.NewRealConfigMapControl(kubeCli, recorder)
	setControl := controller.NewRealStatefuSetControl(kubeCli, setInformer.Lister(), recorder)
	svcControl := controller.NewRealServiceControl(kubeCli, svcInformer.Lister(), recorder)
	pvControl := controller.NewRealPVControl(kubeCli, pvcInformer.Lister(), pvInformer.Lister(), recorder)
	pvcControl := controller.NewRealPVCControl(kubeCli, recorder, pvcInformer.Lister())
	podControl := controller.NewRealPodControl(kubeCli, pdControl, podInformer.Lister(), recorder)
	typedControl := controller.NewTypedControl(controller.NewRealGenericControl(genericCli, recorder))
	pdScaler := mm.NewPDScaler(pdControl, pvcInformer.Lister(), pvcControl)
	tikvScaler := mm.NewTiKVScaler(pdControl, pvcInformer.Lister(), pvcControl, podInformer.Lister())
	tiflashScaler := mm.NewTiFlashScaler(pdControl, pvcInformer.Lister(), pvcControl, podInformer.Lister())
	pdFailover := mm.NewPDFailover(cli, pdControl, pdFailoverPeriod, podInformer.Lister(), podControl, pvcInformer.Lister(), pvcControl, pvInformer.Lister(), recorder)
	tikvFailover := mm.NewTiKVFailover(tikvFailoverPeriod, recorder)
	tidbFailover := mm.NewTiDBFailover(tidbFailoverPeriod, recorder, podInformer.Lister())
	tiflashFailover := mm.NewTiFlashFailover(tiflashFailoverPeriod, recorder)
	pdUpgrader := mm.NewPDUpgrader(pdControl, podControl, podInformer.Lister())
	tikvUpgrader := mm.NewTiKVUpgrader(pdControl, podControl, podInformer.Lister())
	tiflashUpgrader := mm.NewTiFlashUpgrader(pdControl, podControl, podInformer.Lister())
	tidbUpgrader := mm.NewTiDBUpgrader(tidbControl, podInformer.Lister())
	podRestarter := mm.NewPodRestarter(kubeCli, podInformer.Lister())

	tcc := &Controller{
		control: NewDefaultTidbClusterControl(
			tcControl,
			mm.NewPDMemberManager(
				pdControl,
				setControl,
				svcControl,
				podControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				epsInformer.Lister(),
				pvcInformer.Lister(),
				pdScaler,
				pdUpgrader,
				autoFailover,
				pdFailover,
			),
			mm.NewTiKVMemberManager(
				pdControl,
				setControl,
				svcControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				nodeInformer.Lister(),
				autoFailover,
				tikvFailover,
				tikvScaler,
				tikvUpgrader,
				recorder,
			),
			mm.NewTiDBMemberManager(
				setControl,
				svcControl,
				tidbControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				secretInformer.Lister(),
				tidbUpgrader,
				autoFailover,
				tidbFailover,
			),
			meta.NewReclaimPolicyManager(
				pvcInformer.Lister(),
				pvInformer.Lister(),
				pvControl,
			),
			meta.NewMetaManager(
				pvcInformer.Lister(),
				pvcControl,
				pvInformer.Lister(),
				pvControl,
				podInformer.Lister(),
				podControl,
			),
			mm.NewOrphanPodsCleaner(
				podInformer.Lister(),
				podControl,
				pvcInformer.Lister(),
				kubeCli,
			),
			mm.NewRealPVCCleaner(
				kubeCli,
				podInformer.Lister(),
				pvcControl,
				pvcInformer.Lister(),
				pvInformer.Lister(),
				pvControl,
			),
			mm.NewPVCResizer(
				kubeCli,
				pvcInformer,
				scInformer,
			),
			mm.NewPumpMemberManager(
				setControl,
				svcControl,
				typedControl,
				cmControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
			),
			mm.NewTiFlashMemberManager(
				pdControl,
				setControl,
				svcControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				nodeInformer.Lister(),
				autoFailover,
				tiflashFailover,
				tiflashScaler,
				tiflashUpgrader,
			),
			mm.NewTiCDCMemberManager(
				pdControl,
				cdcControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				svcControl,
				setControl,
			),
			mm.NewTidbDiscoveryManager(typedControl),
			mm.NewTidbClusterStatusManager(kubeCli, cli, scalerInformer.Lister()),
			podRestarter,
			&tidbClusterConditionUpdater{},
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
	tc, err := tcc.tcLister.TidbClusters(ns).Get(name)
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
