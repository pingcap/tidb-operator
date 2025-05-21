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

package tiproxygroup

import (
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiproxygroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tiproxygroup
		common.TaskContextObject[scope.TiProxyGroup](state, r.Client),
		// if it's gone just return
		task.IfBreak(common.CondGroupHasBeenDeleted(state)),

		// get cluster
		common.TaskContextCluster[scope.TiProxyGroup](state, r.Client),
		common.TaskFeatureGates(state),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),

		// get all tiproxies
		common.TaskContextTiProxySlice(state, r.Client),

		task.IfBreak(common.CondGroupIsDeleting(state),
			common.TaskGroupFinalizerDel[scope.TiProxyGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.TiProxyGroup](state),
			common.TaskGroupConditionSynced[scope.TiProxyGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.TiProxyGroup](state),
			common.TaskStatusPersister[scope.TiProxyGroup](state, r.Client),
		),
		common.TaskGroupFinalizerAdd[runtime.TiProxyGroupTuple](state, r.Client),

		common.TaskRevision[runtime.TiProxyGroupTuple](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.TiProxyGroup](state),
			common.TaskGroupConditionReady[scope.TiProxyGroup](state),
			common.TaskGroupConditionSynced[scope.TiProxyGroup](state),
			common.TaskStatusPersister[scope.TiProxyGroup](state, r.Client),
		),

		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client),
		common.TaskGroupConditionSuspended[scope.TiProxyGroup](state),
		common.TaskGroupConditionReady[scope.TiProxyGroup](state),
		common.TaskGroupConditionSynced[scope.TiProxyGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.TiProxyGroup](state),
		tasks.TaskStatusAvailable(state),
		common.TaskStatusPersister[scope.TiProxyGroup](state, r.Client),
	)

	return runner
}
