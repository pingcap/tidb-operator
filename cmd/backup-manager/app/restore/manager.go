// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package restore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	bkconstants "github.com/pingcap/tidb-operator/pkg/backup/constants"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	pkgutil "github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

type Manager struct {
	restoreLister  listers.RestoreLister
	StatusUpdater  controller.RestoreConditionUpdaterInterface
	RestoreControl controller.RestoreControlInterface
	Options
}

// NewManager return a RestoreManager
func NewManager(
	restoreLister listers.RestoreLister,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreControl controller.RestoreControlInterface,
	restoreOpts Options) *Manager {
	return &Manager{
		restoreLister,
		statusUpdater,
		restoreControl,
		restoreOpts,
	}
}

func (rm *Manager) setOptions(restore *v1alpha1.Restore) {
	rm.Options.Host = restore.Spec.To.Host

	if restore.Spec.To.Port != 0 {
		rm.Options.Port = restore.Spec.To.Port
	} else {
		rm.Options.Port = v1alpha1.DefaultTiDBServerPort
	}

	if restore.Spec.To.User != "" {
		rm.Options.User = restore.Spec.To.User
	} else {
		rm.Options.User = v1alpha1.DefaultTidbUser
	}

	rm.Options.Password = util.GetOptionValueFromEnv(bkconstants.TidbPasswordKey, bkconstants.BackupManagerEnvVarPrefix)
}

