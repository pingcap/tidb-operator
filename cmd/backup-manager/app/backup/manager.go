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
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupManager mainly used to manage backup related work
type BackupManager struct {
	Cli           versioned.Interface
	StatusUpdater controller.BackupConditionUpdaterInterface
	BackupOpts
}

// NewBackupManager return a BackupManager
func NewBackupManager(
	cli versioned.Interface,
	statusUpdater controller.BackupConditionUpdaterInterface,
	backupOpts BackupOpts) *BackupManager {
	return &BackupManager{
		cli,
		statusUpdater,
		backupOpts,
	}
}

// ProcessBackup used to process the backup logic
func (bm *BackupManager) ProcessBackup() error {
	backup, err := bm.Cli.PingcapV1alpha1().Backups(bm.Namespace).Get(bm.BackupName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.BackupName, err)
	}
	// The backup job has been scheduled successfully, update related status.
	err = bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupScheduled,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	db, err := util.OpenDB(bm.getDSN(constants.TidbMetaDB))
	if err != nil {
		return err
	}
	defer db.Close()
	return bm.performBackup(backup, db)
}

func (bm *BackupManager) performBackup(backup *v1alpha1.Backup, db *sql.DB) error {
	started := time.Now()

	err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRunning,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	oldTikvGCTime, err := bm.getTikvGCLifeTime(db)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("cluster %s %s is %s", bm, constants.TikvGCVariable, oldTikvGCTime)

	err = bm.setTikvGClifeTime(db, constants.TikvGCLifeTime)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "SetTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("increase cluster %s %s to %s success", bm, constants.TikvGCVariable, constants.TikvGCLifeTime)

	backupFullPath, err := bm.dumpTidbClusterData()
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "DumpTidbClusterFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("dump cluster %s data success", bm)

	err = bm.setTikvGClifeTime(db, oldTikvGCTime)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "RestTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("reset cluster %s %s to %s success", bm, constants.TikvGCVariable, oldTikvGCTime)

	// TODO: Concurrent get file size and upload backup data to speed up processing time
	archiveBackupPath := backupFullPath + constants.DefaultArchiveExtention
	err = archiveBackupData(backupFullPath, archiveBackupPath)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ArchiveBackupDataFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("archive cluster %s backup data %s success", bm, archiveBackupPath)

	// TODO: Maybe use rclone size command to get the archived backup file size is more efficiently than du
	size, err := getBackupSize(archiveBackupPath)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupSizeFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("get cluster %s archived backup file %s failed, err: %v", bm, archiveBackupPath, err)

	commitTs, err := getCommitTsFromMetadata(backupFullPath)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetCommitTsFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("get cluster %s commitTs %s success", bm, commitTs)

	bucketURI := bm.getDestBucketURI()
	err = bm.backupDataToRemote(archiveBackupPath, bucketURI)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackupDataToRemoteFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("backup cluster %s data to %s success", bm, bm.StorageType)

	finish := time.Now()

	backup.Status.BackupPath = bucketURI
	backup.Status.TimeStarted = metav1.Time{Time: started}
	backup.Status.TimeCompleted = metav1.Time{Time: finish}
	backup.Status.BackupSize = size
	backup.Status.CommitTs = commitTs

	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: corev1.ConditionTrue,
	})
}

// ProcessCleanBackup used to clean the specific backup
func (bm *BackupManager) ProcessCleanBackup() error {
	backup, err := bm.Cli.PingcapV1alpha1().Backups(bm.Namespace).Get(bm.BackupName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.BackupName, err)
	}
	err = bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupClean,
		Status: corev1.ConditionFalse,
	})
	if err != nil {
		return err
	}
	return bm.performCleanBackup(backup)
}

func (bm *BackupManager) performCleanBackup(backup *v1alpha1.Backup) error {
	if backup.Status.BackupPath == "" {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackupPathIsEmpty",
			Message: fmt.Sprintf("the cluster %s backup path is empty", bm),
		})
	}
	err := bm.cleanRemoteBackupData(backup.Status.BackupPath)
	if err != nil {
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CleanBackupDataFailed",
			Message: err.Error(),
		})
	}
	err = bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupClean,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}
	glog.Infof("clean cluster %s backup %s success", bm, backup.Status.BackupPath)
	return nil
}
