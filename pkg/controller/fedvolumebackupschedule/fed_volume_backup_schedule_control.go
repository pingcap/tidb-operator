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

package fedvolumebackupschedule

import (
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

// ControlInterface implements the control logic for updating VolumeBackupSchedule
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateBackupSchedule implements the control logic for VolumeBackupSchedule
	UpdateBackupSchedule(volumeBackupSchedule *v1alpha1.VolumeBackupSchedule) error
}

// NewDefaultVolumeBackupScheduleControl returns a new instance of the default VolumeBackupSchedule ControlInterface implementation.
func NewDefaultVolumeBackupScheduleControl(
	statusUpdater controller.VolumeBackupScheduleStatusUpdaterInterface,
	backupScheduleManager fedvolumebackup.BackupScheduleManager) ControlInterface {
	return &defaultBackupScheduleControl{
		statusUpdater,
		backupScheduleManager,
	}
}

type defaultBackupScheduleControl struct {
	statusUpdater controller.VolumeBackupScheduleStatusUpdaterInterface
	bsManager     fedvolumebackup.BackupScheduleManager
}

// UpdateBackupSchedule executes the core logic loop for a VolumeBackupSchedule.
func (c *defaultBackupScheduleControl) UpdateBackupSchedule(vbs *v1alpha1.VolumeBackupSchedule) error {
	var errs []error
	oldStatus := vbs.Status.DeepCopy()

	if err := c.updateBackupSchedule(vbs); err != nil {
		errs = append(errs, err)
	}
	if apiequality.Semantic.DeepEqual(&vbs.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	if err := c.statusUpdater.UpdateBackupScheduleStatus(vbs.DeepCopy(), &vbs.Status, oldStatus); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (c *defaultBackupScheduleControl) updateBackupSchedule(vbs *v1alpha1.VolumeBackupSchedule) error {
	return c.bsManager.Sync(vbs)
}

var _ ControlInterface = &defaultBackupScheduleControl{}

// FakeBackupScheduleControl is a fake BackupScheduleControlInterface
type FakeBackupScheduleControl struct {
	bsIndexer       cache.Indexer
	updateBsTracker controller.RequestTracker
}

// NewFakeBackupScheduleControl returns a FakeBackupScheduleControl
func NewFakeBackupScheduleControl(bsInformer informers.VolumeBackupScheduleInformer) *FakeBackupScheduleControl {
	return &FakeBackupScheduleControl{
		bsInformer.Informer().GetIndexer(),
		controller.RequestTracker{},
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (c *FakeBackupScheduleControl) SetUpdateBackupError(err error, after int) {
	c.updateBsTracker.SetError(err).SetAfter(after)
}

// UpdateBackupSchedule updates the VolumeBackupSchedule to BackupScheduleIndexer
func (c *FakeBackupScheduleControl) UpdateBackupSchedule(volumeBackupSchedule *v1alpha1.VolumeBackupSchedule) error {
	defer c.updateBsTracker.Inc()
	if c.updateBsTracker.ErrorReady() {
		defer c.updateBsTracker.Reset()
		return c.updateBsTracker.GetError()
	}

	return c.bsIndexer.Add(volumeBackupSchedule)
}

var _ ControlInterface = &FakeBackupScheduleControl{}
