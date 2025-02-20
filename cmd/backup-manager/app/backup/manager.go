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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Masterminds/semver"
	"github.com/dustin/go-humanize"
	"github.com/pingcap/errors"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/clean"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupMgr "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/backup"
)

const (
	ReasonUpdateStatusFailed = "UpdateStatusFailed"
)

// Manager mainly used to manage backup related work
type Manager struct {
	cli           client.Client
	StatusUpdater backupMgr.BackupConditionUpdaterInterface
	Options
}

// NewManager return a Manager
func NewManager(
	cli client.Client,
	statusUpdater backupMgr.BackupConditionUpdaterInterface,
	backupOpts Options) *Manager {
	return &Manager{
		cli,
		statusUpdater,
		backupOpts,
	}
}

// ProcessBackup used to process the backup logic
func (bm *Manager) ProcessBackup() error {
	ctx, cancel := util.GetContextForTerminationSignals(bm.ResourceName)
	defer cancel()

	var errs []error
	backup := &v1alpha1.Backup{}
	err := bm.cli.Get(ctx, client.ObjectKey{Namespace: bm.Namespace, Name: bm.ResourceName}, backup)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.ResourceName, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "GetBackupCRFailed",
				Message: err.Error(),
			},
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	crData, err := json.Marshal(backup)
	if err != nil {
		klog.Errorf("failed to marshal backup %v to json, err: %v", backup, err)
	} else {
		klog.Infof("start to process backup: %s", string(crData))
	}

	// we treat snapshot backup as restarted if its status is not scheduled when backup pod just start to run
	// we will clean backup data before run br command
	if backup.Spec.Mode == v1alpha1.BackupModeSnapshot && (backup.Status.Phase != v1alpha1.BackupScheduled || v1alpha1.IsBackupRestart(backup)) {
		klog.Infof("snapshot backup %s was restarted, status is %s", bm, backup.Status.Phase)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:   string(v1alpha1.BackupRestart),
				Status: metav1.ConditionTrue,
			},
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

	// skip the DB initialization if spec.from is not specified
	return bm.performBackup(ctx, backup.DeepCopy())
}

// nolint: gocyclo
func (bm *Manager) performBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	started := time.Now()

	var errs []error

	backupFullPath, err := util.GetStoragePath(&backup.Spec.StorageProvider)
	if err != nil {
		errs = append(errs, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "GetBackupRemotePathFailed",
				Message: err.Error(),
			},
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	updatePathStatus := &backupMgr.BackupUpdateStatus{
		BackupPath: &backupFullPath,
	}
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupPrepare),
			Status: metav1.ConditionTrue,
		},
	}, updatePathStatus); err != nil {
		return err
	}

	// clean snapshot backup data if it was restarted
	if backup.Spec.Mode == v1alpha1.BackupModeSnapshot && v1alpha1.IsBackupRestart(backup) && !bm.isBRCanContinueRunByCheckpoint() {
		klog.Infof("clean snapshot backup %s data before run br command, backup path is %s", bm, backup.Status.BackupPath)
		if err := bm.cleanSnapshotBackupEnv(ctx, backup); err != nil {
			return errors.Annotatef(err, "clean snapshot backup %s failed", bm)
		}
	}

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupRunning),
			Status: metav1.ConditionTrue,
		},
	}, nil); err != nil {
		return err
	}

	// run br binary to do the real job
	backupErr := bm.backupData(ctx, backup)
	if backupErr != nil {
		errs = append(errs, backupErr)
		klog.Errorf("backup cluster %s data failed, err: %s", bm, backupErr)
		failedCondition := v1alpha1.BackupFailed
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(failedCondition),
				Status:  metav1.ConditionTrue,
				Reason:  "BackupDataToRemoteFailed",
				Message: backupErr.Error(),
			},
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("backup cluster %s data to %s success", bm, backupFullPath)

	var updateStatus *backupMgr.BackupUpdateStatus
	completeCondition := v1alpha1.BackupComplete
	switch bm.Mode {
	default:
		backupMeta, err := util.GetBRMetaData(ctx, backup.Spec.StorageProvider)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("Get backup metadata for backup files in %s of cluster %s failed, err: %s", backupFullPath, bm, err)
			uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Condition: metav1.Condition{
					Type:    string(v1alpha1.BackupFailed),
					Status:  metav1.ConditionTrue,
					Reason:  "GetBackupMetadataFailed",
					Message: err.Error(),
				},
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		klog.Infof("Get br metadata for backup files in %s of cluster %s success", backupFullPath, bm)
		backupSize := int64(util.GetBRArchiveSize(backupMeta))   //nolint:gosec
		backupSizeReadable := humanize.Bytes(uint64(backupSize)) //nolint: gosec
		commitTS := backupMeta.EndVersion
		klog.Infof("Get size %d for backup files in %s of cluster %s success", backupSize, backupFullPath, bm)
		klog.Infof("Get cluster %s commitTs %d success", bm, commitTS)
		ts := strconv.FormatUint(commitTS, 10)
		updateStatus = &backupMgr.BackupUpdateStatus{
			TimeStarted:        &metav1.Time{Time: started},
			TimeCompleted:      &metav1.Time{Time: time.Now()},
			BackupSize:         &backupSize,
			BackupSizeReadable: &backupSizeReadable,
			CommitTs:           &ts,
		}
	}

	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Condition: metav1.Condition{
			Type:   string(completeCondition),
			Status: metav1.ConditionTrue,
		},
	}, updateStatus)
}

