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

package resourcemanager

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanager/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		common.TaskContextObject[scope.ResourceManager](state, r.Client),
		common.TaskTrack[scope.ResourceManager](state, r.Tracker),
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.ResourceManager](state)),

		common.TaskContextCluster[scope.ResourceManager](state, r.Client),
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(common.CondClusterIsDeleting(state),
			common.TaskInstanceFinalizerDel[scope.ResourceManager](state, r.Client, common.DefaultInstanceSubresourceLister),
		),
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),

		task.IfBreak(common.CondObjectIsDeleting[scope.ResourceManager](state),
			tasks.TaskTransferPrimary(state, r.Client),
			common.TaskInstanceFinalizerDel[scope.ResourceManager](state, r.Client, common.DefaultInstanceSubresourceLister),
			common.TaskInstanceConditionSynced[scope.ResourceManager](state),
			common.TaskInstanceConditionReady[scope.ResourceManager](state),
			common.TaskInstanceConditionRunning[scope.ResourceManager](state),
			common.TaskStatusPersister[scope.ResourceManager](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.ResourceManager](state, r.Client),

		common.TaskContextPod[scope.ResourceManager](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.ResourceManager](state),
			common.TaskInstanceConditionSynced[scope.ResourceManager](state),
			common.TaskInstanceConditionReady[scope.ResourceManager](state),
			common.TaskInstanceConditionRunning[scope.ResourceManager](state),
			common.TaskStatusPersister[scope.ResourceManager](state, r.Client),
		),

		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.ResourceManager](state, r.Client, r.VolumeModifierFactory, common.DefaultPVCNewer[scope.ResourceManager]()),
		tasks.TaskPod(state, r.Client),
		common.TaskInstanceConditionSynced[scope.ResourceManager](state),
		common.TaskInstanceConditionReady[scope.ResourceManager](state),
		common.TaskInstanceConditionRunning[scope.ResourceManager](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
