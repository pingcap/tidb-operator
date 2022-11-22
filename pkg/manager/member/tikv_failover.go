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

package member

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
)

type tikvFailover struct {
	deps *controller.Dependencies
}

// NewTiKVFailover returns a tikv Failover
func NewTiKVFailover(deps *controller.Dependencies) Failover {
	return &tikvFailover{deps: deps}
}

func (f *tikvFailover) isPodDesired(tc *v1alpha1.TidbCluster, podName string) bool {
	ordinals := tc.TiKVStsDesiredOrdinals(true)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func (f *tikvFailover) Failover(tc *v1alpha1.TidbCluster) error {
	// If store downtime exceeds failover time period then create a FailureStore
	// The failure recovery of down store follows this timeline:
	// If HostDown is not set then detect node failure for the pod and set HostDown
	// If HostDown is set then force restart pod once
	// If HostDown is set and Store is Down then delete Store after some gap from the time of pod restart
	// If HostDown is set and Store has been removed or become Tombstone then remove PVC and set StoreDeleted

	if tc.Status.TiKV.FailureStores == nil {
		tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
	}
	// If HostDown is set for any FailureStore then do restart of pod
	if f.deps.CLIConfig.DetectNodeFailure {
		if isFailureTikvPodHostDown(tc) {
			if canAutoFailureRecovery(tc) {
				if err := f.restartPodForHostDown(tc); err != nil {
					if controller.IsIgnoreError(err) {
						return nil
					}
					return err
				}
			}
		} else {
			// If HostDown is not set for any FailureStore then try to detect node failure
			if err := f.checkAndMarkHostDown(tc); err != nil {
				if controller.IsIgnoreError(err) {
					return nil
				}
				return err
			}
		}
	}

	if tc.Spec.TiKV.MaxFailoverCount != nil && *tc.Spec.TiKV.MaxFailoverCount > 0 {
		if err := f.tryMarkAStoreAsFailure(tc); err != nil {
			if controller.IsIgnoreError(err) {
				return nil
			}
			return err
		}
	}

	if canAutoFailureRecovery(tc) {
		// If the store has not come back Up in some time after pod was restarted, then delete store and remove failure PVC
		for storeId := range tc.Status.TiKV.FailureStores {
			failureStore := tc.Status.TiKV.FailureStores[storeId]
			if failureStore.HostDown && !failureStore.StoreDeleted {
				if err := f.invokeDeleteFailureStore(tc, failureStore); err != nil {
					if controller.IsIgnoreError(err) {
						return nil
					}
					return err
				}
				if err := f.checkAndRemoveFailurePVC(tc, failureStore); err != nil {
					if controller.IsIgnoreError(err) {
						return nil
					}
					return err
				}
			}
		}
	}
	return nil
}

// isFailureTikvPodHostDown checks if HostDown is set for any tikv failure store
func isFailureTikvPodHostDown(tc *v1alpha1.TidbCluster) bool {
	for storeID := range tc.Status.TiKV.FailureStores {
		failureStore := tc.Status.TiKV.FailureStores[storeID]
		if failureStore.HostDown {
			return true
		}
	}
	return false
}

func (f *tikvFailover) tryMarkAStoreAsFailure(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for storeID, store := range tc.Status.TiKV.Stores {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		if !f.isPodDesired(tc, podName) {
			// we should ignore the store record of deleted pod, otherwise the
			// record of deleted pod may be added back to failure stores
			// (before it enters into Offline/Tombstone state)
			continue
		}
		deadline := store.LastTransitionTime.Add(f.deps.CLIConfig.TiKVFailoverPeriod)
		exist := false
		for _, failureStore := range tc.Status.TiKV.FailureStores {
			if failureStore.PodName == podName {
				exist = true
				break
			}
		}
		if store.State == v1alpha1.TiKVStateDown && time.Now().After(deadline) {
			if tc.Status.TiKV.FailoverUID == "" {
				tc.Status.TiKV.FailoverUID = uuid.NewUUID()
			}
			if !exist {
				maxFailoverCount := *tc.Spec.TiKV.MaxFailoverCount
				if len(tc.Status.TiKV.FailureStores) >= int(maxFailoverCount) {
					klog.Warningf("%s/%s TiKV failure stores count reached the limit: %d", ns, tcName, tc.Spec.TiKV.MaxFailoverCount)
					return nil
				}
				pvcs, err := getPodPvcs(tc, podName, v1alpha1.TiKVMemberType, f.deps.PVCLister)
				if err != nil {
					return err
				}
				pvcUIDSet := make(map[types.UID]v1alpha1.EmptyStruct)
				for _, pvc := range pvcs {
					pvcUIDSet[pvc.UID] = v1alpha1.EmptyStruct{}
				}
				klog.Infof("tikv failover [tryToMarkAPeerAsFailure] PVCUIDSet for failure store %s is %s", store.ID, pvcUIDSet)
				tc.Status.TiKV.FailureStores[storeID] = v1alpha1.TiKVFailureStore{
					PodName:   podName,
					StoreID:   store.ID,
					PVCUIDSet: pvcUIDSet,
					CreatedAt: metav1.Now(),
				}
				msg := fmt.Sprintf("store[%s] is Down", store.ID)
				f.deps.Recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "tikv", podName, msg))
			}
		}
	}
	return nil
}

