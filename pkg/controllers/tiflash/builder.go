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
	"github.com/pingcap/tidb-operator/pkg/controllers/tiflash/tasks"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

func (r *Reconciler) NewRunner(reporter task.TaskReporter) task.TaskRunner[tasks.ReconcileContext] {
	runner := task.NewTaskRunner(reporter,
		// Get tiflash
		tasks.TaskContextTiFlash(r.Client),
		// If it's deleted just return
		task.NewSwitchTask(tasks.CondTiFlashHasBeenDeleted()),

		// get cluster info, FinalizerDel will use it
		tasks.TaskContextCluster(r.Client),
		// get info from pd
		tasks.TaskContextInfoFromPD(r.PDClientManager),

		task.NewSwitchTask(tasks.CondTiFlashIsDeleting(),
			tasks.TaskFinalizerDel(r.Client),
		),

		// check whether it's paused
		task.NewSwitchTask(tasks.CondClusterIsPaused()),

		// get pod and check whether the cluster is suspending
		tasks.TaskContextPod(r.Client),
		task.NewSwitchTask(tasks.CondClusterIsSuspending(),
			tasks.TaskFinalizerAdd(r.Client),
			tasks.TaskPodSuspend(r.Client),
			tasks.TaskStatusSuspend(r.Client),
		),

		// normal process
		tasks.TaskFinalizerAdd(r.Client),
		tasks.NewTaskConfigMap(r.Logger, r.Client),
		tasks.NewTaskPVC(r.Logger, r.Client, r.VolumeModifier),
		tasks.NewTaskPod(r.Logger, r.Client),
		tasks.NewTaskStoreLabels(r.Logger, r.Client),
		tasks.NewTaskStatus(r.Logger, r.Client),
	)

	return runner
}
