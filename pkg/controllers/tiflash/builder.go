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

package tiflash

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiflash/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// Get tiflash
		common.TaskContextObject[scope.TiFlash](state, r.Client),
		// if it's deleted just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.TiFlash](state)),

		// get cluster info, FinalizerDel will use it
		common.TaskContextCluster[scope.TiFlash](state, r.Client),
		// return if cluster's status is not updated
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),
		// check whether it's paused
		task.IfBreak(common.CondClusterIsPaused(state)),
		// if the cluster is deleting, del all subresources and remove the finalizer directly
		task.IfBreak(common.CondClusterIsDeleting(state),
			tasks.TaskFinalizerDel(state, r.Client),
		),

		// get info from pd
		task.IfNot(common.CondClusterIsDeleting(state),
			tasks.TaskContextInfoFromPD(state, r.PDClientManager),
		),

		// handle offline state machine for two-step store deletion
		tasks.TaskOfflineStore(state),

		task.IfBreak(common.CondObjectIsDeleting[scope.TiFlash](state),
			tasks.TaskFinalizerDel(state, r.Client),
		),

		common.TaskFinalizerAdd[scope.TiFlash](state, r.Client),
		// get pod and check whether the cluster is suspending
		common.TaskContextPod[scope.TiFlash](state, r.Client),

		// check whether the cluster is suspending
		task.IfBreak(common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.TiFlash](state),
			common.TaskInstanceConditionSynced[scope.TiFlash](state),
			common.TaskInstanceConditionReady[scope.TiFlash](state),
			common.TaskStatusPersister[scope.TiFlash](state, r.Client),
		),

		// normal process
		tasks.TaskConfigMap(state, r.Client),
		tasks.TaskPVC(state, r.Logger, r.Client, r.VolumeModifierFactory),
		tasks.TaskPod(state, r.Client),
		tasks.TaskStoreLabels(state, r.Client),
		common.TaskInstanceConditionSynced[scope.TiFlash](state),
		common.TaskInstanceConditionReady[scope.TiFlash](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
