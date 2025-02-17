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

package restore

import (
	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/restore/tasks"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get backup
		tasks.TaskContextRestore(state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondJobHasBeenDeleted(state)),

		// finalizer management
		task.If(task.CondFunc(func() bool {
			return needToAddFinalizer(state.Restore())
		}),
			common.TaskJobFinalizerAdd[*runtime.Restore](state, r.Client),
		),
		task.If(task.CondFunc(func() bool {
			return needToRemoveFinalizer(state.Restore())
		}),
			common.TaskJobFinalizerDel[*runtime.Restore](state, r.Client),
		),

		// get cluster
		common.TaskContextCluster(state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)), // TODO(ideascf): do we need to pause job also?
		task.IfBreak(
			common.CondClusterIsSuspending(state),
		),

		tasks.TaskRestoreManager(state, r.Client, r.EventRecorder),
	)

	return runner
}

func needToAddFinalizer(restore *v1alpha1.Restore) bool {
	return restore.DeletionTimestamp == nil
}

func needToRemoveFinalizer(restore *v1alpha1.Restore) bool {
	return restore.DeletionTimestamp != nil && (
		v1alpha1.IsRestoreComplete(restore) ||
		v1alpha1.IsRestoreFailed(restore)
	)
}
