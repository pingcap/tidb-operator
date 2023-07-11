// Copyright 2021 PingCAP, Inc.
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
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	RequeueInterval = time.Minute
)

// PodController control pods of tidb cluster.
// see docs/design-proposals/2021-11-24-graceful-reschedule-tikv-pod.md
type PodController struct {
	deps  *controller.Dependencies
	queue workqueue.RateLimitingInterface

	podStatsMu sync.Mutex
	podStats   map[string]stat

	// only set in test
	testPDClient                  pdapi.PDClient
	recheckLeaderCountDuration    time.Duration
	recheckClusterStableDuration  time.Duration
	recheckStoreTombstoneDuration time.Duration
}

// NewPodController create a PodController.
func NewPodController(deps *controller.Dependencies) *PodController {
	c := &PodController{
		deps: deps,
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"tidbcluster pods",
		),
		podStats:                      make(map[string]stat),
		recheckLeaderCountDuration:    time.Second * 15,
		recheckClusterStableDuration:  time.Minute * 1,
		recheckStoreTombstoneDuration: time.Second * 15,
	}

	podsInformer := deps.KubeInformerFactory.Core().V1().Pods()
	podsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueuePod,
		UpdateFunc: func(old, cur interface{}) {
			c.enqueuePod(cur)
		},
	})

	return c
}

type stat struct {
	observeAnnotationCounts int
	finishAnnotationCounts  int
}

func (c *PodController) getPodStat(pod *corev1.Pod) stat {
	c.podStatsMu.Lock()
	defer c.podStatsMu.Unlock()

	stat := c.podStats[pod.Namespace+pod.Name]
	return stat
}

func (c *PodController) setPodStat(pod *corev1.Pod, stat stat) {
	c.podStatsMu.Lock()
	defer c.podStatsMu.Unlock()

	c.podStats[pod.Namespace+pod.Name] = stat
}

// enqueueTidbCluster enqueues the given pod in the work queue.
func (c *PodController) enqueuePod(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}

// Name returns the name of the PodController.
func (c *PodController) Name() string {
	return "tidbcluster-pod"
}

// Run the controller.
func (c *PodController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting tidbcluster pod controller")
	defer klog.Info("Shutting down tidbcluster pod controller")

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (c *PodController) worker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (c *PodController) processNextWorkItem() bool {
	metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(1)
	defer metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(-1)

	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	result, err := c.sync(key.(string))
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("TidbCluster pod: %v, sync failed %v, requeuing", key.(string), err))
		c.queue.AddRateLimited(key)
	} else {
		if result.RequeueAfter > 0 {
			c.queue.AddAfter(key, result.RequeueAfter)
		} else if result.Requeue {
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
		}
	}
	return true
}

func (c *PodController) sync(key string) (reconcile.Result, error) {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return reconcile.Result{}, err
	}

	pod, err := c.deps.PodLister.Pods(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("Pod %v has been deleted", key)
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	pod = pod.DeepCopy()

	// labels of TiKV:
	// app.kubernetes.io/component=tikv,app.kubernetes.io/instance=db1369682775200135879,
	// app.kubernetes.io/managed-by=tidb-operator ...
	managedBy := pod.Labels[label.ManagedByLabelKey]
	if managedBy != label.TiDBOperator {
		return reconcile.Result{}, nil
	}

	tcName := pod.Labels[label.InstanceLabelKey]
	if tcName == "" {
		return reconcile.Result{}, nil
	}

	tc, err := c.deps.TiDBClusterLister.TidbClusters(ns).Get(tcName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(4).Infof("TidbCluster %q is not found, skip sync the Pod %s", ns+"/"+tcName, name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, perrors.Annotatef(err, "failed to get TidbCluster %q", ns+"/"+tcName)
	}
	tc = tc.DeepCopy()

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing TidbCluster pod %q (%v)", key, duration)
	}()

	component := pod.Labels[label.ComponentLabelKey]
	ctx := context.Background()
	switch component {
	case label.PDLabelVal:
		return c.syncPDPod(ctx, pod, tc)
	case label.TiKVLabelVal:
		return c.syncTiKVPod(ctx, pod, tc)
	case label.TiDBLabelVal:
		return c.syncTiDBPod(ctx, pod, tc)
	default:
		return reconcile.Result{}, nil
	}
}

