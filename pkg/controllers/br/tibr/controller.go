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

package tibr

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/tibr/tasks"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type Reconciler struct {
	Logger logr.Logger
	Client client.Client
	Scheme *runtime.Scheme
}

func Setup(mgr manager.Manager, c client.Client) error {
	r := &Reconciler{
		Logger: mgr.GetLogger().WithName("TiBR"),
		Client: c,
		Scheme: mgr.GetScheme(),
	}

	h, err := r.makeClusterEventHandler(mgr)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1br.TiBR{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Watches(&v1alpha1.Cluster{}, h).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("tibr", req.NamespacedName)
	reporter := task.NewTableTaskReporter()

	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
		logger.Info("summary: \n" + reporter.Summary())
	}()

	rtx := tasks.NewReconcileContext(ctx, req.NamespacedName, r.Client)

	runner := r.NewRunner(rtx, reporter)

	return runner.Run(ctx)
}
