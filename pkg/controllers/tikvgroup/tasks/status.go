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

func (t *TaskStatus) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	suspendStatus := metav1.ConditionFalse
	suspendMessage := "tikv group is not suspended"
	if rtx.Suspended {
		suspendStatus = metav1.ConditionTrue
		suspendMessage = "tikv group is suspended"
	} else if rtx.Cluster.ShouldSuspendCompute() {
		suspendMessage = "tikv group is suspending"
	}
	conditionChanged := meta.SetStatusCondition(&rtx.TiKVGroup.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiKVGroupCondSuspended,
		Status:             suspendStatus,
		ObservedGeneration: rtx.TiKVGroup.Generation,
		Reason:             v1alpha1.TiKVGroupSuspendReason,
		Message:            suspendMessage,
	})

	// Update the current revision if all instances are synced.
	if int(rtx.TiKVGroup.GetDesiredReplicas()) == len(rtx.Peers) && v1alpha1.AllInstancesSynced(rtx.Peers, rtx.UpdateRevision) {
		if rtx.CurrentRevision != rtx.UpdateRevision || rtx.TiKVGroup.Status.Version != rtx.TiKVGroup.Spec.Version {
			rtx.CurrentRevision = rtx.UpdateRevision
			rtx.TiKVGroup.Status.Version = rtx.TiKVGroup.Spec.Version
			conditionChanged = true
		}
	}
	var readyReplicas int32
	for _, peer := range rtx.Peers {
		if peer.IsHealthy() {
			readyReplicas++
		}
	}

	if conditionChanged || rtx.TiKVGroup.Status.ReadyReplicas != readyReplicas ||
		rtx.TiKVGroup.Status.Replicas != int32(len(rtx.Peers)) || //nolint:gosec // expected type conversion
		!v1alpha1.IsReconciled(rtx.TiKVGroup) ||
		v1alpha1.StatusChanged(rtx.TiKVGroup, rtx.CommonStatus) {
		rtx.TiKVGroup.Status.ReadyReplicas = readyReplicas
		rtx.TiKVGroup.Status.Replicas = int32(len(rtx.Peers)) //nolint:gosec // expected type conversion
		rtx.TiKVGroup.Status.ObservedGeneration = rtx.TiKVGroup.Generation
		rtx.TiKVGroup.Status.CurrentRevision = rtx.CurrentRevision
		rtx.TiKVGroup.Status.UpdateRevision = rtx.UpdateRevision
		rtx.TiKVGroup.Status.CollisionCount = rtx.CollisionCount

		if err := t.Client.Status().Update(ctx, rtx.TiKVGroup); err != nil {
			return task.Fail().With("cannot update status: %w", err)
		}
	}

	return task.Complete().With("status is synced")
}
