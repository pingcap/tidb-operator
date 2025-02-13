// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

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
