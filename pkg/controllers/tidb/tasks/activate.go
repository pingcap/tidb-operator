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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/tidbapi/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskActivate(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Activate", func(ctx context.Context) task.Result {
		obj := state.Object()
		if obj.Spec.Mode == v1alpha1.TiDBModeStandBy {
			return task.Complete().With("tidb is standby")
		}
		if !state.IsHealthy() {
			return task.Retry(defaultTaskWaitDuration).With("tidb is unhealthy ")
		}
		status, err := state.TiDBClient.GetPoolStatus(ctx)
		if err != nil {
			return task.Fail().With("cannot get pool status: %v", err)
		}
		if status.State != tidbapi.PoolStateActivated {
			if err := state.TiDBClient.Activate(ctx, obj.Spec.Keyspace); err != nil {
				return task.Fail().With("cannot activate: %v", err)
			}
		}
		return task.Complete().With("tidb is activated")
	})
}
