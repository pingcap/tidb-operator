package common

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskGroupStatusSuspend[
	GT runtime.GroupTuple[OG, RG],
	OG client.Object,
	RG runtime.Group,
	I runtime.Instance,
](state GroupAndInstanceSliceState[RG, I], c client.Client) task.Task {
	return task.NameTaskFunc("StatusSuspend", func(ctx context.Context) task.Result {
		status := metav1.ConditionFalse
		reason := v1alpha1.ReasonSuspending
		message := "group is suspending"

		suspended := true
		g := state.Group()
		objs := state.Slice()
		for _, obj := range objs {
			if !meta.IsStatusConditionTrue(obj.Conditions(), v1alpha1.CondSuspended) {
				suspended = false
			}
		}
		if suspended {
			status = metav1.ConditionTrue
			reason = v1alpha1.ReasonSuspended
			message = "group is suspended"
		}
		needUpdate := SetStatusCondition(g, &metav1.Condition{
			Type:               v1alpha1.CondSuspended,
			Status:             status,
			ObservedGeneration: g.GetGeneration(),
			Reason:             reason,
			Message:            message,
		})
		needUpdate = SetStatusObservedGeneration(g) || needUpdate

		var t GT
		if needUpdate {
			if err := c.Status().Update(ctx, t.To(g)); err != nil {
				return task.Fail().With("cannot update status: %v", err)
			}
		}

		return task.Complete().With("status is updated")
	})
}

func TaskGroupStatus[
	GT runtime.GroupTuple[OG, RG],
	OG client.Object,
	RG runtime.Group,
	I runtime.Instance,
](state GroupAndInstanceSliceAndRevisionState[RG, I], c client.Client) task.Task {
	var gt GT
	return task.NameTaskFunc("Status", func(ctx context.Context) task.Result {
		g := state.Group()

		needUpdate := SetStatusCondition(g, &metav1.Condition{
			Type:               v1alpha1.CondSuspended,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: g.GetGeneration(),
			Reason:             v1alpha1.ReasonUnsuspended,
			Message:            "group is not suspended",
		})

		updateRevision, currentRevision, collisionCount := state.Revision()

		replicas, readyReplicas, updateReplicas, currentReplicas := calcReplicas(state.Slice(), updateRevision, currentRevision)

		// all instances are updated
		if updateReplicas == replicas {
			// update current revision
			currentRevision = updateRevision
			// update status of pdg version
			// TODO(liubo02): version of a group is hard to understand
			// We need to change it to a more meaningful field
			needUpdate = SetStatusVersion(g)
		}

		needUpdate = SetStatusObservedGeneration(g) || needUpdate
		needUpdate = SetStatusReplicas(g, replicas, readyReplicas, updateReplicas, currentReplicas) || needUpdate
		needUpdate = SetStatusRevision(g, updateRevision, currentRevision, collisionCount) || needUpdate

		if needUpdate {
			if err := c.Status().Update(ctx, gt.To(g)); err != nil {
				return task.Fail().With("cannot update status: %w", err)
			}
		}

		return task.Complete().With("status is synced")
	})
}

func calcReplicas[I runtime.Instance](instances []I, updateRevision, currentRevision string) (
	replicas,
	readyReplicas,
	updateReplicas,
	currentReplicas int32,
) {
	for _, peer := range instances {
		replicas++
		if peer.IsHealthy() {
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

func SetStatusCondition[O runtime.Object](obj O, cond *metav1.Condition) bool {
	conds := obj.Conditions()
	if meta.SetStatusCondition(&conds, *cond) {
		obj.SetConditions(conds)
		return true
	}
	return false
}

func SetStatusVersion[O runtime.Group](obj O) bool {
	v := obj.StatusVersion()
	if SetIfChanged(&v, obj.Version()) {
		obj.SetStatusVersion(v)
		return true
	}

	return false
}

func SetStatusObservedGeneration[O runtime.Object](obj O) bool {
	gen := obj.ObservedGeneration()
	if SetIfChanged(&gen, obj.GetGeneration()) {
		obj.SetObservedGeneration(gen)
		return true
	}

	return false
}

func SetStatusReplicas[O runtime.Group](obj O, newReplicas, newReady, newUpdate, newCurrent int32) bool {
	replicas, ready, update, current := obj.StatusReplicas()
	changed := SetIfChanged(&replicas, newReplicas)
	changed = SetIfChanged(&ready, newReady) || changed
	changed = SetIfChanged(&update, newUpdate) || changed
	changed = SetIfChanged(&current, newCurrent) || changed
	if changed {
		obj.SetStatusReplicas(replicas, ready, update, current)
		return changed
	}

	return false
}

func SetStatusRevision[O runtime.Group](obj O, newUpdate, newCurrent string, newCollisionCount int32) bool {
	update, current, collisionCount := obj.StatusRevision()
	changed := SetIfChanged(&update, newUpdate)
	changed = SetIfChanged(&current, newCurrent) || changed
	changed = NewAndSetIfChanged(&collisionCount, newCollisionCount) || changed
	if changed {
		obj.SetStatusRevision(update, current, collisionCount)
		return changed
	}

	return false
}

func SetIfChanged[T comparable](dst *T, src T) bool {
	if *dst != src {
		*dst = src
		return true
	}

	return false
}

func NewAndSetIfChanged[T comparable](dst **T, src T) bool {
	if *dst == nil {
		zero := new(T)
		if *zero == src {
			return false
		}
		*dst = zero
	}
	return SetIfChanged(*dst, src)
}
