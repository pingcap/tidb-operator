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

	"github.com/Masterminds/semver"
	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"

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

// Controller controls dmclusters.
type Controller struct {
	// kubernetes client interface
	kubeClient kubernetes.Interface
	// operator client interface
	cli versioned.Interface
	// control returns an interface capable of syncing a dm cluster.
	// Abstracted out for testing.
	control ControlInterface
	// dcLister is able to list/get dmclusters from a shared informer's store
	dcLister listers.DMClusterLister
	// dcListerSynced returns true if the dmclusters shared informer has synced at least once
	dcListerSynced cache.InformerSynced
	// setLister is able to list/get stateful sets from a shared informer's store
	setLister appslisters.StatefulSetLister
	// setListerSynced returns true if the statefulset shared informer has synced at least once
	setListerSynced cache.InformerSynced
	// dmclusters that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a dm controller.
func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	autoFailover bool,
) *Controller {
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{QPS: 1})
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidb-controller-manager"})

	dcInformer := informerFactory.Pingcap().V1alpha1().DMClusters()
	setInformer := kubeInformerFactory.Apps().V1().StatefulSets()
	svcInformer := kubeInformerFactory.Core().V1().Services()
	epsInformer := kubeInformerFactory.Core().V1().Endpoints()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	scInformer := kubeInformerFactory.Storage().V1().StorageClasses()
	//nodeInformer := kubeInformerFactory.Core().V1().Nodes()
	//secretInformer := kubeInformerFactory.Core().V1().Secrets()

	dcControl := controller.NewRealDMClusterControl(cli, dcInformer.Lister(), recorder)
	masterControl := dmapi.NewDefaultMasterControl(kubeCli)
	setControl := controller.NewRealStatefuSetControl(kubeCli, setInformer.Lister(), recorder)
	svcControl := controller.NewRealServiceControl(kubeCli, svcInformer.Lister(), recorder)
	pvControl := controller.NewRealPVControl(kubeCli, pvcInformer.Lister(), pvInformer.Lister(), recorder)
	pvcControl := controller.NewRealPVCControl(kubeCli, recorder, pvcInformer.Lister())
	//podControl := controller.NewRealPodControl(kubeCli, nil, podInformer.Lister(), recorder)
	typedControl := controller.NewTypedControl(controller.NewRealGenericControl(genericCli, recorder))
	masterScaler := mm.NewMasterScaler(masterControl, pvcInformer.Lister(), pvcControl)
	masterUpgrader := mm.NewMasterUpgrader(masterControl, podInformer.Lister())

	dcc := &Controller{
		kubeClient: kubeCli,
		cli:        cli,
		control: NewDefaultDMClusterControl(
			dcControl,
			mm.NewMasterMemberManager(
				masterControl,
				setControl,
				svcControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				epsInformer.Lister(),
				pvcInformer.Lister(),
				masterScaler,
				masterUpgrader,
			),
			mm.NewWorkerMemberManager(
				masterControl,
				setControl,
				svcControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
			),
			//meta.NewReclaimPolicyManager(
			//	pvcInformer.Lister(),
			//	pvInformer.Lister(),
			//	pvControl,
			//),
			//meta.NewMetaManager(
			//	pvcInformer.Lister(),
			//	pvcControl,
			//	pvInformer.Lister(),
			//	pvControl,
			//	podInformer.Lister(),
			//	podControl,
			//),
			//mm.NewOrphanPodsCleaner(
			//	podInformer.Lister(),
			//	podControl,
			//	pvcInformer.Lister(),
			//	kubeCli,
			//),
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
			//mm.NewDMClusterStatusManager(kubeCli, cli, scalerInformer.Lister(), tikvGroupInformer.Lister()),
			//podRestarter,
			&dmClusterConditionUpdater{},
			recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"dmcluster",
		),
	}

	dcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: dcc.enqueueDMCluster,
		UpdateFunc: func(old, cur interface{}) {
			dcc.enqueueDMCluster(cur)
		},
		DeleteFunc: dcc.enqueueDMCluster,
	})
	dcc.dcLister = dcInformer.Lister()
	dcc.dcListerSynced = dcInformer.Informer().HasSynced

	setInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: dcc.addStatefulSet,
		UpdateFunc: func(old, cur interface{}) {
			dcc.updateStatefuSet(old, cur)
		},
		DeleteFunc: dcc.deleteStatefulSet,
	})
	dcc.setLister = setInformer.Lister()
	dcc.setListerSynced = setInformer.Informer().HasSynced

	return dcc
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
	dc, err := dcc.dcLister.DMClusters(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("DMCluster has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}
	clusterVersionLT2, err := clusterVersionLessThan2(dc.MasterVersion())
	if err != nil {
		klog.V(4).Infof("cluster version: %s is not semantic versioning compatible", dc.MasterVersion())
	} else if clusterVersionLT2 {
		klog.Errorf("dm version %s not supported, only support to deploy dm from v2.0", dc.MasterVersion())
		return nil
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

// updateStatefuSet adds the dmcluster for the current and old statefulsets to the sync queue.
func (dcc *Controller) updateStatefuSet(old, cur interface{}) {
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
	dc, err := dcc.dcLister.DMClusters(namespace).Get(controllerRef.Name)
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

func clusterVersionLessThan2(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return true, err
	}

	return v.Major() < 2, nil
}
