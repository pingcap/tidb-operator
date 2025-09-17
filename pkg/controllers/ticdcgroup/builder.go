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

package ticdcgroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/ticdcgroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get TiCDCGroup
		common.TaskContextObject[scope.TiCDCGroup](state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.TiCDCGroup](state)),

		// get cluster
		common.TaskContextCluster[scope.TiCDCGroup](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(common.CondFeatureGatesIsNotSynced[scope.TiCDCGroup](state)),

		// get all TiCDCs
		common.TaskContextSlice[scope.TiCDCGroup](state, r.Client),

		task.IfBreak(common.CondObjectIsDeleting[scope.TiCDCGroup](state),
			common.TaskGroupFinalizerDel[scope.TiCDCGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.TiCDCGroup](state),
			common.TaskGroupConditionSynced[scope.TiCDCGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.TiCDCGroup](state),
			common.TaskStatusPersister[scope.TiCDCGroup](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.TiCDCGroup](state, r.Client),

		common.TaskRevision[runtime.TiCDCGroupTuple](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.TiCDCGroup](state),
			common.TaskGroupConditionReady[scope.TiCDCGroup](state),
			common.TaskGroupConditionSynced[scope.TiCDCGroup](state),
			common.TaskStatusPersister[scope.TiCDCGroup](state, r.Client),
		),
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client, r.AllocateFactory),
		common.TaskGroupStatusSelector[scope.TiCDCGroup](state),
		common.TaskGroupConditionSuspended[scope.TiCDCGroup](state),
		common.TaskGroupConditionReady[scope.TiCDCGroup](state),
		common.TaskGroupConditionSynced[scope.TiCDCGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.TiCDCGroup](state),
		common.TaskStatusPersister[scope.TiCDCGroup](state, r.Client),
	)

	return runner
}
