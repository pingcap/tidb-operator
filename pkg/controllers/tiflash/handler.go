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

package tiflash

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
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
			} else if newObj.Status.ID != oldObj.Status.ID {
				r.Logger.Info("cluster id is updating", "from", oldObj.Status.ID, "to", newObj.Status.ID)
			} else if !reflect.DeepEqual(oldObj.Spec.SuspendAction, newObj.Spec.SuspendAction) {
				r.Logger.Info("suspend action is updating", "from", oldObj.Spec.SuspendAction, "to", newObj.Spec.SuspendAction)
			} else if oldObj.Spec.Paused != newObj.Spec.Paused {
				r.Logger.Info("cluster paused is updating", "from", oldObj.Spec.Paused, "to", newObj.Spec.Paused)
			} else {
				return
			}

			var flashl v1alpha1.TiFlashList
			if err := r.Client.List(ctx, &flashl, client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   newObj.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
			}, client.InNamespace(newObj.Namespace)); err != nil {
				r.Logger.Error(err, "cannot list all tiflash instances", "ns", newObj.Namespace, "cluster", newObj.Name)
				return
			}

			for i := range flashl.Items {
				tiflash := &flashl.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      tiflash.Name,
						Namespace: tiflash.Namespace,
					},
				})
			}
		},
		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			obj := event.Object.(*v1alpha1.Cluster)

			r.Logger.Info("enqueue all tiflash instances because of the cluster is deleting", "ns", obj.Namespace, "cluster", obj.Name)

			var fl v1alpha1.TiFlashList
			if err := r.Client.List(ctx, &fl, client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   obj.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
			}, client.InNamespace(obj.Namespace)); err != nil {
				r.Logger.Error(err, "cannot list all tiflash instances", "ns", obj.Namespace, "cluster", obj.Name)
				return
			}

			for i := range fl.Items {
				f := &fl.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      f.Name,
						Namespace: f.Namespace,
					},
				})
			}
		},
	}
}

func (r *Reconciler) StoreEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc: func(ctx context.Context, event event.TypedCreateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			s := event.Object.(*pdv1.Store)
			if s.Engine() != pdv1.StoreEngineTiFlash && s.Engine() != pdv1.StoreEngineTiFlashCompute {
				return
			}
			r.Logger.Info("add tiflash store", "ns", event.Object.GetNamespace(), "name", event.Object.GetName())

			req, err := r.getRequestOfTiFlashStore(ctx, s)
			if err != nil {
				return
			}
			queue.Add(req)
		},

		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			s := event.ObjectNew.(*pdv1.Store)
			if s.Engine() != pdv1.StoreEngineTiFlash && s.Engine() != pdv1.StoreEngineTiFlashCompute {
				return
			}
			r.Logger.Info("update tiflash store",
				"ns", event.ObjectNew.GetNamespace(),
				"name", event.ObjectNew.GetName(),
				"diff", cmp.Diff(event.ObjectOld, event.ObjectNew),
			)

			req, err := r.getRequestOfTiFlashStore(ctx, s)
			if err != nil {
				return
			}
			queue.Add(req)
		},

		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			s := event.Object.(*pdv1.Store)
			if s.Engine() != pdv1.StoreEngineTiFlash && s.Engine() != pdv1.StoreEngineTiFlashCompute {
				return
			}
			r.Logger.Info("delete tiflash store", "ns", event.Object.GetNamespace(), "name", event.Object.GetName())

			req, err := r.getRequestOfTiFlashStore(ctx, s)
			if err != nil {
				return
			}
			queue.Add(req)
		},
	}
}

func (r *Reconciler) getRequestOfTiFlashStore(ctx context.Context, s *pdv1.Store) (reconcile.Request, error) {
	ns, cluster := timanager.SplitPrimaryKey(s.Namespace)
	var tiflashList v1alpha1.TiFlashList
	if err := r.Client.List(ctx, &tiflashList, client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   cluster,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
	}, client.InNamespace(ns)); err != nil {
		r.Logger.Error(err, "cannot list all tiflash instances", "ns", ns, "cluster", cluster)
		return reconcile.Request{}, err
	}

	var c v1alpha1.Cluster
	if err := r.Client.Get(ctx, client.ObjectKey{Name: cluster, Namespace: ns}, &c); err != nil {
		r.Logger.Error(err, "cannot get cluster", "ns", ns, "cluster", cluster)
		return reconcile.Request{}, err
	}

	for i := range tiflashList.Items {
		tiflash := &tiflashList.Items[i]
		if s.Name == coreutil.InstanceAdvertiseAddress[scope.TiFlash](&c, tiflash, coreutil.TiFlashFlashPort(tiflash)) {
			return reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tiflash.Name,
					Namespace: tiflash.Namespace,
				},
			}, nil
		}
	}

	err := fmt.Errorf("store: %v/%v, addr: %v", s.Namespace, s.Name, s.Address)
	r.Logger.Error(err, "failed to find tiflash of store")
	return reconcile.Request{}, err
}
