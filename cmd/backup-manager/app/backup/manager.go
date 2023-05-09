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
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/dustin/go-humanize"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/clean"
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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

const (
	gcPausedKeyword          = "GC is paused"
	pdSchedulesPausedKeyword = "Schedulers are paused"
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

func (bm *Manager) setFromDBOptions(backup *v1alpha1.Backup) {
	bm.Options.Host = backup.Spec.From.Host

	if backup.Spec.From.Port != 0 {
		bm.Options.Port = backup.Spec.From.Port
	} else {
		bm.Options.Port = v1alpha1.DefaultTiDBServerPort
	}

	if backup.Spec.From.User != "" {
		bm.Options.User = backup.Spec.From.User
	} else {
		bm.Options.User = v1alpha1.DefaultTidbUser
	}

	bm.Options.Password = util.GetOptionValueFromEnv(bkconstants.TidbPasswordKey, bkconstants.BackupManagerEnvVarPrefix)
}

// ProcessBackup used to process the backup logic
func (bm *Manager) ProcessBackup() error {
	ctx, cancel := util.GetContextForTerminationSignals(bm.ResourceName)
	defer cancel()

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
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	// we treat snapshot backup as restarted if its status is not scheduled when backup pod just start to run
	// we will clean backup data before run br command
	if backup.Spec.Mode == v1alpha1.BackupModeSnapshot && (backup.Status.Phase != v1alpha1.BackupScheduled || v1alpha1.IsBackupRestart(backup)) {
		klog.Infof("snapshot backup %s was restarted, status is %s", bm, backup.Status.Phase)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:   v1alpha1.BackupRestart,
			Status: corev1.ConditionTrue,
		}, nil)
		if uerr != nil {
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
	}

	if backup.Spec.BR == nil {
		return fmt.Errorf("no br config in %s", bm)
	}

	if bm.Mode == string(v1alpha1.BackupModeLog) {
		return bm.performLogBackup(ctx, backup.DeepCopy())
	}

	if bm.Mode == string(v1alpha1.BackupModeVolumeSnapshot) && bm.Initialize {
		return bm.performVolumeBackupInitialize(ctx, backup.DeepCopy())
	}

	if backup.Spec.From == nil {
		// skip the DB initialization if spec.from is not specified
		return bm.performBackup(ctx, backup.DeepCopy(), nil)
	}

	// validate and create from db
	var db *sql.DB
	db, err = bm.validateAndCreateFromDB(ctx, backup.DeepCopy())
	if err != nil {
		return err
	}

	defer db.Close()
	return bm.performBackup(ctx, backup.DeepCopy(), db)
}

// validateAndCreateFromDB validate and create from db.
func (bm *Manager) validateAndCreateFromDB(ctx context.Context, backup *v1alpha1.Backup) (*sql.DB, error) {
	bm.setFromDBOptions(backup)
	var db *sql.DB
	var dsn string
	var errs []error
	err := wait.PollImmediate(constants.PollInterval, constants.CheckTimeout, func() (done bool, err error) {
		dsn, err = bm.GetDSN(bm.TLSClient)
		if err != nil {
			klog.Errorf("can't get dsn of tidb cluster %s, err: %s", bm, err)
			return false, err
		}
		db, err = pkgutil.OpenDB(ctx, dsn)
		if err != nil {
			klog.Warningf("can't connect to tidb cluster %s, err: %s", bm, err)
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
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
		}, nil)
		errs = append(errs, uerr)
		return nil, errorutils.NewAggregate(errs)
	}

	return db, nil
}

