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
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ProcessBackup used to process the backup logic
func (bm *Manager) ProcessBackup() error {
	backup, err := bm.backupLister.Backups(bm.Namespace).Get(bm.BackupName)
	if err != nil {
		klog.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.BackupName, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupCRFailed",
			Message: err.Error(),
		})
	}

	if backup.Spec.BR == nil {
		return fmt.Errorf("no br config in %s", bm)
	}
	return bm.performBackup(backup.DeepCopy())
}

func (bm *Manager) performBackup(backup *v1alpha1.Backup) error {
	started := time.Now()

	err := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRunning,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	backupFullPath, err := bm.backupData(backup)
	if err != nil {
		klog.Errorf("backup cluster %s data failed, err: %s", bm, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackupDataToRemoteFailed",
			Message: err.Error(),
		})
	}
	klog.Infof("backup cluster %s data to %s success", bm, backupFullPath)

	// Note: The size get from remote may be incorrect because the blobs
	// are eventually consistent.
	size, err := getBackupSize(backup)
	if err != nil {
		klog.Errorf("Get size for backup files in %s of cluster %s failed, err: %s", backupFullPath, bm, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupSizeFailed",
			Message: err.Error(),
		})
	}
	klog.Infof("Get size %d for backup files in %s of cluster %s success", size, backupFullPath, bm)

	commitTs, err := getCommitTs(backup)
	if err != nil {
		klog.Errorf("get cluster %s commitTs failed, err: %s", bm, err)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetCommitTsFailed",
			Message: err.Error(),
		})
	}
	klog.Infof("get cluster %s commitTs %d success", bm, commitTs)

	finish := time.Now()

	backup.Status.BackupPath = backupFullPath
	backup.Status.TimeStarted = metav1.Time{Time: started}
	backup.Status.TimeCompleted = metav1.Time{Time: finish}
	backup.Status.BackupSize = size
	backup.Status.CommitTs = fmt.Sprintf("%d", commitTs)

	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: corev1.ConditionTrue,
	})
}
