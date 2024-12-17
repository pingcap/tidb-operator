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
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type PDContextSetter interface {
	context.Context
	PDKey() types.NamespacedName
	SetPD(pd *v1alpha1.PD)
}

type PDGetter interface {
	GetPD() *v1alpha1.PD
}

func TaskContextPD(ctx PDContextSetter, c client.Client) task.Task {
	return task.NameTaskFunc("ContextPD", func() task.Result {
		var pd v1alpha1.PD
		if err := c.Get(ctx, ctx.PDKey(), &pd); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get pd instance %s: %v", ctx.PDKey(), err)
			}

			return task.Complete().With("pd instance has been deleted")
		}
		ctx.SetPD(&pd)
		return task.Complete().With("pd is set")
	})
}

type ClusterContextSetter interface {
	context.Context
	ClusterKey() types.NamespacedName
	SetCluster(cluster *v1alpha1.Cluster)
}

type ClusterGetter interface {
	GetCluster() *v1alpha1.Cluster
}

func TaskContextCluster(ctx ClusterContextSetter, c client.Client) task.Task {
	return task.NameTaskFunc("ContextCluster", func() task.Result {
		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, ctx.ClusterKey(), &cluster); err != nil {
			return task.Fail().With("cannot find cluster %s: %v", ctx.ClusterKey(), err)
		}
		ctx.SetCluster(&cluster)
		return task.Complete().With("cluster is set")
	})
}

type PodContextSetter interface {
	context.Context
	PodKey() types.NamespacedName
	SetPod(pod *corev1.Pod)
}

type PodGetter interface {
	GetPod() *corev1.Pod
}

func TaskContextPod(ctx PodContextSetter, c client.Client) task.Task {
	return task.NameTaskFunc("ContextPod", func() task.Result {
		var pod corev1.Pod
		if err := c.Get(ctx, ctx.PodKey(), &pod); err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("pod is not created")
			}
			return task.Fail().With("failed to get pod %s: %v", ctx.PodKey(), err)
		}

		ctx.SetPod(&pod)

		return task.Complete().With("pod is set")
	})
}

type PDSliceContextSetter interface {
	context.Context
	ClusterKey() types.NamespacedName
	SetPDSlice(pds []*v1alpha1.PD)
}

// TODO: combine with pd slice context in PDGroup controller
func TaskContextPDSlice(ctx PDSliceContextSetter, c client.Client) task.Task {
	return task.NameTaskFunc("ContextPDSlice", func() task.Result {
		var pdl v1alpha1.PDList
		ck := ctx.ClusterKey()
		if err := c.List(ctx, &pdl, client.InNamespace(ck.Namespace), client.MatchingLabels{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyCluster:   ck.Name,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
		}); err != nil {
			return task.Fail().With("cannot list pd peers: %v", err)
		}

		peers := []*v1alpha1.PD{}
		for i := range pdl.Items {
			peers = append(peers, &pdl.Items[i])
		}
		slices.SortFunc(peers, func(a, b *v1alpha1.PD) int {
			return cmp.Compare(a.Name, b.Name)
		})

		ctx.SetPDSlice(peers)

		return task.Complete().With("peers is set")
	})
}

type PodContext interface {
	context.Context
	PodGetter
}

func TaskSuspendPod(ctx PodContext, c client.Client) task.Task {
	return task.NameTaskFunc("SuspendPod", func() task.Result {
		pod := ctx.GetPod()
		if pod == nil {
			return task.Complete().With("pod has been deleted")
		}
		if !pod.GetDeletionTimestamp().IsZero() {
			return task.Complete().With("pod has been terminating")
		}
		if err := c.Delete(ctx, pod); err != nil {
			return task.Fail().With("can't delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}

		return task.Wait().With("pod is deleting")
	})
}
