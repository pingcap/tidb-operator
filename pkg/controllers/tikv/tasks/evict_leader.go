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

	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskEvictLeader(state *ReconcileContext, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("EvictLeader", func(ctx context.Context) task.Result {
		pc, ok := state.GetPDClient(m)
		if !ok {
			return task.Wait().With("wait if pd client is not registered")
		}
		switch {
		case !state.PDSynced:
			return task.Wait().With("pd is unsynced")
		case state.Store == nil:
			return task.Complete().With("store has been deleted or not created")
		case state.Instance().IsOffline() || state.IsPodTerminating():
			if !state.LeaderEvicting {
				if err := pc.Underlay().BeginEvictLeader(ctx, state.Store.ID); err != nil {
					return task.Fail().With("cannot add evict leader scheduler: %v", err)
				}
			}
			return task.Complete().With("ensure evict leader scheduler exists")
		default:
			if state.LeaderEvicting {
				if err := pc.Underlay().EndEvictLeader(ctx, state.Store.ID); err != nil {
					return task.Fail().With("cannot remove evict leader scheduler: %v", err)
				}
			}
			return task.Complete().With("ensure evict leader scheduler doesn't exist")
		}
	})
}

// TaskEndEvictLeader only be called when object is deleting and store has been removed
// TODO(liubo02): it's not stable because status.ID may be lost
func TaskEndEvictLeader(state *ReconcileContext, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("EndEvictLeader", func(ctx context.Context) task.Result {
		msg := "ensure evict leader scheduler doesn't exist"
		pc, ok := state.GetPDClient(m)
		if !ok {
			return task.Wait().With("wait if pd client is not registered")
		}
		if storeID := state.TiKV().Status.ID; storeID != "" {
			if err := pc.Underlay().EndEvictLeader(ctx, storeID); err != nil {
				return task.Fail().With("cannot remove evict leader scheduler: %v", err)
			}
		} else {
			msg = "can not get the store id"
		}
		return task.Complete().With(msg)
	})
}