func (bm *Manager) performBackup(ctx context.Context, backup *v1alpha1.Backup, db *sql.DB) error {
	started := time.Now()

	var errs []error

	backupFullPath, err := util.GetStoragePath(backup)
	if err != nil {
		errs = append(errs, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupRemotePathFailed",
			Message: err.Error(),
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	updatePathStatus := &controller.BackupUpdateStatus{
		BackupPath: &backupFullPath,
	}
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupPrepare,
		Status: corev1.ConditionTrue,
	}, updatePathStatus); err != nil {
		return err
	}

	var (
		oldTikvGCTime, tikvGCLifeTime             string
		oldTikvGCTimeDuration, tikvGCTimeDuration time.Duration
	)

	// set tikv gc life time to prevent gc when backing up data
	if db != nil {
		oldTikvGCTime, err = bm.GetTikvGCLifeTime(ctx, db)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s get %s failed, err: %s", bm, constants.TikvGCVariable, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "GetTikvGCLifeTimeFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("cluster %s %s is %s", bm, constants.TikvGCVariable, oldTikvGCTime)

		oldTikvGCTimeDuration, err = time.ParseDuration(oldTikvGCTime)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("cluster %s parse old %s failed, err: %s", bm, constants.TikvGCVariable, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ParseOldTikvGCLifeTimeFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}

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
				}, nil)
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
				}, nil)
				errs = append(errs, uerr)
				return errorutils.NewAggregate(errs)
			}
		}

		if oldTikvGCTimeDuration < tikvGCTimeDuration {
			err = bm.SetTikvGCLifeTime(ctx, db, tikvGCLifeTime)
			if err != nil {
				errs = append(errs, err)
				klog.Errorf("cluster %s set tikv GC life time to %s failed, err: %s", bm, tikvGCLifeTime, err)
				uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
					Type:    v1alpha1.BackupFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "SetTikvGCLifeTimeFailed",
					Message: err.Error(),
				}, nil)
				errs = append(errs, uerr)
				return errorutils.NewAggregate(errs)
			}
			klog.Infof("set cluster %s %s to %s success", bm, constants.TikvGCVariable, tikvGCLifeTime)
		}
	}

	// clean snapshot backup data if it was restarted
	if backup.Spec.Mode == v1alpha1.BackupModeSnapshot && v1alpha1.IsBackupRestart(backup) && !bm.isBRCanContinueRunByCheckpoint() {
		klog.Infof("clean snapshot backup %s data before run br command, backup path is %s", bm, backup.Status.BackupPath)
		err := bm.cleanSnapshotBackupEnv(ctx, backup)
		if err != nil {
			return errors.Annotatef(err, "clean snapshot backup %s failed", bm)
		}
	}

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRunning,
		Status: corev1.ConditionTrue,
	}, nil); err != nil {
		return err
	}

	// run br binary to do the real job
	backupErr := bm.backupData(ctx, backup, bm.StatusUpdater)

	if db != nil && oldTikvGCTimeDuration < tikvGCTimeDuration {
		// use another context to revert `tikv_gc_life_time` back.
		// `DefaultTerminationGracePeriodSeconds` for a pod is 30, so we use a smaller timeout value here.
		ctx2, cancel2 := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel2()
		err = bm.SetTikvGCLifeTime(ctx2, db, oldTikvGCTime)
		if err != nil {
			if backupErr != nil {
				errs = append(errs, backupErr)
			}
			errs = append(errs, err)
			klog.Errorf("cluster %s reset tikv GC life time to %s failed, err: %s", bm, oldTikvGCTime, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ResetTikvGCLifeTimeFailed",
				Message: err.Error(),
			}, nil)
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
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("backup cluster %s data to %s success", bm, backupFullPath)

	var updateStatus *controller.BackupUpdateStatus
	completeCondition := v1alpha1.BackupComplete
	switch bm.Mode {
	case string(v1alpha1.BackupModeVolumeSnapshot):
		if !bm.Initialize {
			completeCondition = v1alpha1.VolumeBackupComplete
			// In volume snapshot mode, commitTS have been updated according to the
			// br command output, so we don't need to update it here.
			backupSize, err := util.CalcVolSnapBackupSize(ctx, backup.Spec.StorageProvider)

			if err != nil {
				klog.Warningf("Failed to calc volume snapshot backup size %d bytes, %v", backupSize, err)
			}

			backupSizeReadable := humanize.Bytes(uint64(backupSize))

			updateStatus = &controller.BackupUpdateStatus{
				TimeStarted:        &metav1.Time{Time: started},
				TimeCompleted:      &metav1.Time{Time: time.Now()},
				BackupSize:         &backupSize,
				BackupSizeReadable: &backupSizeReadable,
			}
		}
	default:
		backupMeta, err := util.GetBRMetaData(ctx, backup.Spec.StorageProvider)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("Get backup metadata for backup files in %s of cluster %s failed, err: %s", backupFullPath, bm, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "GetBackupMetadataFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("Get br metadata for backup files in %s of cluster %s success", backupFullPath, bm)
		backupSize := int64(util.GetBRArchiveSize(backupMeta))
		backupSizeReadable := humanize.Bytes(uint64(backupSize))
		commitTS := backupMeta.EndVersion
		klog.Infof("Get size %d for backup files in %s of cluster %s success", backupSize, backupFullPath, bm)
		klog.Infof("Get cluster %s commitTs %d success", bm, commitTS)
		ts := strconv.FormatUint(commitTS, 10)
		updateStatus = &controller.BackupUpdateStatus{
			TimeStarted:        &metav1.Time{Time: started},
			TimeCompleted:      &metav1.Time{Time: time.Now()},
			BackupSize:         &backupSize,
			BackupSizeReadable: &backupSizeReadable,
			CommitTs:           &ts,
		}
	}

	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   completeCondition,
		Status: corev1.ConditionTrue,
	}, updateStatus)
}

