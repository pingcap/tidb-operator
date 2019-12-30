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

package export

import (
	"database/sql"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	glog "k8s.io/klog"
)

// BackupManager mainly used to manage backup related work
type BackupManager struct {
	backupLister  listers.BackupLister
	StatusUpdater controller.BackupConditionUpdaterInterface
	BackupOpts
}

// NewBackupManager return a BackupManager
func NewBackupManager(
	backupLister listers.BackupLister,
	statusUpdater controller.BackupConditionUpdaterInterface,
	backupOpts BackupOpts) *BackupManager {
	return &BackupManager{
		backupLister,
		statusUpdater,
		backupOpts,
	}
}

// ProcessBackup used to process the backup logic
func (bm *BackupManager) ProcessBackup() error {
	backup, err := bm.backupLister.Backups(bm.Namespace).Get(bm.BackupName)
	if err != nil {
		glog.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.BackupName, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupCRFailed",
			Message: err.Error(),
		})
	}

	var db *sql.DB
	err = wait.PollImmediate(constants.PollInterval, constants.CheckTimeout, func() (done bool, err error) {
		db, err = util.OpenDB(bm.getDSN(constants.TidbMetaDB))
		if err != nil {
			glog.Warningf("can't open connection to tidb cluster %s, err: %v", bm, err)
			return false, nil
		}

		if err := db.Ping(); err != nil {
			glog.Warningf("can't connect to tidb cluster %s, err: %s", bm, err)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		glog.Errorf("cluster %s connect failed, err: %s", bm, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ConnectTidbFailed",
			Message: err.Error(),
		})
	}

	defer db.Close()
	return bm.performBackup(backup.DeepCopy(), db)
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
		glog.Errorf("cluster %s get %s failed, err: %s", bm, constants.TikvGCVariable, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("cluster %s %s is %s", bm, constants.TikvGCVariable, oldTikvGCTime)

	oldTikvGCTimeDuration, err := time.ParseDuration(oldTikvGCTime)
	if err != nil {
		glog.Errorf("cluster %s parse old %s failed, err: %s", bm, constants.TikvGCVariable, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ParseOldTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
	}
	tikvGCTimeDuration, err := time.ParseDuration(constants.TikvGCLifeTime)
	if err != nil {
		glog.Errorf("cluster %s parse default %s failed, err: %s", bm, constants.TikvGCVariable, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ParseDefaultTikvGCLifeTimeFailed",
			Message: err.Error(),
		})
	}
	if oldTikvGCTimeDuration < tikvGCTimeDuration {
		err = bm.setTikvGCLifeTime(db, constants.TikvGCLifeTime)
		if err != nil {
			glog.Errorf("cluster %s set tikv GC life time to %s failed, err: %s", bm, constants.TikvGCLifeTime, err)
			return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "SetTikvGCLifeTimeFailed",
				Message: err.Error(),
			})
		}
		glog.Infof("set cluster %s %s to %s success", bm, constants.TikvGCVariable, constants.TikvGCLifeTime)
	}

	backupFullPath, err := bm.dumpTidbClusterData()
	if err != nil {
		glog.Errorf("dump cluster %s data failed, err: %s", bm, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "DumpTidbClusterFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("dump cluster %s data to %s success", bm, backupFullPath)

	if oldTikvGCTimeDuration < tikvGCTimeDuration {
		err = bm.setTikvGCLifeTime(db, oldTikvGCTime)
		if err != nil {
			glog.Errorf("cluster %s reset tikv GC life time to %s failed, err: %s", bm, oldTikvGCTime, err)
			return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "ResetTikvGCLifeTimeFailed",
				Message: err.Error(),
			})
		}
		glog.Infof("reset cluster %s %s to %s success", bm, constants.TikvGCVariable, oldTikvGCTime)
	}
	// TODO: Concurrent get file size and upload backup data to speed up processing time
	archiveBackupPath := backupFullPath + constants.DefaultArchiveExtention
	err = archiveBackupData(backupFullPath, archiveBackupPath)
	if err != nil {
		glog.Errorf("archive cluster %s backup data %s failed, err: %s", bm, archiveBackupPath, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ArchiveBackupDataFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("archive cluster %s backup data %s success", bm, archiveBackupPath)

	size, err := getBackupSize(archiveBackupPath)
	if err != nil {
		glog.Errorf("get cluster %s archived backup file %s size %d failed, err: %s", bm, archiveBackupPath, size, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupSizeFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("get cluster %s archived backup file %s size %d success", bm, archiveBackupPath, size)

	commitTs, err := getCommitTsFromMetadata(backupFullPath)
	if err != nil {
		glog.Errorf("get cluster %s commitTs failed, err: %s", bm, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetCommitTsFailed",
			Message: err.Error(),
		})
	}
	glog.Infof("get cluster %s commitTs %s success", bm, commitTs)

	remotePath := strings.TrimPrefix(archiveBackupPath, constants.BackupRootPath+"/")
	bucketURI := bm.getDestBucketURI(remotePath)
	err = bm.backupDataToRemote(archiveBackupPath, bucketURI)
	if err != nil {
		glog.Errorf("backup cluster %s data to %s failed, err: %s", bm, bm.StorageType, err)
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
