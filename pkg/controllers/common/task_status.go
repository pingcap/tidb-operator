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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination mock_generated.go -package=common ${GO_MODULE}/pkg/controllers/common StatusPersister,InstanceCondSuspendedUpdater,InstanceCondReadyUpdater,GroupCondSuspendedUpdater,GroupCondReadyUpdater,GroupCondSyncedUpdater,StatusRevisionAndReplicasUpdater
package common

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
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
	return task.NameTaskFunc("CondReady", func(ctx context.Context) task.Result {
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
			return task.Wait().With("instance is unready")
		}

		return task.Complete().With("instance is ready")
	})
}

type GroupCondSuspendedUpdater[
	T client.Object,
	I runtime.Instance,
] interface {
	StatusUpdater
	ClusterState

	InstanceSliceState[I]

	Object() T
}

func TaskGroupConditionSuspended[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
	I runtime.Instance,
](state GroupCondSuspendedUpdater[F, I]) task.Task {
	return task.NameTaskFunc("CondSuspended", func(context.Context) task.Result {
		g := state.Object()
		objs := state.Slice()
		cluster := state.Cluster()

		var needUpdate bool

		if coreutil.ShouldSuspendCompute(cluster) {
			suspended := true
			for _, obj := range objs {
				if !meta.IsStatusConditionTrue(obj.Conditions(), v1alpha1.CondSuspended) {
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
	T client.Object,
	I runtime.Instance,
] interface {
	StatusUpdater
	InstanceSliceState[I]

	Object() T
}

func TaskGroupConditionReady[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
	I runtime.Instance,
](state GroupCondReadyUpdater[F, I]) task.Task {
	return task.NameTaskFunc("CondReady", func(context.Context) task.Result {
		g := state.Object()
		var needUpdate bool

		replicas, readyReplicas := calcReadyReplicas(state.Slice())
		if readyReplicas == replicas && replicas != 0 {
			needUpdate = coreutil.SetStatusCondition[S](
				g,
				*coreutil.Ready(),
			) || needUpdate
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
	T client.Object,
	I runtime.Instance,
] interface {
	StatusUpdater
	InstanceSliceState[I]
	RevisionState

	Object() T
}

func TaskGroupConditionSynced[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
	I runtime.Instance,
](state GroupCondSyncedUpdater[F, I]) task.Task {
	return task.NameTaskFunc("CondSynced", func(context.Context) task.Result {
		g := state.Object()
		var needUpdate bool

		updateRevision, _, _ := state.Revision()
		replicas, updateReplicas := calcUpdateReplicas(state.Slice(), updateRevision)

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
				*coreutil.Unsynced(),
			) || needUpdate
		}

		if needUpdate {
			state.SetStatusChanged()
			return task.Complete().With("sync condition is changed")
		}

		return task.Complete().With("sync condition is not changed")
	})
}

type StatusRevisionAndReplicasUpdater[
	T client.Object,
	I runtime.Instance,
] interface {
	StatusUpdater
	InstanceSliceState[I]
	RevisionState

	Object() T
}

func TaskStatusRevisionAndReplicas[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
	I runtime.Instance,
](state StatusRevisionAndReplicasUpdater[F, I]) task.Task {
	return task.NameTaskFunc("Status", func(context.Context) task.Result {
		g := state.Object()

		var needUpdate bool

		updateRevision, currentRevision, collisionCount := state.Revision()
		replicas, readyReplicas, updateReplicas, currentReplicas := calcReplicas(state.Slice(), updateRevision, currentRevision)

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

func calcUpdateReplicas[I runtime.Instance](instances []I, updateRevision string) (
	replicas,
	updateReplicas int32,
) {
	for _, peer := range instances {
		replicas++
		if peer.CurrentRevision() == updateRevision {
			updateReplicas++
		}
	}

	return
}

func calcReadyReplicas[I runtime.Instance](instances []I) (
	replicas,
	readyReplicas int32,
) {
	for _, peer := range instances {
		replicas++
		if peer.IsReady() {
			readyReplicas++
		}
	}

	return
}

func calcReplicas[I runtime.Instance](instances []I, updateRevision, currentRevision string) (
	replicas,
	readyReplicas,
	updateReplicas,
	currentReplicas int32,
) {
	for _, peer := range instances {
		replicas++
		if peer.IsReady() {
			readyReplicas++
		}
		if peer.CurrentRevision() == currentRevision {
			currentReplicas++
		}
		if peer.CurrentRevision() == updateRevision {
			updateReplicas++
		}
	}

	return
}
