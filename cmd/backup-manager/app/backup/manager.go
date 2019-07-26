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

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
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
	db, err := util.OpenDB(bm.getDSN(constants.TidbMetaDB))
	if err != nil {
		return err
	}
	defer db.Close()

	backup, err := bm.Cli.PingcapV1alpha1().Backups(bm.Namespace).Get(bm.BackupName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.BackupName, err)
	}
	return bm.performBackup(backup, db)
}

func (bm *BackupManager) performBackup(backup *v1alpha1.Backup, db *sql.DB) error {
	oldTikvGCTime, err := bm.getTikvGCLifeTime(db)
	if err != nil {
		return err
	}
	glog.Infof("cluster %s %s is %s", bm, constants.TikvGCVariable, oldTikvGCTime)

	err = bm.setTikvGClifeTime(db, constants.TikvGCLifeTime)
	if err != nil {
		return err
	}
	glog.Infof("increase cluster %s %s to %s success", bm, constants.TikvGCVariable, constants.TikvGCLifeTime)

	backupFullPath, err := bm.dumpTidbClusterData()
	if err != nil {
		return err
	}
	glog.Infof("dump cluster %s data success", bm)

	err = bm.setTikvGClifeTime(db, oldTikvGCTime)
	if err != nil {
		return err
	}
	glog.Infof("reset cluster %s %s to %s success", bm, constants.TikvGCVariable, oldTikvGCTime)

	// TODO: Concurrent get file size and upload backup data to speed up processing time
	archiveBackupPath := backupFullPath + constants.DefaultArchiveExtention
	err = archiveBackupData(backupFullPath, archiveBackupPath)
	if err != nil {
		return err
	}
	glog.Infof("archive cluster %s backup data %s success", bm, archiveBackupPath)

	// TODO: Maybe use rclone size command to get the archived backup file size is more efficiently than du
	_, err = getBackupSize(archiveBackupPath)
	if err != nil {
		return err
	}
	glog.Infof("get cluster %s archived backup file %s failed, err: %v", bm, archiveBackupPath, err)

	commitTs, err := getCommitTsFromMetadata(backupFullPath)
	if err != nil {
		return err
	}
	glog.Infof("get cluster %s commitTs %s success", bm, commitTs)

	err = bm.backupDataToRemote()
	if err != nil {
		return err
	}
	glog.Infof("backup cluster %s data to %s success", bm, bm.StorageType)
	return nil
}

// CleanBackup used to clean the specific backup
func (bm *BackupManager) CleanBackup() error {
	return nil
}
