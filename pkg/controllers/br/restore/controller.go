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

package restore

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/restore/tasks"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
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
		Logger:        mgr.GetLogger().WithName("Restore"),
		Client:        c,
		PDCM:          pdcm,
		EventRecorder: mgr.GetEventRecorderFor("restore"),
		Config:        conf,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Restore{}).
		// for PD instance controller, we only need to care about the cluster spec change event now
		Watches(&batchv1.Job{}, r.JobEventHandler()).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("restore", req.NamespacedName)
	reporter := task.NewTableTaskReporter()

	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
		logger.Info("summary: \n" + reporter.Summary())
	}()

	restore := &v1alpha1.Restore{}
	if err := r.Client.Get(ctx, req.NamespacedName, restore); err != nil {
		return ctrl.Result{}, err
	}
	result, err := common.JobLifecycleManager.Sync(ctx, runtime.FromRestore(restore), r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result != nil {
		return *result, nil
	}

	rtx := &tasks.ReconcileContext{
		PDClientManager: r.PDCM,
		State:           tasks.NewState(restore),
		Config:          tasks.Config(r.Config),
	}

	runner := r.NewRunner(rtx, reporter)

	return runner.Run(ctx)
}

func (r *Reconciler) JobEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			newObj := event.ObjectNew.(*batchv1.Job)
			restore, err := r.resolveRestoreFromJob(ctx, newObj.Namespace, newObj)
			if err != nil {
				r.Logger.Error(err, "cannot resolve restore from job", "namespace", newObj.Namespace, "job", newObj.Name)
				return
			}
			if restore == nil {
				r.Logger.Info("restore not found by Job, skip reconcile", "namespace", newObj.Namespace, "job", newObj.Name)
				return
			}

			r.Logger.Info("queue add", "reason", "k8s job change", "namespace", restore.Namespace, "restore", restore.Name)
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      restore.Name,
					Namespace: restore.Namespace,
				},
			})
		},
	}
}

func (r *Reconciler) resolveRestoreFromJob(ctx context.Context, namespace string, job *batchv1.Job) (*v1alpha1.Restore, error) {
	owner := metav1.GetControllerOf(job)
	if owner == nil {
		return nil, nil
	}

	if owner.Kind != v1alpha1.RestoreControllerKind.Kind {
		return nil, nil
	}

	restore := &v1alpha1.Restore{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      owner.Name,
		Namespace: namespace,
	}, restore); err != nil {
		return nil, err
	}
	if owner.UID != restore.UID {
		return nil, fmt.Errorf("restore %s/%s is not the owner of job %s/%s", namespace, owner.Name, namespace, job.Name)
	}
	return restore, nil
}

// TODO(ideascf):
// metrics?
// IgnoredError?
// RateLimiter?
// webhook to check invalid, completed, failed?
