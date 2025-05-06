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

package tikvgroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/tikvgroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tikvgroup
		common.TaskContextTiKVGroup(state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondGroupHasBeenDeleted(state)),

		// get cluster
		common.TaskContextCluster[scope.TiKVGroup](state, r.Client),
		common.TaskFeatureGates(state),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get all tikvs
		common.TaskContextTiKVSlice(state, r.Client),

		task.IfBreak(common.CondGroupIsDeleting(state),
			common.TaskGroupFinalizerDel[scope.TiKVGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.TiKVGroup](state),
			common.TaskGroupConditionSynced[scope.TiKVGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.TiKVGroup](state),
			common.TaskStatusPersister[scope.TiKVGroup](state, r.Client),
		),
		common.TaskGroupFinalizerAdd[runtime.TiKVGroupTuple](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.TiKVGroup](state),
			common.TaskGroupConditionReady[scope.TiKVGroup](state),
			common.TaskGroupConditionSynced[scope.TiKVGroup](state),
			common.TaskStatusPersister[scope.TiKVGroup](state, r.Client),
		),

		common.TaskRevision[runtime.TiKVGroupTuple](state, r.Client),
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client),
		common.TaskGroupConditionSuspended[scope.TiKVGroup](state),
		common.TaskGroupConditionReady[scope.TiKVGroup](state),
		common.TaskGroupConditionSynced[scope.TiKVGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.TiKVGroup](state),
		common.TaskStatusPersister[scope.TiKVGroup](state, r.Client),
	)

	return runner
}
