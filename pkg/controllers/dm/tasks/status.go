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
		dm := state.DM()
		pod := state.Pod()
		ready := coreutil.IsReady[scope.DM](dm)

		needUpdate = syncSuspendCond(dm) || needUpdate

		needUpdate = compare.SetIfNotEmptyAndChanged(&dm.Status.MemberID, state.MemberID) || needUpdate
		needUpdate = compare.SetIfChanged(&dm.Status.ObservedGeneration, dm.Generation) || needUpdate
		needUpdate = compare.SetIfNotEmptyAndChanged(
			&dm.Status.UpdateRevision,
			dm.Labels[v1alpha1.LabelKeyInstanceRevisionHash],
		) || needUpdate

		if ready && pod != nil {
			needUpdate = compare.SetIfNotEmptyAndChanged(
				&dm.Status.CurrentRevision,
				pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash],
			) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, dm); err != nil {
				return task.Fail().With("cannot update status: %v", err)
			}
		}

		if !ready {
			if state.IsPodTerminating() {
				return task.Retry(defaultTaskWaitDuration).With("pod may be terminating, requeue to retry")
			}
			return task.Retry(defaultTaskWaitDuration).With("dm-master is not ready, requeue to retry")
		}

		return task.Complete().With("status is synced")
	})
}

func syncSuspendCond(dm *v1alpha1.DM) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&dm.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: dm.Generation,
		Reason:             v1alpha1.ReasonUnsuspended,
		Message:            "instance is not suspended",
	})
}