// checkAndMarkHostDown checks the availability of nodes of failure pods and marks HostDown for one failure store at a time
func (f *tikvFailover) checkAndMarkHostDown(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	for storeId := range tc.Status.TiKV.FailureStores {
		failureStore := tc.Status.TiKV.FailureStores[storeId]
		// for backward compatibility, if there exists failureStores and user upgrades operator to newer version there
		// will be failure store structures with empty PVCUIDSet set from api server, we should not handle those failure
		// stores for failure recovery
		if !failureStore.HostDown && len(failureStore.PVCUIDSet) > 0 {
			tikvStore, exists := tc.Status.TiKV.Stores[failureStore.StoreID]
			if exists && tikvStore.State == v1alpha1.TiKVStateDown {
				pod, err := f.deps.PodLister.Pods(ns).Get(tikvStore.PodName)
				if err != nil && !errors.IsNotFound(err) {
					return fmt.Errorf("tikv failover [tryFailurePodRestart]: failed to get pod %s for tc %s/%s, error: %s", tikvStore.PodName, ns, tcName, err)
				}
				if pod == nil {
					return nil
				}

				// Check node and pod conditions and set HostDown in FailureStore
				var reason string
				if f.deps.CLIConfig.PodHardRecoveryPeriod > 0 && time.Now().After(tikvStore.LastTransitionTime.Add(f.deps.CLIConfig.PodHardRecoveryPeriod)) {
					reason = hdReasonStoreDownTimeExceeded
				}
				if len(reason) == 0 {
					hostingNodeUnavailable, roDiskFound, detectErr := isHostingNodeUnavailable(f.deps.NodeLister, pod)
					if detectErr != nil {
						return detectErr
					}
					if hostingNodeUnavailable {
						reason = hdReasonNodeFailure
					}
					if roDiskFound {
						reason = hdReasonRODiskFound
					}
				}

				if len(reason) > 0 {
					failureStore.HostDown = true
					tc.Status.TiKV.FailureStores[failureStore.StoreID] = failureStore
					klog.Infof("tikv failover [checkAndMarkHostDown]: Set HostDown for tikv store '%s' in cluster %s/%s. Host down reason: %s", failureStore.StoreID, tc.GetName(), tc.GetName(), reason)
					return controller.IgnoreErrorf("failure tikv pod %s/%s can be force deleted for recovery", ns, tikvStore.PodName)
				}
			}
		}
	}
	return nil
}

// restartPodForHostDown restarts pod of one tikv failure store if HostDown is set for the tikv failure store
func (f *tikvFailover) restartPodForHostDown(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	for storeId := range tc.Status.TiKV.FailureStores {
		failureStore := tc.Status.TiKV.FailureStores[storeId]
		tikvStore, exists := tc.Status.TiKV.Stores[failureStore.StoreID]
		if failureStore.HostDown && exists && tikvStore.State == v1alpha1.TiKVStateDown {
			pod, err := f.deps.PodLister.Pods(ns).Get(tikvStore.PodName)
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("tikv failover [restartPodForHostDown]: failed to get pod %s for tc %s/%s, error: %s", tikvStore.PodName, ns, tcName, err)
			}
			if pod == nil {
				return nil
			}
			// If the failed pod has already been restarted once, its CreationTimestamp will be after FailureMember.CreatedAt
			if failureStore.CreatedAt.After(pod.CreationTimestamp.Time) {
				// Use force option to delete the pod
				if err = f.deps.PodControl.ForceDeletePod(tc, pod); err != nil {
					return err
				}
				msg := fmt.Sprintf("Failed tikv pod %s/%s is force deleted for recovery", ns, tikvStore.PodName)
				klog.Infof(msg)
				return controller.IgnoreErrorf(msg)
			}
		}
	}
	return nil
}

// invokeDeleteFailureStore invokes Delete of a tikv failure store. A gap after a pod restart which will give the pod a
// chance to be created properly and store to come up in cases like EBS backed storage in which delete will not be done.
func (f *tikvFailover) invokeDeleteFailureStore(tc *v1alpha1.TidbCluster, failureStore v1alpha1.TiKVFailureStore) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if failureStore.StoreDeleted {
		return nil
	}
	tikvStore, storeExists := tc.Status.TiKV.Stores[failureStore.StoreID]
	// Delete store
	if storeExists && tikvStore.State == v1alpha1.TiKVStateDown {
		pod, err := f.deps.PodLister.Pods(ns).Get(tikvStore.PodName)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("tikv failover [invokeDeleteFailureStore]: failed to get pod %s for tc %s/%s, error: %s", tikvStore.PodName, ns, tcName, err)
		}
		if pod == nil {
			return nil
		}
		// Check if pod was restarted. The CreationTimestamp of new pod should be after FailureMember.CreatedAt
		if pod.CreationTimestamp.After(failureStore.CreatedAt.Time) && pod.CreationTimestamp.Add(restartToDeleteStoreGap).Before(time.Now()) {
			storeUintId, parseErr := strconv.ParseUint(failureStore.StoreID, 10, 64)
			if parseErr != nil {
				return parseErr
			}
			pdCli := controller.GetPDClient(f.deps.PDControl, tc)
			if deleteErr := pdCli.DeleteStore(storeUintId); deleteErr != nil {
				return deleteErr
			}
			msg := fmt.Sprintf("Invoked delete on tikv store '%s' in cluster %s/%s", failureStore.StoreID, ns, tcName)
			f.deps.Recorder.Event(tc, corev1.EventTypeWarning, recoveryEventReason, msg)
			return controller.RequeueErrorf(msg)
		}
	}
	return nil
}

