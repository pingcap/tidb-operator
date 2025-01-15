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
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
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
		tikv := state.TiKV()
		pod := state.Pod()
		// TODO(liubo02): simplify it
		var healthy bool
		if pod != nil &&
			statefulset.IsPodRunningAndReady(pod) &&
			!state.PodIsTerminating &&
			state.StoreState == v1alpha1.StoreStateServing {
			healthy = true
		}
		needUpdate = syncHealthCond(tikv, healthy) || needUpdate
		needUpdate = syncSuspendCond(tikv) || needUpdate
		needUpdate = syncLeadersEvictedCond(tikv, state.Store, state.LeaderEvicting, state.IsPDAvailable) || needUpdate
		needUpdate = SetIfChanged(&tikv.Status.ID, state.StoreID) || needUpdate
		needUpdate = SetIfChanged(&tikv.Status.State, state.StoreState) || needUpdate

		needUpdate = SetIfChanged(&tikv.Status.ObservedGeneration, tikv.Generation) || needUpdate
		needUpdate = SetIfChanged(&tikv.Status.UpdateRevision, tikv.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate

		if healthy {
			needUpdate = SetIfChanged(&tikv.Status.CurrentRevision, pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]) || needUpdate
		}

		if needUpdate {
			if err := c.Status().Update(ctx, tikv); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		if state.PodIsTerminating {
			return task.Retry(defaultTaskWaitDuration).With("pod is terminating, retry after it's terminated")
		}

		if state.LeaderEvicting {
			return task.Wait().With("tikv is evicting leader, wait")
		}

		// TODO: use a condition to refactor it
		if !healthy || tikv.Status.ID == "" {
			return task.Retry(task.DefaultRequeueAfter).With("tikv may not be ready, wait")
		}

		return task.Complete().With("status is synced")
	})
}

func syncHealthCond(tikv *v1alpha1.TiKV, healthy bool) bool {
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

	return meta.SetStatusCondition(&tikv.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondHealth,
		Status:             status,
		ObservedGeneration: tikv.Generation,
		Reason:             reason,
		Message:            msg,
	})
}

func syncSuspendCond(tikv *v1alpha1.TiKV) bool {
	// always set it as unsuspended
	return meta.SetStatusCondition(&tikv.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSuspended,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: tikv.Generation,
		Reason:             v1alpha1.ReasonUnsuspended,
		Message:            "instace is not suspended",
	})
}

// Status of this condition can only transfer as the below
func syncLeadersEvictedCond(tikv *v1alpha1.TiKV, store *pdv1.Store, isEvicting, isPDAvail bool) bool {
	status := metav1.ConditionFalse
	reason := "NotEvicted"
	msg := "leaders are not all evicted"
	switch {
	case isPDAvail && store == nil:
		status = metav1.ConditionTrue
		reason = "StoreIsRemoved"
		msg = "store does not exist"
	case isPDAvail && isEvicting && store.LeaderCount == 0:
		status = metav1.ConditionTrue
		reason = "Evicted"
		msg = "all leaders are evicted"
	case !isPDAvail:
		status = metav1.ConditionUnknown
		reason = "Unknown"
		msg = "cannot get leaders info from pd"
	}

	return meta.SetStatusCondition(&tikv.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiKVCondLeadersEvicted,
		Status:             status,
		ObservedGeneration: tikv.Generation,
		Reason:             reason,
		Message:            msg,
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
