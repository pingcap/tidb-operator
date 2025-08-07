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

package tibrgc

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/br/tibrgc/tasks"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(rtx *tasks.ReconcileContext, reporter t.TaskReporter) t.TaskRunner {
	runner := t.NewTaskRunner(reporter,
		tasks.TaskContextRefreshTiBRGC(rtx),
		t.IfBreak(tasks.CondTiBRGCNotFound(rtx), tasks.TaskSkipReconcile(rtx, "TiBRGC not found")),
		// escape if it's deleting, let owner reference handle deletion
		t.IfBreak(tasks.CondTiBRGCIsDeleting(rtx), tasks.TaskSkipReconcile(rtx, "TiBRGC is deleting")),
		// Only allow one TiBRGC created for a cluster
		tasks.TaskContextRefreshTiBRGCList4SameCluster(rtx),
		t.IfBreak(tasks.CondMoreThanOneTiBRGCCreated(rtx), tasks.TaskSkipReconcile(rtx, "more than one TiBRGC created for a cluster")),

		// escape if cluster not found or paused
		tasks.TaskContextRefreshCluster(rtx),
		t.IfBreak(tasks.CondClusterNotFound(rtx), tasks.TaskSkipReconcile(rtx, "cluster not found")),
		t.IfBreak(common.CondClusterIsPaused(rtx), tasks.TaskSkipReconcile(rtx, "cluster is paused")),

		// cleanup sub-resources if cluster is suspending
		t.IfBreak(common.CondClusterIsSuspending(rtx), tasks.TaskEnsureSubResourcesCleanup(rtx)),

		// ensure sub-resources
		t.IfBreak(tasks.CondClusterIsNotReadyForBRGC(rtx), tasks.TaskSkipReconcile(rtx, "cluster is not ready for TiBRGC")),
		tasks.TaskEnsureSubResources(rtx),
	)

	return runner
}
