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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination mock_generated.go -package=common ${GO_MODULE}/pkg/controllers/common StatusPersister,InstanceCondSuspendedUpdater,InstanceCondReadyUpdater,InstanceCondSyncedUpdater,InstanceCondOfflineUpdater,GroupCondSuspendedUpdater,GroupCondReadyUpdater,GroupCondSyncedUpdater,GroupStatusSelectorUpdater,StatusRevisionAndReplicasUpdater
package common

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/podutil"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/third_party/kubernetes/pkg/controller/statefulset"
)

type StatusPersister[T client.Object] interface {
	Object() T
	IsStatusChanged() bool
}

func TaskStatusPersister[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](state StatusPersister[F], c client.Client) task.Task {
	return task.NameTaskFunc("UpdateStatus", func(ctx context.Context) task.Result {
		obj := state.Object()
		needUpdate := coreutil.SetStatusObservedGeneration[S](obj)

		if state.IsStatusChanged() || needUpdate {
			if err := c.Status().Update(ctx, obj); err != nil {
				return task.Fail().With("cannot update status: %v", err)
			}
		}

		return task.Complete().With("status is updated")
	})
}

type StatusUpdater interface {
	SetStatusChanged()
}

type InstanceCondSuspendedUpdater[T client.Object] interface {
	StatusUpdater
	PodState
	ClusterState
	Object() T
}

func TaskInstanceConditionSuspended[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state InstanceCondSuspendedUpdater[F]) task.Task {
	return task.NameTaskFunc("CondSuspended", func(context.Context) task.Result {
		instance := state.Object()
		pod := state.Pod()
		cluster := state.Cluster()

		var needUpdate bool
		if coreutil.ShouldSuspendCompute(cluster) {
			if pod == nil {
				needUpdate = coreutil.SetStatusCondition[S](
					instance,
					*coreutil.Suspended(),
				) || needUpdate
			} else {
				needUpdate = coreutil.SetStatusCondition[S](
					instance,
					*coreutil.Suspending(),
				) || needUpdate
			}
		} else {
			needUpdate = coreutil.SetStatusCondition[S](
				instance,
				*coreutil.Unsuspended(),
			) || needUpdate
		}

		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("suspended condition is changed")
		}

		return task.Complete().With("suspended condition is not changed")
	})
}

func isPodRunning(state PodState, comp string) (isRunning bool, reason, msg string) {
	pod := state.Pod()
	switch {
	case pod == nil:
		reason = v1alpha1.ReasonPodNotCreated
	case state.IsPodTerminating():
		reason = v1alpha1.ReasonPodTerminating
	default:
		if err := podutil.IsContainerRunning(pod, comp); err != nil {
			return false, v1alpha1.ReasonPodNotRunning, err.Error()
		}
		isRunning = true
	}

	return isRunning, reason, ""
}

type InstanceCondRunningUpdater[T client.Object] interface {
	StatusUpdater
	PodState
	Object() T
}

func TaskInstanceConditionRunning[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state InstanceCondRunningUpdater[F]) task.Task {
	return task.NameTaskFunc("CondRunning", func(context.Context) task.Result {
		instance := state.Object()

		var needUpdate bool
		isRunning, reason, msg := isPodRunning(state, scope.Component[S]())

		var cond *metav1.Condition
		if isRunning {
			cond = coreutil.Running()
		} else {
			cond = coreutil.NotRunning(reason, msg)
		}

		needUpdate = coreutil.SetStatusCondition[S](
			instance,
			*cond,
		) || needUpdate

		if needUpdate {
			state.SetStatusChanged()
		}

		if !isRunning {
			return task.Wait().With("instance is not running: %s", coreutil.SprintCondition(cond))
		}

		return task.Complete().With("instance is running")
	})
}

type InstanceCondReadyUpdater[T client.Object] interface {
	StatusUpdater
	PodState
	HealthyState
	Object() T
}

func TaskInstanceConditionReady[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state InstanceCondReadyUpdater[F]) task.Task {
	return task.NameTaskFunc("CondReady", func(context.Context) task.Result {
		instance := state.Object()
		pod := state.Pod()

		var needUpdate, isReady bool
		var reason string
		switch {
		case pod == nil:
			reason = v1alpha1.ReasonPodNotCreated
		case state.IsPodTerminating():
			reason = v1alpha1.ReasonPodTerminating
		case !statefulset.IsPodRunningAndReady(pod):
			reason = v1alpha1.ReasonPodNotReady
		case !state.IsHealthy():
			reason = v1alpha1.ReasonInstanceNotHealthy
		default:
			isReady = true
			podReadyCond := statefulset.GetPodReadyCondition(&pod.Status)
			instanceReadyCond := coreutil.FindStatusCondition[S](instance, v1alpha1.CondReady)
			// If pod is not ready and then ready again, instance controller may not capture the unready stage
			// The updater depends on instance's ready condition to do next action
			if podReadyCond != nil &&
				instanceReadyCond != nil &&
				podReadyCond.LastTransitionTime.After(instanceReadyCond.LastTransitionTime.Time) {
				// reset ready condition to update last transition time
				coreutil.RemoveStatusCondition[S](instance, v1alpha1.CondReady)
			}
		}

		var cond *metav1.Condition
		if isReady {
			cond = coreutil.Ready()
		} else {
			cond = coreutil.Unready(reason)
		}

		needUpdate = coreutil.SetStatusCondition[S](
			instance,
			*cond,
		) || needUpdate

		if needUpdate {
			state.SetStatusChanged()
		}

		if !isReady {
			return task.Wait().With("instance is unready: %s", coreutil.SprintCondition(cond))
		}

		return task.Complete().With("instance is ready")
	})
}

