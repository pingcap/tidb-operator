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

package backup

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/backup/tasks"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type Config tasks.Config
type Reconciler struct {
	Logger        logr.Logger
	Client        client.Client
	PDCM          pdm.PDClientManager
	EventRecorder record.EventRecorder

	Config Config
}

func Setup(mgr manager.Manager, c client.Client, pdcm pdm.PDClientManager, conf Config) error {
	r := &Reconciler{
		Logger:        mgr.GetLogger().WithName("Backup"),
		Client:        c,
		PDCM:          pdcm,
		EventRecorder: mgr.GetEventRecorderFor("backup"),
		Config:        conf,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Backup{}).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("backup", req.NamespacedName)
	reporter := task.NewTableTaskReporter()

	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
		logger.Info("summary: \n" + reporter.Summary())
	}()

	rtx := &tasks.ReconcileContext{
		PDClientManager: r.PDCM,
		State:           tasks.NewState(req.NamespacedName),
		Config:          tasks.Config(r.Config),
	}

	runner := r.NewRunner(rtx, reporter)

	return runner.Run(ctx)
}
