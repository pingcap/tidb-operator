// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State

	PDClient pdm.PDClient

	LeaderEvicting bool

	Store *pdv1.Store
}

// GetStoreID returns the store ID for PD operations
func (r *ReconcileContext) GetStoreID() string {
	if r.Store == nil {
		return ""
	}
	return r.Store.ID
}

// GetStoreNotExists returns true if the store does not exist in PD
func (r *ReconcileContext) GetStoreNotExists() bool {
	return r.Store == nil
}

// GetPDClient returns the PD client for API operations
func (r *ReconcileContext) GetPDClient() pdm.PDClient {
	return r.PDClient
}

// tikvInstanceAdapter adapts TiKV instance to StoreOfflineInstance interface
type tikvInstanceAdapter struct {
	tikv *v1alpha1.TiKV
}

// IsOffline returns true if offline operation is requested
func (a *tikvInstanceAdapter) IsOffline() bool {
	return a.tikv.Spec.Offline
}

// GetConditions returns the conditions slice
func (a *tikvInstanceAdapter) GetConditions() []metav1.Condition {
	return a.tikv.Status.Conditions
}

// SetConditions sets the conditions slice
func (a *tikvInstanceAdapter) SetConditions(conditions []metav1.Condition) {
	a.tikv.Status.Conditions = conditions
}

// NewTiKVInstanceAdapter creates a new adapter for TiKV instance
func NewTiKVInstanceAdapter(tikv *v1alpha1.TiKV) *tikvInstanceAdapter {
	return &tikvInstanceAdapter{tikv: tikv}
}

// GetStoreID returns the store ID for PD operations
func (r *ReconcileContext) GetStoreID() string {
	return r.StoreID
}

// GetStoreNotExists returns true if the store does not exist in PD
func (r *ReconcileContext) GetStoreNotExists() bool {
	return r.StoreNotExists
}

// GetPDClient returns the PD client for API operations
func (r *ReconcileContext) GetPDClient() pdm.PDClient {
	return r.PDClient
}

// tikvInstanceAdapter adapts TiKV instance to StoreOfflineInstance interface
type tikvInstanceAdapter struct {
	tikv *v1alpha1.TiKV
}

// IsOffline returns true if offline operation is requested
func (a *tikvInstanceAdapter) IsOffline() bool {
	return a.tikv.Spec.Offline
}

// GetConditions returns the conditions slice
func (a *tikvInstanceAdapter) GetConditions() []metav1.Condition {
	return a.tikv.Status.Conditions
}

// SetConditions sets the conditions slice
func (a *tikvInstanceAdapter) SetConditions(conditions []metav1.Condition) {
	a.tikv.Status.Conditions = conditions
}

// NewTiKVInstanceAdapter creates a new adapter for TiKV instance
func NewTiKVInstanceAdapter(tikv *v1alpha1.TiKV) *tikvInstanceAdapter {
	return &tikvInstanceAdapter{tikv: tikv}
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		c, ok := cm.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Fail().With("pd client is not registered")
		}
		state.PDClient = c

		if !c.HasSynced() {
			return task.Fail().With("store info is not synced, just wait for next sync")
		}

		s, err := c.Stores().Get(coreutil.TiKVAdvertiseClientURLs(state.TiKV()))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %w", err)
			}
			return task.Complete().With("store does not exist")
		}

		state.Store = s
		state.SetStoreState(string(s.NodeState))
		state.SetLeaderCount(s.LeaderCount)
		state.SetRegionCount(s.RegionCount)
		state.SetStoreBusy(s.IsBusy)

		// TODO: cache evict leader scheduler info, then we don't need to check suspend here
		if coreutil.ShouldSuspendCompute(state.Cluster()) {
			return task.Complete().With("cluster is suspending")
		}
		scheduler, err := state.PDClient.Underlay().GetEvictLeaderScheduler(ctx, state.Store.ID)
		if err != nil {
			return task.Fail().With("pd is unexpectedly crashed: %w", err)
		}
		if scheduler != "" {
			state.LeaderEvicting = true
		}

		return task.Complete().With("get store info")
	})
}
