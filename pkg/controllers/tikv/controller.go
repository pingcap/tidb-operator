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

package tikv

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/tikv/tasks"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/tracker"
	"github.com/pingcap/tidb-operator/pkg/volumes"
)

type Reconciler struct {
	Logger                logr.Logger
	Client                client.Client
	PDClientManager       pdm.PDClientManager
	VolumeModifierFactory volumes.ModifierFactory
	Tracker               tracker.Tracker
}

func Setup(mgr manager.Manager, c client.Client, pdcm pdm.PDClientManager, vm volumes.ModifierFactory, t tracker.Tracker) error {
	r := &Reconciler{
		Logger:                mgr.GetLogger().WithName("TiKV"),
		Client:                c,
		PDClientManager:       pdcm,
		VolumeModifierFactory: vm,
		Tracker:               t,
	}

	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.TiKV{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Watches(&v1alpha1.Cluster{}, r.ClusterEventHandler()).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		WatchesRawSource(pdcm.Source(&pdv1.Store{}, r.StoreEventHandler())).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logr.FromContextOrDiscard(ctx)
	reporter := task.NewTableTaskReporter(uuid.NewString())

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
