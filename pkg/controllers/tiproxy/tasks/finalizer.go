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

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskDrainPodForDelete(state State, c client.Client) task.Task {
	return task.NameTaskFunc("DrainPodForDelete", func(ctx context.Context) task.Result {
		pod := state.Pod()
		if pod == nil {
			return task.Complete().With("pod doesn't exist")
		}

		retryAfter, err := drainPodForGracefulShutdown(ctx, c, state, pod, false)
		if err != nil {
			return task.Fail().With("cannot delete pod of tiproxy: %v", err)
		}
		if retryAfter > 0 {
			return task.Retry(retryAfter).With("wait for tiproxy pod to be deleted")
		}
		return task.Complete().With("pod is deleted")
	})
}
