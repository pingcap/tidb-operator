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

package tiproxygroup

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/tidbgroup/tasks"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type Reconciler struct {
	Logger logr.Logger
	Client client.Client
}

func Setup(mgr manager.Manager, c client.Client) error {
	r := &Reconciler{
		Logger: mgr.GetLogger().WithName("TiProxyGroup"),
		Client: c,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TiProxyGroup{}).
		Owns(&v1alpha1.TiProxy{}).
		// Only care about the generation change (i.e. spec update)
		Watches(&v1alpha1.Cluster{}, r.ClusterEventHandler(), builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func (r *Reconciler) ClusterEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			cluster := event.ObjectNew.(*v1alpha1.Cluster)

			var list v1alpha1.TiProxyGroupList
			if err := r.Client.List(ctx, &list, client.InNamespace(cluster.Namespace),
				client.MatchingFields{"spec.cluster.name": cluster.Name}); err != nil {
				if !errors.IsNotFound(err) {
					r.Logger.Error(err, "cannot list all tiproxy groups", "ns", cluster.Namespace, "cluster", cluster.Name)
				}
				return
			}

			for i := range list.Items {
				dbg := &list.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      dbg.Name,
						Namespace: dbg.Namespace,
					},
				})
			}
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("tiproxygroup", req.NamespacedName)
	reporter := task.NewTableTaskReporter()

	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
		logger.Info("summay: \n" + reporter.Summary())
	}()

	rtx := &tasks.ReconcileContext{
		State: tasks.NewState(req.NamespacedName),
	}

	runner := r.NewRunner(rtx, reporter)

	return runner.Run(ctx)
}
