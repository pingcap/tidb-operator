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
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

type backupManager struct {
	backupLister listers.BackupLister
	jobControl   controller.JobControlInterface
}

// NewBackupManager return a *backupManager
func NewBackupManager(
	backupLister listers.BackupLister,
	jobControl controller.JobControlInterface,
) backup.BackupManager {
	return &backupManager{
		backupLister,
		jobControl,
	}
}

func (bm *backupManager) Sync(backup *v1alpha1.Backup) error {
	// TODO: Implement this method
	return nil
}

var _ backup.BackupManager = &backupManager{}
