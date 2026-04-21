// Copyright 2026 PingCAP, Inc.
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

	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/metrics"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

// TaskObserveInstance refreshes the AbnormalInstance gauge for the reconciled
// instance every time the pipeline runs, and clears it when the instance CR
// has been removed from the API server (state.Object() == nil).
//
// It mirrors the shape of TaskTrack so observe and clear live in one place:
// the same branch that decides whether the object still exists also decides
// whether to publish or retract the gauge. Placing this after TaskTrack and
// before the `CondObjectHasBeenDeleted` IfBreak means it always runs once
// per reconcile, including the reconcile triggered by the informer's DELETE
// event - covering graceful delete, force-delete that strips finalizers, and
// any other path that lands in the watch stream.
func TaskObserveInstance[
	S scope.Instance[F, T],
	F Object[P],
	T runtime.Instance,
	P any,
](state TrackState[F]) task.Task {
	return task.NameTaskFunc("ObserveInstance", func(context.Context) task.Result {
		obj := state.Object()
		if obj == nil {
			key := state.Key()
			metrics.ClearInstanceConditionMetricsByKey(key.Namespace, key.Name)
			return task.Complete().With("cleared metrics for deleted %s", key)
		}
		conds := coreutil.StatusConditions[S](obj)
		metrics.ObserveConditions(obj, conds)
		return task.Complete().With("observed metrics for %s/%s", obj.GetNamespace(), obj.GetName())
	})
}