func (c *PodController) getPDClient(tc *v1alpha1.TidbCluster) pdapi.PDClient {
	if c.testPDClient != nil {
		return c.testPDClient
	}

	pdClient := controller.GetPDClient(c.deps.PDControl, tc)
	return pdClient
}

func (c *PodController) syncPDPod(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	value, ok := needPDLeaderTransfer(pod)
	if !ok {
		// No need to transfer leader
		return reconcile.Result{}, nil
	}

	switch value {
	case v1alpha1.TransferLeaderValueNone:
	case v1alpha1.TransferLeaderValueDeletePod:
	default:
		klog.Warningf("Ignore unknown value %q of annotation %q for Pod %s/%s", value, v1alpha1.PDLeaderTransferAnnKey, pod.Namespace, pod.Name)
		return reconcile.Result{}, nil
	}

	if c.isEvictLeaderExpired(pod, v1alpha1.PDLeaderTransferExpirationTimeAnnKey) {
		return reconcile.Result{}, c.cleanupLeaderEvictionAnnotations(pod, tc, []string{v1alpha1.PDLeaderTransferAnnKey, v1alpha1.PDLeaderTransferExpirationTimeAnnKey})
	}

	// Check if there's any ongoing updates in PD.
	if tc.Status.PD.Phase != v1alpha1.NormalPhase {
		return reconcile.Result{RequeueAfter: RequeueInterval}, nil
	}

	pdClient := c.getPDClient(tc)

	if unstableReason := pdapi.IsPDStable(pdClient); unstableReason != "" {
		klog.Infof("PD cluster in %s is unstable: %s", tc.Name, unstableReason)
		return reconcile.Result{RequeueAfter: c.recheckClusterStableDuration}, nil
	}

	// Transfer leader to other peer if necessary.
	var err error
	pdName := getPdName(pod, tc)
	if tc.Status.PD.Leader.Name == pod.Name || tc.Status.PD.Leader.Name == pdName {
		err = transferPDLeader(tc, pdClient)
		if err != nil {
			return reconcile.Result{}, nil
		}
	}

	// Delete pod after leader transfer if configured.
	if value == v1alpha1.TransferLeaderValueDeletePod {
		err = c.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to delete pod %q", pod.Name)
		}
	}

	return reconcile.Result{}, nil
}

func (c *PodController) syncTiKVPod(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	result, err := c.syncTiKVPodForEviction(ctx, pod, tc)
	if err != nil || result.Requeue || result.RequeueAfter > 0 {
		return result, err
	}
	return c.syncTiKVPodForReplaceDisk(ctx, pod, tc)
}

