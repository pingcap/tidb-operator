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

package replicationworker

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/replicationworker/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get replicationworker
		common.TaskContextObject[scope.ReplicationWorker](state, r.Client),
		common.TaskTrack[scope.ReplicationWorker](state, r.Tracker),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.ReplicationWorker](state)),

		// get cluster
		common.TaskContextCluster[scope.ReplicationWorker](state, r.Client),
		// return if cluster's status is not updated
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get info from pd
		// tasks.TaskContextInfoFromPD(state, r.PDClientManager),
		task.IfBreak(common.CondObjectIsDeleting[scope.ReplicationWorker](state),
			tasks.TaskFinalizerDel(state, r.Client),
			common.TaskInstanceConditionSynced[scope.ReplicationWorker](state),
			common.TaskInstanceConditionReady[scope.ReplicationWorker](state),
			common.TaskInstanceConditionRunning[scope.ReplicationWorker](state),
			common.TaskStatusPersister[scope.ReplicationWorker](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.ReplicationWorker](state, r.Client),

		// get pod
		common.TaskContextPod[scope.ReplicationWorker](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.ReplicationWorker](state),
			common.TaskInstanceConditionSynced[scope.ReplicationWorker](state),
			common.TaskInstanceConditionReady[scope.ReplicationWorker](state),
			common.TaskInstanceConditionRunning[scope.ReplicationWorker](state),
			common.TaskStatusPersister[scope.ReplicationWorker](state, r.Client),
		),

		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.ReplicationWorker](state, r.Client, r.VolumeModifierFactory, tasks.PVCNewer()),
		tasks.TaskPod(state, r.Client),
		common.TaskInstanceConditionSynced[scope.ReplicationWorker](state),
		common.TaskInstanceConditionReady[scope.ReplicationWorker](state),
		common.TaskInstanceConditionRunning[scope.ReplicationWorker](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
