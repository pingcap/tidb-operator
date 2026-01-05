// Copyright 2026 PingCAP, Inc.
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

package resourcemanagergroup

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
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
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanagergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

type Reconciler struct {
	Logger          logr.Logger
	Client          client.Client
	AllocateFactory tracker.AllocateFactory
}

func Setup(mgr manager.Manager, c client.Client, af tracker.AllocateFactory) error {
	r := &Reconciler{
		Logger:          mgr.GetLogger().WithName("ResourceManagerGroup"),
		Client:          c,
		AllocateFactory: af,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ResourceManagerGroup{}).
		Owns(&v1alpha1.ResourceManager{}).
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

			var list v1alpha1.ResourceManagerGroupList
			if err := r.Client.List(ctx, &list,
				client.InNamespace(cluster.Namespace),
				client.MatchingFields{"spec.cluster.name": cluster.Name},
			); err != nil {
				if !errors.IsNotFound(err) {
					r.Logger.Error(err, "cannot list all resource manager groups", "ns", cluster.Namespace, "cluster", cluster.Name)
				}
				return
			}

			for i := range list.Items {
				rmg := &list.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      rmg.Name,
						Namespace: rmg.Namespace,
					},
				})
			}
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logr.FromContextOrDiscard(ctx)
	reporter := task.NewTableTaskReporter(uuid.NewString())

	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
		summary := fmt.Sprintf("summary for %v\n%s", req.NamespacedName, reporter.Summary())
		logger.Info(summary)
	}()

	rtx := &tasks.ReconcileContext{State: tasks.NewState(req.NamespacedName)}
	runner := r.NewRunner(rtx, reporter)
	return runner.Run(ctx)
}
