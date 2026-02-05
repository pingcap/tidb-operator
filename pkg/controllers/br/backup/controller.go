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
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/backup/tasks"
	backupMgr "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type (
	Config     tasks.Config
	Reconciler struct {
		Logger        logr.Logger
		Client        client.Client
		PDCM          pdm.PDClientManager
		EventRecorder record.EventRecorder

		Config        Config
		BackupManager backupMgr.BackupManager
		BackupCleaner backupMgr.BackupCleaner
		StatusUpdater backupMgr.BackupConditionUpdaterInterface
	}
)

func Setup(mgr manager.Manager, c client.Client, pdcm pdm.PDClientManager, conf Config) error {
	eventRecorder := mgr.GetEventRecorderFor("backup")
	statusUpdater := backupMgr.NewRealBackupConditionUpdater(c, eventRecorder)

	r := &Reconciler{
		Logger:        mgr.GetLogger().WithName("Backup"),
		Client:        c,
		PDCM:          pdcm,
		EventRecorder: eventRecorder,

		Config:        conf,
		BackupManager: backupMgr.NewBackupManager(c, pdcm, statusUpdater, conf.BackupManagerImage),
		BackupCleaner: backupMgr.NewBackupCleaner(c, statusUpdater, conf.BackupManagerImage),
		StatusUpdater: statusUpdater,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Backup{}).
		Watches(&batchv1.Job{}, r.JobEventHandler()).
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

	backup := &v1alpha1.Backup{}
	if err := r.Client.Get(ctx, req.NamespacedName, backup); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	err := common.JobLifecycleManager.Sync(ctx, runtime.FromBackup(backup), r.Client)
	if err != nil {
		requeueErr := &common.RequeueError{}
		if errors.As(err, &requeueErr) {
			logger.Info("requeue backup", "namespace", backup.Namespace, "backup", backup.Name, "duration", requeueErr.Duration)
			return ctrl.Result{RequeueAfter: requeueErr.Duration}, nil
		}
		if common.IsIgnoreError(err) {
			logger.Info("ignore backup", "namespace", backup.Namespace, "backup", backup.Name, "reason", err)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if v1alpha1.IsBackupInvalid(backup) {
		logger.Info("backup is invalid, skip reconcile", "namespace", backup.Namespace, "backup", backup.Name)
		return ctrl.Result{}, nil
	}

	rtx := &tasks.ReconcileContext{
		State:         tasks.NewState(backup),
		Config:        tasks.Config(r.Config),
		BackupManager: r.BackupManager,
		BackupCleaner: r.BackupCleaner,
	}
	runner := r.NewRunner(rtx, reporter)
	return runner.Run(ctx)
}

func (r *Reconciler) JobEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			newObj := event.ObjectNew.(*batchv1.Job)
			if err := r.resolveBackupFromJob(ctx, newObj.Namespace, newObj, queue); err != nil {
				r.Logger.Error(err, "failed to resolve backup from job", "namespace", newObj.Namespace, "job", newObj.Name)
			}
		},
		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			job := event.Object.(*batchv1.Job)
			if err := r.resolveBackupFromJob(ctx, job.Namespace, job, queue); err != nil {
				r.Logger.Error(err, "failed to resolve backup from job", "namespace", job.Namespace, "job", job.Name)
			}
		},
	}
}

func (r *Reconciler) resolveBackupFromJob(
	ctx context.Context,
	namespace string,
	job *batchv1.Job,
	queue workqueue.TypedRateLimitingInterface[reconcile.Request],
) error {
	logger := log.FromContext(ctx)
	logger.Info("job event handler", "namespace", job.Namespace, "job", job.Name)

	owner := metav1.GetControllerOf(job)
	if owner == nil {
		return nil
	}

	if owner.Kind != v1alpha1.BackupControllerKind.Kind {
		return nil
	}

	backup := &v1alpha1.Backup{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      owner.Name,
		Namespace: namespace,
	}, backup); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("backup not found by Job, skip reconcile", "namespace", job.Namespace, "job", job.Name)
			return nil
		}
		return err
	}
	if owner.UID != backup.UID {
		return fmt.Errorf("backup %s/%s is not the owner of job %s/%s", namespace, owner.Name, namespace, job.Name)
	}

	logger.Info("queue add", "reason", "k8s job change", "namespace", backup.Namespace, "backup", backup.Name)
	queue.Add(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
	})
	return nil
}
