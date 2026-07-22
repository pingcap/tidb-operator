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

package tikvgroup

import (
	"context"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func (r *Reconciler) ClusterEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			cluster := event.ObjectNew.(*v1alpha1.Cluster)

			groups, err := apicall.ListGroups[scope.TiKVGroup](ctx, r.Client, cluster.Namespace, cluster.Name)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					r.Logger.Error(err, "cannot list all tikv groups", "ns", cluster.Namespace, "cluster", cluster.Name)
				}
				return
			}

			for _, kvg := range groups {
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      kvg.Name,
						Namespace: kvg.Namespace,
					},
				})
			}
		},
	}
}

func (r *Reconciler) PlacementPolicyEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc: func(_ context.Context, event event.TypedCreateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			policy := event.Object.(*v1alpha1.PlacementPolicy)
			enqueueTiKVGroupsForPlacementPolicy(queue, policy)
		},
		UpdateFunc: func(_ context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			oldPolicy := event.ObjectOld.(*v1alpha1.PlacementPolicy)
			policy := event.ObjectNew.(*v1alpha1.PlacementPolicy)
			if oldPolicy.Spec.Cluster.Name == policy.Spec.Cluster.Name &&
				reflect.DeepEqual(oldPolicy.Spec.GroupRefs, policy.Spec.GroupRefs) {
				return
			}

			enqueueTiKVGroupsForPlacementPolicy(queue, oldPolicy)
			enqueueTiKVGroupsForPlacementPolicy(queue, policy)
		},
		DeleteFunc: func(_ context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			policy := event.Object.(*v1alpha1.PlacementPolicy)
			enqueueTiKVGroupsForPlacementPolicy(queue, policy)
		},
	}
}

func enqueueTiKVGroupsForPlacementPolicy(
	queue workqueue.TypedRateLimitingInterface[reconcile.Request],
	policy *v1alpha1.PlacementPolicy,
) {
	for _, name := range coreutil.PlacementPolicyTiKVGroupRefNames(policy) {
		queue.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: policy.Namespace,
			},
		})
	}
}