// performLogBackup execute log backup commands according to backup cr.
func (bm *Manager) performLogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	var (
		err          error
		reason       string
		resultStatus *controller.BackupUpdateStatus
	)

	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogSubCommandType(bm.SubCommand),
		Type:    v1alpha1.BackupPrepare,
		Status:  corev1.ConditionTrue,
	}, nil); err != nil {
		return err
	}

	// start/stop/truncate log backup
	switch bm.SubCommand {
	case string(v1alpha1.LogStartCommand):
		resultStatus, reason, err = bm.startLogBackup(ctx, backup)
	case string(v1alpha1.LogStopCommand):
		resultStatus, reason, err = bm.stopLogBackup(ctx, backup)
	case string(v1alpha1.LogTruncateCommand):
		resultStatus, reason, err = bm.truncateLogBackup(ctx, backup)
	default:
		return fmt.Errorf("log backup %s unknown log subcommand %s", bm, bm.SubCommand)
	}

	// handle error
	if err != nil {
		errs := make([]error, 0)
		errs = append(errs, err)
		// update failed status
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: v1alpha1.LogSubCommandType(bm.SubCommand),
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogSubCommandType(bm.SubCommand),
		Type:    v1alpha1.BackupComplete,
		Status:  corev1.ConditionTrue,
	}, resultStatus)
}

// startLogBackup starts log backup.
func (bm *Manager) startLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*controller.BackupUpdateStatus, string, error) {
	started := time.Now()
	backupFullPath, err := util.GetStoragePath(backup)
	if err != nil {
		klog.Errorf("Get backup full path of cluster %s failed, err: %s", bm, err)
		return nil, "GetBackupRemotePathFailed", err
	}
	klog.Infof("Get backup full path %s of cluster %s success", backupFullPath, bm)

	updatePathStatus := &controller.BackupUpdateStatus{
		BackupPath: &backupFullPath,
	}

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStartCommand,
		Type:    v1alpha1.BackupRunning,
		Status:  corev1.ConditionTrue,
	}, updatePathStatus); err != nil {
		return nil, "UpdateStatusFailed", err
	}

	// run br binary to do the real job
	backupErr := bm.doStartLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Start log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "StartLogBackuFailed", backupErr
	}
	klog.Infof("Start log backup of cluster %s to %s success", bm, backupFullPath)

	// get Meta info
	backupMeta, err := util.GetBRMetaData(ctx, backup.Spec.StorageProvider)
	if err != nil {
		klog.Errorf("Get log backup metadata for backup files in %s of cluster %s failed, err: %s", backupFullPath, bm, err)
		return nil, "GetLogBackupMetadataFailed", err
	}
	klog.Infof("Get log backup metadata for backup files in %s of cluster %s success", backupFullPath, bm)
	commitTs := backupMeta.StartVersion
	klog.Infof("Get cluster %s commitTs %d success", bm, commitTs)
	finish := time.Now()

	ts := strconv.FormatUint(commitTs, 10)
	updateStatus := &controller.BackupUpdateStatus{
		TimeStarted:   &metav1.Time{Time: started},
		TimeCompleted: &metav1.Time{Time: finish},
		CommitTs:      &ts,
	}
	return updateStatus, "", nil
}

