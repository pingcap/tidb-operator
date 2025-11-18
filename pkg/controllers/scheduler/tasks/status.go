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
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/compare"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	defaultTaskWaitDuration = 5 * time.Second
)

func TaskStatus(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		needUpdate := state.IsStatusChanged()
		obj := state.Object()
		pod := state.Pod()
		// ready condition is updated in previous task
		ready := coreutil.IsReady[scope.Scheduler](obj)

		needUpdate = compare.SetIfChanged(&obj.Status.ObservedGeneration, obj.Generation) || needUpdate
		needUpdate = compare.SetIfNotEmptyAndChanged(
			&obj.Status.UpdateRevision,
			obj.Labels[v1alpha1.LabelKeyInstanceRevisionHash],
		) || needUpdate

		if ready {
			needUpdate = compare.SetIfNotEmptyAndChanged(
				&obj.Status.CurrentRevision,
				pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash],
			) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, obj); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		if state.IsPodTerminating() {
			return task.Retry(defaultTaskWaitDuration).With("pod is terminating, retry after it's terminated")
		}

		if !ready {
			return task.Wait().With("scheduler may not be ready, wait")
		}

		return task.Complete().With("status is synced")
	})
}
