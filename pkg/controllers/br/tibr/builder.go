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

package tibr

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/tibr/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	t "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(rtx *tasks.ReconcileContext, reporter t.TaskReporter) t.TaskRunner {
	runner := t.NewTaskRunner(reporter,
		tasks.TaskContextRefreshTiBR(rtx),
		t.IfBreak(tasks.CondTiBRNotFound(rtx)),

		// let owner reference handle deletion
		t.IfBreak(tasks.CondTiBRIsDeleting(rtx), tasks.TaskUpdateStatusIfNeeded(rtx)),

		// escape if cluster not found or paused
		tasks.TaskContextRefreshCluster(rtx),
		t.IfBreak(tasks.CondClusterNotFound(rtx), tasks.TaskUpdateStatusIfNeeded(rtx)),
		t.IfBreak(common.CondClusterIsPaused(rtx), tasks.TaskUpdateStatusIfNeeded(rtx)),

		// refresh the sub-resources
		tasks.TaskContextRefreshConfigMap(rtx),
		tasks.TaskContextRefreshStatefulSet(rtx),
		tasks.TaskContextRefreshHeadlessSvc(rtx),

		// cleanup sub-resources if cluster is suspending
		t.IfBreak(common.CondClusterIsSuspending(rtx), tasks.TaskEnsureSubResourcesCleanup(rtx), tasks.TaskUpdateStatusIfNeeded(rtx)),

		// ensure sub-resources
		t.IfBreak(tasks.CondClusterIsNotReadyForBR(rtx), tasks.TaskUpdateStatusIfNeeded(rtx)),
		tasks.TaskEnsureSubResources(rtx),
		tasks.TaskUpdateStatusIfNeeded(rtx),
	)

	return runner
}
