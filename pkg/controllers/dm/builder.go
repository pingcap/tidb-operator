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

package dm

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/dm/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get DM instance
		common.TaskContextObject[scope.DM](state, r.Client),
		common.TaskTrack[scope.DM](state, r.Tracker),
		// refresh the abnormal_instance gauge, or clear it if the CR is gone
		common.TaskObserveInstance[scope.DM](state),
		// if it's deleted just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.DM](state)),

		// get cluster info
		common.TaskContextCluster[scope.DM](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),
		// if the cluster is deleting, del all subresources and remove the finalizer directly
		task.IfBreak(common.CondClusterIsDeleting(state),
			common.TaskInstanceFinalizerDel[scope.DM](state, r.Client, common.DefaultInstanceSubresourceLister),
		),

		task.IfBreak(common.CondObjectIsDeleting[scope.DM](state),
			common.TaskInstanceFinalizerDel[scope.DM](state, r.Client, common.DefaultInstanceSubresourceLister),
			common.TaskInstanceConditionSynced[scope.DM](state),
			common.TaskInstanceConditionReady[scope.DM](state),
			common.TaskInstanceConditionRunning[scope.DM](state),
			common.TaskStatusPersister[scope.DM](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.DM](state, r.Client),

		// get pod and check whether the cluster is suspending
		common.TaskContextPod[scope.DM](state, r.Client),
		task.IfBreak(common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.DM](state),
			common.TaskInstanceConditionSynced[scope.DM](state),
			common.TaskInstanceConditionReady[scope.DM](state),
			common.TaskInstanceConditionRunning[scope.DM](state),
			common.TaskStatusPersister[scope.DM](state, r.Client),
		),

		// list peer DM instances (for config generation)
		common.TaskContextPeerSlice[scope.DM](state, r.Client),
		// health check and member ID
		tasks.TaskContextInfoFromDM(state, r.Client),

		// normal process
		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.DM](state, r.Client, r.VolumeModifierFactory, tasks.PVCNewer()),
		tasks.TaskPod(state, r.Client),
		common.TaskInstanceConditionSynced[scope.DM](state),
		common.TaskInstanceConditionReady[scope.DM](state),
		common.TaskInstanceConditionRunning[scope.DM](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