// performLogBackup execute log backup commands according to backup cr.
func (bm *Manager) performLogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	var (
		err          error
		reason       string
		resultStatus *backupMgr.BackupUpdateStatus
	)

	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogSubCommandType(bm.SubCommand),
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupPrepare),
			Status: metav1.ConditionTrue,
		},
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
	case string(v1alpha1.LogResumeCommand):
		resultStatus, reason, err = bm.resumeLogBackup(ctx, backup)
	case string(v1alpha1.LogPauseCommand):
		resultStatus, reason, err = bm.pauseLogBackup(ctx, backup)
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
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupFailed),
				Status:  metav1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			},
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogSubCommandType(bm.SubCommand),
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupComplete),
			Status: metav1.ConditionTrue,
		},
	}, resultStatus)
}

// startLogBackup starts log backup.
func (bm *Manager) startLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*backupMgr.BackupUpdateStatus, string, error) {
	started := time.Now()
	backupFullPath, err := util.GetStoragePath(&backup.Spec.StorageProvider)
	if err != nil {
		klog.Errorf("Get backup full path of cluster %s failed, err: %s", bm, err)
		return nil, "GetBackupRemotePathFailed", err
	}
	klog.Infof("Get backup full path %s of cluster %s success", backupFullPath, bm)

	updatePathStatus := &backupMgr.BackupUpdateStatus{
		BackupPath: &backupFullPath,
	}

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStartCommand,
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupRunning),
			Status: metav1.ConditionTrue,
		},
	}, updatePathStatus); err != nil {
		return nil, ReasonUpdateStatusFailed, err
	}

	// run br binary to do the real job
	backupErr := bm.doStartLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Start log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "StartLogBackupFailed", backupErr
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
	updateStatus := &backupMgr.BackupUpdateStatus{
		TimeStarted:   &metav1.Time{Time: started},
		TimeCompleted: &metav1.Time{Time: finish},
		CommitTs:      &ts,
	}
	return updateStatus, "", nil
}

// resumeLogBackup resume log backup.
func (bm *Manager) resumeLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*backupMgr.BackupUpdateStatus, string, error) {
	started := time.Now()

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogResumeCommand,
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupRunning),
			Status: metav1.ConditionTrue,
		},
	}, nil); err != nil {
		return nil, ReasonUpdateStatusFailed, err
	}

	// run br binary to do the real job
	backupErr := bm.doResumeLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Resume log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "ResumeLogBackuFailed", backupErr
	}
	klog.Infof("Resume log backup of cluster %s success", bm)

	finish := time.Now()
	updateStatus := &backupMgr.BackupUpdateStatus{
		TimeStarted:   &metav1.Time{Time: started},
		TimeCompleted: &metav1.Time{Time: finish},
	}
	return updateStatus, "", nil
}

// stopLogBackup stops log backup.
func (bm *Manager) stopLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*backupMgr.BackupUpdateStatus, string, error) {
	started := time.Now()

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStopCommand,
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupRunning),
			Status: metav1.ConditionTrue,
		},
	}, nil); err != nil {
		return nil, ReasonUpdateStatusFailed, err
	}

	// run br binary to do the real job
	backupErr := bm.doStopLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Stop log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "StopLogBackupFailed", backupErr
	}
	klog.Infof("Stop log backup of cluster %s success", bm)

	finish := time.Now()

	updateStatus := &backupMgr.BackupUpdateStatus{
		TimeStarted:   &metav1.Time{Time: started},
		TimeCompleted: &metav1.Time{Time: finish},
	}
	return updateStatus, "", nil
}

// pauseLogBackup pauses log backup.
func (bm *Manager) pauseLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*backupMgr.BackupUpdateStatus, string, error) {
	started := time.Now()

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogPauseCommand,
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupRunning),
			Status: metav1.ConditionTrue,
		},
	}, nil); err != nil {
		return nil, ReasonUpdateStatusFailed, err
	}

	// run br binary to do the real job
	backupErr := bm.doPauseLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Pause log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "PauseLogBackupFailed", backupErr
	}
	klog.Infof("Pause log backup of cluster %s success", bm)

	finish := time.Now()

	updateStatus := &backupMgr.BackupUpdateStatus{
		TimeStarted:   &metav1.Time{Time: started},
		TimeCompleted: &metav1.Time{Time: finish},
	}
	return updateStatus, "", nil
}

// truncateLogBackup truncates log backup.
func (bm *Manager) truncateLogBackup(ctx context.Context, backup *v1alpha1.Backup) (*backupMgr.BackupUpdateStatus, string, error) {
	started := time.Now()

	// change Prepare to Running before real backup process start
	if err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogTruncateCommand,
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupRunning),
			Status: metav1.ConditionTrue,
		},
	}, nil); err != nil {
		return nil, ReasonUpdateStatusFailed, err
	}

	// run br binary to do the real job
	backupErr := bm.doTruncateLogBackup(ctx, backup)

	if backupErr != nil {
		klog.Errorf("Truncate log backup of cluster %s failed, err: %s", bm, backupErr)
		return nil, "TruncateLogBackuFailed", backupErr
	}
	klog.Infof("Truncate log backup of cluster %s success", bm)

	finish := time.Now()

	updateStatus := &backupMgr.BackupUpdateStatus{
		TimeStarted:             &metav1.Time{Time: started},
		TimeCompleted:           &metav1.Time{Time: finish},
		LogSuccessTruncateUntil: &bm.TruncateUntil,
	}
	return updateStatus, "", nil
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
