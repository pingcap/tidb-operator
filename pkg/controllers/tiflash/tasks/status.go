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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

func TaskStatusSuspend(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("StatusSuspend", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		rtx.TiFlash.Status.ObservedGeneration = rtx.TiFlash.Generation

		var (
			suspendStatus  = metav1.ConditionFalse
			suspendMessage = "tiflash is suspending"

			// when suspending, the health status should be false
			healthStatus  = metav1.ConditionFalse
			healthMessage = "tiflash is not healthy"
		)

		if rtx.Pod == nil {
			suspendStatus = metav1.ConditionTrue
			suspendMessage = "tiflash is suspended"
		}
		needUpdate := meta.SetStatusCondition(&rtx.TiFlash.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.TiFlashCondSuspended,
			Status:             suspendStatus,
			ObservedGeneration: rtx.TiFlash.Generation,
			// TODO: use different reason for suspending and suspended
			Reason:  v1alpha1.TiFlashSuspendReason,
			Message: suspendMessage,
		})

		needUpdate = meta.SetStatusCondition(&rtx.TiFlash.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.TiFlashCondHealth,
			Status:             healthStatus,
			ObservedGeneration: rtx.TiFlash.Generation,
			Reason:             v1alpha1.TiFlashHealthReason,
			Message:            healthMessage,
		}) || needUpdate

		if needUpdate {
			if err := c.Status().Update(ctx, rtx.TiFlash); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		return task.Complete().With("status of suspend tiflash is updated")
	})
}

type TaskStatus struct {
	Client client.Client
	Logger logr.Logger
}

func NewTaskStatus(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskStatus{
		Client: c,
		Logger: logger,
	}
}

func (*TaskStatus) Name() string {
	return "Status"
}

//nolint:gocyclo // refactor is possible
func (t *TaskStatus) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	var (
		healthStatus  = metav1.ConditionFalse
		healthMessage = "tiflash is not healthy"

		suspendStatus  = metav1.ConditionFalse
		suspendMessage = "tiflash is not suspended"

		needUpdate = false
	)

	if rtx.StoreID != "" {
		if rtx.TiFlash.Status.ID != rtx.StoreID {
			rtx.TiFlash.Status.ID = rtx.StoreID
			needUpdate = true
		}

		info, err := rtx.PDClient.GetStore(ctx, rtx.StoreID)
		if err == nil && info != nil && info.Store != nil {
			rtx.StoreState = info.Store.NodeState.String()
		} else {
			t.Logger.Error(err, "failed to get tiflash store info", "store", rtx.StoreID)
		}
	}
	if rtx.StoreState != "" && rtx.TiFlash.Status.State != rtx.StoreState {
		rtx.TiFlash.Status.State = rtx.StoreState
		needUpdate = true
	}

	needUpdate = meta.SetStatusCondition(&rtx.TiFlash.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiFlashCondSuspended,
		Status:             suspendStatus,
		ObservedGeneration: rtx.TiFlash.Generation,
		Reason:             v1alpha1.TiFlashSuspendReason,
		Message:            suspendMessage,
	}) || needUpdate

	if needUpdate || !v1alpha1.IsReconciled(rtx.TiFlash) ||
		rtx.TiFlash.Status.UpdateRevision != rtx.TiFlash.Labels[v1alpha1.LabelKeyInstanceRevisionHash] {
		rtx.TiFlash.Status.ObservedGeneration = rtx.TiFlash.Generation
		rtx.TiFlash.Status.UpdateRevision = rtx.TiFlash.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
		needUpdate = true
	}

	if rtx.Pod == nil || rtx.PodIsTerminating {
	} else if statefulset.IsPodRunningAndReady(rtx.Pod) && rtx.StoreState == v1alpha1.StoreStateServing {
		rtx.Healthy = true
		if rtx.TiFlash.Status.CurrentRevision != rtx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash] {
			rtx.TiFlash.Status.CurrentRevision = rtx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
			needUpdate = true
		}
	} else {
		rtx.Healthy = false
	}

	if rtx.Healthy {
		healthStatus = metav1.ConditionTrue
		healthMessage = "tiflash is healthy"
	}
	needUpdate = meta.SetStatusCondition(&rtx.TiFlash.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiFlashCondHealth,
		Status:             healthStatus,
		ObservedGeneration: rtx.TiFlash.Generation,
		Reason:             v1alpha1.TiFlashHealthReason,
		Message:            healthMessage,
	}) || needUpdate

	if needUpdate {
		if err := t.Client.Status().Update(ctx, rtx.TiFlash); err != nil {
			return task.Fail().With("cannot update status: %w", err)
		}
	}

	// TODO: use a condition to refactor it
	if rtx.TiFlash.Status.ID == "" || rtx.TiFlash.Status.State != v1alpha1.StoreStateServing || !v1alpha1.IsUpToDate(rtx.TiFlash) {
		return task.Fail().With("tiflash may not be initialized, retry")
	}

	return task.Complete().With("status is synced")
}
