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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
)

// RestoreManager mainly used to manage backup related work
type RestoreManager struct {
	restoreLister listers.RestoreLister
	statusUpdater controller.RestoreStatusUpdaterInterface
	RestoreOpts
}

// NewRestoreManager return a RestoreManager
func NewRestoreManager(
	restoreLister listers.RestoreLister,
	statusUpdater controller.RestoreStatusUpdaterInterface,
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
		v1alpha1.UpdateRestoreCondition(&restore.Status, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackupPathIsEmpty",
			Message: fmt.Sprintf("backup %s path is empty", rm.BackupName),
		})
		return rm.statusUpdater.UpdateRestoreStatus(restore.DeepCopy(), &restore.Status)
	}

	var errs []error
	oldStatus := restore.Status.DeepCopy()

	if err := rm.performRestore(restore); err != nil {
		errs = append(errs, err)
	}

	if apiequality.Semantic.DeepEqual(&restore.Status, oldStatus) {
		// without status update, return directly
		return errorutils.NewAggregate(errs)
	}

	if err := rm.statusUpdater.UpdateRestoreStatus(restore.DeepCopy(), &restore.Status); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (rm *RestoreManager) performRestore(restore *v1alpha1.Restore) error {
	started := time.Now()

	v1alpha1.UpdateRestoreCondition(&restore.Status, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreRunning,
		Status: corev1.ConditionTrue,
	})
	if err := rm.statusUpdater.UpdateRestoreStatus(restore.DeepCopy(), &restore.Status); err != nil {
		return err
	}

	restoreDataPath := rm.getRestoreDataPath()
	if err := rm.downloadBackupData(restoreDataPath); err != nil {
		v1alpha1.UpdateRestoreCondition(&restore.Status, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "DownloadBackupDataFailed",
			Message: err.Error(),
		})
		return err
	}
	glog.Infof("download cluster %s backup %s data success", rm, rm.BackupPath)

	restoreDataDir := filepath.Dir(restoreDataPath)
	unarchiveDataPath, err := unarchiveBackupData(restoreDataPath, restoreDataDir)
	if err != nil {
		v1alpha1.UpdateRestoreCondition(&restore.Status, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "UnarchiveBackupDataFailed",
			Message: err.Error(),
		})
		return err
	}
	glog.Infof("unarchive cluster %s backup %s data success", rm, restoreDataPath)

	err = rm.loadTidbClusterData(unarchiveDataPath)
	if err != nil {
		v1alpha1.UpdateRestoreCondition(&restore.Status, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "LoaderBackupDataFailed",
			Message: err.Error(),
		})
		return err
	}
	glog.Infof("restore cluster %s from backup %s success", rm, rm.BackupPath)

	finish := time.Now()

	restore.Status.TimeStarted = metav1.Time{Time: started}
	restore.Status.TimeCompleted = metav1.Time{Time: finish}

	v1alpha1.UpdateRestoreCondition(&restore.Status, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
	return nil
}
