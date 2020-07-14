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

package backup

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
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

// Manager mainly used to manage backup related work
type Manager struct {
	backupLister  listers.BackupLister
	StatusUpdater controller.BackupConditionUpdaterInterface
	Options
}

// NewManager return a Manager
func NewManager(
	backupLister listers.BackupLister,
	statusUpdater controller.BackupConditionUpdaterInterface,
	backupOpts Options) *Manager {
	return &Manager{
		backupLister,
		statusUpdater,
		backupOpts,
	}
}

func (bm *Manager) setOptions(backup *v1alpha1.Backup) {
	bm.Options.Host = backup.Spec.From.Host

	if backup.Spec.From.Port != 0 {
		bm.Options.Port = backup.Spec.From.Port
	} else {
		bm.Options.Port = bkconstants.DefaultTidbPort
	}

	if backup.Spec.From.User != "" {
		bm.Options.User = backup.Spec.From.User
	} else {
		bm.Options.User = bkconstants.DefaultTidbUser
	}

	bm.Options.Password = util.GetOptionValueFromEnv(bkconstants.TidbPasswordKey, bkconstants.BackupManagerEnvVarPrefix)
}

// ProcessBackup used to process the backup logic
func (bm *Manager) ProcessBackup() error {
	var errs []error
	backup, err := bm.backupLister.Backups(bm.Namespace).Get(bm.ResourceName)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.ResourceName, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupCRFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	if backup.Spec.BR == nil {
		return fmt.Errorf("no br config in %s", bm)
	}

	bm.setOptions(backup)

	var db *sql.DB
	var dsn string
	err = wait.PollImmediate(constants.PollInterval, constants.CheckTimeout, func() (done bool, err error) {
		dsn, err = bm.GetDSN(bm.TLSClient)
		if err != nil {
			klog.Errorf("can't get dsn of tidb cluster %s, err: %s", bm, err)
			return false, err
		}
		db, err = util.OpenDB(dsn)
		if err != nil {
			klog.Warningf("can't connect to tidb cluster %s, err: %s", bm, err)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		errs = append(errs, err)
		klog.Errorf("cluster %s connect failed, err: %s", bm, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ConnectTidbFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	defer db.Close()
	return bm.performBackup(backup.DeepCopy(), db)
}

func (bm *Manager) performBackup(backup *v1alpha1.Backup, db *sql.DB) error {
	started := time.Now()

	var errs []error
	err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRunning,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	backupFullPath, err := util.GetRemotePath(backup)
	if err != nil {
		errs = append(errs, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupRemotePathFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	backup.Status.BackupPath = backupFullPath
	err = bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupPrepare,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		errs = append(errs, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "UpdatePrepareBackupFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	oldTikvGCTime, err := bm.GetTikvGCLifeTime(db)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("cluster %s get %s failed, err: %s", bm, constants.TikvGCVariable, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("cluster %s %s is %s", bm, constants.TikvGCVariable, oldTikvGCTime)

	oldTikvGCTimeDuration, err := time.ParseDuration(oldTikvGCTime)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("cluster %s parse old %s failed, err: %s", bm, constants.TikvGCVariable, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ParseOldTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	var tikvGCTimeDuration time.Duration
	var tikvGCLifeTime string
	if backup.Spec.TikvGCLifeTime != nil {
		tikvGCLifeTime = *backup.Spec.TikvGCLifeTime
		tikvGCTimeDuration, err = time.ParseDuration(tikvGCLifeTime)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s parse configured %s failed, err: %s", bm, constants.TikvGCVariable, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
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
			klog.Errorf("cluster %s parse default %s failed, err: %s", bm, constants.TikvGCVariable, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ParseDefaultTikvGCLifeTimeFailed",
				Message: err.Error(),
			})
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
	}

	if oldTikvGCTimeDuration < tikvGCTimeDuration {
		err = bm.SetTikvGCLifeTime(db, tikvGCLifeTime)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s set tikv GC life time to %s failed, err: %s", bm, tikvGCLifeTime, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "SetTikvGCLifeTimeFailed",
				Message: err.Error(),
			})
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("set cluster %s %s to %s success", bm, constants.TikvGCVariable, tikvGCLifeTime)
	}

	backupErr := bm.backupData(backup)
	if oldTikvGCTimeDuration < tikvGCTimeDuration {
		err = bm.SetTikvGCLifeTime(db, oldTikvGCTime)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s reset tikv GC life time to %s failed, err: %s", bm, oldTikvGCTime, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ResetTikvGCLifeTimeFailed",
				Message: err.Error(),
			})
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("reset cluster %s %s to %s success", bm, constants.TikvGCVariable, oldTikvGCTime)
	}

	if backupErr != nil {
		errs = append(errs, backupErr)
		klog.Errorf("backup cluster %s data failed, err: %s", bm, backupErr)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackupDataToRemoteFailed",
			Message: backupErr.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("backup cluster %s data to %s success", bm, backupFullPath)

	// Note: The size get from remote may be incorrect because the blobs
	// are eventually consistent.
	size, err := getBackupSize(backup)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("Get size for backup files in %s of cluster %s failed, err: %s", backupFullPath, bm, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupSizeFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("Get size %d for backup files in %s of cluster %s success", size, backupFullPath, bm)

	commitTs, err := util.GetCommitTsFromBRMetaData(backup.Spec.StorageProvider)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("get cluster %s commitTs failed, err: %s", bm, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetCommitTsFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("get cluster %s commitTs %d success", bm, commitTs)

	finish := time.Now()

	backup.Status.TimeStarted = metav1.Time{Time: started}
	backup.Status.TimeCompleted = metav1.Time{Time: finish}
	backup.Status.BackupSize = size
	backup.Status.BackupSizeReadable = humanize.Bytes(uint64(size))
	backup.Status.CommitTs = strconv.FormatUint(commitTs, 10)

	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: corev1.ConditionTrue,
	})
}
