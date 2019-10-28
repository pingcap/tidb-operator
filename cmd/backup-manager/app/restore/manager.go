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
	"fmt"
	"path/filepath"
	"time"

	"github.com/golang/glog"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestoreManager mainly used to manage backup related work
type RestoreManager struct {
	restoreLister listers.RestoreLister
	StatusUpdater controller.RestoreConditionUpdaterInterface
	RestoreOpts
}

// NewRestoreManager return a RestoreManager
func NewRestoreManager(
	restoreLister listers.RestoreLister,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	backupOpts RestoreOpts) *RestoreManager {
	return &RestoreManager{
		restoreLister,
		statusUpdater,
		backupOpts,
	}
}

// ProcessRestore used to process the restore logic
func (rm *RestoreManager) ProcessRestore() error {
	restore, err := rm.restoreLister.Restores(rm.Namespace).Get(rm.RestoreName)
	if err != nil {
		return fmt.Errorf("can't find cluster %s restore %s CRD object, err: %v", rm, rm.RestoreName, err)
	}

	if rm.BackupPath == "" {
		glog.Errorf("backup %s path is empty", rm.BackupName)
		return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackupPathIsEmpty",
			Message: fmt.Sprintf("backup %s path is empty", rm.BackupName),
		})
	}
	return rm.performRestore(restore.DeepCopy())
}

func (rm *RestoreManager) performRestore(restore *v1alpha1.Restore) error {
	started := time.Now()

	err := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreRunning,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	restoreDataPath := rm.getRestoreDataPath()
	if err := rm.downloadBackupData(restoreDataPath); err != nil {
		glog.Errorf("download cluster %s backup %s data failed, err: %s", rm, rm.BackupPath, err)
		return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "DownloadBackupDataFailed",
			Message: fmt.Sprintf("download backup %s data failed, err: %v", rm.BackupPath, err),
		})
	}
	glog.Infof("download cluster %s backup %s data success", rm, rm.BackupPath)

	restoreDataDir := filepath.Dir(restoreDataPath)
	unarchiveDataPath, err := unarchiveBackupData(restoreDataPath, restoreDataDir)
	if err != nil {
		glog.Errorf("unarchive cluster %s backup %s data failed, err: %s", rm, restoreDataPath, err)
		return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "UnarchiveBackupDataFailed",
			Message: fmt.Sprintf("unarchive backup %s data failed, err: %v", restoreDataPath, err),
		})
	}
	glog.Infof("unarchive cluster %s backup %s data success", rm, restoreDataPath)

	err = rm.loadTidbClusterData(unarchiveDataPath)
	if err != nil {
		glog.Errorf("restore cluster %s from backup %s failed, err: %s", rm, rm.BackupPath, err)
		return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "LoaderBackupDataFailed",
			Message: fmt.Sprintf("loader backup %s data failed, err: %v", restoreDataPath, err),
		})
	}
	glog.Infof("restore cluster %s from backup %s success", rm, rm.BackupPath)

	finish := time.Now()

	restore.Status.TimeStarted = metav1.Time{Time: started}
	restore.Status.TimeCompleted = metav1.Time{Time: finish}

	return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
}