func (c *PodController) syncTiKVPodForEviction(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	key, value, ok := needEvictLeader(pod)

	if ok {
		switch value {
		case v1alpha1.EvictLeaderValueNone:
		case v1alpha1.EvictLeaderValueDeletePod:
		default:
			klog.Warningf("Ignore unknown value %q of annotation %q for Pod %s/%s", value, key, pod.Namespace, pod.Name)
			return reconcile.Result{}, nil
		}
	}

	if ok {
		if c.isEvictLeaderExpired(pod, v1alpha1.TiKVEvictLeaderExpirationTimeAnnKey) {
			return reconcile.Result{}, c.cleanupLeaderEvictionAnnotations(pod, tc, append(v1alpha1.EvictLeaderAnnKeys, v1alpha1.TiKVEvictLeaderExpirationTimeAnnKey))
		}

		evictStatus := &v1alpha1.EvictLeaderStatus{
			PodCreateTime: pod.CreationTimestamp,
			BeginTime:     metav1.Now(),
			Value:         value,
		}
		nowStatus := tc.Status.TiKV.EvictLeader[pod.Name]
		if nowStatus != nil && !nowStatus.BeginTime.IsZero() {
			evictStatus.BeginTime = nowStatus.BeginTime
		}

		// update status of eviction
		if nowStatus == nil || *nowStatus != *evictStatus {
			if tc.Status.TiKV.EvictLeader == nil {
				tc.Status.TiKV.EvictLeader = make(map[string]*v1alpha1.EvictLeaderStatus)
			}
			tc.Status.TiKV.EvictLeader[pod.Name] = evictStatus
			var err error
			key := fmt.Sprintf("%s/%s", tc.Namespace, tc.Name)
			tc, err = c.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(ctx, tc, metav1.UpdateOptions{})
			if err != nil {
				return reconcile.Result{}, perrors.Annotatef(err, "failed to update tc %q status", key)
			}

			stat := c.getPodStat(pod)
			stat.observeAnnotationCounts++
			c.setPodStat(pod, stat)
		}

		// begin to evict leader
		pdClient := c.getPDClient(tc)
		storeID, err := member.TiKVStoreIDFromStatus(tc, pod.Name)
		if err != nil {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to get tikv store id from status for pod %s/%s", pod.Namespace, pod.Name)
		}
		if unstableReason := pdapi.IsTiKVStable(pdClient); unstableReason != "" {
			klog.Infof("Cluster %s is unstable: %s", tc.Name, unstableReason)
			return reconcile.Result{RequeueAfter: c.recheckClusterStableDuration}, nil
		}
		err = pdClient.BeginEvictLeader(storeID)
		if err != nil {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to evict leader for store %d (Pod %s/%s)", storeID, pod.Namespace, pod.Name)
		}

		// delete pod after eviction finished if needed
		if value == v1alpha1.EvictLeaderValueDeletePod {
			tlsEnabled := tc.IsTLSClusterEnabled()
			kvClient := c.deps.TiKVControl.GetTiKVPodClient(tc.Namespace, tc.Name, pod.Name, tlsEnabled)
			leaderCount, err := kvClient.GetLeaderCount()
			if err != nil {
				return reconcile.Result{}, perrors.Annotatef(err, "failed to get leader count for pod %s/%s", pod.Namespace, pod.Name)
			}

			klog.Infof("Region leader count is %d for Pod %s/%s", leaderCount, pod.Namespace, pod.Name)

			if leaderCount == 0 {
				err = c.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					return reconcile.Result{}, perrors.Annotatef(err, "failed to delete pod %q", pod.Name)
				}
			} else {
				// re-check leader count next time
				return reconcile.Result{RequeueAfter: c.recheckLeaderCountDuration}, nil
			}
		}
	} else {
		// 1. delete evict-leader scheduler
		// 2. delete pod from tc.Status.TiKV.EvictLeader and update it to api-server
		endEvict := func() error {
			pdClient := c.getPDClient(tc)
			storeID, err := member.TiKVStoreIDFromStatus(tc, pod.Name)
			if err != nil {
				return perrors.Annotatef(err, "failed to get tikv store id from status for pod %s/%s", pod.Namespace, pod.Name)
			}

			err = pdClient.EndEvictLeader(storeID)
			if err != nil {
				return perrors.Annotatef(err, "failed to remove evict leader scheduler for store %d, pod %s/%s", storeID, pod.Namespace, pod.Name)
			}

			err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				delete(tc.Status.TiKV.EvictLeader, pod.Name)
				_, updateErr := c.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(ctx, tc, metav1.UpdateOptions{})
				if updateErr == nil {
					return nil
				}

				if updated, err := c.deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name); err == nil {
					// // make a copy so we don't mutate the shared cache
					tc = updated.DeepCopy()
				} else {
					utilruntime.HandleError(fmt.Errorf("error getting updated tc %s/%s from lister: %v", tc.Namespace, tc.Name, err))
				}

				return updateErr
			})
			if err != nil {
				return perrors.Annotatef(err, "failed to update status for tc %s/%s", tc.Namespace, tc.Name)
			}

			return nil
		}

		evictStatus := tc.Status.TiKV.EvictLeader[pod.Name]
		if evictStatus != nil {
			if evictStatus.Value == v1alpha1.EvictLeaderValueDeletePod {
				if podutil.IsPodReady(pod) {
					err := endEvict()
					if err != nil {
						return reconcile.Result{}, err
					}
				}
			} else if evictStatus.Value == v1alpha1.EvictLeaderValueNone {
				err := endEvict()
				if err != nil {
					return reconcile.Result{}, err
				}
			}
			stat := c.getPodStat(pod)
			stat.finishAnnotationCounts++
			c.setPodStat(pod, stat)
		}
	}

	return reconcile.Result{}, nil
}
func (c *PodController) syncTiKVPodForReplaceDisk(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	value, exist := pod.Annotations[v1alpha1.ReplaceDiskAnnKey]
	if exist {
		if value != v1alpha1.ReplaceDiskValueTrue {
			klog.Warningf("Ignore unknown value %q of annotation %q for Pod %s/%s", value, v1alpha1.ReplaceDiskAnnKey, pod.Namespace, pod.Name)
			return reconcile.Result{}, nil
		}
		pdClient := c.getPDClient(tc)
		storeID, err := member.TiKVStoreIDFromStatus(tc, pod.Name)
		if err != nil {
			storeIDStr, exist := pod.Labels[label.StoreIDLabelKey]
			if !exist {
				return reconcile.Result{}, perrors.Annotatef(err, "failed to get tikv store id from status or label for pod %s/%s", pod.Namespace, pod.Name)
			}
			storeID, err = strconv.ParseUint(storeIDStr, 10, 64)
			if err != nil {
				return reconcile.Result{}, perrors.Annotatef(err, "Could not parse storeId (%s) from label for pod %s/%s", storeIDStr, pod.Namespace, pod.Name)
			}
		}
		if pod.Labels[label.StoreIDLabelKey] == "" {
			return reconcile.Result{Requeue: true}, fmt.Errorf("StoreID not yet updated on pod label")
		}
		storeInfo, err := pdClient.GetStore(storeID)
		if err != nil {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to get tikv store info from pd for storeid %d pod %s/%s", storeID, pod.Namespace, pod.Name)
		}
		if storeInfo.Store.StateName == v1alpha1.TiKVStateUp {
			if !tc.TiKVAllStoresReady() {
				return reconcile.Result{Requeue: true}, fmt.Errorf("Not all TIKV stores ready before replace")
			}
			// 1. Delete store
			klog.Infof("storeid %d is Up, deleting due to replace disk annotation.", storeID)
			pdClient.DeleteStore(storeID)
			return reconcile.Result{RequeueAfter: c.recheckStoreTombstoneDuration}, nil
		} else if storeInfo.Store.StateName == v1alpha1.TiKVStateOffline {
			// 2. Wait for Tombstone
			return reconcile.Result{RequeueAfter: c.recheckStoreTombstoneDuration}, fmt.Errorf("StoreID %d not yet Tombstone", storeID)
		} else if storeInfo.Store.StateName == v1alpha1.TiKVStateTombstone {
			// 3. Delete PVCs
			var pvcs []*corev1.PersistentVolumeClaim = nil
			for _, vol := range pod.Spec.Volumes {
				if vol.PersistentVolumeClaim != nil {
					pvc, err := c.deps.PVCLister.PersistentVolumeClaims(pod.Namespace).Get(vol.PersistentVolumeClaim.ClaimName)
					if err != nil {
						klog.Warningf("Vol %s, claim %s did not find pvc: %s", vol.Name, vol.PersistentVolumeClaim.ClaimName, err)
						continue // Skip missing (deleted?) pvc.
					}
					if pvc.DeletionTimestamp != nil {
						klog.Warningf("pvc %s already marked for deletion %s", pvc.Name, pvc.DeletionTimestamp)
						continue // already marked for deletion.
					}
					pvcs = append(pvcs, pvc)
				}
			}
			if pvcs != nil {
				// Delete any PVCs not yet deleted.
				var anyErr error = nil
				for _, pvc := range pvcs {
					err = c.deps.PVCControl.DeletePVC(tc, pvc)
					klog.Infof("Deleting pvc: %s", pvc.Name)
					if err != nil {
						anyErr = err
					}
				}
				// Requeue next iteration, to verify pvc marked for deletion before moving on to pod deletion.
				return reconcile.Result{Requeue: true}, anyErr
			}
			// 4. Delete pod
			if pod.DeletionTimestamp != nil {
				// Already marked for deletion, wait for it to delete.
				return reconcile.Result{RequeueAfter: RequeueInterval}, nil
			}
			klog.Infof("Deleting pod: %s", pod.Name)
			err := c.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return reconcile.Result{}, perrors.Annotatef(err, "failed to delete pod %s", pod.Name)
			}
		} else {
			return reconcile.Result{}, perrors.Annotatef(err, "Cannot replace disk when store in state: %s for storeid %d pod %s/%s", storeInfo.Store.StateName, storeID, pod.Namespace, pod.Name)
		}
	}
	return reconcile.Result{}, nil
}

