// Copyright 2023 PingCAP, Inc.
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

package fedvolumebackup

import (
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
)

// BackupManager implements the logic for manage federation VolumeBackup.
type BackupManager interface {
	// Sync	implements the logic for syncing VolumeBackup.
	Sync(volumeBackup *v1alpha1.VolumeBackup) error
	// UpdateStatus updates the status for a VolumeBackup, include condition and status info.
	UpdateStatus(volumeBackup *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error
}

// RestoreManager implements the logic for manage federation VolumeRestore.
type RestoreManager interface {
	// Sync	implements the logic for syncing VolumeRestore.
	Sync(volumeRestore *v1alpha1.VolumeRestore) error
	// UpdateCondition updates the condition for a VolumeRestore.
	UpdateCondition(volumeRestore *v1alpha1.VolumeRestore, condition *v1alpha1.VolumeRestoreCondition) error
}

// BackupScheduleManager implements the logic for manage federation VolumeBackupSchedule.
type BackupScheduleManager interface {
	// Sync	implements the logic for syncing VolumeBackupSchedule.
	Sync(volumeBackup *v1alpha1.VolumeBackupSchedule) error
}
