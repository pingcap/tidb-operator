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

package pd

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/pd/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get pd
		common.TaskContextObject[scope.PD](state, r.Client),
		common.TaskTrack[scope.PD](state, r.Tracker),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.PD](state)),

		// get cluster
		common.TaskContextCluster[scope.PD](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get info from pd
		tasks.TaskContextInfoFromPD(state, r.PDClientManager),
		task.IfBreak(common.CondObjectIsDeleting[scope.PD](state),
			tasks.TaskFinalizerDel(state, r.Client),
			// TODO(liubo02): if the finalizer has been removed, no need to update status
			common.TaskInstanceConditionSynced[scope.PD](state),
			common.TaskInstanceConditionReady[scope.PD](state),
			common.TaskInstanceConditionRunning[scope.PD](state),
			common.TaskStatusPersister[scope.PD](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.PD](state, r.Client),

		// get pod
		common.TaskContextPod[scope.PD](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.PD](state), common.TaskInstanceConditionSynced[scope.PD](state),
			common.TaskInstanceConditionReady[scope.PD](state),
			common.TaskInstanceConditionRunning[scope.PD](state),
			common.TaskStatusPersister[scope.PD](state, r.Client),
		),

		common.TaskContextPeerSlice[scope.PD](state, r.Client),
		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.PD](state, r.Client, r.VolumeModifierFactory, tasks.PVCNewer()),
		tasks.TaskPod(state, r.Client),
		common.TaskInstanceConditionSynced[scope.PD](state),
		common.TaskInstanceConditionReady[scope.PD](state),
		common.TaskInstanceConditionRunning[scope.PD](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
