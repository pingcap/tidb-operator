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
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils"
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
		needUpdate := state.IsStatusChanged()
		tidb := state.TiDB()
		pod := state.Pod()

		ready := coreutil.IsReady[scope.TiDB](tidb)

		needUpdate = syncSuspendCond(tidb) || needUpdate

		needUpdate = utils.SetIfChanged(&tidb.Status.ObservedGeneration, tidb.Generation) || needUpdate
		needUpdate = utils.SetIfChanged(&tidb.Status.UpdateRevision, tidb.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate

		if ready {
			needUpdate = utils.SetIfChanged(&tidb.Status.CurrentRevision, pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, state.TiDB()); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		if !ready {
			// TODO(liubo02): delete pod should retrigger the events, try to change to wait
			if state.IsPodTerminating() {
				return task.Retry(defaultTaskWaitDuration).With("pod may be terminating, requeue to retry")
			}

			if pod != nil && statefulset.IsPodRunningAndReady(pod) {
				return task.Retry(defaultTaskWaitDuration).With("tidb is not healthy, requeue to retry")
			}

			return task.Wait().With("pod of tidb is not ready, wait")
		}

		return task.Complete().With("status is synced")
	})
}

func syncSuspendCond(tidb *v1alpha1.TiDB) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&tidb.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: tidb.Generation,
		Reason:             v1alpha1.ReasonUnsuspended,
		Message:            "instance is not suspended",
	})
}
