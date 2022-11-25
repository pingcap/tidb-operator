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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
)

type tiflashFailover struct {
	sharedFailover sharedStoreFailover
}

// NewTiFlashFailover returns a tiflash Failover
func NewTiFlashFailover(deps *controller.Dependencies) Failover {
	return &tiflashFailover{sharedFailover: sharedStoreFailover{storeAccess: &tiflashStoreAccess{}, deps: deps}}
}

func (f *tiflashFailover) Failover(tc *v1alpha1.TidbCluster) error {
	if err := f.sharedFailover.tryMarkAStoreAsFailure(tc, f.sharedFailover.deps.CLIConfig.TiFlashFailoverPeriod); err != nil {
		if controller.IsIgnoreError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (f *tiflashFailover) RemoveUndesiredFailures(tc *v1alpha1.TidbCluster) {
	f.sharedFailover.RemoveUndesiredFailures(tc)
}

func (f *tiflashFailover) Recover(tc *v1alpha1.TidbCluster) {
	f.sharedFailover.Recover(tc)
}

// tiflashStoreAccess is a folder of access functions for TiFlash store
type tiflashStoreAccess struct {
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

func (tsa *tiflashStoreAccess) CreateFailureStoresIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiFlash.FailureStores == nil {
		tc.Status.TiFlash.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
	}
}

func (tsa *tiflashStoreAccess) GetFailureStores(tc *v1alpha1.TidbCluster) map[string]v1alpha1.TiKVFailureStore {
	return tc.Status.TiFlash.FailureStores
}
func (tsa *tiflashStoreAccess) SetFailoverUIDIfAbsent(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiFlash.FailoverUID == "" {
		tc.Status.TiFlash.FailoverUID = uuid.NewUUID()
	}
}

func (tsa *tiflashStoreAccess) SetFailureStores(tc *v1alpha1.TidbCluster, storeID string, failureStore v1alpha1.TiKVFailureStore) {
	tc.Status.TiFlash.FailureStores[storeID] = failureStore
}

func (tsa *tiflashStoreAccess) GetStsDesiredOrdinals(tc *v1alpha1.TidbCluster, excludeFailover bool) sets.Int32 {
	return tc.TiFlashStsDesiredOrdinals(excludeFailover)
}

func (tsa *tiflashStoreAccess) ClearFailStatus(tc *v1alpha1.TidbCluster) {
	tc.Status.TiFlash.FailureStores = nil
	tc.Status.TiFlash.FailoverUID = ""
}

type fakeTiFlashFailover struct{}

// NewFakeTiFlashFailover returns a fake Failover
func NewFakeTiFlashFailover() Failover {
	return &fakeTiFlashFailover{}
}

func (_ *fakeTiFlashFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (_ *fakeTiFlashFailover) Recover(_ *v1alpha1.TidbCluster) {
}

func (_ *fakeTiFlashFailover) RemoveUndesiredFailures(_ *v1alpha1.TidbCluster) {
}