func (c *PodController) syncTiDBPod(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	result, err := c.syncTiDBPodForGracefulShutdown(ctx, pod, tc)
	if err != nil || result.Requeue || result.RequeueAfter > 0 {
		return result, err
	}
	return c.syncTiDBPodForReplaceDisk(ctx, pod, tc)
}

func (c *PodController) syncTiDBPodForGracefulShutdown(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	value := needDeleteTiDBPod(pod)
	if value == "" {
		// No need to delete tidb pod
		return reconcile.Result{}, nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.PDUpgrading() || tc.PDScaling() || tc.TiKVUpgrading() || tc.TiKVScaling() || tc.TiDBUpgrading() || tc.TiDBScaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %s, "+
			"tikv status is %s, tiflash status is %s, pump status is %s, "+
			"tidb status is %s, can not delete tidb",
			ns, tcName,
			tc.Status.PD.Phase, tc.Status.TiKV.Phase, tc.Status.TiFlash.Phase,
			tc.Status.Pump.Phase, tc.Status.TiDB.Phase)
		return reconcile.Result{RequeueAfter: RequeueInterval}, nil
	}

	switch value {
	case v1alpha1.TiDBPodDeletionValueNone:
	case v1alpha1.TiDBPodDeletionDeletePod:
	default:
		klog.Warningf("Ignore unknown value %q of annotation %q for Pod %s/%s", value, v1alpha1.TiDBGracefulShutdownAnnKey, pod.Namespace, pod.Name)
		return reconcile.Result{}, nil
	}

	if value == v1alpha1.TiDBPodDeletionDeletePod {
		err := c.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to delete pod %q", pod.Name)
		}
	}

	return reconcile.Result{}, nil
}

