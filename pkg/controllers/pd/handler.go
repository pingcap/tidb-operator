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

package pd

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
)

func (r *Reconciler) EventLogger() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
			r.Logger.Info("create event", "type", reflect.TypeOf(e.Object), "name", e.Object.GetName())
			return false
		},
		UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
			r.Logger.Info("update event", "type", reflect.TypeOf(e.ObjectNew), "name", e.ObjectNew.GetName())
			return false
		},
		DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
			r.Logger.Info("delete event", "type", reflect.TypeOf(e.Object), "name", e.Object.GetName())
			return false
		},
		GenericFunc: func(e event.TypedGenericEvent[client.Object]) bool {
			r.Logger.Info("generic event", "type", reflect.TypeOf(e.Object), "name", e.Object.GetName())
			return false
		},
	}
}

func (r *Reconciler) ClusterEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			oldObj := event.ObjectOld.(*v1alpha1.Cluster)
			newObj := event.ObjectNew.(*v1alpha1.Cluster)

			if !reflect.DeepEqual(oldObj.Spec.SuspendAction, newObj.Spec.SuspendAction) {
				r.Logger.Info("suspend action is updating", "from", oldObj.Spec.SuspendAction, "to", newObj.Spec.SuspendAction)
			} else if oldObj.Spec.Paused != newObj.Spec.Paused {
				r.Logger.Info("cluster paused is updating", "from", oldObj.Spec.Paused, "to", newObj.Spec.Paused)
			} else {
				return
			}

			var pdl v1alpha1.PDList
			if err := r.Client.List(ctx, &pdl, client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   newObj.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
			}, client.InNamespace(newObj.Namespace)); err != nil {
				r.Logger.Error(err, "cannot list all pd instances", "ns", newObj.Namespace, "cluster", newObj.Name)
				return
			}

			for i := range pdl.Items {
				pd := &pdl.Items[i]
				r.Logger.Info("queue add", "reason", "cluster change", "namespace", pd.Namespace, "cluster", pd.Spec.Cluster, "name", pd.Name)
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      pd.Name,
						Namespace: pd.Namespace,
					},
				})
			}
		},
	}
}

func (r *Reconciler) MemberEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc: func(_ context.Context, event event.TypedCreateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			m := event.Object.(*pdv1.Member)
			ns, cluster := pdm.SplitPrimaryKey(m.Namespace)

			r.Logger.Info("add member", "namespace", ns, "cluster", cluster, "name", m.Name, "health", m.Health, "invalid", m.Invalid)

			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      m.Name,
					Namespace: ns,
				},
			})
		},

		UpdateFunc: func(_ context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			m := event.ObjectNew.(*pdv1.Member)
			ns, cluster := pdm.SplitPrimaryKey(m.Namespace)

			r.Logger.Info("update member", "namespace", ns, "cluster", cluster, "name", m.Name, "health", m.Health, "invalid", m.Invalid)

			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      m.Name,
					Namespace: ns,
				},
			})
		},
		DeleteFunc: func(_ context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			m := event.Object.(*pdv1.Member)
			ns, cluster := pdm.SplitPrimaryKey(m.Namespace)

			r.Logger.Info("delete member", "namespace", ns, "cluster", cluster, "name", m.Name)

			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      m.Name,
					Namespace: ns,
				},
			})
		},
	}
}
