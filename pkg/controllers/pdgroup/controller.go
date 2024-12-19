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

package pdgroup

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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/pdgroup/tasks"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type Reconciler struct {
	Logger          logr.Logger
	Client          client.Client
	PDClientManager pdm.PDClientManager
}

func Setup(mgr manager.Manager, c client.Client, pdcm pdm.PDClientManager) error {
	r := &Reconciler{
		Logger:          mgr.GetLogger().WithName("PDGroup"),
		Client:          c,
		PDClientManager: pdcm,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PDGroup{}).
		Owns(&v1alpha1.PD{}).
		// Only care about the generation change (i.e. spec update)
		Watches(&v1alpha1.Cluster{}, r.ClusterEventHandler(), builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func (r *Reconciler) ClusterEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			cluster := event.ObjectNew.(*v1alpha1.Cluster)

			var list v1alpha1.PDGroupList
			if err := r.Client.List(ctx, &list, client.InNamespace(cluster.Namespace),
				client.MatchingFields{"spec.cluster.name": cluster.Name}); err != nil {
				if !errors.IsNotFound(err) {
					r.Logger.Error(err, "cannot list all pd groups", "ns", cluster.Namespace, "cluster", cluster.Name)
				}
				return
			}

			for i := range list.Items {
				pdg := &list.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      pdg.Name,
						Namespace: pdg.Namespace,
					},
				})
			}
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("pdgroup", req.NamespacedName)
	reporter := task.NewTableTaskReporter()

	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
		logger.Info("summary: \n" + reporter.Summary())
	}()

	rtx := &tasks.ReconcileContext{
		// some fields will be set in the context task
		Context: ctx,
		Key:     req.NamespacedName,
	}

	runner := task.NewTaskRunner[tasks.ReconcileContext](reporter)
	runner.AddTasks(
		tasks.NewTaskContext(logger, r.Client, r.PDClientManager),
		tasks.NewTaskFinalizer(logger, r.Client, r.PDClientManager),
		tasks.NewTaskBoot(logger, r.Client),
		tasks.NewTaskService(logger, r.Client),
		tasks.NewTaskUpdater(logger, r.Client),
		tasks.NewTaskStatus(logger, r.Client),
	)

	return runner.Run(rtx)
}