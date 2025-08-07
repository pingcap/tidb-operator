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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
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
		if state.Object().GetDeletionTimestamp().IsZero() {
			return task.Complete().With("tikv is not deleting, no need to offline the store")
		}
		if !state.IsStoreUp() {
			return task.Wait().With("store has been %s, no need to offline it", state.GetStoreState())
		}
		var reason string
		delTime := state.Object().GetDeletionTimestamp()
		switch {
		// leaders evicted
		case state.LeaderEvicting && state.GetLeaderCount() == 0:
			reason = "leaders have been all evicted"
		case !delTime.IsZero() && delTime.Add(defaultLeaderEvictTimeout).Before(time.Now()):
			reason = "leader eviction timeout"
		}

		if reason != "" {
			if err := state.PDClient.Underlay().DeleteStore(ctx, state.Store.ID); err != nil {
				return task.Fail().With("cannot delete store %s: %w", state.Store.ID, err)
			}
			state.SetStoreState(v1alpha1.StoreStateRemoving)
			return task.Wait().With("%s, try to removing the store", reason)
		}

		return task.Retry(defaultLeaderEvictTimeout + jitter).With("waiting for leaders evicted or timeout")
	})
}
