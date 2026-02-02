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

package router

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/router/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		common.TaskContextObject[scope.Router](state, r.Client),
		common.TaskTrack[scope.Router](state, r.Tracker),
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.Router](state)),

		task.IfBreak(common.CondObjectIsDeleting[scope.Router](state),
			common.TaskInstanceFinalizerDel[scope.Router](state, r.Client, common.DefaultInstanceSubresourceLister),
			common.TaskInstanceConditionSynced[scope.Router](state),
			common.TaskInstanceConditionReady[scope.Router](state),
			common.TaskInstanceConditionRunning[scope.Router](state),
			common.TaskStatusPersister[scope.Router](state, r.Client),
		),

		common.TaskContextCluster[scope.Router](state, r.Client),
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(common.CondClusterIsDeleting(state),
			common.TaskInstanceFinalizerDel[scope.Router](state, r.Client, common.DefaultInstanceSubresourceLister),
		),
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),
		common.TaskFinalizerAdd[scope.Router](state, r.Client),

		common.TaskContextPod[scope.Router](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.Router](state),
			common.TaskInstanceConditionSynced[scope.Router](state),
			common.TaskInstanceConditionReady[scope.Router](state),
			common.TaskInstanceConditionRunning[scope.Router](state),
			common.TaskStatusPersister[scope.Router](state, r.Client),
		),

		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.Router](state, r.Client, r.VolumeModifierFactory, common.DefaultPVCNewer[scope.Router]()),
		tasks.TaskPod(state, r.Client),
		common.TaskInstanceConditionSynced[scope.Router](state),
		common.TaskInstanceConditionReady[scope.Router](state),
		common.TaskInstanceConditionRunning[scope.Router](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
