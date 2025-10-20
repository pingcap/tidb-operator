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

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

type ReconcileContext struct {
	State

	PDClient pdm.PDClient

	LeaderEvicting bool

	Store    *pdv1.Store
	PDSynced bool

	// IsStoreReady will be set only when pd is synced and the store is ok
	// It may be outdated so the tikv is healthy only when the pod is also available
	// If it's true and the pod is ready but not available,
	// the operator will retry to avoid unexpectedly missing next reconciliation
	IsStoreReady bool
}

// GetStoreID returns the store ID for PD operations
func (r *ReconcileContext) GetStoreID() string {
	if r.Store == nil {
		return ""
	}
	return r.Store.ID
}

// StoreNotExists returns true if the store does not exist in PD
func (r *ReconcileContext) StoreNotExists() bool {
	return r.Store == nil
}

// GetPDClient returns the PD client for API operations
func (r *ReconcileContext) GetPDClient() pdm.PDClient {
	return r.PDClient
}

func TaskContextInfoFromPD(state *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		c, ok := cm.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			// We have to retry here because the store may be removed and cannot trigger the changes
			return task.Retry(defaultTaskWaitDuration).With("pd client is not registered")
		}
		state.PDClient = c

		if !c.HasSynced() {
			// We have to retry here because the store may be removed and cannot trigger the changes
			return task.Retry(defaultTaskWaitDuration).With("store info is not synced, just wait for next sync")
		}

		state.PDSynced = true

		s, err := c.Stores().Get(coreutil.TiKVAdvertiseClientURLs(state.TiKV()))
		if err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("failed to get store info: %v", err)
			}
			return task.Complete().With("store does not exist")
		}

		state.Store = s
		state.SetStoreState(s.NodeState)
		state.SetLeaderCount(s.LeaderCount)
		state.SetRegionCount(s.RegionCount)
		state.SetStoreBusy(s.IsBusy)
		state.IsStoreReady = IsStoreReady(state)
		pod := state.Pod()
		if state.IsStoreReady && pod != nil && statefulset.IsPodAvailable(pod, minReadySeconds, metav1.Now()) {
			state.SetHealthy()
		}

		if state.FeatureGates().Enabled(metav1alpha1.UseTiKVReadyAPI) {
			// always set it as healthy
			state.SetHealthy()
		}

		// TODO: cache evict leader scheduler info, then we don't need to check suspend here
		if coreutil.ShouldSuspendCompute(state.Cluster()) {
			return task.Complete().With("cluster is suspending")
		}
		scheduler, err := state.PDClient.Underlay().GetEvictLeaderScheduler(ctx, state.Store.ID)
		if err != nil {
			return task.Fail().With("pd is unexpectedly crashed: %v", err)
		}
		if scheduler != "" {
			state.LeaderEvicting = true
		}

		return task.Complete().With("get store info, state: %s, leader count: %v, region count: %v, busy: %v, leader evicting: %v",
			state.GetStoreState(),
			state.GetLeaderCount(),
			state.GetRegionCount(),
			state.IsStoreBusy(),
			state.LeaderEvicting,
		)
	})
}
