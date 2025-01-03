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

package common

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerr "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func taskContextResource[T any, PT Object[T]](name string, w ResourceInitializer[T], c client.Client, shouldExist bool) task.Task {
	return task.NameTaskFunc("Context"+name, func(ctx context.Context) task.Result {
		var obj PT = new(T)
		key := types.NamespacedName{
			Namespace: w.Namespace(),
			Name:      w.Name(),
		}
		if err := c.Get(ctx, key, obj); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get %s: %v", key, err)
			}

			if shouldExist {
				return task.Fail().With("cannot find %s: %v", key, err)
			}

			return task.Complete().With("obj %s does not exist", key)
		}
		w.Set(obj)
		return task.Complete().With("%s is set", strings.ToLower(name))
	})
}

func taskContextResourceSlice[T any, L any, PT Object[T], PL ObjectList[L]](
	name string,
	w ResourceSliceInitializer[T],
	c client.Client,
) task.Task {
	return task.NameTaskFunc("Context"+name, func(ctx context.Context) task.Result {
		var l PL = new(L)
		ns := w.Namespace()
		labels := w.Labels()

		if err := c.List(ctx, l, client.InNamespace(ns), client.MatchingLabels(labels)); err != nil {
			return task.Fail().With("cannot list objs: %v", err)
		}

		objs := make([]*T, 0, meta.LenList(l))
		if err := meta.EachListItem(l, func(item kuberuntime.Object) error {
			obj, ok := item.(PT)
			if !ok {
				// unreachable
				return fmt.Errorf("cannot convert item")
			}
			objs = append(objs, obj)
			return nil
		}); err != nil {
			// unreachable
			return task.Fail().With("cannot extract list objs: %v", err)
		}

		slices.SortFunc(objs, func(a, b *T) int {
			var pa, pb PT = a, b
			return cmp.Compare(pa.GetName(), pb.GetName())
		})

		w.Set(objs)

		return task.Complete().With("peers is set")
	})
}

func TaskContextPD(state PDStateInitializer, c client.Client) task.Task {
	w := state.PDInitializer()
	return taskContextResource("PD", w, c, false)
}

func TaskContextCluster(state ClusterStateInitializer, c client.Client) task.Task {
	w := state.ClusterInitializer()
	return taskContextResource("Cluster", w, c, true)
}

func TaskContextPod(state PodStateInitializer, c client.Client) task.Task {
	w := state.PodInitializer()
	return taskContextResource("Pod", w, c, false)
}

func TaskContextPDGroup(state PDGroupStateInitializer, c client.Client) task.Task {
	w := state.PDGroupInitializer()
	return taskContextResource("PDGroup", w, c, false)
}

func TaskContextTiKVGroup(state TiKVGroupStateInitializer, c client.Client) task.Task {
	w := state.TiKVGroupInitializer()
	return taskContextResource("TiKVGroup", w, c, false)
}

func TaskContextTiDBGroup(state TiDBGroupStateInitializer, c client.Client) task.Task {
	w := state.TiDBGroupInitializer()
	return taskContextResource("TiDBGroup", w, c, false)
}

func TaskContextPDSlice(state PDSliceStateInitializer, c client.Client) task.Task {
	w := state.PDSliceInitializer()
	return taskContextResourceSlice[v1alpha1.PD, v1alpha1.PDList]("PDSlice", w, c)
}

func TaskContextTiKVSlice(state TiKVSliceStateInitializer, c client.Client) task.Task {
	w := state.TiKVSliceInitializer()
	return taskContextResourceSlice[v1alpha1.TiKV, v1alpha1.TiKVList]("TiKVSlice", w, c)
}

func TaskContextTiDBSlice(state TiDBSliceStateInitializer, c client.Client) task.Task {
	w := state.TiDBSliceInitializer()
	return taskContextResourceSlice[v1alpha1.TiDB, v1alpha1.TiDBList]("TiDBSlice", w, c)
}

func TaskSuspendPod(state PodState, c client.Client) task.Task {
	return task.NameTaskFunc("SuspendPod", func(ctx context.Context) task.Result {
		pod := state.Pod()
		if pod == nil {
			return task.Complete().With("pod has been deleted")
		}
		if !pod.GetDeletionTimestamp().IsZero() {
			return task.Complete().With("pod has been terminating")
		}
		if err := c.Delete(ctx, pod); err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("pod is deleted")
			}
			return task.Fail().With("can't delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}

		return task.Wait().With("pod is deleting")
	})
}

func TaskGroupFinalizerAdd[
	GT runtime.GroupTuple[OG, RG],
	OG client.Object,
	RG runtime.Group,
](state GroupState[RG], c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerAdd", func(ctx context.Context) task.Result {
		var t GT
		if err := k8s.EnsureFinalizer(ctx, c, t.To(state.Group())); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %v", err)
		}
		return task.Complete().With("finalizer is added")
	})
}

const defaultDelWaitTime = 10 * time.Second

func TaskGroupFinalizerDel[
	GT runtime.GroupTuple[OG, RG],
	IT runtime.InstanceTuple[OI, RI],
	OG client.Object,
	RG runtime.Group,
	OI client.Object,
	RI runtime.Instance,
](state GroupAndInstanceSliceState[RG, RI], c client.Client) task.Task {
	var it IT
	var gt GT
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		var errList []error
		var names []string
		for _, peer := range state.Slice() {
			names = append(names, peer.GetName())
			if peer.GetDeletionTimestamp().IsZero() {
				if err := c.Delete(ctx, it.To(peer)); err != nil {
					if errors.IsNotFound(err) {
						continue
					}
					errList = append(errList, fmt.Errorf("try to delete the instance %v failed: %w", peer.GetName(), err))
					continue
				}
			}
		}

		if len(errList) != 0 {
			return task.Fail().With("failed to delete all instances: %v", utilerr.NewAggregate(errList))
		}

		if len(names) != 0 {
			return task.Retry(defaultDelWaitTime).With("wait for all instances being removed, %v still exists", names)
		}

		wait, err := k8s.DeleteGroupSubresource(ctx, c, state.Group(), &corev1.ServiceList{})
		if err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}
		if wait {
			return task.Wait().With("wait all subresources deleted")
		}

		if err := k8s.RemoveFinalizer(ctx, c, gt.To(state.Group())); err != nil {
			return task.Fail().With("failed to ensure finalizer has been removed: %w", err)
		}

		return task.Complete().With("finalizer has been removed")
	})
}

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
		needUpdate := SetStatusConditionIfChanged(g, &metav1.Condition{
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

func SetStatusConditionIfChanged[O runtime.Object](obj O, cond *metav1.Condition) bool {
	conds := obj.Conditions()
	if meta.SetStatusCondition(&conds, *cond) {
		obj.SetConditions(conds)
		return true
	}
	return false
}

func SetStatusObservedGeneration[O runtime.Object](obj O) bool {
	actual := obj.GetGeneration()
	current := obj.ObservedGeneration()

	if current != actual {
		obj.SetObservedGeneration(actual)
		return true
	}

	return false
}
