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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

// ControlInterface implements the control logic for updating Backup
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateBackup implements the control logic for backup job and backup clean job's creation, deletion
	UpdateBackup(schedule *v1alpha1.Backup) error
}

// NewDefaultBackupControl returns a new instance of the default implementation BackupControlInterface that
// implements the documented semantics for Backup.
func NewDefaultBackupControl(
	statusUpdater controller.BackupConditionUpdaterInterface,
	backupManager backup.BackupManager,
	recorder record.EventRecorder) ControlInterface {
	return &defaultBackupControl{
		statusUpdater,
		backupManager,
		recorder,
	}
}

type defaultBackupControl struct {
	statusUpdater controller.BackupConditionUpdaterInterface
	backupManager backup.BackupManager
	recorder      record.EventRecorder
}

// UpdateBackup executes the core logic loop for a Backup.
func (bc *defaultBackupControl) UpdateBackup(backup *v1alpha1.Backup) error {
	var errs []error
	oldStatus := backup.Status.DeepCopy()

	if err := bc.updateBackup(backup); err != nil {
		errs = append(errs, err)
	}
	if apiequality.Semantic.DeepEqual(&backup.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}

	return errorutils.NewAggregate(errs)
}

func (bc *defaultBackupControl) updateBackup(backup *v1alpha1.Backup) error {
	return bc.backupManager.Sync(backup)
}
