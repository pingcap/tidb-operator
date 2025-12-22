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

package scheduling

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/scheduling/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get scheduling
		common.TaskContextObject[scope.Scheduling](state, r.Client),
		common.TaskTrack[scope.Scheduling](state, r.Tracker),
		// if it's gone just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.Scheduling](state)),

		// get cluster
		common.TaskContextCluster[scope.Scheduling](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),
		// if the cluster is deleting, del all subresources and remove the finalizer directly
		task.IfBreak(common.CondClusterIsDeleting(state),
			common.TaskInstanceFinalizerDel[scope.Scheduling](state, r.Client, common.DefaultInstanceSubresourceLister),
		),
		// return if cluster's status is not updated
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),

		// get info from pd
		// tasks.TaskContextInfoFromPD(state, r.PDClientManager),
		task.IfBreak(common.CondObjectIsDeleting[scope.Scheduling](state),
			common.TaskInstanceFinalizerDel[scope.Scheduling](state, r.Client, common.DefaultInstanceSubresourceLister),
			common.TaskInstanceConditionSynced[scope.Scheduling](state),
			common.TaskInstanceConditionReady[scope.Scheduling](state),
			common.TaskInstanceConditionRunning[scope.Scheduling](state),
			common.TaskStatusPersister[scope.Scheduling](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.Scheduling](state, r.Client),

		// get pod
		common.TaskContextPod[scope.Scheduling](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.Scheduling](state),
			common.TaskInstanceConditionSynced[scope.Scheduling](state),
			common.TaskInstanceConditionReady[scope.Scheduling](state),
			common.TaskInstanceConditionRunning[scope.Scheduling](state),
			common.TaskStatusPersister[scope.Scheduling](state, r.Client),
		),

		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.Scheduling](state, r.Client, r.VolumeModifierFactory, tasks.PVCNewer()),
		tasks.TaskPod(state, r.Client),
		common.TaskInstanceConditionSynced[scope.Scheduling](state),
		common.TaskInstanceConditionReady[scope.Scheduling](state),
		common.TaskInstanceConditionRunning[scope.Scheduling](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
