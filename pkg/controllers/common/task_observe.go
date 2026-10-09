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

	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/metrics"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

// TaskObserveInstance refreshes the AbnormalInstance gauge for the reconciled
// instance after its Cluster is loaded, and clears it when the instance CR
// has been removed from the API server (state.Object() == nil).
//
// Run after TaskContextCluster for live instances so suspend filtering uses the
// current desired state. Also run in the CondObjectHasBeenDeleted branch to
// clear metrics on DELETE events, including force-deletion without finalizers.
func TaskObserveInstance[
	S scope.Instance[F, T],
	F Object[P],
	T runtime.Instance,
	P any,
](state interface {
	TrackState[F]
	ClusterState
}) task.Task {
	return task.NameTaskFunc("ObserveInstance", func(context.Context) task.Result {
		obj := state.Object()
		if obj == nil {
			key := state.Key()
			// scope.Component[S]() is a compile-time constant for the reconcile
			// kind, so we can qualify the sweep even after the CR is gone and
			// its labels are no longer readable.
			metrics.ClearInstanceConditionMetricsByKey(key.Namespace, scope.Component[S](), key.Name)
			return task.Complete().With("cleared metrics for deleted %s", key)
		}
		conds := coreutil.StatusConditions[S](obj)
		metrics.ObserveConditions(obj, conds, state.Cluster())
		return task.Complete().With("observed metrics for %s/%s", obj.GetNamespace(), obj.GetName())
	})
}
