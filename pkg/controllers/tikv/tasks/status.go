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
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

func TaskStatusSuspend(c client.Client) task.Task[ReconcileContext] {
	return task.NameTaskFunc("StatusSuspend", func(ctx task.Context[ReconcileContext]) task.Result {
		rtx := ctx.Self()
		rtx.TiKV.Status.ObservedGeneration = rtx.TiKV.Generation

		var (
			suspendStatus  = metav1.ConditionFalse
			suspendMessage = "tikv is suspending"

			// when suspending, the health status should be false
			healthStatus  = metav1.ConditionFalse
			healthMessage = "tikv is not healthy"
		)

		if rtx.Pod == nil {
			suspendStatus = metav1.ConditionTrue
			suspendMessage = "tikv is suspended"
		}
		needUpdate := meta.SetStatusCondition(&rtx.TiKV.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.TiKVCondSuspended,
			Status:             suspendStatus,
			ObservedGeneration: rtx.TiKV.Generation,
			// TODO: use different reason for suspending and suspended
			Reason:  v1alpha1.TiKVSuspendReason,
			Message: suspendMessage,
		})

		needUpdate = meta.SetStatusCondition(&rtx.TiKV.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.TiKVCondHealth,
			Status:             healthStatus,
			ObservedGeneration: rtx.TiKV.Generation,
			Reason:             v1alpha1.TiKVHealthReason,
			Message:            healthMessage,
		}) || needUpdate

		if needUpdate {
			if err := c.Status().Update(ctx, rtx.TiKV); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		return task.Complete().With("status of suspend tikv is updated")
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
		healthMessage = "tikv is not healthy"

		suspendStatus  = metav1.ConditionFalse
		suspendMessage = "tikv is not suspended"

		needUpdate = false
	)

	if rtx.StoreID != "" {
		if rtx.TiKV.Status.ID != rtx.StoreID {
			rtx.TiKV.Status.ID = rtx.StoreID
			needUpdate = true
		}

		info, err := rtx.PDClient.GetStore(ctx, rtx.StoreID)
		if err == nil && info != nil && info.Store != nil {
			rtx.StoreState = info.Store.NodeState.String()
		} else {
			t.Logger.Error(err, "failed to get tikv store info", "store", rtx.StoreID)
		}
	}
	if rtx.StoreState != "" && rtx.TiKV.Status.State != rtx.StoreState {
		rtx.TiKV.Status.State = rtx.StoreState
		needUpdate = true
	}

	needUpdate = meta.SetStatusCondition(&rtx.TiKV.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiKVCondSuspended,
		Status:             suspendStatus,
		ObservedGeneration: rtx.TiKV.Generation,
		Reason:             v1alpha1.TiKVSuspendReason,
		Message:            suspendMessage,
	}) || needUpdate

	if needUpdate || !v1alpha1.IsReconciled(rtx.TiKV) ||
		rtx.TiKV.Status.UpdateRevision != rtx.TiKV.Labels[v1alpha1.LabelKeyInstanceRevisionHash] {
		rtx.TiKV.Status.ObservedGeneration = rtx.TiKV.Generation
		rtx.TiKV.Status.UpdateRevision = rtx.TiKV.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
		needUpdate = true
	}

	if rtx.Pod == nil || rtx.PodIsTerminating {
		rtx.Healthy = false
	} else if statefulset.IsPodRunningAndReady(rtx.Pod) && rtx.StoreState == v1alpha1.StoreStateServing {
		rtx.Healthy = true
		if rtx.TiKV.Status.CurrentRevision != rtx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash] {
			rtx.TiKV.Status.CurrentRevision = rtx.Pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
			needUpdate = true
		}
	} else {
		rtx.Healthy = false
	}

	if rtx.Healthy {
		healthStatus = metav1.ConditionTrue
		healthMessage = "tikv is healthy"
	}
	needUpdate = meta.SetStatusCondition(&rtx.TiKV.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiKVCondHealth,
		Status:             healthStatus,
		ObservedGeneration: rtx.TiKV.Generation,
		Reason:             v1alpha1.TiKVHealthReason,
		Message:            healthMessage,
	}) || needUpdate

	if t.syncLeadersEvictedCond(rtx.TiKV, rtx.Store, rtx.LeaderEvicting) {
		needUpdate = true
	}

	if needUpdate {
		if err := t.Client.Status().Update(ctx, rtx.TiKV); err != nil {
			return task.Fail().With("cannot update status: %w", err)
		}
	}

	// TODO: use a condition to refactor it
	if rtx.TiKV.Status.ID == "" || rtx.TiKV.Status.State != v1alpha1.StoreStateServing || !v1alpha1.IsUpToDate(rtx.TiKV) {
		// can we only rely on the PD member events for this condition?
		//nolint:mnd // refactor to use a constant
		return task.Retry(5 * time.Second).With("tikv may not be initialized, retry")
	}

	return task.Complete().With("status is synced")
}

// Status of this condition can only transfer as the below
func (*TaskStatus) syncLeadersEvictedCond(tikv *v1alpha1.TiKV, store *pdv1.Store, isEvicting bool) bool {
	status := metav1.ConditionFalse
	reason := "NotEvicted"
	msg := "leaders are not all evicted"
	switch {
	case store == nil:
		status = metav1.ConditionTrue
		reason = "StoreIsRemoved"
		msg = "store does not exist"
	case isEvicting && store.LeaderCount == 0:
		status = metav1.ConditionTrue
		reason = "Evicted"
		msg = "all leaders are evicted"
	}

	return meta.SetStatusCondition(&tikv.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiKVCondLeadersEvicted,
		Status:             status,
		ObservedGeneration: tikv.Generation,
		Reason:             reason,
		Message:            msg,
	})
}