// stopLogBackup stops log backup.
func (bm *Manager) stopLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*controller.BackupUpdateStatus, string, error) {
	started := time.Now()

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStopCommand,
		Type:    v1alpha1.BackupRunning,
		Status:  corev1.ConditionTrue,
	}, nil); err != nil {
		return nil, "UpdateStatusFailed", err
	}

	// run br binary to do the real job
	backupErr := bm.doStopLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Stop log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "StopLogBackupFailed", backupErr
	}
	klog.Infof("Stop log backup of cluster %s success", bm)

	finish := time.Now()

	updateStatus := &controller.BackupUpdateStatus{
		TimeStarted:   &metav1.Time{Time: started},
		TimeCompleted: &metav1.Time{Time: finish},
	}
	return updateStatus, "", nil
}

// truncateLogBackup truncates log backup.
func (bm *Manager) truncateLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*controller.BackupUpdateStatus, string, error) {
	started := time.Now()

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogTruncateCommand,
		Type:    v1alpha1.BackupRunning,
		Status:  corev1.ConditionTrue,
	}, nil); err != nil {
		return nil, "UpdateStatusFailed", err
	}

	// run br binary to do the real job
	backupErr := bm.doTruncateLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Truncate log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "TruncateLogBackuFailed", backupErr
	}
	klog.Infof("Truncate log backup of cluster %s success", bm)

	finish := time.Now()

	updateStatus := &controller.BackupUpdateStatus{
		TimeStarted:             &metav1.Time{Time: started},
		TimeCompleted:           &metav1.Time{Time: finish},
		LogSuccessTruncateUntil: &bm.TruncateUntil,
	}
	return updateStatus, "", nil
}

// performVolumeBackupInitialize execute br to stop GC and PD schedules
// it will keep running until the process is killed
func (bm *Manager) performVolumeBackupInitialize(ctx context.Context, backup *v1alpha1.Backup) error {
	err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRunning,
		Status: corev1.ConditionTrue,
	}, nil)
	if err != nil {
		return err
	}

	if err = bm.doInitializeVolumeBackup(ctx, backup, bm.StatusUpdater); err != nil {
		errs := make([]error, 0, 2)
		errs = append(errs, err)
		updateErr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "InitializeVolumeBackupFailed",
			Message: err.Error(),
		}, nil)
		if updateErr != nil {
			errs = append(errs, updateErr)
		}
		return errorutils.NewAggregate(errs)
	}

	return nil
}

func (bm *Manager) cleanSnapshotBackupEnv(ctx context.Context, backup *v1alpha1.Backup) error {
	if backup.Spec.Mode != v1alpha1.BackupModeSnapshot {
		return nil
	}

	cleanOpt := clean.Options{Namespace: bm.Namespace, BackupName: bm.ResourceName}
	return cleanOpt.CleanBRRemoteBackupData(ctx, backup)
}

func (bm *Manager) isBRCanContinueRunByCheckpoint() bool {
	v, err := semver.NewVersion(bm.TiKVVersion)
	if err != nil {
		klog.Errorf("Parse version %s failure, error: %v", bm.TiKVVersion, err)
		return false
	}
	lessThanV651, _ := semver.NewConstraint("<v6.5.1-0")
	return !lessThanV651.Check(v)
}

// VolumeBackupInitializeManager manages volume backup initializing status
type VolumeBackupInitializeManager struct {
	done               bool
	gcStopped          bool
	pdSchedulesStopped bool

	backup        *v1alpha1.Backup
	statusUpdater controller.BackupConditionUpdaterInterface
}

// UpdateBackupStatus extracts information from log line and update backup status to VolumeBackupInitialized
// when GC and PD schedules are all stopped
func (vb *VolumeBackupInitializeManager) UpdateBackupStatus(logLine string) {
	if vb.done {
		return
	}

	if strings.Contains(logLine, gcPausedKeyword) {
		vb.gcStopped = true
	} else if strings.Contains(logLine, pdSchedulesPausedKeyword) {
		vb.pdSchedulesStopped = true
	}
	vb.tryUpdateBackupStatus()
}

// tryUpdateBackupStatus tries to update backup status
func (vb *VolumeBackupInitializeManager) tryUpdateBackupStatus() {
	if !vb.gcStopped || !vb.pdSchedulesStopped {
		return
	}

	err := vb.statusUpdater.Update(vb.backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.VolumeBackupInitialized,
		Status: corev1.ConditionTrue,
	}, nil)
	if err == nil {
		vb.done = true
	} else {
		klog.Warningf("backup %s/%s update status to VolumeBackupInitialized failed, err: %s",
			vb.backup.Namespace, vb.backup.Name, err.Error())
	}
}
