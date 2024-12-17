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
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	defaultTaskWaitDuration = 5 * time.Second
)

func TaskStatusSuspend(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("StatusSuspend", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		rtx.TiDB.Status.ObservedGeneration = rtx.TiDB.Generation

		var (
			suspendStatus  = metav1.ConditionFalse
			suspendMessage = "tidb is suspending"

			// when suspending, the health status should be false
			healthStatus  = metav1.ConditionFalse
			healthMessage = "tidb is not healthy"
		)

		if rtx.Pod == nil {
			suspendStatus = metav1.ConditionTrue
			suspendMessage = "tidb is suspended"
		}
		needUpdate := meta.SetStatusCondition(&rtx.TiDB.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.TiDBCondSuspended,
			Status:             suspendStatus,
			ObservedGeneration: rtx.TiDB.Generation,
			// TODO: use different reason for suspending and suspended
			Reason:  v1alpha1.TiDBSuspendReason,
			Message: suspendMessage,
		})

		needUpdate = meta.SetStatusCondition(&rtx.TiDB.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.TiDBCondHealth,
			Status:             healthStatus,
			ObservedGeneration: rtx.TiDB.Generation,
			Reason:             v1alpha1.TiDBHealthReason,
			Message:            healthMessage,
		}) || needUpdate

		if needUpdate {
			if err := c.Status().Update(ctx, rtx.TiDB); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		return task.Complete().With("status is suspend tidb is updated")
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

func (t *TaskStatus) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	var (
		healthStatus  = metav1.ConditionFalse
		healthMessage = "tidb is not healthy"

		suspendStatus  = metav1.ConditionFalse
		suspendMessage = "tidb is not suspended"

		needUpdate = false
	)

	conditionChanged := meta.SetStatusCondition(&rtx.TiDB.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiDBCondSuspended,
		Status:             suspendStatus,
		ObservedGeneration: rtx.TiDB.Generation,
		Reason:             v1alpha1.TiDBSuspendReason,
		Message:            suspendMessage,
	})

	if !v1alpha1.IsReconciled(rtx.TiDB) || rtx.TiDB.Status.UpdateRevision != rtx.TiDB.Labels[v1alpha1.LabelKeyInstanceRevisionHash] {
		rtx.TiDB.Status.ObservedGeneration = rtx.TiDB.Generation
		rtx.TiDB.Status.UpdateRevision = rtx.TiDB.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
		needUpdate = true
	}

	if rtx.Pod == nil || rtx.PodIsTerminating {
		rtx.Healthy = false
	} else if statefulset.IsPodRunningAndReady(rtx.Pod) && rtx.Healthy {
		if rtx.TiDB.Status.CurrentRevision != rtx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash] {
			rtx.TiDB.Status.CurrentRevision = rtx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
			needUpdate = true
		}
	} else {
		rtx.Healthy = false
	}

	if rtx.Healthy {
		healthStatus = metav1.ConditionTrue
		healthMessage = "tidb is healthy"
	}
	updateCond := metav1.Condition{
		Type:               v1alpha1.TiDBCondHealth,
		Status:             healthStatus,
		ObservedGeneration: rtx.TiDB.Generation,
		Reason:             v1alpha1.TiDBHealthReason,
		Message:            healthMessage,
	}
	conditionChanged = meta.SetStatusCondition(&rtx.TiDB.Status.Conditions, updateCond) || conditionChanged

	if needUpdate || conditionChanged {
		if err := t.Client.Status().Update(ctx, rtx.TiDB); err != nil {
			return task.Fail().With("cannot update status: %w", err)
		}
	}

	if !rtx.Healthy || !v1alpha1.IsUpToDate(rtx.TiDB) {
		// can we only rely on Pod status events to trigger the retry?
		return task.Retry(defaultTaskWaitDuration).With("tidb may not be healthy, requeue to retry")
	}

	return task.Complete().With("status is synced")
}
