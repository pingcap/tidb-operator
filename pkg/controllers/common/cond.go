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
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func CondPDHasBeenDeleted(ctx PDState) task.Condition {
	return task.CondFunc(func() bool {
		return ctx.PD() == nil
	})
}

func CondPDIsDeleting(ctx PDState) task.Condition {
	return task.CondFunc(func() bool {
		return !ctx.PD().GetDeletionTimestamp().IsZero()
	})
}

func CondClusterIsSuspending(ctx ClusterState) task.Condition {
	return task.CondFunc(func() bool {
		return ctx.Cluster().ShouldSuspendCompute()
	})
}

func CondClusterIsPaused(ctx ClusterState) task.Condition {
	return task.CondFunc(func() bool {
		return ctx.Cluster().ShouldPauseReconcile()
	})
}

func CondGroupIsDeleting[G runtime.Group](state GroupState[G]) task.Condition {
	return task.CondFunc(func() bool {
		return !state.Group().GetDeletionTimestamp().IsZero()
	})
}

func CondGroupHasBeenDeleted[RG runtime.GroupT[G], G runtime.GroupSet](state GroupState[RG]) task.Condition {
	return task.CondFunc(func() bool {
		return state.Group() == nil
	})
}

func CondInstanceIsDeleting[I runtime.Instance](state InstanceState[I]) task.Condition {
	return task.CondFunc(func() bool {
		return !state.Instance().GetDeletionTimestamp().IsZero()
	})
}

func CondInstanceHasBeenDeleted[RI runtime.InstanceT[I], I runtime.InstanceSet](state InstanceState[RI]) task.Condition {
	return task.CondFunc(func() bool {
		return state.Instance() == nil
	})
}
