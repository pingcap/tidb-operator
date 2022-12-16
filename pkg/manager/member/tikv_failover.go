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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// NewTiKVFailover returns a tikv Failover
func NewTiKVFailover(deps *controller.Dependencies) Failover {
	storeAccess := &tikvStoreAccess{}
	failureRecovery := commonStatefulFailureRecovery{deps: deps, failureObjectAccess: &failureStoreAccess{storeAccess: storeAccess}}
	return &commonStoreFailover{deps: deps, storeAccess: storeAccess, failureRecovery: failureRecovery}
}

// tikvStoreAccess is a folder of access functions for TiKV store and implements StoreAccess
type tikvStoreAccess struct{}

var _ StoreAccess = (*tikvStoreAccess)(nil)

func (tsa *tikvStoreAccess) GetFailoverPeriod(cliConfig *controller.CLIConfig) time.Duration {
	return cliConfig.TiKVFailoverPeriod
}

func (tsa *tikvStoreAccess) GetMemberType() v1alpha1.MemberType {
	return v1alpha1.TiKVMemberType
}

func (tsa *tikvStoreAccess) GetMaxFailoverCount(tc *v1alpha1.TidbCluster) *int32 {
	return tc.Spec.TiKV.MaxFailoverCount
}

func (tsa *tikvStoreAccess) GetStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVStore {
	return tc.Status.TiKV.Stores
}

func (tsa *tikvStoreAccess) GetStore(tc *v1alpha1.TidbCluster, storeID string) (v1alpha1.TiKVStore, bool) {
	store, exists := tc.Status.TiKV.Stores[storeID]
	return store, exists
}

func (tsa *tikvStoreAccess) SetFailoverUIDIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiKV.FailoverUID == "" {
		tc.Status.TiKV.FailoverUID = uuid.NewUUID()
	}
}

func (tsa *tikvStoreAccess) CreateFailureStoresIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiKV.FailureStores == nil {
		tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
	}
}

func (tsa *tikvStoreAccess) GetFailureStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVFailureStore {
	return tc.Status.TiKV.FailureStores
}

func (tsa *tikvStoreAccess) GetFailureStore(tc *v1alpha1.TidbCluster, storeID string) (v1alpha1.TiKVFailureStore, bool) {
	failureStore, exists := tc.Status.TiKV.FailureStores[storeID]
	return failureStore, exists
}

func (tsa *tikvStoreAccess) SetFailureStore(tc *v1alpha1.TidbCluster, storeID string, failureStore v1alpha1.TiKVFailureStore) {
	tc.Status.TiKV.FailureStores[storeID] = failureStore
}

func (tsa *tikvStoreAccess) GetStsDesiredOrdinals(tc *v1alpha1.TidbCluster, excludeFailover bool) sets.Int32 {
	return tc.TiKVStsDesiredOrdinals(excludeFailover)
}

func (tsa *tikvStoreAccess) ClearFailStatus(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.FailureStores = nil
	tc.Status.TiKV.FailoverUID = ""
}

// IsHostDownForFailurePod checks if HostDown is set for any tikv failure store
func (tsa *tikvStoreAccess) IsHostDownForFailurePod(tc *v1alpha1.TidbCluster) bool {
	for storeID := range tc.Status.TiKV.FailureStores {
		failureStore := tc.Status.TiKV.FailureStores[storeID]
		if failureStore.HostDown {
			return true
		}
	}
	return false
}

// NewFakeTiKVFailover returns a fake Failover
func NewFakeTiKVFailover() Failover {
	return &fakeStoreFailover{}
}
