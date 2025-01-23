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

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tiflashconfig "github.com/pingcap/tidb-operator/pkg/configs/tiflash"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
)

func (r *Reconciler) ClusterEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
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
	}
}

func (r *Reconciler) StoreEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc: func(ctx context.Context, event event.TypedCreateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			s := event.Object.(*pdv1.Store)
			req, err := r.getRequestOfTiFlashStore(ctx, s)
			if err != nil {
				return
			}
			queue.Add(req)
		},

		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			s := event.ObjectNew.(*pdv1.Store)
			req, err := r.getRequestOfTiFlashStore(ctx, s)
			if err != nil {
				return
			}
			queue.Add(req)
		},

		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			s := event.Object.(*pdv1.Store)
			req, err := r.getRequestOfTiFlashStore(ctx, s)
			if err != nil {
				return
			}
			queue.Add(req)
		},
	}
}

func (r *Reconciler) getRequestOfTiFlashStore(ctx context.Context, s *pdv1.Store) (reconcile.Request, error) {
	if s.Engine() != pdv1.StoreEngineTiFlash {
		return reconcile.Request{}, fmt.Errorf("store is not tiflash")
	}

	ns, cluster := pdm.SplitPrimaryKey(s.Namespace)
	var tiflashList v1alpha1.TiFlashList
	if err := r.Client.List(ctx, &tiflashList, client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   cluster,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
	}, client.InNamespace(ns)); err != nil {
		r.Logger.Error(err, "cannot list all tiflash instances", "ns", ns, "cluster", cluster)
		return reconcile.Request{}, err
	}

	for i := range tiflashList.Items {
		tiflash := &tiflashList.Items[i]
		if s.Name == tiflashconfig.GetServiceAddr(tiflash) {
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
