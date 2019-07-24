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

package backupschedule

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

// ControlInterface implements the control logic for updating BackupSchedule
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateBackupSchedule implements the control logic for backup schedule
	UpdateBackupSchedule(schedule *v1alpha1.BackupSchedule) error
}

// NewDefaultBackupScheduleControl returns a new instance of the default implementation BackupScheduleControlInterface that
// implements the documented semantics for BackupSchedule.
func NewDefaultBackupScheduleControl(
	statusUpdater controller.BackupScheduleStatusUpdaterInterface,
	bsManager backup.BackupScheduleManager,
	recorder record.EventRecorder) ControlInterface {
	return &defaultBackupScheduleControl{
		statusUpdater,
		bsManager,
		recorder,
	}
}

type defaultBackupScheduleControl struct {
	statusUpdater controller.BackupScheduleStatusUpdaterInterface
	bsManager     backup.BackupScheduleManager
	recorder      record.EventRecorder
}

// UpdateBackupSchedule executes the core logic loop for a BackupSchedule.
func (bsc *defaultBackupScheduleControl) UpdateBackupSchedule(bs *v1alpha1.BackupSchedule) error {
	var errs []error
	oldStatus := bs.Status.DeepCopy()

	if err := bsc.updateBackupSchedule(bs); err != nil {
		errs = append(errs, err)
	}
	if apiequality.Semantic.DeepEqual(&bs.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	if err := bsc.statusUpdater.UpdateBackupScheduleStatus(bs.DeepCopy(), &bs.Status, oldStatus); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (bsc *defaultBackupScheduleControl) updateBackupSchedule(bs *v1alpha1.BackupSchedule) error {
	return bsc.bsManager.Sync(bs)
}
