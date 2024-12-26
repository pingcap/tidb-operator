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

package pdgroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/pdgroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get pdgroup
		common.TaskContextPDGroup(state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondPDGroupHasBeenDeleted(state)),

		// get cluster
		common.TaskContextCluster(state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get all pds
		common.TaskContextPDSlice(state, r.Client),

		task.IfBreak(common.CondPDGroupIsDeleting(state),
			tasks.TaskFinalizerDel(state, r.Client, r.PDClientManager),
		),
		tasks.TaskFinalizerAdd(state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			tasks.TaskStatusSuspend(state, r.Client),
		),
		tasks.TaskContextPDClient(state, r.PDClientManager),
		tasks.TaskBoot(state, r.Client),
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
