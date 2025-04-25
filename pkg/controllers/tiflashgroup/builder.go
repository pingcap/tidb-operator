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

package tiflashgroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiflashgroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tiflashgroup
		common.TaskContextTiFlashGroup(state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondGroupHasBeenDeleted(state)),

		// get cluster
		common.TaskContextCluster[scope.TiFlashGroup](state, r.Client),
		common.TaskFeatureGates(state),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get all tiflashes
		common.TaskContextTiFlashSlice(state, r.Client),

		task.IfBreak(common.CondGroupIsDeleting(state),
			common.TaskGroupFinalizerDel[runtime.TiFlashGroupTuple, runtime.TiFlashTuple](state, r.Client),
			common.TaskGroupConditionReady[scope.TiFlashGroup](state),
			common.TaskGroupConditionSynced[scope.TiFlashGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.TiFlashGroup](state),
			common.TaskStatusPersister[scope.TiFlashGroup](state, r.Client),
		),
		common.TaskGroupFinalizerAdd[runtime.TiFlashGroupTuple](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.TiFlashGroup](state),
			common.TaskGroupConditionReady[scope.TiFlashGroup](state),
			common.TaskGroupConditionSynced[scope.TiFlashGroup](state),
			common.TaskStatusPersister[scope.TiFlashGroup](state, r.Client),
		),

		common.TaskRevision[runtime.TiFlashGroupTuple](state, r.Client),
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client),
		common.TaskGroupConditionSuspended[scope.TiFlashGroup](state),
		common.TaskGroupConditionReady[scope.TiFlashGroup](state),
		common.TaskGroupConditionSynced[scope.TiFlashGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.TiFlashGroup](state),
		common.TaskStatusPersister[scope.TiFlashGroup](state, r.Client),
	)

	return runner
}
