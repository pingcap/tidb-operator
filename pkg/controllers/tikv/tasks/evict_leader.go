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
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskEvictLeader(state *ReconcileContext, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("EvictLeader", func(ctx context.Context) task.Result {
		pc, ok := state.GetPDClient(m)
		if !ok {
			return task.Wait().With("wait if pd client is not registered")
		}
		if state.Store == nil {
			if syncLeadersEvictedCond(state.TiKV(), nil, state.LeaderEvicting) {
				state.SetStatusChanged()
			}
			return task.Complete().With("store has been deleted or not created")
		}

		if state.ShouldEvictLeader && !state.LeaderEvicting {
			if err := pc.Underlay().BeginEvictLeader(ctx, state.Store.ID); err != nil {
				return task.Fail().With("cannot add evict leader scheduler: %v", err)
			}
			state.LeaderEvicting = true
		}

		if state.LeaderEvicting && !state.ShouldEvictLeader {
			if err := pc.Underlay().EndEvictLeader(ctx, state.Store.ID); err != nil {
				return task.Fail().With("cannot remove evict leader scheduler: %v", err)
			}
			state.LeaderEvicting = false
		}

		needUpdate := syncLeadersEvictedCond(state.TiKV(), state.Store, state.LeaderEvicting)
		if needUpdate {
			state.SetStatusChanged()
		}

		return task.Complete().With("sync evict leader scheduler, expected: %v, actual: %v", state.ShouldEvictLeader, state.LeaderEvicting)
	})
}

// TaskEndEvictLeader only be called when object is deleting and store has been removed
// TODO(liubo02): it's not stable because status.ID may be lost
func TaskEndEvictLeader(state *ReconcileContext, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("EndEvictLeader", func(ctx context.Context) task.Result {
		msg := "ensure evict leader scheduler doesn't exist"
		pc, ok := state.GetPDClient(m)
		if !ok {
			return task.Wait().With("wait if pd client is not registered")
		}
		if storeID := state.TiKV().Status.ID; storeID != "" {
			if err := pc.Underlay().EndEvictLeader(ctx, storeID); err != nil {
				return task.Fail().With("cannot remove evict leader scheduler: %v", err)
			}
		} else {
			msg = "can not get the store id"
		}
		return task.Complete().With(msg)
	})
}

// Status of this condition can only transfer as the below
func syncLeadersEvictedCond(tikv *v1alpha1.TiKV, store *pdv1.Store, isEvicting bool) bool {
	status := metav1.ConditionFalse
	reason := v1alpha1.ReasonNotEvicted
	msg := "leaders are not all evicted"
	switch {
	case store == nil:
		status = metav1.ConditionTrue
		reason = v1alpha1.ReasonStoreNotExist
		msg = "store does not exist"
	case isEvicting && store.LeaderCount == 0:
		status = metav1.ConditionTrue
		reason = v1alpha1.ReasonEvicted
		msg = "all leaders are evicted"
	case isEvicting:
		status = metav1.ConditionFalse
		reason = v1alpha1.ReasonEvicting
		msg = fmt.Sprintf("not all leaders are evicted, still: %v", store.LeaderCount)
	}

	return meta.SetStatusCondition(&tikv.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.TiKVCondLeadersEvicted,
		Status:             status,
		ObservedGeneration: tikv.Generation,
		Reason:             reason,
		Message:            msg,
	})
}
