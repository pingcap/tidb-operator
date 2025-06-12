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

package schedulinggroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/schedulinggroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get schedulinggroup
		common.TaskContextObject[scope.SchedulingGroup](state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.SchedulingGroup](state)),

		// get cluster
		common.TaskContextCluster[scope.SchedulingGroup](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get all schedulings
		common.TaskContextSlice[scope.SchedulingGroup](state, r.Client),

		task.IfBreak(common.CondObjectIsDeleting[scope.SchedulingGroup](state),
			common.TaskGroupFinalizerDel[scope.SchedulingGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.SchedulingGroup](state),
			common.TaskGroupConditionSynced[scope.SchedulingGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.SchedulingGroup](state),
			common.TaskStatusPersister[scope.SchedulingGroup](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.SchedulingGroup](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.SchedulingGroup](state),
			common.TaskGroupConditionReady[scope.SchedulingGroup](state),
			common.TaskGroupConditionSynced[scope.SchedulingGroup](state),
			common.TaskStatusPersister[scope.SchedulingGroup](state, r.Client),
		),

		common.TaskRevision[runtime.SchedulingGroupTuple](state, r.Client), // TODO: Define runtimeSchedulingGroupupTuple
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client),
		common.TaskGroupStatusSelector[scope.SchedulingGroup](state),
		common.TaskGroupConditionSuspended[scope.SchedulingGroup](state),
		common.TaskGroupConditionReady[scope.SchedulingGroup](state),
		common.TaskGroupConditionSynced[scope.SchedulingGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.SchedulingGroup](state),
		common.TaskStatusPersister[scope.SchedulingGroup](state, r.Client),
	)

	return runner
}
