package resourcemanager
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





import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanager/tasks"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
	"github.com/pingcap/tidb-operator/v2/pkg/volumes"
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
		Logger:                mgr.GetLogger().WithName("ResourceManager"),
		Client:                c,
		PDClientManager:       pdcm,
		VolumeModifierFactory: vm,
		Tracker:               t,
	}
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.ResourceManager{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Watches(&v1alpha1.Cluster{}, r.ClusterEventHandler()).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
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
		summary := fmt.Sprintf("summary for %v\n%s", req.NamespacedName, reporter.Summary())
		logger.Info(summary)
	}()

	rtx := &tasks.ReconcileContext{State: tasks.NewState(req.NamespacedName)}
	runner := r.NewRunner(rtx, reporter)
	return runner.Run(ctx)
}