type InstanceCondSyncedUpdater[T client.Object] interface {
	StatusUpdater
	PodState
	ClusterState
	Object() T
}

func TaskInstanceConditionSynced[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state InstanceCondSyncedUpdater[F]) task.Task {
	return task.NameTaskFunc("CondSynced", func(context.Context) task.Result {
		instance := state.Object()
		pod := state.Pod()
		cluster := state.Cluster()

		var needUpdate, isSynced bool
		var reason string
		if !instance.GetDeletionTimestamp().IsZero() || coreutil.ShouldSuspendCompute(cluster) {
			if pod == nil {
				isSynced = true
			} else {
				reason = v1alpha1.ReasonPodNotDeleted
			}
		} else {
			if pod == nil ||
				state.IsPodTerminating() ||
				pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash] != coreutil.UpdateRevision[S](instance) {
				reason = v1alpha1.ReasonPodNotUpToDate
			} else {
				// TODO(liubo02): now only check pod, pvcs should also be checked.
				isSynced = true
			}
		}

		var cond *metav1.Condition
		if isSynced {
			cond = coreutil.Synced()
		} else {
			cond = coreutil.Unsynced(reason)
		}

		needUpdate = coreutil.SetStatusCondition[S](
			instance,
			*cond,
		) || needUpdate

		if needUpdate {
			state.SetStatusChanged()
		}

		if !isSynced {
			return task.Wait().With("instance is unsynced: %s", coreutil.SprintCondition(cond))
		}

		return task.Complete().With("instance is synced")
	})
}

type GroupCondSuspendedUpdater[
	G client.Object,
	I client.Object,
] interface {
	StatusUpdater
	ClusterState

	SliceState[I]

	Object() G
}

func TaskGroupConditionSuspended[
	S scope.GroupInstance[F, T, IS],
	IS scope.Instance[IF, IT],
	F client.Object,
	T runtime.Group,
	IF client.Object,
	IT runtime.Instance,
](state GroupCondSuspendedUpdater[F, IF]) task.Task {
	return task.NameTaskFunc("CondSuspended", func(context.Context) task.Result {
		g := state.Object()
		objs := state.InstanceSlice()
		cluster := state.Cluster()

		var needUpdate bool

		if coreutil.ShouldSuspendCompute(cluster) {
			suspended := true
			for _, obj := range objs {
				if !meta.IsStatusConditionTrue(coreutil.StatusConditions[IS](obj), v1alpha1.CondSuspended) {
					suspended = false
				}
			}
			if suspended {
				needUpdate = coreutil.SetStatusCondition[S](
					g,
					*coreutil.Suspended(),
				) || needUpdate
			} else {
				needUpdate = coreutil.SetStatusCondition[S](
					g,
					*coreutil.Suspending(),
				) || needUpdate
			}
		} else {
			needUpdate = coreutil.SetStatusCondition[S](
				g,
				*coreutil.Unsuspended(),
			) || needUpdate
		}

		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("suspended condition is changed")
		}

		return task.Complete().With("suspended condition is not changed")
	})
}

type GroupCondReadyUpdater[
	G client.Object,
	I client.Object,
] interface {
	StatusUpdater
	SliceState[I]

	Object() G
}

func TaskGroupConditionReady[
	S scope.GroupInstance[F, T, IS],
	IS scope.Instance[IF, IT],
	F client.Object,
	T runtime.Group,
	IF client.Object,
	IT runtime.Instance,
](state GroupCondReadyUpdater[F, IF]) task.Task {
	return task.NameTaskFunc("CondReady", func(context.Context) task.Result {
		g := state.Object()
		var needUpdate bool

		replicas, readyReplicas := calcReadyReplicas[IS](state.InstanceSlice())
		specReplicas := coreutil.Replicas[S](g)

		if readyReplicas == replicas && replicas == specReplicas {
			needUpdate = coreutil.SetStatusCondition[S](g, *coreutil.Ready()) || needUpdate
		} else {
			// TODO(liubo02): more info when unready
			needUpdate = coreutil.SetStatusCondition[S](
				g,
				*coreutil.Unready(v1alpha1.ReasonNotAllInstancesReady),
			) || needUpdate
		}

		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("ready condition is changed")
		}

		return task.Complete().With("ready condition is not changed")
	})
}

