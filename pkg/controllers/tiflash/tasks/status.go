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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	defaultTaskWaitDuration = 5 * time.Second
)

// TODO(liubo02): extract to common task
//
//nolint:gocyclo // refactor is possible
func TaskStatus(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		needUpdate := false
		tiflash := state.TiFlash()
		pod := state.Pod()
		// TODO(liubo02): simplify it
		var healthy bool
		if pod != nil &&
			statefulset.IsPodRunningAndReady(pod) &&
			!state.PodIsTerminating &&
			state.Store.NodeState == v1alpha1.StoreStateServing {
			healthy = true
		}
		needUpdate = syncHealthCond(tiflash, healthy) || needUpdate
		needUpdate = syncSuspendCond(tiflash) || needUpdate
		needUpdate = SetIfChanged(&tiflash.Status.ID, state.StoreID) || needUpdate
		needUpdate = SetIfChanged(&tiflash.Status.State, state.StoreState) || needUpdate

		needUpdate = SetIfChanged(&tiflash.Status.ObservedGeneration, tiflash.Generation) || needUpdate
		needUpdate = SetIfChanged(&tiflash.Status.UpdateRevision, tiflash.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate

		if healthy {
			needUpdate = SetIfChanged(&tiflash.Status.CurrentRevision, pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, tiflash); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		// TODO: use a condition to refactor it
		if tiflash.Status.ID == "" || tiflash.Status.State != v1alpha1.StoreStateServing || !v1alpha1.IsUpToDate(tiflash) {
			// can we only rely on the PD member events for this condition?
			// TODO(liubo02): change to task.Wait
			return task.Retry(defaultTaskWaitDuration).With("tiflash may not be initialized, retry")
		}

		return task.Complete().With("status is synced")
	})
}

func syncHealthCond(tiflash *v1alpha1.TiFlash, healthy bool) bool {
	var (
		status = metav1.ConditionFalse
		reason = "Unhealthy"
		msg    = "instance is not healthy"
	)
	if healthy {
		status = metav1.ConditionTrue
		reason = "Healthy"
		msg = "instance is healthy"
	}

	return meta.SetStatusCondition(&tiflash.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondHealth,
		Status:             status,
		ObservedGeneration: tiflash.Generation,
		Reason:             reason,
		Message:            msg,
	})
}

func syncSuspendCond(tiflash *v1alpha1.TiFlash) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&tiflash.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: tiflash.Generation,
		Reason:             v1alpha1.ReasonUnsuspended,
		Message:            "instace is not suspended",
	})
}

// TODO: move to utils
func SetIfChanged[T comparable](dst *T, src T) bool {
	if src == *new(T) {
		return false
	}
	if *dst != src {
		*dst = src
		return true
	}

	return false
}
