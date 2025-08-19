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

package replicationworkergroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/replicationworkergroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		common.TaskContextObject[scope.ReplicationWorkerGroup](state, r.Client),
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.ReplicationWorkerGroup](state)),

		// get cluster
		common.TaskContextCluster[scope.ReplicationWorkerGroup](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get all schedulers
		common.TaskContextSlice[scope.ReplicationWorkerGroup](state, r.Client),

		task.IfBreak(common.CondObjectIsDeleting[scope.ReplicationWorkerGroup](state),
			common.TaskGroupFinalizerDel[scope.ReplicationWorkerGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.ReplicationWorkerGroup](state),
			common.TaskGroupConditionSynced[scope.ReplicationWorkerGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.ReplicationWorkerGroup](state),
			common.TaskStatusPersister[scope.ReplicationWorkerGroup](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.ReplicationWorkerGroup](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.ReplicationWorkerGroup](state),
			common.TaskGroupConditionReady[scope.ReplicationWorkerGroup](state),
			common.TaskGroupConditionSynced[scope.ReplicationWorkerGroup](state),
			common.TaskStatusPersister[scope.ReplicationWorkerGroup](state, r.Client),
		),

		common.TaskRevision[runtime.ReplicationWorkerGroupTuple](state, r.Client), // TODO: Define runtime.ReplicationWorkerGroupTuple
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client, r.Tracker),
		common.TaskGroupStatusSelector[scope.ReplicationWorkerGroup](state),
		common.TaskGroupConditionSuspended[scope.ReplicationWorkerGroup](state),
		common.TaskGroupConditionReady[scope.ReplicationWorkerGroup](state),
		common.TaskGroupConditionSynced[scope.ReplicationWorkerGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.ReplicationWorkerGroup](state),
		common.TaskStatusPersister[scope.ReplicationWorkerGroup](state, r.Client),
	)

	return runner
}
