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

package backupschedule

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

type nowFn func() time.Time

type backupScheduleManager struct {
	deps *controller.BrFedDependencies
	now  nowFn
}

// NewBackupScheduleManager return backupScheduleManager
func NewBackupScheduleManager(deps *controller.BrFedDependencies) fedvolumebackup.BackupScheduleManager {
	return &backupScheduleManager{
		deps: deps,
		now:  time.Now,
	}
}

func (bm *backupScheduleManager) Sync(volumeBackupSchedule *v1alpha1.VolumeBackupSchedule) error {
	ns := volumeBackupSchedule.GetNamespace()
	name := volumeBackupSchedule.GetName()
	// TODO(federation): implement the main logic of backupSchedule
	klog.Infof("sync VolumeBackupSchedule %s/%s", ns, name)

	return nil
}

var _ fedvolumebackup.BackupScheduleManager = &backupScheduleManager{}

type FakeBackupScheduleManager struct {
	err error
}

func NewFakeBackupScheduleManager() *FakeBackupScheduleManager {
	return &FakeBackupScheduleManager{}
}

func (m *FakeBackupScheduleManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeBackupScheduleManager) Sync(_ *v1alpha1.VolumeBackupSchedule) error {
	return m.err
}

var _ fedvolumebackup.BackupScheduleManager = &FakeBackupScheduleManager{}
