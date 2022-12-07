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

// StoreAccess contains the common set of functions to access the properties of TiKV and TiFlash types
type StoreAccess interface {
	GetFailoverPeriod(cliConfig *controller.CLIConfig) time.Duration
	GetMemberType() v1alpha1.MemberType
	GetMaxFailoverCount(tc *v1alpha1.TidbCluster) *int32
	GetStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVStore
	SetFailoverUIDIfAbsent(tc *v1alpha1.TidbCluster)
	CreateFailureStoresIfAbsent(tc *v1alpha1.TidbCluster)
	GetFailureStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVFailureStore
	SetFailureStore(tc *v1alpha1.TidbCluster, storeID string, failureStore v1alpha1.TiKVFailureStore)
	ClearFailStatus(tc *v1alpha1.TidbCluster)
	GetStsDesiredOrdinals(tc *v1alpha1.TidbCluster, excludeFailover bool) sets.Int32
}

// commonStoreFailover handles the common failover logic for TiKV and TiFlash
type commonStoreFailover struct {
	deps        *controller.Dependencies
	storeAccess StoreAccess
}

func (sf *commonStoreFailover) Failover(tc *v1alpha1.TidbCluster) error {
	if err := sf.tryMarkAStoreAsFailure(tc); err != nil {
		if controller.IsIgnoreError(err) {
			return nil
		}
		return err
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
					sf.storeAccess.SetFailureStore(tc, storeID, v1alpha1.TiKVFailureStore{
						PodName:   podName,
						StoreID:   store.ID,
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

type fakeStoreFailover struct{}

func (fsf *fakeStoreFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (fsf *fakeStoreFailover) Recover(_ *v1alpha1.TidbCluster) {}

func (fsf *fakeStoreFailover) RemoveUndesiredFailures(_ *v1alpha1.TidbCluster) {}

var _ Failover = (*fakeStoreFailover)(nil)
