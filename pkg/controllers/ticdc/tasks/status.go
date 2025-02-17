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
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	defaultTaskWaitDuration = 5 * time.Second
)

// TODO(liubo02): extract to common task
//
//nolint:gocyclo // refactor if possible
func TaskStatus(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		needUpdate := false
		ticdc := state.TiCDC()
		pod := state.Pod()
		// TODO(liubo02): simplify it
		var healthy bool

		if pod != nil &&
			statefulset.IsPodRunningAndReady(pod) &&
			!state.PodIsTerminating &&
			state.Healthy {
			healthy = true
		}

		needUpdate = syncHealthCond(ticdc, healthy) || needUpdate
		needUpdate = syncSuspendCond(ticdc) || needUpdate

		if state.MemberID != "" {
			needUpdate = SetIfChanged(&ticdc.Status.ID, state.MemberID) || needUpdate
		}
		needUpdate = SetIfChanged(&ticdc.Status.ObservedGeneration, ticdc.Generation) || needUpdate
		needUpdate = SetIfChanged(&ticdc.Status.UpdateRevision, ticdc.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate

		if healthy {
			needUpdate = SetIfChanged(&ticdc.Status.CurrentRevision, pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, state.TiCDC()); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		if !healthy {
			// TODO(liubo02): delete pod should retrigger the events, try to change to wait
			if state.PodIsTerminating {
				return task.Retry(defaultTaskWaitDuration).With("pod may be terminating, requeue to retry")
			}

			if pod != nil && statefulset.IsPodRunningAndReady(pod) {
				return task.Retry(defaultTaskWaitDuration).With("ticdc is not healthy, requeue to retry")
			}

			return task.Wait().With("pod of ticdc is not ready, wait")
		}

		return task.Complete().With("status is synced")
	})
}

func syncHealthCond(ticdc *v1alpha1.TiCDC, healthy bool) bool {
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

	return meta.SetStatusCondition(&ticdc.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondHealth,
		Status:             status,
		ObservedGeneration: ticdc.Generation,
		Reason:             reason,
		Message:            msg,
	})
}

func syncSuspendCond(ticdc *v1alpha1.TiCDC) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&ticdc.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: ticdc.Generation,
		Reason:             v1alpha1.ReasonUnsuspended,
		Message:            "instance is not suspended",
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
