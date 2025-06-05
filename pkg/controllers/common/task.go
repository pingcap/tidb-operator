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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func taskContextResourceSlice[T any, PT Object[T]](
	name string,
	w ResourceSliceInitializer[T],
	l client.ObjectList,
	c client.Client,
) task.Task {
	return task.NameTaskFunc("Context"+name, func(ctx context.Context) task.Result {
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

func TaskContextPDSlice(state PDSliceStateInitializer, c client.Client) task.Task {
	w := state.PDSliceInitializer()
	return taskContextResourceSlice("PDSlice", w, &v1alpha1.PDList{}, c)
}

func TaskContextTiKVSlice(state TiKVSliceStateInitializer, c client.Client) task.Task {
	w := state.TiKVSliceInitializer()
	return taskContextResourceSlice("TiKVSlice", w, &v1alpha1.TiKVList{}, c)
}

func TaskContextTiDBSlice(state TiDBSliceStateInitializer, c client.Client) task.Task {
	w := state.TiDBSliceInitializer()
	return taskContextResourceSlice("TiDBSlice", w, &v1alpha1.TiDBList{}, c)
}

func TaskContextTiFlashSlice(state TiFlashSliceStateInitializer, c client.Client) task.Task {
	w := state.TiFlashSliceInitializer()
	return taskContextResourceSlice("TiFlashSlice", w, &v1alpha1.TiFlashList{}, c)
}

func TaskContextTiCDCSlice(state TiCDCSliceStateInitializer, c client.Client) task.Task {
	w := state.TiCDCSliceInitializer()
	return taskContextResourceSlice("TiCDCSlice", w, &v1alpha1.TiCDCList{}, c)
}

func TaskContextTiProxySlice(state TiProxySliceStateInitializer, c client.Client) task.Task {
	w := state.TiProxySliceInitializer()
	return taskContextResourceSlice("TiProxySlice", w, &v1alpha1.TiProxyList{}, c)
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

		return task.Retry(task.DefaultRequeueAfter).With("pod is deleting")
	})
}

type ContextObjectNewer[
	F client.Object,
] interface {
	Key() types.NamespacedName
	SetObject(f F)
}

func TaskContextObject[
	S scope.Object[F, T],
	F Object[O],
	T runtime.Object,
	O any,
](state ContextObjectNewer[F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextObject", func(ctx context.Context) task.Result {
		key := state.Key()
		var obj F = new(O)
		if err := c.Get(ctx, key, obj); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get %s: %v", key, err)
			}

			return task.Complete().With("obj %s does not exist", key)
		}
		state.SetObject(obj)
		return task.Complete().With("object is set")
	})
}

type ContextSliceNewer[
	GF client.Object,
	IF client.Object,
] interface {
	ObjectState[GF]
	SetInstanceSlice(f []IF)
}

func TaskContextSlice[
	S scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](state ContextSliceNewer[GF, I], c client.Client) task.Task {
	return task.NameTaskFunc("ContextSlice", func(ctx context.Context) task.Result {
		g := state.Object()
		objs, err := apicall.ListInstances[S](ctx, c, g)
		if err != nil {
			return task.Fail().With("cannot get instance slice: %v", err)
		}

		state.SetInstanceSlice(objs)
		return task.Complete().With("instance slice is set")
	})
}

type ContextClusterNewer[
	F client.Object,
] interface {
	ObjectState[F]
	SetCluster(c *v1alpha1.Cluster)
}

func TaskContextCluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](state ContextClusterNewer[F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextCluster", func(ctx context.Context) task.Result {
		cluster, err := apicall.GetCluster[S](ctx, c, state.Object())
		if err != nil {
			return task.Fail().With("cannot get cluster: %v", err)
		}
		state.SetCluster(cluster)
		return task.Complete().With("cluster is set")
	})
}

type ContextPodNewer[
	F client.Object,
] interface {
	ObjectState[F]
	SetPod(pod *corev1.Pod)
}

func TaskContextPod[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state ContextPodNewer[F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextPod", func(ctx context.Context) task.Result {
		pod, err := apicall.GetPod[S](ctx, c, state.Object())
		if err != nil {
			return task.Fail().With("cannot get pod: %v", err)
		}
		if pod == nil {
			return task.Complete().With("pod doesn't exist")
		}

		state.SetPod(pod)
		return task.Complete().With("pod is set")
	})
}