func (c *PodController) syncTiDBPodForReplaceDisk(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	value, exist := pod.Annotations[v1alpha1.ReplaceDiskAnnKey]
	if exist {
		if value != v1alpha1.ReplaceDiskValueTrue {
			klog.Warningf("Ignore unknown value %q of annotation %q for Pod %s/%s", value, v1alpha1.ReplaceDiskAnnKey, pod.Namespace, pod.Name)
			return reconcile.Result{}, nil
		}
		// Verify okay to delete tidb pod now.
		ns := tc.GetNamespace()
		tcName := tc.GetName()

		if tc.PDUpgrading() || tc.PDScaling() || tc.TiKVUpgrading() || tc.TiKVScaling() || tc.TiDBScaling() {
			klog.Infof("TidbCluster: [%s/%s]'s pd status is %s, "+
				"tikv status is %s, tiflash status is %s, pump status is %s, "+
				"tidb status is %s, can not replace tidb disk",
				ns, tcName,
				tc.Status.PD.Phase, tc.Status.TiKV.Phase, tc.Status.TiFlash.Phase,
				tc.Status.Pump.Phase, tc.Status.TiDB.Phase)
			return reconcile.Result{RequeueAfter: RequeueInterval}, nil
		}
		if !tc.TiDBAllMembersReady() {
			return reconcile.Result{Requeue: true}, nil
		}
		// 1. Delete PVCs
		var pvcs []*corev1.PersistentVolumeClaim = nil
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvc, err := c.deps.PVCLister.PersistentVolumeClaims(pod.Namespace).Get(vol.PersistentVolumeClaim.ClaimName)
				if err != nil {
					klog.Warningf("Vol %s, claim %s did not find pvc: %s", vol.Name, vol.PersistentVolumeClaim.ClaimName, err)
					continue // Skip missing (deleted?) pvc.
				}
				if pvc.DeletionTimestamp != nil {
					klog.Warningf("pvc %s already marked for deletion %s", pvc.Name, pvc.DeletionTimestamp)
					continue // already marked for deletion.
				}
				pvcs = append(pvcs, pvc)
			}
		}
		if pvcs != nil {
			// Delete any PVCs not yet deleted.
			var anyErr error = nil
			for _, pvc := range pvcs {
				err := c.deps.PVCControl.DeletePVC(tc, pvc)
				klog.Infof("Deleting pvc: %s", pvc.Name)
				if err != nil {
					anyErr = err
				}
			}
			// Requeue next iteration, to verify pvc marked for deletion before moving on to pod deletion.
			return reconcile.Result{Requeue: true}, anyErr
		}
		// 2. Delete pod
		if pod.DeletionTimestamp != nil {
			// Already marked for deletion, wait for it to delete.
			return reconcile.Result{RequeueAfter: RequeueInterval}, nil
		}
		klog.Infof("Deleting pod: %s", pod.Name)
		err := c.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to delete pod %s", pod.Name)
		}
	}
	return reconcile.Result{}, nil
}

