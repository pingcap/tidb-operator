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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskContextResource[T any, PT Object[T]](name string, w ResourceInitializer[T], c client.Client, shouldExist bool) task.Task {
	return taskContextResource[T, PT](name, w, c, shouldExist)
}

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

func TaskContextPD(state PDStateInitializer, c client.Client) task.Task {
	w := state.PDInitializer()
	return taskContextResource("PD", w, c, false)
}

func TaskContextTiKV(state TiKVStateInitializer, c client.Client) task.Task {
	w := state.TiKVInitializer()
	return taskContextResource("TiKV", w, c, false)
}

func TaskContextTiDB(state TiDBStateInitializer, c client.Client) task.Task {
	w := state.TiDBInitializer()
	return taskContextResource("TiDB", w, c, false)
}

func TaskContextTiFlash(state TiFlashStateInitializer, c client.Client) task.Task {
	w := state.TiFlashInitializer()
	return taskContextResource("TiFlash", w, c, false)
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

func TaskContextTiFlashGroup(state TiFlashGroupStateInitializer, c client.Client) task.Task {
	w := state.TiFlashGroupInitializer()
	return taskContextResource("TiFlashGroup", w, c, false)
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

func TaskFeatureGates(state ClusterState) task.Task {
	return task.NameTaskFunc("FeatureGates", func(context.Context) task.Result {
		if err := features.Verify(state.Cluster()); err != nil {
			return task.Fail().With("feature gates are not up to date: %v", err)
		}

		return task.Complete().With("feature gates are initialized")
	})
}

type ContextClusterNewer[
	F client.Object,
] interface {
	Object() F
	SetCluster(c *v1alpha1.Cluster)
}

func TaskContextCluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](state ContextClusterNewer[F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextCluster", func(ctx context.Context) task.Result {
		cluster := v1alpha1.Cluster{}
		obj := state.Object()

		key := types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      coreutil.Cluster[S](obj),
		}
		if err := c.Get(ctx, key, &cluster); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get %s: %v", key, err)
			}

			return task.Fail().With("cannot find %s: %v", key, err)
		}
		state.SetCluster(&cluster)
		return task.Complete().With("cluster is set")
	})
}
