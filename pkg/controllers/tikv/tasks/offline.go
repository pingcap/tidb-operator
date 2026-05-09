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
	"time"

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
		if state.Store != nil && isOffline {
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

		if err := common.TaskOfflineStore[scope.TiKV](
			ctx,
			pc.Underlay(),
			tikv,
			state.GetStoreID(),
			state.GetStoreState(),
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
