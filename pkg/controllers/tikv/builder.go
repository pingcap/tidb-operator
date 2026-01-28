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

package tikv

import (
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikv/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func (r *Reconciler) NewRunner(state *tasks.ReconcileContext, reporter task.TaskReporter) task.TaskRunner {
	runner := task.NewTaskRunner(reporter,
		// get tikv
		common.TaskContextObject[scope.TiKV](state, r.Client),
		common.TaskTrack[scope.TiKV](state, r.Tracker),
		// if it's deleted just return
		task.IfBreak(common.CondObjectHasBeenDeleted[scope.TiKV](state)),

		// get cluster info, FinalizerDel will use it
		common.TaskContextCluster[scope.TiKV](state, r.Client),

		// check whether it's paused
		task.IfBreak(common.CondClusterIsPaused(state)),

		// if the cluster is deleting, del all subresources and remove the finalizer directly
		task.IfBreak(common.CondClusterIsDeleting(state),
			common.TaskInstanceFinalizerDel[scope.TiKV](state, r.Client, tasks.SubresourceLister),
		),

		// return if cluster's status is not updated
		task.IfBreak(common.CondClusterPDAddrIsNotRegistered(state)),

		// if instance is not deleting but store is offlined
		task.IfBreak(common.CondObjectIsNotDeletingButOfflined[scope.TiKV](state),
			common.TaskDeleteOfflinedStore[scope.TiKV](state, r.Client),
		),

		// get pod and check whether the cluster is suspending
		common.TaskContextPod[scope.TiKV](state, r.Client),
		// get info from pd
		tasks.TaskContextInfoFromPD(state, r.PDClientManager),

		// if instance is deleting and store is removed
		task.IfBreak(ObjectIsDeletingAndStoreIsRemoved(state),
			tasks.TaskEndEvictLeader(state, r.PDClientManager),
			common.TaskInstanceFinalizerDel[scope.TiKV](state, r.Client, tasks.SubresourceLister),
		),

		// if instance is deleting, ensure spec.offline is set
		task.IfBreak(ObjectIsDeletingAndNotOffline(state),
			common.TaskSetOffline[scope.TiKV](state, r.Client),
		),

		common.TaskFinalizerAdd[scope.TiKV](state, r.Client),

		// check whether the cluster is suspending
		// if cluster is suspending, we cannot handle any tikv deletion
		task.IfBreak(common.CondClusterIsSuspending(state),
			// NOTE: suspend tikv pod should delete with grace peroid
			// TODO(liubo02): combine with the common one
			tasks.TaskSuspendPod(state, r.Client),
			common.TaskInstanceConditionSuspended[scope.TiKV](state),
			common.TaskInstanceConditionSynced[scope.TiKV](state),
			common.TaskInstanceConditionReady[scope.TiKV](state),
			common.TaskInstanceConditionRunning[scope.TiKV](state),
			common.TaskInstanceConditionOffline[scope.TiKV](state),
			common.TaskStatusPersister[scope.TiKV](state, r.Client),
		),

		// normal process
		tasks.TaskOfflineStore(state, r.PDClientManager),
		tasks.TaskConfigMap(state, r.Client),
		common.TaskPVC[scope.TiKV](state, r.Client, r.VolumeModifierFactory, tasks.PVCNewer()),
		tasks.TaskPod(state, r.Client),
		tasks.TaskStoreLabels(state, r.Client, r.PDClientManager),
		tasks.TaskEvictLeader(state, r.PDClientManager),
		common.TaskInstanceConditionSynced[scope.TiKV](state),
		// only set ready if pd is synced
		task.If(PDIsSynced(state),
			common.TaskInstanceConditionReady[scope.TiKV](state),
			common.TaskInstanceConditionOffline[scope.TiKV](state),
		),
		common.TaskInstanceConditionRunning[scope.TiKV](state),
		tasks.TaskStatus(state, r.Client),
	)

	return runner
}

func ObjectIsDeletingAndStoreIsRemoved(state *tasks.ReconcileContext) task.Condition {
	return task.CondFunc(func() bool {
		return !state.Object().GetDeletionTimestamp().IsZero() && state.PDSynced &&
			(state.GetStoreState() == pdv1.NodeStateRemoved || state.Store == nil)
	})
}

func ObjectIsDeletingAndNotOffline(state *tasks.ReconcileContext) task.Condition {
	return task.CondFunc(func() bool {
		obj := state.Object()
		return !obj.GetDeletionTimestamp().IsZero() && !coreutil.IsOffline[scope.TiKV](obj)
	})
}

func PDIsSynced(state *tasks.ReconcileContext) task.Condition {
	return task.CondFunc(func() bool {
		return state.PDSynced
	})
}
