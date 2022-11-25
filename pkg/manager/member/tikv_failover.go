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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
)

type tikvFailover struct {
	sharedFailover sharedStoreFailover
}

// NewTiKVFailover returns a tikv Failover
func NewTiKVFailover(deps *controller.Dependencies) Failover {
	return &tikvFailover{sharedFailover: sharedStoreFailover{storeAccess: &tikvStoreAccess{}, deps: deps}}
}

func (f *tikvFailover) Failover(tc *v1alpha1.TidbCluster) error {
	return f.sharedFailover.doFailover(tc, f.sharedFailover.deps.CLIConfig.TiKVFailoverPeriod)
}

func (f *tikvFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
	f.sharedFailover.RemoveUndesiredFailures(tc)
}

func (f *tikvFailover) Recover(tc *v1alpha1.TidbCluster) {
	f.sharedFailover.Recover(tc)
}

// tikvStoreAccess is a folder of access functions for TiKV store
type tikvStoreAccess struct {
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

func (tsa *tikvStoreAccess) CreateFailureStoresIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiKV.FailureStores == nil {
		tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
	}
}

func (tsa *tikvStoreAccess) GetFailureStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVFailureStore {
	return tc.Status.TiKV.FailureStores
}
func (tsa *tikvStoreAccess) SetFailoverUIDIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiKV.FailoverUID == "" {
		tc.Status.TiKV.FailoverUID = uuid.NewUUID()
	}
}

func (tsa *tikvStoreAccess) SetFailureStores(tc *v1alpha1.TidbCluster, storeID string, failureStore v1alpha1.TiKVFailureStore) {
	tc.Status.TiKV.FailureStores[storeID] = failureStore
}

func (tsa *tikvStoreAccess) GetStsDesiredOrdinals(tc *v1alpha1.TidbCluster, excludeFailover bool) sets.Int32 {
	return tc.TiKVStsDesiredOrdinals(excludeFailover)
}

func (tsa *tikvStoreAccess) ClearFailStatus(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.FailureStores = nil
	tc.Status.TiKV.FailoverUID = ""
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
