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
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

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

//nolint:gocyclo // refactor if possible
func (t *TaskStatus) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	availStatus := metav1.ConditionFalse
	availMessage := "tidb group is not available"
	if rtx.IsAvailable {
		availStatus = metav1.ConditionTrue
		availMessage = "tidb group is available"
	}
	conditionChanged := meta.SetStatusCondition(&rtx.TiDBGroup.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiDBGroupCondAvailable,
		Status:             availStatus,
		ObservedGeneration: rtx.TiDBGroup.Generation,
		Reason:             v1alpha1.TiDBGroupAvailableReason,
		Message:            availMessage,
	})

	suspendStatus := metav1.ConditionFalse
	suspendMessage := "tidb group is not suspended"
	if rtx.Suspended {
		suspendStatus = metav1.ConditionTrue
		suspendMessage = "tidb group is suspended"
	} else if rtx.Cluster.ShouldSuspendCompute() {
		suspendMessage = "tidb group is suspending"
	}
	conditionChanged = meta.SetStatusCondition(&rtx.TiDBGroup.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiDBGroupCondSuspended,
		Status:             suspendStatus,
		ObservedGeneration: rtx.TiDBGroup.Generation,
		Reason:             v1alpha1.TiDBGroupSuspendReason,
		Message:            suspendMessage,
	}) || conditionChanged

	// Update the current revision if all instances are synced.
	if int(rtx.TiDBGroup.GetDesiredReplicas()) == len(rtx.TiDBs) && v1alpha1.AllInstancesSynced(rtx.TiDBs, rtx.UpdateRevision) {
		conditionChanged = true
		rtx.CurrentRevision = rtx.UpdateRevision
		rtx.TiDBGroup.Status.Version = rtx.TiDBGroup.Spec.Version
	}
	var readyReplicas int32
	for _, tidb := range rtx.TiDBs {
		if tidb.IsHealthy() {
			readyReplicas++
		}
	}

	if conditionChanged || rtx.TiDBGroup.Status.ReadyReplicas != readyReplicas ||
		rtx.TiDBGroup.Status.Replicas != int32(len(rtx.TiDBs)) || //nolint:gosec // expected type conversion
		!v1alpha1.IsReconciled(rtx.TiDBGroup) ||
		v1alpha1.StatusChanged(rtx.TiDBGroup, rtx.CommonStatus) {
		rtx.TiDBGroup.Status.ReadyReplicas = readyReplicas
		rtx.TiDBGroup.Status.Replicas = int32(len(rtx.TiDBs)) //nolint:gosec// expected type conversion
		rtx.TiDBGroup.Status.ObservedGeneration = rtx.TiDBGroup.Generation
		rtx.TiDBGroup.Status.CurrentRevision = rtx.CurrentRevision
		rtx.TiDBGroup.Status.UpdateRevision = rtx.UpdateRevision
		rtx.TiDBGroup.Status.CollisionCount = rtx.CollisionCount

		if err := t.Client.Status().Update(ctx, rtx.TiDBGroup); err != nil {
			return task.Fail().With("cannot update status: %w", err)
		}
	}

	if !rtx.IsAvailable && !rtx.Suspended {
		return task.Fail().With("tidb group may not be available, requeue to retry")
	}

	return task.Complete().With("status is synced")
}