type GroupCondSyncedUpdater[
	G client.Object,
	I client.Object,
] interface {
	StatusUpdater
	SliceState[I]
	RevisionState

	Object() G
}

func TaskGroupConditionSynced[
	S scope.GroupInstance[F, T, IS],
	IS scope.Instance[IF, IT],
	F client.Object,
	T runtime.Group,
	IF client.Object,
	IT runtime.Instance,
](state GroupCondSyncedUpdater[F, IF]) task.Task {
	return task.NameTaskFunc("CondSynced", func(context.Context) task.Result {
		g := state.Object()
		var needUpdate bool

		updateRevision, _, _ := state.Revision()
		replicas, updateReplicas := calcUpdateReplicas[IS](state.InstanceSlice(), updateRevision)

		// instances are updated and num of instances is as expected
		if updateReplicas == replicas && replicas == coreutil.Replicas[S](g) {
			needUpdate = coreutil.SetStatusCondition[S](
				g,
				*coreutil.Synced(),
			) || needUpdate
		} else {
			// TODO(liubo02): more info when unsynced
			needUpdate = coreutil.SetStatusCondition[S](
				g,
				*coreutil.Unsynced(v1alpha1.ReasonNotAllInstancesUpToDate),
			) || needUpdate
		}

		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("sync condition is changed")
		}

		return task.Complete().With("sync condition is not changed")
	})
}

type GroupStatusSelectorUpdater[
	G client.Object,
] interface {
	StatusUpdater

	Object() G
}

func TaskGroupStatusSelector[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](state GroupStatusSelectorUpdater[F]) task.Task {
	return task.NameTaskFunc("GroupStatusSelector", func(context.Context) task.Result {
		g := state.Object()

		needUpdate := coreutil.SetStatusSelector[S](g)

		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("selector is changed")
		}

		return task.Complete().With("selector is not changed")
	})
}

type StatusRevisionAndReplicasUpdater[
	G client.Object,
	I client.Object,
] interface {
	StatusUpdater
	SliceState[I]
	RevisionState

	Object() G
}

func TaskStatusRevisionAndReplicas[
	S scope.GroupInstance[F, T, IS],
	IS scope.Instance[IF, IT],
	F client.Object,
	T runtime.Group,
	IF client.Object,
	IT runtime.Instance,
](state StatusRevisionAndReplicasUpdater[F, IF]) task.Task {
	return task.NameTaskFunc("Status", func(context.Context) task.Result {
		g := state.Object()

		var needUpdate bool

		updateRevision, currentRevision, collisionCount := state.Revision()
		replicas, readyReplicas, updateReplicas, currentReplicas := calcReplicas[IS](
			state.InstanceSlice(),
			updateRevision,
			currentRevision,
		)

		// all instances are updated
		if updateReplicas == replicas && replicas == coreutil.Replicas[S](g) {
			currentRevision = updateRevision
			currentReplicas = updateReplicas

			// TODO(liubo02): version of a group is not reasonable
			// We need to change it to a more meaningful field
			needUpdate = coreutil.SetStatusVersion[S](g)
		}

		needUpdate = coreutil.SetStatusReplicas[S](g, replicas, readyReplicas, updateReplicas, currentReplicas) || needUpdate
		needUpdate = coreutil.SetStatusRevision[S](g, updateRevision, currentRevision, collisionCount) || needUpdate

		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("revision or replicas is changed")
		}

		return task.Complete().With("revision or replicas is not changed")
	})
}

func calcUpdateReplicas[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](instances []F, updateRevision string) (
	replicas,
	updateReplicas int32,
) {
	for _, peer := range instances {
		replicas++
		if coreutil.CurrentRevision[S](peer) == updateRevision {
			updateReplicas++
		}
	}

	return
}

func calcReadyReplicas[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](instances []F) (
	replicas,
	readyReplicas int32,
) {
	for _, peer := range instances {
		replicas++
		if coreutil.IsReady[S](peer) {
			readyReplicas++
		}
	}

	return
}

func calcReplicas[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](instances []F, updateRevision, currentRevision string) (
	replicas,
	readyReplicas,
	updateReplicas,
	currentReplicas int32,
) {
	for _, peer := range instances {
		replicas++
		if coreutil.IsReady[S](peer) {
			readyReplicas++
		}
		if coreutil.CurrentRevision[S](peer) == currentRevision {
			currentReplicas++
		}
		if coreutil.CurrentRevision[S](peer) == updateRevision {
			updateReplicas++
		}
	}

	return
}
