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

package routergroup

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/routergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		common.TaskContextObject[scope.RouterGroup](state, r.Client),
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.RouterGroup](state)),

		task.IfBreak(common.CondObjectIsDeleting[scope.RouterGroup](state),
			common.TaskContextSlice[scope.RouterGroup](state, r.Client),
			common.TaskGroupFinalizerDel[scope.RouterGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.RouterGroup](state),
			common.TaskGroupConditionSynced[scope.RouterGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.RouterGroup](state),
			common.TaskStatusPersister[scope.RouterGroup](state, r.Client),
		),

		common.TaskContextCluster[scope.RouterGroup](state, r.Client),
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(common.CondFeatureGatesIsNotSynced[scope.RouterGroup](state)),

		common.TaskContextSlice[scope.RouterGroup](state, r.Client),
		common.TaskFinalizerAdd[scope.RouterGroup](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.RouterGroup](state),
			common.TaskGroupConditionReady[scope.RouterGroup](state),
			common.TaskGroupConditionSynced[scope.RouterGroup](state),
			common.TaskStatusPersister[scope.RouterGroup](state, r.Client),
		),

		common.TaskRevision[runtime.RouterGroupTuple](state, r.Client),
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client, r.AllocateFactory),
		common.TaskGroupStatusSelector[scope.RouterGroup](state),
		common.TaskGroupConditionSuspended[scope.RouterGroup](state),
		common.TaskGroupConditionReady[scope.RouterGroup](state),
		common.TaskGroupConditionSynced[scope.RouterGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.RouterGroup](state),
		common.TaskStatusPersister[scope.RouterGroup](state, r.Client),
	)

	return runner
}
