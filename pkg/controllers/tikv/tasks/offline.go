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
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	defaultLeaderEvictTimeout = 5 * time.Minute
	jitter                    = 10 * time.Second
)

func TaskOfflineStore(state *ReconcileContext, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("OfflineStore", func(ctx context.Context) task.Result {
		if !state.PDSynced {
			return task.Wait().With("pd is not synced")
		}

		tikv := state.Object()
		isOffline := coreutil.IsOffline[scope.TiKV](tikv)

		// If the store is nil, it means the store has been deleted or not created yet.
		// No need to check if leaders are evicted.
		if state.Store != nil && isOffline && !state.FeatureGates().Enabled(meta.TiKVSequentialScaleIn) {
			state.ShouldEvictLeader = true

			if err := CheckTiKVLeadersEvictedOrTimeout(tikv, defaultLeaderEvictTimeout); err != nil {
				return task.Retry(defaultLeaderEvictTimeout+jitter).
					With("waiting for leaders evicted or timeout: %v", err)
			}
		}

		pc, ok := state.GetPDClient(m)
		if !ok {
			return task.Wait().With("pd client is not registered")
		}

		isSelected := !state.FeatureGates().Enabled(meta.TiKVSequentialScaleIn) ||
			isSelectedForScaleIn(tikv, state.TiKVSlice())

		if err := common.TaskOfflineStore[scope.TiKV](
			ctx,
			pc.Underlay(),
			tikv,
			state.GetStoreID(),
			state.GetStoreState(),
			isSelected,
		); err != nil {
			// refresh store state
			pc.Stores().Refresh()

			if task.IsWaitError(err) {
				return task.Wait().With("%v", err)
			}

			return task.Fail().With("%v", err)
		}

		return task.Complete().With("offline is completed or no need, spec.offline: %v", isOffline)
	})
}

func isSelectedForScaleIn(current *v1alpha1.TiKV, peers []*v1alpha1.TiKV) bool {
	candidates := make([]*v1alpha1.TiKV, 0, len(peers))
	for _, peer := range peers {
		if peer.Status.State != v1alpha1.StoreStateRemoved &&
			(coreutil.IsOffline[scope.TiKV](peer) || peer.Status.State == v1alpha1.StoreStateRemoving) {
			candidates = append(candidates, peer)
		}
	}
	slices.SortFunc(candidates, func(a, b *v1alpha1.TiKV) int {
		aRemoving, bRemoving := a.Status.State == v1alpha1.StoreStateRemoving, b.Status.State == v1alpha1.StoreStateRemoving
		if aRemoving != bRemoving {
			if aRemoving {
				return -1
			}
			return 1
		}
		if order := a.CreationTimestamp.Compare(b.CreationTimestamp.Time); order != 0 {
			return order
		}
		if order := cmp.Compare(a.Name, b.Name); order != 0 {
			return order
		}
		return cmp.Compare(a.UID, b.UID)
	})
	return len(candidates) > 0 && candidates[0].UID == current.UID
}
