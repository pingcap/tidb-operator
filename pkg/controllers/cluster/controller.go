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

package cluster

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/cluster/tasks"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
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
		Logger:          mgr.GetLogger().WithName("Cluster"),
		Client:          c,
		PDClientManager: pdcm,
	}
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Cluster{}).
		Watches(&v1alpha1.PDGroup{}, handler.EnqueueRequestsFromMapFunc(enqueueForGroupFunc[scope.PDGroup]())).
		Watches(&v1alpha1.TiKVGroup{}, handler.EnqueueRequestsFromMapFunc(enqueueForGroupFunc[scope.TiKVGroup]())).
		Watches(&v1alpha1.TiDBGroup{}, handler.EnqueueRequestsFromMapFunc(enqueueForGroupFunc[scope.TiDBGroup]())).
		Watches(&v1alpha1.TiFlashGroup{}, handler.EnqueueRequestsFromMapFunc(enqueueForGroupFunc[scope.TiFlashGroup]())).
		Watches(&v1alpha1.TiCDCGroup{}, handler.EnqueueRequestsFromMapFunc(enqueueForGroupFunc[scope.TiCDCGroup]())).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func enqueueForGroupFunc[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
]() handler.MapFunc {
	return handler.MapFunc(
		func(_ context.Context, obj client.Object) []reconcile.Request {
			t := obj.(F)
			return []reconcile.Request{
				{
					NamespacedName: client.ObjectKey{
						Namespace: obj.GetNamespace(),
						Name:      coreutil.Cluster[S](t),
					},
				},
			}
		},
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("cluster", req.NamespacedName)
	reporter := task.NewTableTaskReporter()

	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
		logger.Info("summay: \n" + reporter.Summary())
	}()

	rtx := &tasks.ReconcileContext{
		// some fields will be set in the context task
		Context: ctx,
		Key:     req.NamespacedName,
	}

	runner := task.NewTaskRunner[tasks.ReconcileContext](reporter)
	runner.AddTasks(
		tasks.NewTaskContext(logger, r.Client),
		tasks.NewTaskFinalizer(logger, r.Client),
		tasks.NewTaskStatus(logger, r.Client, r.PDClientManager),
	)

	return runner.Run(rtx)
}
