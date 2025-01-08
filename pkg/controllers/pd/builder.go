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
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/pd/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get pd
		common.TaskContextPD(state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondInstanceHasBeenDeleted(state)),

		// get cluster
		common.TaskContextCluster(state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get info from pd
		tasks.TaskContextInfoFromPD(state, r.PDClientManager),
		task.IfBreak(common.CondInstanceIsDeleting(state),
			tasks.TaskFinalizerDel(state, r.Client),
		),
		common.TaskInstanceFinalizerAdd[runtime.PDTuple](state, r.Client),

		// get pod
		common.TaskContextPod(state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceStatusSuspend[runtime.PDTuple](state, r.Client),
		),

		common.TaskContextPDSlice(state, r.Client),
		tasks.TaskConfigMap(state, r.Logger, r.Client),
		tasks.TaskPVC(state, r.Logger, r.Client, r.VolumeModifier),
		tasks.TaskPod(state, r.Logger, r.Client),
		// If pd client has not been registered yet, do not update status of the pd
		task.IfBreak(tasks.CondPDClientIsNotRegisterred(state),
			tasks.TaskStatusUnknown(),
		),
		tasks.TaskStatus(state, r.Logger, r.Client),
	)

	return runner
}
