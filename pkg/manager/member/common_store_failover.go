// Copyright 2022 PingCAP, Inc.
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

// StoreAccess contains the common set of functions to access the properties of TiKV and TiFlash types
type StoreAccess interface {
	GetFailoverPeriod(cliConfig *controller.CLIConfig) time.Duration
	GetMemberType() v1alpha1.MemberType
	GetMaxFailoverCount(tc *v1alpha1.TidbCluster) *int32
	GetStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVStore
	GetStore(tc *v1alpha1.TidbCluster, storeID string) (v1alpha1.TiKVStore, bool)
	SetFailoverUIDIfAbsent(tc *v1alpha1.TidbCluster)
	CreateFailureStoresIfAbsent(tc *v1alpha1.TidbCluster)
	GetFailureStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVFailureStore
	GetFailureStore(tc *v1alpha1.TidbCluster, storeID string) (v1alpha1.TiKVFailureStore, bool)
	SetFailureStore(tc *v1alpha1.TidbCluster, storeID string, failureStore v1alpha1.TiKVFailureStore)
	ClearFailStatus(tc *v1alpha1.TidbCluster)
	GetStsDesiredOrdinals(tc *v1alpha1.TidbCluster, excludeFailover bool) sets.Int32
	IsHostDownForFailurePod(tc *v1alpha1.TidbCluster) bool
}

// commonStoreFailover has the common logic to handle the failover of TiKV and TiFlash store
type commonStoreFailover struct {
	deps            *controller.Dependencies
	storeAccess     StoreAccess
	failureRecovery commonStatefulFailureRecovery
}

var _ Failover = (*commonStoreFailover)(nil)

