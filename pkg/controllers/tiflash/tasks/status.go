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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/compare"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	defaultTaskWaitDuration = 5 * time.Second
)

// TODO(liubo02): extract to common task
//
//nolint:gocyclo,staticcheck // refactor is possible
func TaskStatus(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		needUpdate := state.IsStatusChanged()
		tiflash := state.TiFlash()
		pod := state.Pod()
		ready := coreutil.IsReady[scope.TiFlash](tiflash)
		needUpdate = syncSuspendCond(tiflash) || needUpdate
		if state.Store != nil {
			needUpdate = compare.SetIfNotEmptyAndChanged(&tiflash.Status.ID, state.Store.ID) || needUpdate
		}
		needUpdate = compare.SetIfNotEmptyAndChanged(&tiflash.Status.State, state.GetStoreState()) || needUpdate

		needUpdate = compare.SetIfChanged(&tiflash.Status.ObservedGeneration, tiflash.Generation) || needUpdate
		needUpdate = compare.SetIfNotEmptyAndChanged(
			&tiflash.Status.UpdateRevision,
			tiflash.Labels[v1alpha1.LabelKeyInstanceRevisionHash],
		) || needUpdate

		if ready {
			needUpdate = compare.SetIfNotEmptyAndChanged(
				&tiflash.Status.CurrentRevision,
				pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash],
			) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, tiflash); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		if state.IsPodTerminating() {
			return task.Retry(defaultTaskWaitDuration).With("pod may be terminating, requeue to retry")
		}

		// TODO: use a condition to refactor it
		if !ready || tiflash.Status.ID == "" {
			return task.Wait().With("tiflash may not be synced, wait")
		}

		return task.Complete().With("status is synced")
	})
}

func TaskStoreStatus(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("StoreStatus", func(ctx context.Context) task.Result {
		needUpdate := state.IsStatusChanged()
		tiflash := state.TiFlash()
		if state.Store != nil {
			needUpdate = compare.SetIfChanged(&tiflash.Status.ID, state.Store.ID) || needUpdate
		}
		needUpdate = compare.SetIfChanged(&tiflash.Status.State, state.GetStoreState()) || needUpdate
		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("store state is changed")
		}

		return task.Complete().With("store state is not changed")
	})
}

func syncSuspendCond(tiflash *v1alpha1.TiFlash) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&tiflash.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: tiflash.Generation,
		Reason:             v1alpha1.ReasonUnsuspended,
		Message:            "instance is not suspended",
	})
}