// ProcessRestore used to process the restore logic
func (rm *Manager) ProcessRestore() error {
	ctx, cancel := util.GetContextForTerminationSignals(rm.ResourceName)
	defer cancel()

	var errs []error
	restore, err := rm.restoreLister.Restores(rm.Namespace).Get(rm.ResourceName)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("can't find cluster %s restore %s CRD object, err: %v", rm, rm.ResourceName, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetRestoreCRFailed",
			Message: err.Error(),
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	if restore.Spec.BR == nil {
		return fmt.Errorf("no br config in %s", rm)
	}

	crData, err := json.Marshal(restore)
	if err != nil {
		klog.Errorf("failed to marshal restore %v to json, err: %s", restore, err)
	} else {
		klog.Infof("start to process restore: %s", string(crData))
	}

	if restore.Spec.To == nil {
		return rm.performRestore(ctx, restore.DeepCopy(), nil)
	}

	rm.setOptions(restore)

	klog.Infof("start to connect to tidb server (%s:%d) as the .spec.to field is specified",
		restore.Spec.To.Host, restore.Spec.To.Port)

	var db *sql.DB
	var dsn string
	err = wait.PollImmediate(constants.PollInterval, constants.CheckTimeout, func() (done bool, err error) {
		dsn, err = rm.GetDSN(rm.TLSClient)
		if err != nil {
			klog.Errorf("can't get dsn of tidb cluster %s, err: %s", rm, err)
			return false, err
		}

		db, err = pkgutil.OpenDB(ctx, dsn)
		if err != nil {
			klog.Warningf("can't connect to tidb cluster %s, err: %s", rm, err)
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		errs = append(errs, err)
		klog.Errorf("cluster %s connect failed, err: %s", rm, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ConnectTidbFailed",
			Message: err.Error(),
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	defer db.Close()
	return rm.performRestore(ctx, restore.DeepCopy(), db)
}

func (rm *Manager) performRestore(ctx context.Context, restore *v1alpha1.Restore, db *sql.DB) error {
	started := time.Now()

	err := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreRunning,
		Status: corev1.ConditionTrue,
	}, nil)
	if err != nil {
		return err
	}

	var errs []error

	var (
		oldTikvGCTime, tikvGCLifeTime             string
		oldTikvGCTimeDuration, tikvGCTimeDuration time.Duration
	)

	// set tikv gc life time to prevent gc when restoring data
	if db != nil {
		oldTikvGCTime, err = rm.GetTikvGCLifeTime(ctx, db)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s get %s failed, err: %s", rm, constants.TikvGCVariable, err)
			uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "GetTikvGCLifeTimeFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("cluster %s %s is %s", rm, constants.TikvGCVariable, oldTikvGCTime)

		oldTikvGCTimeDuration, err = time.ParseDuration(oldTikvGCTime)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s parse old %s failed, err: %s", rm, constants.TikvGCVariable, err)
			uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ParseOldTikvGCLifeTimeFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}

		if restore.Spec.TikvGCLifeTime != nil {
			tikvGCLifeTime = *restore.Spec.TikvGCLifeTime
			tikvGCTimeDuration, err = time.ParseDuration(tikvGCLifeTime)
			if err != nil {
				errs = append(errs, err)
				klog.Errorf("cluster %s parse configured %s failed, err: %s", rm, constants.TikvGCVariable, err)
				uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
					Type:    v1alpha1.RestoreFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "ParseConfiguredTikvGCLifeTimeFailed",
					Message: err.Error(),
				}, nil)
				errs = append(errs, uerr)
				return errorutils.NewAggregate(errs)
			}
		} else {
			tikvGCLifeTime = constants.TikvGCLifeTime
			tikvGCTimeDuration, err = time.ParseDuration(tikvGCLifeTime)
			if err != nil {
				errs = append(errs, err)
				klog.Errorf("cluster %s parse default %s failed, err: %s", rm, constants.TikvGCVariable, err)
				uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
					Type:    v1alpha1.RestoreFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "ParseDefaultTikvGCLifeTimeFailed",
					Message: err.Error(),
				}, nil)
				errs = append(errs, uerr)
				return errorutils.NewAggregate(errs)
			}
		}

		if oldTikvGCTimeDuration < tikvGCTimeDuration {
			err = rm.SetTikvGCLifeTime(ctx, db, tikvGCLifeTime)
			if err != nil {
				errs = append(errs, err)
				klog.Errorf("cluster %s set tikv GC life time to %s failed, err: %s", rm, tikvGCLifeTime, err)
				uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
					Type:    v1alpha1.RestoreFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "SetTikvGCLifeTimeFailed",
					Message: err.Error(),
				}, nil)
				errs = append(errs, uerr)
				return errorutils.NewAggregate(errs)
			}
			klog.Infof("set cluster %s %s to %s success", rm, constants.TikvGCVariable, tikvGCLifeTime)
		}
	}

	restoreErr := rm.restoreData(ctx, restore, rm.StatusUpdater, rm.RestoreControl)

	if db != nil && oldTikvGCTimeDuration < tikvGCTimeDuration {
		// use another context to revert `tikv_gc_life_time` back.
		// `DefaultTerminationGracePeriodSeconds` for a pod is 30, so we use a smaller timeout value here.
		ctx2, cancel2 := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel2()
		err = rm.SetTikvGCLifeTime(ctx2, db, oldTikvGCTime)
		if err != nil {
			if restoreErr != nil {
				errs = append(errs, restoreErr)
			}
			errs = append(errs, err)
			klog.Errorf("cluster %s reset tikv GC life time to %s failed, err: %s", rm, oldTikvGCTime, err)
			uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ResetTikvGCLifeTimeFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("reset cluster %s %s to %s success", rm, constants.TikvGCVariable, oldTikvGCTime)
	}

	if restoreErr != nil {
		errs = append(errs, restoreErr)
		klog.Errorf("restore cluster %s from %s failed, err: %s", rm, restore.Spec.Type, restoreErr)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "RestoreDataFromRemoteFailed",
			Message: restoreErr.Error(),
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("restore cluster %s from %s succeed", rm, restore.Spec.Type)

	var (
		commitTS    *string
		restoreType v1alpha1.RestoreConditionType
		allFinished bool
	)
	switch rm.Mode {
	case string(v1alpha1.RestoreModeVolumeSnapshot):
		// In volume snapshot mode, commitTS and size have been updated according to the
		// br command output, so we don't need to update them here.
		if rm.Prepare {
			restoreType = v1alpha1.RestoreVolumeComplete
		} else {
			restoreType = v1alpha1.RestoreDataComplete
		}
	default:
		ts, err := util.GetCommitTsFromBRMetaData(ctx, restore.Spec.StorageProvider)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("get cluster %s commitTs failed, err: %s", rm, err)
			uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "GetCommitTsFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		restoreType = v1alpha1.RestoreComplete
		tsStr := strconv.FormatUint(ts, 10)
		commitTS = &tsStr
		allFinished = true
	}

	updateStatus := &controller.RestoreUpdateStatus{
		CommitTs: commitTS,
	}
	if restore.Status.TimeStarted.Unix() <= 0 {
		updateStatus.TimeStarted = &metav1.Time{Time: started}
	}
	if allFinished {
		updateStatus.TimeCompleted = &metav1.Time{Time: time.Now()}
	}
	return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   restoreType,
		Status: corev1.ConditionTrue,
	}, updateStatus)
}
