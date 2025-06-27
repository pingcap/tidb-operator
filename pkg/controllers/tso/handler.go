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

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
)

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