func (c *PodController) cleanupLeaderEvictionAnnotations(pod *corev1.Pod, tc *v1alpha1.TidbCluster, annKeys []string) error {
	for _, ann := range annKeys {
		delete(pod.Annotations, ann)
	}

	if _, err := c.deps.PodControl.UpdatePod(tc, pod); err != nil {
		return perrors.Annotatef(err, "failed delete expired annotations for tc %s/%s", pod.Namespace, pod.Name)
	}
	return nil
}

func (c *PodController) isEvictLeaderExpired(pod *corev1.Pod, annKey string) bool {
	timeToExpireAnnValue, exist := pod.Annotations[annKey]
	if exist {
		evictionExpirationTime, err := time.Parse(time.RFC3339, timeToExpireAnnValue)
		if err == nil {
			if metav1.Now().Time.After(evictionExpirationTime) {
				klog.Infof("Annotation to evict leader on the Pod %s/%s is expired", pod.Namespace, pod.Name)
				return true
			}
		} else {
			klog.Warningf("Can't parse %s value %s on %s/%s. Mark pod as expired right away. Err: ", annKey, evictionExpirationTime, pod.Namespace, pod.Name, err)
			return true
		}
	}
	return false
}

func needDeleteTiDBPod(pod *corev1.Pod) string {
	if pod.Annotations == nil {
		return ""
	}
	return pod.Annotations[v1alpha1.TiDBGracefulShutdownAnnKey]
}

func needEvictLeader(pod *corev1.Pod) (string, string, bool) {
	for _, key := range v1alpha1.EvictLeaderAnnKeys {
		value, exist := pod.Annotations[key]
		if exist {
			return key, value, true
		}
	}

	return "", "", false
}

func needPDLeaderTransfer(pod *corev1.Pod) (string, bool) {
	value, exist := pod.Annotations[v1alpha1.PDLeaderTransferAnnKey]
	if !exist {
		return "", false
	}
	return value, true
}

func transferPDLeader(tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) error {
	// find a target from peer members
	target := pickNewLeader(tc)
	if len(target) == 0 {
		return fmt.Errorf("can't find a target pd for leader transfer")
	}

	return pdClient.TransferPDLeader(target)
}

func pickNewLeader(tc *v1alpha1.TidbCluster) string {
	for _, pdMember := range tc.Status.PD.Members {
		if pdMember.Name != tc.Status.PD.Leader.Name && pdMember.Health {
			return pdMember.Name
		}
	}
	for _, peerMember := range tc.Status.PD.PeerMembers {
		if peerMember.Name != tc.Status.PD.Leader.Name && peerMember.Health {
			return peerMember.Name
		}
	}
	return ""
}

func getPdName(pod *corev1.Pod, tc *v1alpha1.TidbCluster) string {
	if len(tc.Spec.ClusterDomain) > 0 {
		return fmt.Sprintf("%s.%s-pd-peer.%s.svc.%s", pod.Name, tc.Name, tc.Namespace, tc.Spec.ClusterDomain)
	}

	if tc.Spec.AcrossK8s {
		return fmt.Sprintf("%s.%s-pd-peer.%s.svc", pod.Name, tc.Name, tc.Namespace)
	}

	return pod.Name
}
