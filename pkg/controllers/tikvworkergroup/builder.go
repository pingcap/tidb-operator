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

package tikvworkergroup

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvworkergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tikv worker group
		common.TaskContextObject[scope.TiKVWorkerGroup](state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.TiKVWorkerGroup](state)),

		// get cluster
		common.TaskContextCluster[scope.TiKVWorkerGroup](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(common.CondFeatureGatesIsNotSynced[scope.TiKVWorkerGroup](state)),

		// get all tikv workers
		common.TaskContextSlice[scope.TiKVWorkerGroup](state, r.Client),

		task.IfBreak(common.CondObjectIsDeleting[scope.TiKVWorkerGroup](state),
			common.TaskGroupFinalizerDel[scope.TiKVWorkerGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.TiKVWorkerGroup](state),
			common.TaskGroupConditionSynced[scope.TiKVWorkerGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.TiKVWorkerGroup](state),
			common.TaskStatusPersister[scope.TiKVWorkerGroup](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.TiKVWorkerGroup](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.TiKVWorkerGroup](state),
			common.TaskGroupConditionReady[scope.TiKVWorkerGroup](state),
			common.TaskGroupConditionSynced[scope.TiKVWorkerGroup](state),
			common.TaskStatusPersister[scope.TiKVWorkerGroup](state, r.Client),
		),

		common.TaskRevision[runtime.TiKVWorkerGroupTuple](state, r.Client),
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client, r.AllocateFactory),
		common.TaskGroupStatusSelector[scope.TiKVWorkerGroup](state),
		common.TaskGroupConditionSuspended[scope.TiKVWorkerGroup](state),
		common.TaskGroupConditionReady[scope.TiKVWorkerGroup](state),
		common.TaskGroupConditionSynced[scope.TiKVWorkerGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.TiKVWorkerGroup](state),
		common.TaskStatusPersister[scope.TiKVWorkerGroup](state, r.Client),
	)

	return runner
}
