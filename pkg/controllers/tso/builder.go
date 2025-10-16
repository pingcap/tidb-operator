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

package tso

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/tso/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tso
		common.TaskContextObject[scope.TSO](state, r.Client),
		common.TaskTrack[scope.TSO](state, r.Tracker),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.TSO](state)),

		// get cluster
		common.TaskContextCluster[scope.TSO](state, r.Client),
		// return if cluster's status is not updated
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get info from pd
		// tasks.TaskContextInfoFromPD(state, r.PDClientManager),
		task.IfBreak(common.CondObjectIsDeleting[scope.TSO](state),
			tasks.TaskFinalizerDel(state, r.Client),
			// TODO(liubo02): if the finalizer has been removed, no need to update status
			common.TaskInstanceConditionSynced[scope.TSO](state),
			common.TaskInstanceConditionReady[scope.TSO](state),
			common.TaskInstanceConditionRunning[scope.TSO](state),
			common.TaskStatusPersister[scope.TSO](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.TSO](state, r.Client),

		// get pod
		common.TaskContextPod[scope.TSO](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.TSO](state),
			common.TaskInstanceConditionSynced[scope.TSO](state),
			common.TaskInstanceConditionReady[scope.TSO](state),
			common.TaskInstanceConditionRunning[scope.TSO](state),
			common.TaskStatusPersister[scope.TSO](state, r.Client),
		),

		// get peers
		common.TaskContextPeerSlice[scope.TSO](state, r.Client),
		// init client
		tasks.TaskContextClient(state, r.PDClientManager, r.TSOClientManager),

		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.TSO](state, r.Client, r.VolumeModifierFactory, tasks.PVCNewer()),
		tasks.TaskPod(state, r.Client),
		common.TaskInstanceConditionSynced[scope.TSO](state),
		common.TaskInstanceConditionReady[scope.TSO](state),
		common.TaskInstanceConditionRunning[scope.TSO](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
