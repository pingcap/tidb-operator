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
	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

type TaskEvictLeader struct {
	Client client.Client
	Logger logr.Logger
}

func NewTaskEvictLeader(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskEvictLeader{
		Client: c,
		Logger: logger,
	}
}

func (*TaskEvictLeader) Name() string {
	return "EvictLeader"
}

func (*TaskEvictLeader) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	switch {
	case rtx.Store == nil:
		return task.Complete().With("store has been deleted or not created")
	case rtx.PodIsTerminating:
		if !rtx.LeaderEvicting {
			if err := rtx.PDClient.BeginEvictLeader(ctx, rtx.StoreID); err != nil {
				return task.Fail().With("cannot add evict leader scheduler: %v", err)
			}
		}
		return task.Complete().With("ensure evict leader scheduler exists")
	default:
		if rtx.LeaderEvicting {
			if err := rtx.PDClient.EndEvictLeader(ctx, rtx.StoreID); err != nil {
				return task.Fail().With("cannot remove evict leader scheduler: %v", err)
			}
		}
		return task.Complete().With("ensure evict leader scheduler doesn't exist")
	}
}
