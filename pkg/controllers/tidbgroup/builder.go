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

package tidbgroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/tidbgroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tidbgroup
		common.TaskContextObject[scope.TiDBGroup](state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.TiDBGroup](state)),

		// get cluster
		common.TaskContextCluster[scope.TiDBGroup](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(common.CondFeatureGatesIsNotSynced[scope.TiDBGroup](state)),

		// get all tidbs
		common.TaskContextSlice[scope.TiDBGroup](state, r.Client),

		task.IfBreak(common.CondObjectIsDeleting[scope.TiDBGroup](state),
			common.TaskGroupFinalizerDel[scope.TiDBGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.TiDBGroup](state),
			common.TaskGroupConditionSynced[scope.TiDBGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.TiDBGroup](state),
			common.TaskStatusPersister[scope.TiDBGroup](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.TiDBGroup](state, r.Client),

		common.TaskRevision[runtime.TiDBGroupTuple](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.TiDBGroup](state),
			common.TaskGroupConditionReady[scope.TiDBGroup](state),
			common.TaskGroupConditionSynced[scope.TiDBGroup](state),
			common.TaskStatusPersister[scope.TiDBGroup](state, r.Client),
		),

		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client, r.AllocateFactory, r.AdoptManager),
		common.TaskGroupStatusSelector[scope.TiDBGroup](state),
		common.TaskGroupConditionSuspended[scope.TiDBGroup](state),
		common.TaskGroupConditionReady[scope.TiDBGroup](state),
		common.TaskGroupConditionSynced[scope.TiDBGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.TiDBGroup](state),
		tasks.TaskStatusAvailable(state),
		common.TaskStatusPersister[scope.TiDBGroup](state, r.Client),
	)

	return runner
}
