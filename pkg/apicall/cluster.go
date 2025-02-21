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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func ListGroups[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](ctx context.Context, c client.Client, ns, cluster string) ([]F, error) {
	l := scope.NewList[S]()
	if err := c.List(ctx, l, client.InNamespace(ns), client.MatchingFields{
		"spec.cluster.name": cluster,
	}); err != nil {
		return nil, err
	}

	objs := make([]F, 0, meta.LenList(l))
	if err := meta.EachListItem(l, func(item kuberuntime.Object) error {
		obj, ok := item.(F)
		if !ok {
			// unreachable
			return fmt.Errorf("cannot convert item")
		}
		objs = append(objs, obj)
		return nil
	}); err != nil {
		// unreachable
		return nil, err
	}

	return objs, nil
}

func GetCluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](ctx context.Context, c client.Client, obj F) (*v1alpha1.Cluster, error) {
	cluster := &v1alpha1.Cluster{}

	key := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      coreutil.Cluster[S](obj),
	}
	if err := c.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}
