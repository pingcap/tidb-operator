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

package tidb

import (
	"context"

	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tidb/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tidb
		common.TaskContextObject[scope.TiDB](state, r.Client),
		common.TaskTrack[scope.TiDB](state, r.Tracker),
		// if it's deleted just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.TiDB](state)),

		// get cluster info, FinalizerDel will use it
		common.TaskContextCluster[scope.TiDB](state, r.Client),
		// if it's paused just return
		task.IfBreak(common.CondClusterIsPaused(state)),
		// if the cluster is deleting, del all subresources and remove the finalizer directly
		task.IfBreak(common.CondClusterIsDeleting(state),
			tasks.TaskFinalizerDel(state, r.Client),
		),
		// return if cluster's status is not updated
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),

		task.IfBreak(common.CondObjectIsDeleting[scope.TiDB](state),
			tasks.TaskFinalizerDel(state, r.Client),
			// TODO(liubo02): if the finalizer has been removed, no need to update status
			common.TaskInstanceConditionSynced[scope.TiDB](state),
			common.TaskInstanceConditionReady[scope.TiDB](state),
			common.TaskInstanceConditionRunning[scope.TiDB](state),
			common.TaskStatusPersister[scope.TiDB](state, r.Client),
		),
		common.TaskFinalizerAdd[scope.TiDB](state, r.Client),

		// get pod and check whether the cluster is suspending
		common.TaskContextPod[scope.TiDB](state, r.Client),
		task.IfBreak(common.CondClusterIsSuspending(state),
			common.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.TiDB](state),
			common.TaskInstanceConditionSynced[scope.TiDB](state),
			common.TaskInstanceConditionReady[scope.TiDB](state),
			common.TaskInstanceConditionRunning[scope.TiDB](state),
			common.TaskStatusPersister[scope.TiDB](state, r.Client),
		),

		// normal process
		tasks.TaskContextInfoFromPDAndTiDB(state, r.Client, r.PDClientManager),
		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.TiDB](state, r.Client, r.VolumeModifierFactory, tasks.PVCNewer()),
		tasks.TaskPod(state, r.Client),
		common.TaskServerLabels[scope.TiDB](state, r.Client, func(ctx context.Context, labels map[string]string) error {
			return state.TiDBClient.SetServerLabels(ctx, labels)
		}),
		common.TaskInstanceConditionSynced[scope.TiDB](state),
		common.TaskInstanceConditionReady[scope.TiDB](state),
		common.TaskInstanceConditionRunning[scope.TiDB](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}
