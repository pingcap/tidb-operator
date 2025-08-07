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
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	defaultLeaderEvictTimeout = 5 * time.Minute
)

// TaskOfflineStore handles the two-step store deletion process based on spec.offline field.
// This implements the state machine for offline operations: Pending -> Active -> Completed/Failed/Canceled.
func TaskOfflineStore(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("OfflineTiKVStore", func(ctx context.Context) task.Result {
		if !state.PDSynced {
			return task.Wait().With("pd is not synced")
		}
		tikv := state.Instance()
		if tikv.GetDeletionTimestamp().IsZero() {
			return task.Complete().With("tikv is not deleting, no need to offline the store")
		}
		tikv.Spec.Offline = true
		return common.TaskOfflineStoreStateMachine(ctx, state, tikv, "tikv", &evictLeaderHook{state})
	})
}

type evictLeaderHook struct {
	*ReconcileContext
}

var _ common.StoreOfflineHook = &evictLeaderHook{}

func (h *evictLeaderHook) BeforeDeleteStore(ctx context.Context) (wait bool, err error) {
	logger := logr.FromContextOrDiscard(ctx).WithName("EvictLeaderHook")

	if !h.LeaderEvicting {
		logger.Info("evict leader before deletion")
		if err = h.PDClient.Underlay().BeginEvictLeader(ctx, h.Store.ID); err != nil {
			logger.Error(err, "failed to evict leader")
		}
		return true, err
	}

	var reason string
	delTime, leaderCnt := h.TiKV().GetDeletionTimestamp(), h.GetLeaderCount()
	switch {
	// leaders evicted
	case h.LeaderEvicting && leaderCnt == 0:
		reason = "leaders have been all evicted"
	case !delTime.IsZero() && delTime.Add(defaultLeaderEvictTimeout).Before(time.Now()):
		reason = "leader eviction timeout"
	}
	if reason != "" {
		logger.Info("can delete store", "reason", reason)
		return false, nil
	}

	logger.Info("wait for evicting leader", "leaders", leaderCnt)
	return true, nil
}

func (h *evictLeaderHook) AfterStoreIsDeleted(ctx context.Context) (wait bool, err error) {
	return endEvictLeader(ctx, h.PDClient, h.LeaderEvicting, h.TiKV().Status.ID)
}

func (h *evictLeaderHook) AfterCancelDeleteStore(ctx context.Context) (wait bool, err error) {
	return endEvictLeader(ctx, h.PDClient, h.LeaderEvicting, h.TiKV().Status.ID)
}

func endEvictLeader(ctx context.Context, pdClient pd.PDClient, leaderEvicting bool, storeID string) (wait bool, err error) {
	if leaderEvicting && storeID != "" {
		if err = pdClient.Underlay().EndEvictLeader(ctx, storeID); err != nil {
			return true, fmt.Errorf("failed to end evict leader")
		}
	}
	return false, nil
}
