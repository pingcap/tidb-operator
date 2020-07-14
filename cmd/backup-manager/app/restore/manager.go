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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

type Manager struct {
	restoreLister listers.RestoreLister
	StatusUpdater controller.RestoreConditionUpdaterInterface
	Options
}

// NewManager return a RestoreManager
func NewManager(
	restoreLister listers.RestoreLister,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreOpts Options) *Manager {
	return &Manager{
		restoreLister,
		statusUpdater,
		restoreOpts,
	}
}

func (rm *Manager) setOptions(restore *v1alpha1.Restore) {
	rm.Options.Host = restore.Spec.To.Host

	if restore.Spec.To.Port != 0 {
		rm.Options.Port = restore.Spec.To.Port
	} else {
		rm.Options.Port = bkconstants.DefaultTidbPort
	}

	if restore.Spec.To.User != "" {
		rm.Options.User = restore.Spec.To.User
	} else {
		rm.Options.User = bkconstants.DefaultTidbUser
	}

	rm.Options.Password = util.GetOptionValueFromEnv(bkconstants.TidbPasswordKey, bkconstants.BackupManagerEnvVarPrefix)
}

// ProcessRestore used to process the restore logic
func (rm *Manager) ProcessRestore() error {
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
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	if restore.Spec.BR == nil {
		return fmt.Errorf("no br config in %s", rm)
	}

	rm.setOptions(restore)

	var db *sql.DB
	var dsn string
	err = wait.PollImmediate(constants.PollInterval, constants.CheckTimeout, func() (done bool, err error) {
		dsn, err = rm.GetDSN(rm.TLSClient)
		if err != nil {
			klog.Errorf("can't get dsn of tidb cluster %s, err: %s", rm, err)
			return false, err
		}

		db, err = util.OpenDB(dsn)
		if err != nil {
			klog.Warningf("can't connect to tidb cluster %s, err: %s", rm, err)
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
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	defer db.Close()
	return rm.performRestore(restore.DeepCopy(), db)
}

func (rm *Manager) performRestore(restore *v1alpha1.Restore, db *sql.DB) error {
	started := time.Now()

	err := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreRunning,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	var errs []error
	oldTikvGCTime, err := rm.GetTikvGCLifeTime(db)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("cluster %s get %s failed, err: %s", rm, constants.TikvGCVariable, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("cluster %s %s is %s", rm, constants.TikvGCVariable, oldTikvGCTime)

	oldTikvGCTimeDuration, err := time.ParseDuration(oldTikvGCTime)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("cluster %s parse old %s failed, err: %s", rm, constants.TikvGCVariable, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ParseOldTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	var tikvGCTimeDuration time.Duration
	var tikvGCLifeTime string
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
			})
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
			})
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
	}

	if oldTikvGCTimeDuration < tikvGCTimeDuration {
		err = rm.SetTikvGCLifeTime(db, tikvGCLifeTime)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s set tikv GC life time to %s failed, err: %s", rm, tikvGCLifeTime, err)
			uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "SetTikvGCLifeTimeFailed",
				Message: err.Error(),
			})
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("set cluster %s %s to %s success", rm, constants.TikvGCVariable, tikvGCLifeTime)
	}

	restoreErr := rm.restoreData(restore)
	if oldTikvGCTimeDuration < tikvGCTimeDuration {
		err = rm.SetTikvGCLifeTime(db, oldTikvGCTime)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s reset tikv GC life time to %s failed, err: %s", rm, oldTikvGCTime, err)
			uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ResetTikvGCLifeTimeFailed",
				Message: err.Error(),
			})
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
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("restore cluster %s from %s succeed", rm, restore.Spec.Type)

	commitTs, err := util.GetCommitTsFromBRMetaData(restore.Spec.StorageProvider)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("get cluster %s commitTs failed, err: %s", rm, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetCommitTsFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	finish := time.Now()
	restore.Status.TimeStarted = metav1.Time{Time: started}
	restore.Status.TimeCompleted = metav1.Time{Time: finish}
	restore.Status.CommitTs = strconv.FormatUint(commitTs, 10)

	return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
}