// checkAndRemoveFailurePVC removes failure pod and pvc if the store is Tombstone or is removed. At the end it marks
// StoreDeleted for the failure store.
func (f *tikvFailover) checkAndRemoveFailurePVC(tc *v1alpha1.TidbCluster, failureStore v1alpha1.TiKVFailureStore) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	tikvStore, storeExists := tc.Status.TiKV.Stores[failureStore.StoreID]
	if !storeExists || tikvStore.State == v1alpha1.TiKVStateTombstone {
		pod, pvcs, err := getPodAndPvcs(tc, failureStore.PodName, v1alpha1.TiKVMemberType, f.deps.PodLister, f.deps.PVCLister)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		if pod == nil {
			klog.Infof("tikv failover[checkAndRemoveFailurePVC]: failure pod %s/%s not found, skip", ns, failureStore.PodName)
			return nil
		}
		if pod.DeletionTimestamp == nil {
			// If Scheduled condition of pod is True, then it would have re-used the old PVC after the pod restart and
			// clean up of pvc will not happen. It is expected that scheduling be disabled on the bad node by cordoning
			// it before pod and pvc delete.
			podScheduled := isPodConditionScheduledTrue(pod.Status.Conditions)
			klog.Infof("tikv failover[checkAndRemoveFailurePVC]: Scheduled condition of pod %s of tc %s/%s: %t", failureStore.PodName, ns, tcName, podScheduled)
			if deleteErr := f.deps.PodControl.DeletePod(tc, pod); deleteErr != nil {
				return deleteErr
			}
		} else {
			klog.Infof("pod %s/%s has DeletionTimestamp set to %s", ns, pod.Name, pod.DeletionTimestamp)
		}
		pvcUids := make([]types.UID, 0, len(pvcs))
		for p := range pvcs {
			pvcUids = append(pvcUids, pvcs[p].ObjectMeta.UID)
		}
		klog.Infof("tikv failover[checkAndRemoveFailurePVC]: UIDs of PVCs to be deleted in cluster %s/%s: %s", ns, tcName, pvcUids)
		for p := range pvcs {
			pvc := pvcs[p]
			if _, pvcUIDExist := failureStore.PVCUIDSet[pvc.ObjectMeta.UID]; pvcUIDExist {
				if pvc.DeletionTimestamp == nil {
					if deleteErr := f.deps.PVCControl.DeletePVC(tc, pvc); deleteErr != nil {
						klog.Errorf("tikv failover[checkAndRemoveFailurePVC]: failed to delete PVC: %s/%s, error: %s", ns, pvc.Name, deleteErr)
						return deleteErr
					}
					klog.Infof("tikv failover[checkAndRemoveFailurePVC]: delete PVC %s/%s successfully", ns, pvc.Name)
				} else {
					klog.Infof("pvc %s/%s has DeletionTimestamp set to %s", ns, pvc.Name, pvc.DeletionTimestamp)
				}
			}
		}
		failureStore.StoreDeleted = true
		tc.Status.TiKV.FailureStores[failureStore.StoreID] = failureStore
		klog.Infof("tikv failover: Set StoreDeleted for tikv store '%s' in cluster %s/%s", failureStore.StoreID, tc.GetNamespace(), tc.GetName())
	}
	return nil
}

func (f *tikvFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
	for key, failureStore := range tc.Status.TiKV.FailureStores {
		if !f.isPodDesired(tc, failureStore.PodName) {
			// If we delete the pods, e.g. by using advanced statefulset delete
			// slots feature. We should remove the record of undesired pods,
			// otherwise an extra replacement pod will be created.
			delete(tc.Status.TiKV.FailureStores, key)
		}
	}
}

func (f *tikvFailover) Recover(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.FailureStores = nil
	tc.Status.TiKV.FailoverUID = ""
	klog.Infof("TiKV recover: clear FailureStores, %s/%s", tc.GetNamespace(), tc.GetName())
}

type fakeTiKVFailover struct{}

// NewFakeTiKVFailover returns a fake Failover
func NewFakeTiKVFailover() Failover {
	return &fakeTiKVFailover{}
}

func (ftf *fakeTiKVFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (ftf *fakeTiKVFailover) Recover(_ *v1alpha1.TidbCluster) {
}

func (ftf *fakeTiKVFailover) RemoveUndesiredFailures(_ *v1alpha1.TidbCluster) {
}
