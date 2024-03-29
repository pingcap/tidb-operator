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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

// BackupManager implements the logic for manage backup.
type BackupManager interface {
	// Sync	implements the logic for syncing Backup.
	Sync(backup *v1alpha1.Backup) error
	// UpdateStatus updates the status for a Backup, include condition and status info.
	UpdateStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *controller.BackupUpdateStatus) error
}

// RestoreManager implements the logic for manage restore.
type RestoreManager interface {
	// Sync	implements the logic for syncing Restore.
	Sync(backup *v1alpha1.Restore) error
	// UpdateCondition updates the condition for a Restore.
	UpdateCondition(restore *v1alpha1.Restore, condition *v1alpha1.RestoreCondition) error
}

// BackupScheduleManager implements the logic for manage backupSchedule.
type BackupScheduleManager interface {
	// Sync	implements the logic for syncing BackupSchedule.
	Sync(backup *v1alpha1.BackupSchedule) error
}
