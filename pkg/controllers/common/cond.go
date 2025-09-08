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
	"slices"

	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func CondClusterIsSuspending(ctx ClusterState) task.Condition {
	return task.CondFunc(func() bool {
		return coreutil.ShouldSuspendCompute(ctx.Cluster())
	})
}

func CondClusterIsPaused(ctx ClusterState) task.Condition {
	return task.CondFunc(func() bool {
		return coreutil.ShouldPauseReconcile(ctx.Cluster())
	})
}

func CondClusterIsDeleting(ctx ClusterState) task.Condition {
	return task.CondFunc(func() bool {
		return !ctx.Cluster().DeletionTimestamp.IsZero()
	})
}

func CondClusterPDAddrIsNotRegistered(ctx ClusterState) task.Condition {
	return task.CondFunc(func() bool {
		return ctx.Cluster().Status.PD == ""
	})
}

func CondObjectHasBeenDeleted[
	S scope.Object[F, T],
	F Object[O],
	T runtime.Object,
	O any,
](state ObjectState[F]) task.Condition {
	return task.CondFunc(func() bool {
		return state.Object() == nil
	})
}

func CondObjectIsDeleting[
	S scope.Object[F, T],
	F Object[O],
	T runtime.Object,
	O any,
](state ObjectState[F]) task.Condition {
	return task.CondFunc(func() bool {
		return !state.Object().GetDeletionTimestamp().IsZero()
	})
}

type ContextFeatureGates[
	F client.Object,
] interface {
	ObjectState[F]
	ClusterState
}

// CondFeatureGatesIsNotSynced is defined to ensure features are synced before all actions
func CondFeatureGatesIsNotSynced[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](state ContextFeatureGates[F]) task.Condition {
	return task.CondFunc(func() bool {
		obj := state.Object()
		cluster := state.Cluster()
		return !slices.Equal(coreutil.Features[S](obj), coreutil.EnabledFeatures(cluster))
	})
}
