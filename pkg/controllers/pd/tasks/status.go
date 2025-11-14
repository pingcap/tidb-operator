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

//nolint:gocyclo // refactor if possible
func TaskStatus(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		needUpdate := state.IsStatusChanged()
		pd := state.PD()
		pod := state.Pod()
		ready := coreutil.IsReady[scope.PD](pd)

		needUpdate = syncInitializedCond(pd, state.Initialized) || needUpdate
		needUpdate = syncSuspendCond(pd) || needUpdate

		needUpdate = compare.SetIfNotEmptyAndChanged(&pd.Status.ID, state.MemberID) || needUpdate
		needUpdate = compare.SetIfChanged(&pd.Status.IsLeader, state.IsLeader) || needUpdate
		needUpdate = compare.SetIfChanged(&pd.Status.ObservedGeneration, pd.Generation) || needUpdate
		needUpdate = compare.SetIfNotEmptyAndChanged(&pd.Status.UpdateRevision, pd.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate
		if ready {
			needUpdate = compare.SetIfNotEmptyAndChanged(&pd.Status.CurrentRevision, pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, pd); err != nil {
				return task.Fail().With("cannot update status: %v", err)
			}
		}
		if state.IsPodTerminating() {
			//nolint:mnd // refactor to use a constant
			return task.Retry(5 * time.Second).With("pod is terminating, retry after it's terminated")
		}

		if !ready || !state.Initialized {
			return task.Wait().With("pd may not be initialized or healthy, wait for next event")
		}
		// TODO(csuzhangxc): if we reach here, is "ClusterID" always set?

		return task.Complete().With("status is synced")
	})
}

func syncSuspendCond(pd *v1alpha1.PD) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&pd.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: pd.Generation,
		Reason:             v1alpha1.ReasonUnsuspended,
		Message:            "instance is not suspended",
	})
}

// TODO(liubo02): remove it, it seems useless
// Status of this condition can only transfer as the below
// 1. false => true
// 2. true <=> unknown
func syncInitializedCond(pd *v1alpha1.PD, initialized bool) bool {
	cond := meta.FindStatusCondition(pd.Status.Conditions, v1alpha1.PDCondInitialized)
	status := metav1.ConditionUnknown
	reason := "Unavailable"
	msg := "pd is unavailable"
	switch {
	case initialized:
		status = metav1.ConditionTrue
		reason = "Initialized"
		msg = "instance is initialized"
	case !initialized && (cond == nil || cond.Status == metav1.ConditionFalse):
		status = metav1.ConditionFalse
		reason = "Uninitialized"
		msg = "instance has not been initialized yet"
	}

	return meta.SetStatusCondition(&pd.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.PDCondInitialized,
		Status:             status,
		ObservedGeneration: pd.Generation,
		Reason:             reason,
		Message:            msg,
	})
}
