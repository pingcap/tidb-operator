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

package clean

import (
	"fmt"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
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

// ProcessCleanBackup used to clean the specific backup
func (bm *Manager) ProcessCleanBackup() error {
	backup, err := bm.backupLister.Backups(bm.Namespace).Get(bm.BackupName)
	if err != nil {
		return fmt.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.BackupName, err)
	}

	return bm.performCleanBackup(backup.DeepCopy())
}

func (bm *Manager) performCleanBackup(backup *v1alpha1.Backup) error {
	if backup.Status.BackupPath == "" {
		klog.Errorf("cluster %s backup path is empty", bm)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "BackupPathIsEmpty",
			Message: fmt.Sprintf("the cluster %s backup path is empty", bm),
		})
	}

	var errs []error
	var err error
	if backup.Spec.BR != nil {
		err = bm.cleanBRRemoteBackupData(backup)
	} else {
		opts := util.GetOptions(backup.Spec.StorageProvider)
		err = bm.cleanRemoteBackupData(backup.Status.BackupPath, opts)
	}

	if err != nil {
		errs = append(errs, err)
		klog.Errorf("clean cluster %s backup %s failed, err: %s", bm, backup.Status.BackupPath, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CleanBackupDataFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	klog.Infof("clean cluster %s backup %s success", bm, backup.Status.BackupPath)
	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupClean,
		Status: corev1.ConditionTrue,
	})
}
