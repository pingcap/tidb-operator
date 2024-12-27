// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskBoot(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Boot", func(ctx context.Context) task.Result {
		pdg := state.PDGroup()
		if !state.IsBootstrapped && !pdg.Spec.Bootstrapped {
			return task.Wait().With("skip the task and wait until the pd svc is available")
		}
		if !pdg.Spec.Bootstrapped {
			pdg.Spec.Bootstrapped = true
			if err := c.Update(ctx, pdg); err != nil {
				return task.Fail().With("pd cluster is available but not marked as bootstrapped: %w", err)
			}
		}

		return task.Complete().With("pd cluster is bootstrapped")
	})
}
