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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

// StoreCommonAccess contains the set of functions to access the properties of TiKV and TiFlash types in a common way
type StoreCommonAccess interface {
	GetMemberType() v1alpha1.MemberType
	GetMaxFailoverCount(tc *v1alpha1.TidbCluster) *int32
	GetStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVStore
	CreateFailureStoresIfAbsent(tc *v1alpha1.TidbCluster)
	GetFailureStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVFailureStore
	SetFailoverUIDIfAbsent(tc *v1alpha1.TidbCluster)
	SetFailureStores(tc *v1alpha1.TidbCluster, storeID string, failureStore v1alpha1.TiKVFailureStore)
	ClearFailStatus(tc *v1alpha1.TidbCluster)
	GetStsDesiredOrdinals(tc *v1alpha1.TidbCluster, excludeFailover bool) sets.Int32
}

// sharedStoreFailover contains the shared failover logic of TiKV and TiFlash
type sharedStoreFailover struct {
	storeAccess StoreCommonAccess
	deps        *controller.Dependencies
}

func (ssf *sharedStoreFailover) doFailover(tc *v1alpha1.TidbCluster, failoverPeriod time.Duration) error {
	if err := ssf.tryMarkAStoreAsFailure(tc, failoverPeriod); err != nil {
		if controller.IsIgnoreError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (ssf *sharedStoreFailover) tryMarkAStoreAsFailure(tc *v1alpha1.TidbCluster, failoverPeriod time.Duration) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for storeID, store := range ssf.storeAccess.GetStores(tc) {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		if !ssf.isPodDesired(tc, podName) {
			// we should ignore the store record of deleted pod, otherwise the
			// record of deleted pod may be added back to failure stores
			// (before it enters into Offline/Tombstone state)
			continue
		}
		deadline := store.LastTransitionTime.Add(failoverPeriod)
		exist := false
		for _, failureStore := range ssf.storeAccess.GetFailureStores(tc) {
			if failureStore.PodName == podName {
				exist = true
				break
			}
		}
		if store.State == v1alpha1.TiKVStateDown && time.Now().After(deadline) {
			maxFailoverCount := ssf.storeAccess.GetMaxFailoverCount(tc)
			if maxFailoverCount != nil && *maxFailoverCount > 0 {
				ssf.storeAccess.SetFailoverUIDIfAbsent(tc)
				if !exist {
					ssf.storeAccess.CreateFailureStoresIfAbsent(tc)
					if len(ssf.storeAccess.GetFailureStores(tc)) >= int(*maxFailoverCount) {
						klog.Warningf("%s/%s %s failure stores count reached the limit: %d", ns, tcName, ssf.storeAccess.GetMemberType(), maxFailoverCount)
						return nil
					}
					ssf.storeAccess.SetFailureStores(tc, storeID, v1alpha1.TiKVFailureStore{
						PodName:   podName,
						StoreID:   store.ID,
						CreatedAt: metav1.Now(),
					})
					msg := fmt.Sprintf("store[%s] is Down", store.ID)
					ssf.deps.Recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, ssf.storeAccess.GetMemberType(), podName, msg))
				}
			}
		}
	}
	return nil
}

func (ssf *sharedStoreFailover) isPodDesired(tc *v1alpha1.TidbCluster, podName string) bool {
	ordinals := ssf.storeAccess.GetStsDesiredOrdinals(tc, true)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func (ssf *sharedStoreFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
	for key, failureStore := range ssf.storeAccess.GetFailureStores(tc) {
		if !ssf.isPodDesired(tc, failureStore.PodName) {
			// If we delete the pods, e.g. by using advanced statefulset delete
			// slots feature. We should remove the record of undesired pods,
			// otherwise an extra replacement pod will be created.
			delete(ssf.storeAccess.GetFailureStores(tc), key)
		}
	}
}

func (ssf *sharedStoreFailover) Recover(tc *v1alpha1.TidbCluster) {
	ssf.storeAccess.ClearFailStatus(tc)
	klog.Infof("%s recover: clear FailureStores, %s/%s", ssf.storeAccess.GetMemberType(), tc.GetNamespace(), tc.GetName())
}
