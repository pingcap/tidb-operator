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

package member

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// NewTiFlashFailover returns a tiflash Failover
func NewTiFlashFailover(deps *controller.Dependencies) Failover {
	storeAccess := &tiflashStoreAccess{}
	failureRecovery := commonStatefulFailureRecovery{deps: deps, failureObjectAccess: &failureStoreAccess{storeAccess: storeAccess}}
	return &commonStoreFailover{deps: deps, storeAccess: storeAccess, failureRecovery: failureRecovery}
}

// tiflashStoreAccess is a folder of access functions for TiFlash store and implements StoreAccess
type tiflashStoreAccess struct{}

var _ StoreAccess = (*tiflashStoreAccess)(nil)

func (tsa *tiflashStoreAccess) GetFailoverPeriod(cliConfig *controller.CLIConfig) time.Duration {
	return cliConfig.TiFlashFailoverPeriod
}

func (tsa *tiflashStoreAccess) GetMemberType() v1alpha1.MemberType {
	return v1alpha1.TiFlashMemberType
}

func (tsa *tiflashStoreAccess) GetMaxFailoverCount(tc *v1alpha1.TidbCluster) *int32 {
	return tc.Spec.TiFlash.MaxFailoverCount
}

func (tsa *tiflashStoreAccess) GetStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVStore {
	return tc.Status.TiFlash.Stores
}

func (tsa *tiflashStoreAccess) GetStore(tc *v1alpha1.TidbCluster, storeID string) (v1alpha1.TiKVStore, bool) {
	store, exists := tc.Status.TiFlash.Stores[storeID]
	return store, exists
}

func (tsa *tiflashStoreAccess) SetFailoverUIDIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiFlash.FailoverUID == "" {
		tc.Status.TiFlash.FailoverUID = uuid.NewUUID()
	}
}

func (tsa *tiflashStoreAccess) CreateFailureStoresIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiFlash.FailureStores == nil {
		tc.Status.TiFlash.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
	}
}

func (tsa *tiflashStoreAccess) GetFailureStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVFailureStore {
	return tc.Status.TiFlash.FailureStores
}

func (tsa *tiflashStoreAccess) GetFailureStore(tc *v1alpha1.TidbCluster, storeID string) (v1alpha1.TiKVFailureStore, bool) {
	failureStore, exists := tc.Status.TiFlash.FailureStores[storeID]
	return failureStore, exists
}

func (tsa *tiflashStoreAccess) SetFailureStore(tc *v1alpha1.TidbCluster, storeID string, failureStore v1alpha1.TiKVFailureStore) {
	tc.Status.TiFlash.FailureStores[storeID] = failureStore
}

func (tsa *tiflashStoreAccess) GetStsDesiredOrdinals(tc *v1alpha1.TidbCluster, excludeFailover bool) sets.Int32 {
	return tc.TiFlashStsDesiredOrdinals(excludeFailover)
}

func (tsa *tiflashStoreAccess) ClearFailStatus(tc *v1alpha1.TidbCluster) {
	tc.Status.TiFlash.FailureStores = nil
	tc.Status.TiFlash.FailoverUID = ""
}

// IsHostDownForFailurePod checks if HostDown is set for any tiflash failure store
func (tsa *tiflashStoreAccess) IsHostDownForFailurePod(tc *v1alpha1.TidbCluster) bool {
	for storeID := range tc.Status.TiFlash.FailureStores {
		failureStore := tc.Status.TiFlash.FailureStores[storeID]
		if failureStore.HostDown {
			return true
		}
	}
	return false
}

// NewFakeTiFlashFailover returns a fake Failover
func NewFakeTiFlashFailover() Failover {
	return &fakeStoreFailover{}
}
