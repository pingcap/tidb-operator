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
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(ctx *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get pd
		common.TaskContextPD(ctx, r.Client),
		// if it's deleted just return
		task.IfBreak(common.CondPDHasBeenDeleted(ctx)),

		// get info from pd
		tasks.TaskContextInfoFromPD(ctx, r.PDClientManager),
		task.IfBreak(common.CondPDIsDeleting(ctx),
			tasks.TaskFinalizerDel(ctx, r.Client),
		),

		// get cluster and check whether it's paused
		common.TaskContextCluster(ctx, r.Client),
		task.IfBreak(
			common.CondClusterIsPaused(ctx),
		),

		// get pod and check whether the cluster is suspending
		common.TaskContextPod(ctx, r.Client),
		task.IfBreak(
			common.CondClusterIsSuspending(ctx),
			tasks.TaskFinalizerAdd(ctx, r.Client),
			common.TaskSuspendPod(ctx, r.Client),
			// TODO: extract as a common task
			tasks.TaskStatusSuspend(ctx, r.Client),
		),

		tasks.TaskContextPeers(ctx, r.Client),
		tasks.TaskFinalizerAdd(ctx, r.Client),
		tasks.TaskConfigMap(ctx, r.Logger, r.Client),
		tasks.TaskPVC(ctx, r.Logger, r.Client, r.VolumeModifier),
		tasks.TaskPod(ctx, r.Logger, r.Client),
		// If pd client has not been registered yet, do not update status of the pd
		task.IfBreak(tasks.CondPDClientIsNotRegisterred(ctx),
			tasks.TaskStatusUnknown(),
		),
		tasks.TaskStatus(ctx, r.Logger, r.Client),
	)

	return runner
}
