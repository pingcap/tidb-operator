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

package resourcemanagergroup

import (
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanagergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		common.TaskContextObject[scope.ResourceManagerGroup](state, r.Client),
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.ResourceManagerGroup](state)),

		common.TaskContextCluster[scope.ResourceManagerGroup](state, r.Client),
		task.IfBreak(common.CondClusterIsPaused(state)),
		task.IfBreak(common.CondFeatureGatesIsNotSynced[scope.ResourceManagerGroup](state)),

		common.TaskContextSlice[scope.ResourceManagerGroup](state, r.Client),

		task.IfBreak(common.CondObjectIsDeleting[scope.ResourceManagerGroup](state),
			common.TaskGroupFinalizerDel[scope.ResourceManagerGroup](state, r.Client),
			common.TaskGroupConditionReady[scope.ResourceManagerGroup](state),
			common.TaskGroupConditionSynced[scope.ResourceManagerGroup](state),
			common.TaskStatusRevisionAndReplicas[scope.ResourceManagerGroup](state),
			common.TaskStatusPersister[scope.ResourceManagerGroup](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.ResourceManagerGroup](state, r.Client),

		task.IfBreak(
			common.CondClusterIsSuspending(state),
			common.TaskGroupConditionSuspended[scope.ResourceManagerGroup](state),
			common.TaskGroupConditionReady[scope.ResourceManagerGroup](state),
			common.TaskGroupConditionSynced[scope.ResourceManagerGroup](state),
			common.TaskStatusPersister[scope.ResourceManagerGroup](state, r.Client),
		),

		common.TaskRevision[runtime.ResourceManagerGroupTuple](state, r.Client),
		tasks.TaskService(state, r.Client),
		tasks.TaskUpdater(state, r.Client, r.AllocateFactory),
		common.TaskGroupStatusSelector[scope.ResourceManagerGroup](state),
		common.TaskGroupConditionSuspended[scope.ResourceManagerGroup](state),
		common.TaskGroupConditionReady[scope.ResourceManagerGroup](state),
		common.TaskGroupConditionSynced[scope.ResourceManagerGroup](state),
		common.TaskStatusRevisionAndReplicas[scope.ResourceManagerGroup](state),
		common.TaskStatusPersister[scope.ResourceManagerGroup](state, r.Client),
	)

	return runner
}
