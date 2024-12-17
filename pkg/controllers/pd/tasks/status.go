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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

func TaskStatusSuspend(ctx *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("StatusSuspend", func() task.Result {
		ctx.PD.Status.ObservedGeneration = ctx.PD.Generation
		var (
			suspendStatus  = metav1.ConditionFalse
			suspendMessage = "pd is suspending"

			// when suspending, the health status should be false
			healthStatus  = metav1.ConditionFalse
			healthMessage = "pd is not healthy"
		)

		if ctx.Pod == nil {
			suspendStatus = metav1.ConditionTrue
			suspendMessage = "pd is suspended"
		}
		needUpdate := meta.SetStatusCondition(&ctx.PD.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.PDCondSuspended,
			Status:             suspendStatus,
			ObservedGeneration: ctx.PD.Generation,
			// TODO: use different reason for suspending and suspended
			Reason:  v1alpha1.PDSuspendReason,
			Message: suspendMessage,
		})

		needUpdate = meta.SetStatusCondition(&ctx.PD.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.PDCondHealth,
			Status:             healthStatus,
			ObservedGeneration: ctx.PD.Generation,
			Reason:             v1alpha1.PDHealthReason,
			Message:            healthMessage,
		}) || needUpdate
		if needUpdate {
			if err := c.Status().Update(ctx, ctx.PD); err != nil {
				return task.Fail().With("cannot update status: %v", err)
			}
		}

		return task.Complete().With("status of suspend pd is updated")
	})
}

func TaskStatusUnknown() task.Task {
	return task.NameTaskFunc("StatusUnknown", func() task.Result {
		return task.Wait().With("status of the pd is unknown")
	})
}

//nolint:gocyclo // refactor if possible
func TaskStatus(ctx *ReconcileContext, _ logr.Logger, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func() task.Result {
		var (
			healthStatus  = metav1.ConditionFalse
			healthMessage = "pd is not healthy"

			suspendStatus  = metav1.ConditionFalse
			suspendMessage = "pd is not suspended"

			needUpdate = false
		)

		if ctx.MemberID != "" {
			needUpdate = SetIfChanged(&ctx.PD.Status.ID, ctx.MemberID) || needUpdate
		}

		needUpdate = SetIfChanged(&ctx.PD.Status.IsLeader, ctx.IsLeader) || needUpdate
		needUpdate = syncInitializedCond(ctx.PD, ctx.Initialized) || needUpdate

		needUpdate = meta.SetStatusCondition(&ctx.PD.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.PDCondSuspended,
			Status:             suspendStatus,
			ObservedGeneration: ctx.PD.Generation,
			Reason:             v1alpha1.PDSuspendReason,
			Message:            suspendMessage,
		}) || needUpdate

		needUpdate = SetIfChanged(&ctx.PD.Status.ObservedGeneration, ctx.PD.Generation) || needUpdate
		needUpdate = SetIfChanged(&ctx.PD.Status.UpdateRevision, ctx.PD.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate

		if ctx.Pod == nil || ctx.PodIsTerminating {
			ctx.Healthy = false
		} else if statefulset.IsPodRunningAndReady(ctx.Pod) && ctx.Healthy {
			if ctx.PD.Status.CurrentRevision != ctx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash] {
				ctx.PD.Status.CurrentRevision = ctx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
				needUpdate = true
			}
		} else {
			ctx.Healthy = false
		}

		if ctx.Healthy {
			healthStatus = metav1.ConditionTrue
			healthMessage = "pd is healthy"
		}
		needUpdate = meta.SetStatusCondition(&ctx.PD.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.PDCondHealth,
			Status:             healthStatus,
			ObservedGeneration: ctx.PD.Generation,
			Reason:             v1alpha1.PDHealthReason,
			Message:            healthMessage,
		}) || needUpdate

		if needUpdate {
			if err := c.Status().Update(ctx, ctx.PD); err != nil {
				return task.Fail().With("cannot update status: %v", err)
			}
		}
		if ctx.PodIsTerminating {
			//nolint:mnd // refactor to use a constant
			return task.Retry(5 * time.Second).With("pod is terminating, retry after it's terminated")
		}

		if !ctx.Initialized || !ctx.Healthy {
			return task.Wait().With("pd may not be initialized or healthy, wait for next event")
		}

		return task.Complete().With("status is synced")
	})
}

// Status of this condition can only transfer as the below
// 1. false => true
// 2. true <=> unknown
func syncInitializedCond(pd *v1alpha1.PD, initialized bool) bool {
	cond := meta.FindStatusCondition(pd.Status.Conditions, v1alpha1.PDCondInitialized)
	status := metav1.ConditionUnknown
	switch {
	case initialized:
		status = metav1.ConditionTrue
	case !initialized && (cond == nil || cond.Status == metav1.ConditionFalse):
		status = metav1.ConditionFalse
	}

	return meta.SetStatusCondition(&pd.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.PDCondInitialized,
		Status:             status,
		ObservedGeneration: pd.Generation,
		Reason:             "initialized",
		Message:            "instance has joined the cluster",
	})
}

// TODO: move to utils
func SetIfChanged[T comparable](dst *T, src T) bool {
	if *dst != src {
		*dst = src
		return true
	}

	return false
}