func (sf *commonStoreFailover) Failover(tc *v1alpha1.TidbCluster) error {
	// If store downtime exceeds failover time period then create a FailureStore
	// The failure recovery of down store follows this timeline:
	// If HostDown is not set then detect node failure for the pod and set HostDown
	// If HostDown is set then force restart pod once
	// If HostDown is set and Store is Down then delete Store after some gap from the time of pod restart
	// If HostDown is set and Store has been removed or become Tombstone then remove PVC and set StoreDeleted

	if err := sf.failureRecovery.RestartPodOnHostDown(tc); err != nil {
		if controller.IsIgnoreError(err) {
			return nil
		}
		return err
	}

	if err := sf.tryMarkAStoreAsFailure(tc); err != nil {
		if controller.IsIgnoreError(err) {
			return nil
		}
		return err
	}

	if canAutoFailureRecovery(tc) {
		// If the store has not come back Up in some time after pod was restarted, then delete store and remove failure PVC
		failureStores := sf.storeAccess.GetFailureStores(tc)
		for _, failureStore := range failureStores {
			if failureStore.HostDown && !failureStore.StoreDeleted {
				if err := sf.invokeDeleteFailureStore(tc, failureStore); err != nil {
					if controller.IsIgnoreError(err) {
						return nil
					}
					return err
				}
				if err := sf.checkAndRemoveFailurePVC(tc, failureStore); err != nil {
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

func (sf *commonStoreFailover) tryMarkAStoreAsFailure(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for storeID, store := range sf.storeAccess.GetStores(tc) {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		if !sf.isPodDesired(tc, podName) {
			// we should ignore the store record of deleted pod, otherwise the
			// record of deleted pod may be added back to failure stores
			// (before it enters into Offline/Tombstone state)
			continue
		}
		deadline := store.LastTransitionTime.Add(sf.storeAccess.GetFailoverPeriod(sf.deps.CLIConfig))
		exist := false
		for _, failureStore := range sf.storeAccess.GetFailureStores(tc) {
			if failureStore.PodName == podName {
				exist = true
				break
			}
		}
		if store.State == v1alpha1.TiKVStateDown && time.Now().After(deadline) {
			maxFailoverCount := sf.storeAccess.GetMaxFailoverCount(tc)
			if maxFailoverCount != nil && *maxFailoverCount > 0 {
				sf.storeAccess.SetFailoverUIDIfAbsent(tc)
				if !exist {
					sf.storeAccess.CreateFailureStoresIfAbsent(tc)
					if len(sf.storeAccess.GetFailureStores(tc)) >= int(*maxFailoverCount) {
						klog.Warningf("%s/%s %s failure stores count reached the limit: %d", ns, tcName, sf.storeAccess.GetMemberType(), maxFailoverCount)
						return nil
					}
					pvcs, err := sf.failureRecovery.getPodPvcs(tc, podName)
					if err != nil {
						return err
					}
					pvcUIDSet := make(map[types.UID]v1alpha1.EmptyStruct)
					for _, pvc := range pvcs {
						pvcUIDSet[pvc.UID] = v1alpha1.EmptyStruct{}
					}
					klog.Infof("%s failover [tryMarkAStoreAsFailure] PVCUIDSet for failure store %s is %s", sf.storeAccess.GetMemberType(), store.ID, pvcUIDSet)
					sf.storeAccess.SetFailureStore(tc, storeID, v1alpha1.TiKVFailureStore{
						PodName:   podName,
						StoreID:   store.ID,
						PVCUIDSet: pvcUIDSet,
						CreatedAt: metav1.Now(),
					})
					msg := fmt.Sprintf("store[%s] is Down", store.ID)
					sf.deps.Recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, sf.storeAccess.GetMemberType(), podName, msg))
				}
			}
		}
	}
	return nil
}

// invokeDeleteFailureStore invokes delete of a failure store. A time gap is given after a pod restart to allow the pod to
// be created properly and the store to come up, for ex., in cases like EBS volumes.
func (sf *commonStoreFailover) invokeDeleteFailureStore(tc *v1alpha1.TidbCluster, failureStore v1alpha1.TiKVFailureStore) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if failureStore.StoreDeleted {
		return nil
	}
	store, storeExists := sf.storeAccess.GetStore(tc, failureStore.StoreID)
	// Delete store
	if storeExists && store.State == v1alpha1.TiKVStateDown {
		if sf.failureRecovery.canDoCleanUpNow(tc, failureStore.StoreID) {
			storeUintId, parseErr := strconv.ParseUint(failureStore.StoreID, 10, 64)
			if parseErr != nil {
				return parseErr
			}
			pdCli := controller.GetPDClient(sf.deps.PDControl, tc)
			if deleteErr := pdCli.DeleteStore(storeUintId); deleteErr != nil {
				return deleteErr
			}
			msg := fmt.Sprintf("Invoked delete on %s store '%s' in cluster %s/%s", sf.storeAccess.GetMemberType(), failureStore.StoreID, ns, tcName)
			sf.deps.Recorder.Event(tc, corev1.EventTypeWarning, recoveryEventReason, msg)
			return controller.RequeueErrorf(msg)
		}
	}
	return nil
}

// checkAndRemoveFailurePVC removes failure pod and pvc if the store is Tombstone or is removed. At the end it marks
// StoreDeleted for the failure store.
func (sf *commonStoreFailover) checkAndRemoveFailurePVC(tc *v1alpha1.TidbCluster, failureStore v1alpha1.TiKVFailureStore) error {
	store, storeExists := sf.storeAccess.GetStore(tc, failureStore.StoreID)
	if !storeExists || store.State == v1alpha1.TiKVStateTombstone {
		err := sf.failureRecovery.deletePodAndPvcs(tc, failureStore.StoreID)
		if err != nil {
			return err
		}
		failureStore.StoreDeleted = true
		sf.storeAccess.SetFailureStore(tc, failureStore.StoreID, failureStore)
		klog.Infof("%s failover: Set StoreDeleted for store '%s' in cluster %s/%s", sf.storeAccess.GetMemberType(), failureStore.StoreID, tc.GetNamespace(), tc.GetName())
	}
	return nil
}

func (sf *commonStoreFailover) isPodDesired(tc *v1alpha1.TidbCluster, podName string) bool {
	ordinals := sf.storeAccess.GetStsDesiredOrdinals(tc, true)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func (sf *commonStoreFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
	for key, failureStore := range sf.storeAccess.GetFailureStores(tc) {
		if !sf.isPodDesired(tc, failureStore.PodName) {
			// If we delete the pods, e.g. by using advanced statefulset delete
			// slots feature. We should remove the record of undesired pods,
			// otherwise an extra replacement pod will be created.
			delete(sf.storeAccess.GetFailureStores(tc), key)
		}
	}
}

func (sf *commonStoreFailover) Recover(tc *v1alpha1.TidbCluster) {
	sf.storeAccess.ClearFailStatus(tc)
	klog.Infof("%s recover: clear FailureStores, %s/%s", sf.storeAccess.GetMemberType(), tc.GetNamespace(), tc.GetName())
}

// failureStoreAccess implements the FailureObjectAccess interface for TiKV and TiFlash store
type failureStoreAccess struct {
	storeAccess StoreAccess
}

var _ FailureObjectAccess = (*failureStoreAccess)(nil)

func (fsa *failureStoreAccess) GetMemberType() v1alpha1.MemberType {
	return fsa.storeAccess.GetMemberType()
}

// GetFailureObjects returns the set of failure store ids of the particular store type
func (fsa *failureStoreAccess) GetFailureObjects(tc *v1alpha1.TidbCluster) map[string]v1alpha1.EmptyStruct {
	failureStores := make(map[string]v1alpha1.EmptyStruct, len(fsa.storeAccess.GetFailureStores(tc)))
	for storeId := range fsa.storeAccess.GetFailureStores(tc) {
		failureStores[storeId] = v1alpha1.EmptyStruct{}
	}
	return failureStores
}

// IsFailing returns if the particular store is in down state
func (fsa *failureStoreAccess) IsFailing(tc *v1alpha1.TidbCluster, storeId string) bool {
	store, exists := fsa.storeAccess.GetStore(tc, storeId)
	return exists && store.State == v1alpha1.TiKVStateDown
}

// GetPodName returns the pod name of the given failure store
func (fsa *failureStoreAccess) GetPodName(tc *v1alpha1.TidbCluster, storeId string) string {
	failureStore, _ := fsa.storeAccess.GetFailureStore(tc, storeId)
	return failureStore.PodName
}

// IsHostDownForFailedPod checks if HostDown is set for any failure store in the particular store type
func (fsa *failureStoreAccess) IsHostDownForFailedPod(tc *v1alpha1.TidbCluster) bool {
	return fsa.storeAccess.IsHostDownForFailurePod(tc)
}

// IsHostDown returns true if HostDown is set for the given failure store
func (fsa *failureStoreAccess) IsHostDown(tc *v1alpha1.TidbCluster, storeId string) bool {
	failureStore, _ := fsa.storeAccess.GetFailureStore(tc, storeId)
	return failureStore.HostDown
}

// SetHostDown sets the HostDown property in the given failure store
func (fsa *failureStoreAccess) SetHostDown(tc *v1alpha1.TidbCluster, storeId string, hostDown bool) {
	failureStore, _ := fsa.storeAccess.GetFailureStore(tc, storeId)
	failureStore.HostDown = hostDown
	fsa.storeAccess.SetFailureStore(tc, storeId, failureStore)
}

// GetCreatedAt returns the CreatedAt timestamp of the given failure store
func (fsa *failureStoreAccess) GetCreatedAt(tc *v1alpha1.TidbCluster, storeId string) metav1.Time {
	failureStore, _ := fsa.storeAccess.GetFailureStore(tc, storeId)
	return failureStore.CreatedAt
}

// GetLastTransitionTime returns the LastTransitionTime timestamp of the given failure store
func (fsa *failureStoreAccess) GetLastTransitionTime(tc *v1alpha1.TidbCluster, storeId string) metav1.Time {
	store, _ := fsa.storeAccess.GetStore(tc, storeId)
	return store.LastTransitionTime
}

// GetPVCUIDSet returns the PVC UID set of the given failure store
func (fsa *failureStoreAccess) GetPVCUIDSet(tc *v1alpha1.TidbCluster, storeId string) map[types.UID]v1alpha1.EmptyStruct {
	failureStore, _ := fsa.storeAccess.GetFailureStore(tc, storeId)
	return failureStore.PVCUIDSet
}

type fakeStoreFailover struct{}

var _ Failover = (*fakeStoreFailover)(nil)

func (fsf *fakeStoreFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (fsf *fakeStoreFailover) Recover(_ *v1alpha1.TidbCluster) {}

func (fsf *fakeStoreFailover) RemoveUndesiredFailures(_ *v1alpha1.TidbCluster) {}
