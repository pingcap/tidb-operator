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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskStatusSuspend(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("StatusSuspend", func(ctx context.Context) task.Result {
		suspendStatus := metav1.ConditionFalse
		suspendMessage := "pd group is suspending"

		suspended := true
		pdg := state.PDGroup()
		pds := state.PDSlice()
		for _, pd := range pds {
			if !meta.IsStatusConditionTrue(pd.Status.Conditions, v1alpha1.PDCondSuspended) {
				suspended = false
			}
		}
		if suspended {
			suspendStatus = metav1.ConditionTrue
			suspendMessage = "pd group is suspended"
		}

		needUpdate := meta.SetStatusCondition(&pdg.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.PDGroupCondSuspended,
			Status:             suspendStatus,
			ObservedGeneration: pdg.Generation,
			Reason:             v1alpha1.PDGroupSuspendReason,
			Message:            suspendMessage,
		})
		needUpdate = SetIfChanged(&pdg.Status.ObservedGeneration, pdg.Generation) || needUpdate

		if needUpdate {
			if err := c.Status().Update(ctx, state.PDGroup()); err != nil {
				return task.Fail().With("cannot update status: %v", err)
			}
		}

		return task.Complete().With("status of suspend pd is updated")
	})
}

func TaskStatus(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		pdg := state.PDGroup()

		suspendStatus := metav1.ConditionFalse
		suspendMessage := "pd group is not suspended"
		needUpdate := meta.SetStatusCondition(&pdg.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.PDGroupCondSuspended,
			Status:             suspendStatus,
			ObservedGeneration: pdg.Generation,
			Reason:             v1alpha1.PDGroupSuspendReason,
			Message:            suspendMessage,
		})

		replicas, readyReplicas, updateReplicas, currentReplicas := calcReplicas(state.PDSlice(), state.CurrentRevision, state.UpdateRevision)

		// all instances are updated
		if updateReplicas == replicas {
			// update current revision
			state.CurrentRevision = state.UpdateRevision
			// update status of pdg version
			// TODO(liubo02): version of a group is hard to understand
			// We need to change it to a more meaningful field
			needUpdate = SetIfChanged(&pdg.Status.Version, pdg.Spec.Version) || needUpdate
		}

		needUpdate = SetIfChanged(&pdg.Status.ObservedGeneration, pdg.Generation) || needUpdate
		needUpdate = SetIfChanged(&pdg.Status.Replicas, replicas) || needUpdate
		needUpdate = SetIfChanged(&pdg.Status.ReadyReplicas, readyReplicas) || needUpdate
		needUpdate = SetIfChanged(&pdg.Status.UpdateReplicas, updateReplicas) || needUpdate
		needUpdate = SetIfChanged(&pdg.Status.CurrentReplicas, currentReplicas) || needUpdate
		needUpdate = SetIfChanged(&pdg.Status.CurrentRevision, state.CurrentRevision) || needUpdate
		needUpdate = SetIfChanged(&pdg.Status.UpdateRevision, state.UpdateRevision) || needUpdate
		needUpdate = SetIfChanged(&pdg.Status.CollisionCount, state.CollisionCount) || needUpdate

		if needUpdate {
			if err := c.Status().Update(ctx, pdg); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		if !state.IsBootstrapped {
			return task.Wait().With("pd group may not be available, wait")
		}

		return task.Complete().With("status is synced")
	})
}

func calcReplicas(pds []*v1alpha1.PD, currentRevision, updateRevision string) (
	replicas,
	readyReplicas,
	updateReplicas,
	currentReplicas int32,
) {
	for _, peer := range pds {
		replicas++
		if peer.IsHealthy() {
			readyReplicas++
		}
		if peer.CurrentRevision() == currentRevision {
			currentReplicas++
		}
		if peer.CurrentRevision() == updateRevision {
			updateReplicas++
		}
	}

	return
}

func SetIfChanged[T comparable](dst *T, src T) bool {
	if *dst != src {
		*dst = src
		return true
	}

	return false
}
