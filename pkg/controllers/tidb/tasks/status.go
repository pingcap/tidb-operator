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
func TaskStatus(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		needUpdate := false
		tidb := state.TiDB()
		pod := state.Pod()
		// TODO(liubo02): simplify it
		var healthy bool

		if pod != nil &&
			statefulset.IsPodRunningAndReady(pod) &&
			!state.PodIsTerminating &&
			state.Healthy {
			healthy = true
		}

		needUpdate = syncHealthCond(tidb, healthy) || needUpdate
		needUpdate = syncSuspendCond(tidb) || needUpdate

		needUpdate = SetIfChanged(&tidb.Status.ObservedGeneration, tidb.Generation) || needUpdate
		needUpdate = SetIfChanged(&tidb.Status.UpdateRevision, tidb.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate

		if healthy {
			needUpdate = SetIfChanged(&tidb.Status.CurrentRevision, pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, state.TiDB()); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		if !state.Healthy || !v1alpha1.IsUpToDate(state.TiDB()) {
			// can we only rely on Pod status events to trigger the retry?
			// TODO(liubo02): change to task.Wait
			return task.Retry(defaultTaskWaitDuration).With("tidb may not be healthy, requeue to retry")
		}

		return task.Complete().With("status is synced")
	})
}

func syncHealthCond(tidb *v1alpha1.TiDB, healthy bool) bool {
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

	return meta.SetStatusCondition(&tidb.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondHealth,
		Status:             status,
		ObservedGeneration: tidb.Generation,
		Reason:             reason,
		Message:            msg,
	})
}

func syncSuspendCond(tidb *v1alpha1.TiDB) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&tidb.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: tidb.Generation,
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
