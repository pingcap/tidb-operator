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

package common

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

// TaskOfflineStore tries to delete store or cancel store deletion
//
// A: spec.offline
// - A1: spec.offline == false
// - A2: spec.offline == true
// B: state
// - B1: state == Preparing || state == Serving
// - B2: state == Removing
// - B3: state == Removed
// 1. (A1, B1): do nothing, waiting for changes of spec.offline
// 2. (A2, B1): call delete store api
// 3. (A1, B2): call cancel api
// 4. (A2, B2): do nothing, waiting for changes of state
// 5. (A1, B3): do nothing, waiting for instance being removed
// 6. (A2, B3): do nothing, waiting for instance being removed
func TaskOfflineStore[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](
	ctx context.Context,
	c pdapi.PDClient,
	obj F,
	// storeID may be empty if the store is not found or info from pd is not synced
	storeID string,
	state pdv1.NodeState,
) error {
	isOffline := coreutil.IsOffline[S](obj)

	switch {
	case !isOffline && storeID == "":
		// do nothing because store does not exist
		return nil
	case isOffline && storeID == "":
		return fmt.Errorf("%w: store does not exist and is offline", task.ErrWait)
	case isOffline && (state == pdv1.NodeStatePreparing || state == pdv1.NodeStateServing):
		if err := c.DeleteStore(ctx, storeID); err != nil {
			return err
		}

		return fmt.Errorf("%w: node state should be changed to Removing", task.ErrWait)
	case isOffline && state == pdv1.NodeStateRemoving:
		return fmt.Errorf("%w: node state should be changed to Removed", task.ErrWait)
	case isOffline && state == pdv1.NodeStateRemoved:
		return nil
	case !isOffline && (state == pdv1.NodeStatePreparing || state == pdv1.NodeStateServing):
		// do nothing
		return nil
	case !isOffline && state == pdv1.NodeStateRemoving:
		if err := c.CancelDeleteStore(ctx, storeID); err != nil {
			return err
		}
		return fmt.Errorf("%w: node state should be changed to Preparing or Serving", task.ErrWait)
	case !isOffline && state == pdv1.NodeStateRemoved:
		// store has been removed, cannot cancel the deletion
		return nil
	}

	return fmt.Errorf("unreachable, just retry again")
}

type InstanceCondOfflineUpdater[T client.Object] interface {
	StatusUpdater
	StoreState

	Object() T
}

// TaskInstanceConditionOffline set offline condition of an instance
func TaskInstanceConditionOffline[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](s InstanceCondOfflineUpdater[F]) task.Task {
	return task.NameTaskFunc("CondOffline", func(context.Context) task.Result {
		instance := s.Object()
		isOffline := coreutil.IsOffline[S](instance)
		state := s.GetStoreState()

		var needUpdate, isCompleted bool
		var reason string
		switch {
		case isOffline && state == "":
			reason = v1alpha1.ReasonOfflineCompleted
			isCompleted = true
		case state == pdv1.NodeStateRemoved:
			reason = v1alpha1.ReasonOfflineCompleted
			isCompleted = true
		case isOffline:
			reason = v1alpha1.ReasonOfflineProcessing
		case !isOffline && (state == "" || state == pdv1.NodeStatePreparing || state == pdv1.NodeStateServing):
			reason = ""
		case !isOffline && state == pdv1.NodeStateRemoving:
			reason = v1alpha1.ReasonOfflineCanceling
		}

		// not offline and store is up
		if reason == "" {
			needUpdate = coreutil.RemoveStatusCondition[S](
				instance,
				v1alpha1.StoreOfflinedConditionType,
			) || needUpdate
			if needUpdate {
				s.SetStatusChanged()
			}
			return task.Complete().With("instance is not offline")
		}

		var cond *metav1.Condition
		if isCompleted {
			cond = coreutil.Offlined()
		} else {
			cond = coreutil.NotOfflined(reason)
		}

		needUpdate = coreutil.SetStatusCondition[S](
			instance,
			*cond,
		) || needUpdate

		if needUpdate {
			s.SetStatusChanged()
		}

		if !isCompleted {
			return task.Wait().With("instance is processing or canceling: %s", coreutil.SprintCondition(cond))
		}

		return task.Complete().With("instance is offlined")
	})
}
