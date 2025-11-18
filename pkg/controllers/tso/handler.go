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

package tso

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
)

func (r *Reconciler) ClusterEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			oldObj := event.ObjectOld.(*v1alpha1.Cluster)
			newObj := event.ObjectNew.(*v1alpha1.Cluster)

			if newObj.Status.PD != oldObj.Status.PD {
				r.Logger.Info("pd url is updating", "from", oldObj.Status.PD, "to", newObj.Status.PD)
			} else if !reflect.DeepEqual(oldObj.Spec.SuspendAction, newObj.Spec.SuspendAction) {
				r.Logger.Info("suspend action is updating", "from", oldObj.Spec.SuspendAction, "to", newObj.Spec.SuspendAction)
			} else if oldObj.Spec.Paused != newObj.Spec.Paused {
				r.Logger.Info("cluster paused is updating", "from", oldObj.Spec.Paused, "to", newObj.Spec.Paused)
			} else {
				return
			}

			var tl v1alpha1.TSOList
			if err := r.Client.List(ctx, &tl, client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   newObj.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTSO,
			}, client.InNamespace(newObj.Namespace)); err != nil {
				r.Logger.Error(err, "cannot list all tso instances", "ns", newObj.Namespace, "cluster", newObj.Name)
				return
			}

			for i := range tl.Items {
				tso := &tl.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      tso.Name,
						Namespace: tso.Namespace,
					},
				})
			}
		},
		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			obj := event.Object.(*v1alpha1.Cluster)

			r.Logger.Info("enqueue all tso instances because of the cluster is deleting", "ns", obj.Namespace, "cluster", obj.Name)

			var tl v1alpha1.TSOList
			if err := r.Client.List(ctx, &tl, client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   obj.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTSO,
			}, client.InNamespace(obj.Namespace)); err != nil {
				r.Logger.Error(err, "cannot list all tso instances", "ns", obj.Namespace, "cluster", obj.Name)
				return
			}

			for i := range tl.Items {
				tso := &tl.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      tso.Name,
						Namespace: tso.Namespace,
					},
				})
			}
		},
	}
}

func (r *Reconciler) TSOMemberEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc: func(ctx context.Context, event event.TypedCreateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			m := event.Object.(*pdv1.TSOMember)
			ns, cluster := timanager.SplitPrimaryKey(m.Namespace)

			r.Logger.Info("add tso member", "namespace", ns, "cluster", cluster, "name", m.Name, "invalid", m.Invalid)

			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      m.Name,
				},
			})
		},

		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			m := event.ObjectNew.(*pdv1.TSOMember)
			ns, cluster := timanager.SplitPrimaryKey(m.Namespace)

			r.Logger.Info("update tso member", "namespace", ns, "cluster", cluster, "name", m.Name, "invalid", m.Invalid)

			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      m.Name,
				},
			})
		},
		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			m := event.Object.(*pdv1.TSOMember)
			ns, cluster := timanager.SplitPrimaryKey(m.Namespace)

			r.Logger.Info("delete tso member", "namespace", ns, "cluster", cluster, "name", m.Name)

			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      m.Name,
				},
			})
		},
	}
}
