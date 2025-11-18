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

package apicall

import (
	"cmp"
	"context"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

// ListInstances returns instances managed by a specified group
func ListInstances[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](ctx context.Context, c client.Client, g GF) ([]I, error) {
	l := scope.NewList[IS]()
	if err := c.List(ctx, l, client.InNamespace(g.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   coreutil.Cluster[GS](g),
		v1alpha1.LabelKeyGroup:     g.GetName(),
		v1alpha1.LabelKeyComponent: scope.Component[GS](),
	}); err != nil {
		return nil, err
	}

	objs := scope.GetItems[IS](l)

	// always sort instances
	slices.SortFunc(objs, func(a, b I) int {
		return cmp.Compare(a.GetName(), b.GetName())
	})

	return objs, nil
}

// ListPeerInstances returns peers of an instance in a same cluster
// NOTE:
// - All peers in the same cluster will be returned
// - Input instance will also be returned
func ListPeerInstances[
	S scope.InstanceList[F, T, L],
	F client.Object,
	T runtime.Instance,
	L client.ObjectList,
](ctx context.Context, c client.Client, in F) ([]F, error) {
	return ListClusterInstances[S](ctx, c, in.GetNamespace(), coreutil.Cluster[S](in))
}

// NewInstanceListerWatcher returns a cache.ListerWatcher for a group
func NewInstanceListerWatcher[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](ctx context.Context, c client.Client, g GF) cache.ListerWatcher {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (kuberuntime.Object, error) {
			list := scope.NewList[IS]()
			if err := c.List(ctx, list, &client.ListOptions{
				Namespace: g.GetNamespace(),
				LabelSelector: labels.SelectorFromSet(labels.Set{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   coreutil.Cluster[GS](g),
					v1alpha1.LabelKeyGroup:     g.GetName(),
					v1alpha1.LabelKeyComponent: scope.Component[GS](),
				}),
				Raw: &options,
			}); err != nil {
				return nil, err
			}
			return list, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			list := scope.NewList[IS]()
			return c.Watch(ctx, list, &client.ListOptions{
				Namespace: g.GetNamespace(),
				LabelSelector: labels.SelectorFromSet(labels.Set{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   coreutil.Cluster[GS](g),
					v1alpha1.LabelKeyGroup:     g.GetName(),
					v1alpha1.LabelKeyComponent: scope.Component[GS](),
				}),
				Raw: &options,
			})
		},
	}

	return lw
}
