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

	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskEvictLeader(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("EvictLeader", func(ctx context.Context) task.Result {
		switch {
		case state.Store == nil:
			return task.Complete().With("store has been deleted or not created")
		case state.PodIsTerminating:
			if !state.LeaderEvicting {
				if err := state.PDClient.BeginEvictLeader(ctx, state.StoreID); err != nil {
					return task.Fail().With("cannot add evict leader scheduler: %v", err)
				}
			}
			return task.Complete().With("ensure evict leader scheduler exists")
		default:
			if state.LeaderEvicting {
				if err := state.PDClient.EndEvictLeader(ctx, state.StoreID); err != nil {
					return task.Fail().With("cannot remove evict leader scheduler: %v", err)
				}
			}
			return task.Complete().With("ensure evict leader scheduler doesn't exist")
		}
	})
}
