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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	defaultLeaderEvictTimeout = 5 * time.Minute
	jitter                    = 10 * time.Second
)

func TaskOfflineStore(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("OfflineStore", func(ctx context.Context) task.Result {
		if !state.PDSynced {
			return task.Wait().With("pd is not synced")
		}

		tikv := state.Object()
		isOffline := coreutil.IsOffline[scope.TiKV](tikv)

		// If the store is nil, it means the store has been deleted or not created yet.
		// No need to check if leaders are evicted.
		if state.Store != nil && isOffline {
			var reason string
			beginTime := getBeginEvictLeaderTime(tikv)
			switch {
			case state.LeaderEvicting && state.GetLeaderCount() == 0:
				reason = "leaders have been all evicted"
			case beginTime != nil && beginTime.Add(defaultLeaderEvictTimeout).Before(time.Now()):
				reason = "leader eviction timeout"
			}

			if reason == "" {
				return task.Retry(defaultLeaderEvictTimeout+jitter).
					With("waiting for leaders evicted or timeout, current leader count: %d", state.GetLeaderCount())
			}
		}

		if err := common.TaskOfflineStore[scope.TiKV](
			ctx,
			state.PDClient.Underlay(),
			tikv,
			state.GetStoreID(),
			state.GetStoreState(),
		); err != nil {
			// refresh store state
			state.PDClient.Stores().Refresh()

			if task.IsWaitError(err) {
				return task.Wait().With("%v", err)
			}

			return task.Fail().With("%v", err)
		}

		return task.Complete().With("offline is completed or no need, spec.offline: %v", isOffline)
	})
}

// getBeginEvictLeaderTime returns the time when the leader eviction started.
// If the condition is not found or the status is not False, it returns nil.
func getBeginEvictLeaderTime(tikv *v1alpha1.TiKV) *time.Time {
	cond := meta.FindStatusCondition(tikv.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted)
	if cond != nil && cond.Status == metav1.ConditionFalse {
		return ptr.To(cond.LastTransitionTime.Time)
	}
	return nil
}
