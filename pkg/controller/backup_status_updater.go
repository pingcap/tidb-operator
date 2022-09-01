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

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// BackupUpdateStatus represents the status of a backup to be updated.
// This structure should keep synced with the fields in `BackupStatus`
// except for `Phase` and `Conditions`.
type BackupUpdateStatus struct {
	// BackupPath is the location of the backup.
	BackupPath *string
	// TimeStarted is the time at which the backup was started.
	TimeStarted *metav1.Time
	// TimeCompleted is the time at which the backup was completed.
	TimeCompleted *metav1.Time
	// BackupSizeReadable is the data size of the backup.
	// the difference with BackupSize is that its format is human readable
	BackupSizeReadable *string
	// BackupSize is the data size of the backup.
	BackupSize *int64
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs *string
	// LogCheckpointTs is the ts of log backup process.
	LogCheckpointTs *string
	// LogSuccessTruncateUntil is log backup already successfully truncate until timestamp.
	LogSuccessTruncateUntil *string
	// LogTruncatingUntil is log backup truncate until timestamp which is used to mark the truncate command.
	LogTruncatingUntil *string
}

// BackupConditionUpdaterInterface enables updating Backup conditions.
type BackupConditionUpdaterInterface interface {
	Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error
}

type realBackupConditionUpdater struct {
	cli          versioned.Interface
	backupLister listers.BackupLister
	recorder     record.EventRecorder
}

// returns a BackupConditionUpdaterInterface that updates the Status of a Backup,
func NewRealBackupConditionUpdater(
	cli versioned.Interface,
	backupLister listers.BackupLister,
	recorder record.EventRecorder) BackupConditionUpdaterInterface {
	return &realBackupConditionUpdater{
		cli,
		backupLister,
		recorder,
	}
}

func (u *realBackupConditionUpdater) Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error {
	ns := backup.GetNamespace()
	backupName := backup.GetName()
	// try best effort to guarantee backup is updated.
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		var isStatusUpdate, isConditionUpdate, isLogSubcommandStatusUpdate bool
		// log backup status update should be handled specially before backup is delete.
		// it needs update both subcommand status and whole backup status.
		if backup.Spec.Mode == v1alpha1.BackupModeLog && backup.DeletionTimestamp == nil {
			isLogSubcommandStatusUpdate = updateLogBackupStatus(backup, condition, newStatus)
		} else {
			isStatusUpdate = updateBackupStatus(&backup.Status, newStatus)
			isConditionUpdate = v1alpha1.UpdateBackupCondition(&backup.Status, condition)
		}
		if isStatusUpdate || isConditionUpdate || isLogSubcommandStatusUpdate {
			_, updateErr := u.cli.PingcapV1alpha1().Backups(ns).Update(context.TODO(), backup, metav1.UpdateOptions{})
			if updateErr == nil {
				klog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
				return nil
			}
			klog.Errorf("Failed to update backup [%s/%s], error: %v", ns, backupName, updateErr)
			if updated, err := u.backupLister.Backups(ns).Get(backupName); err == nil {
				// make a copy so we don't mutate the shared cache
				backup = updated.DeepCopy()
			} else {
				utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
			}
			return updateErr
		}
		return nil
	})
	return err
}

// updateBackupStatus updates existing Backup status.
// from the fields in BackupUpdateStatus.
func updateBackupStatus(status *v1alpha1.BackupStatus, newStatus *BackupUpdateStatus) bool {
	isUpdate := false
	if newStatus == nil {
		return isUpdate
	}
	if newStatus.BackupPath != nil && status.BackupPath != *newStatus.BackupPath {
		status.BackupPath = *newStatus.BackupPath
		isUpdate = true
	}
	if newStatus.TimeStarted != nil && status.TimeStarted != *newStatus.TimeStarted {
		status.TimeStarted = *newStatus.TimeStarted
		isUpdate = true
	}
	if newStatus.TimeCompleted != nil && status.TimeCompleted != *newStatus.TimeCompleted {
		status.TimeCompleted = *newStatus.TimeCompleted
		isUpdate = true
	}
	if newStatus.BackupSizeReadable != nil && status.BackupSizeReadable != *newStatus.BackupSizeReadable {
		status.BackupSizeReadable = *newStatus.BackupSizeReadable
		isUpdate = true
	}
	if newStatus.BackupSize != nil && status.BackupSize != *newStatus.BackupSize {
		status.BackupSize = *newStatus.BackupSize
		isUpdate = true
	}
	if newStatus.CommitTs != nil && status.CommitTs != *newStatus.CommitTs {
		status.CommitTs = *newStatus.CommitTs
		isUpdate = true
	}
	if newStatus.LogCheckpointTs != nil && status.LogCheckpointTs != *newStatus.LogCheckpointTs {
		status.LogCheckpointTs = *newStatus.LogCheckpointTs
		isUpdate = true
	}
	if newStatus.LogSuccessTruncateUntil != nil && status.LogSuccessTruncateUntil != *newStatus.LogSuccessTruncateUntil {
		status.LogSuccessTruncateUntil = *newStatus.LogSuccessTruncateUntil
		isUpdate = true
	}
	return isUpdate
}

// updateLogBackupStatus update log backup status.
// it will update both the log backup sub command status and the whole log backup status.
func updateLogBackupStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) bool {
	if condition == nil || condition.Command == "" {
		return false
	}
	isSubCommandStatusUpdate := updateLogSubcommandStatus(backup, condition, newStatus)
	isWholeStatusUpdate := updateWholeLogBackupStatus(backup, condition, newStatus)
	return isSubCommandStatusUpdate || isWholeStatusUpdate
}

// updateLogSubcommandStatus update log backup subcommand status.
func updateLogSubcommandStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) bool {
	// init log subcommand status map
	if backup.Status.LogSubCommandStatuses == nil {
		backup.Status.LogSubCommandStatuses = make(map[v1alpha1.LogSubCommandType]v1alpha1.LogSubCommandStatus)
	}
	// update subcommand status
	subStatus, ok := backup.Status.LogSubCommandStatuses[condition.Command]
	if !ok {
		// init subcommand status
		subStatus = v1alpha1.LogSubCommandStatus{
			Command:     condition.Command,
			TimeStarted: metav1.Time{Time: time.Now()},
			Conditions:  make([]v1alpha1.BackupCondition, 0),
		}
	}

	// update the status info and the condition info
	subcommandStatusUpdate := updateLogSubCommandStatusOnly(&subStatus, newStatus)
	subcomandConditionUpdate := updateLogSubCommandConditionOnly(&subStatus, condition)
	if subcommandStatusUpdate || subcomandConditionUpdate {
		backup.Status.LogSubCommandStatuses[condition.Command] = subStatus
		return true
	}
	return false
}

// updateWholeLogBackupStatus updates the whole log backup status.
func updateWholeLogBackupStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) bool {
	var (
		copyCondition  v1alpha1.BackupCondition
		copyStatus     BackupUpdateStatus
		copyConditionP *v1alpha1.BackupCondition
		copyStatusP    *BackupUpdateStatus
	)
	copyCondition = *condition
	copyConditionP = &copyCondition
	if newStatus != nil {
		copyStatus = *newStatus
		copyStatusP = &copyStatus
	}

	switch condition.Command {
	case v1alpha1.LogStartCommand:
		// start command should not update TimeCompleted and condition when start complete.
		// start command other status and condition info can be synced to the whole log backup status.
		if condition.Type == v1alpha1.BackupComplete {
			if copyStatusP != nil {
				copyStatusP.TimeCompleted = nil
			}
			copyConditionP = nil
		}
	case v1alpha1.LogStopCommand:
		// stop command should not update TimeStarted, but should update condition when stop complete.
		// stop command other status should not be synced to whole log backup status.
		if condition.Type == v1alpha1.BackupComplete {
			if copyStatusP != nil {
				copyStatusP.TimeStarted = nil
			}
		} else {
			copyStatusP = nil
			copyConditionP = nil
		}
	case v1alpha1.LogTruncateCommand:
		// truncate command shoudld not update TimeStarted, TimeCompleted when truncate complete.
		// truncate command other status should not be synced to whole log backup status.
		if condition.Type == v1alpha1.BackupComplete {
			if copyStatusP != nil {
				copyStatusP.TimeCompleted = nil
				copyStatusP.TimeStarted = nil
			}
		} else {
			copyStatusP = nil
		}
		copyConditionP = nil
	default:
		// should not hanpen
		return false
	}

	// update whole log status and condition
	wholeStatusUpdate := updateBackupStatus(&backup.Status, copyStatusP)
	wholeConditionUpdate := v1alpha1.UpdateBackupCondition(&backup.Status, copyConditionP)

	return wholeStatusUpdate || wholeConditionUpdate
}

// updateLogSubCommandStatusOnly only updates log subcommand's status info.
func updateLogSubCommandStatusOnly(status *v1alpha1.LogSubCommandStatus, newStatus *BackupUpdateStatus) bool {
	isUpdate := false
	if newStatus == nil {
		return isUpdate
	}
	if newStatus.TimeStarted != nil && status.TimeStarted != *newStatus.TimeStarted {
		status.TimeStarted = *newStatus.TimeStarted
		isUpdate = true
	}
	if newStatus.TimeCompleted != nil && status.TimeCompleted != *newStatus.TimeCompleted {
		status.TimeCompleted = *newStatus.TimeCompleted
		isUpdate = true
	}
	if newStatus.LogTruncatingUntil != nil && status.LogTruncatingUntil != *newStatus.LogTruncatingUntil {
		status.LogTruncatingUntil = *newStatus.LogTruncatingUntil
		isUpdate = true
	}
	return isUpdate
}

// updateLogSubCommandConditionOnly only updates log subcommand's condition info.
func updateLogSubCommandConditionOnly(status *v1alpha1.LogSubCommandStatus, condition *v1alpha1.BackupCondition) bool {
	isUpdate := false
	if condition == nil {
		return isUpdate
	}

	if status.Phase != condition.Type {
		status.Phase = condition.Type
		isUpdate = true
	}

	var oldCondition *v1alpha1.BackupCondition
	// find old Backup condition.
	for _, c := range status.Conditions {
		if c.Type == condition.Type {
			oldCondition = &c
			break
		}
	}
	if oldCondition == nil {
		//add new Backup condition.
		oldCondition = &v1alpha1.BackupCondition{Type: condition.Type}
		status.Conditions = append(status.Conditions, *oldCondition)
		oldCondition = &status.Conditions[len(status.Conditions)-1]
		isUpdate = true
	}

	klog.Infof("status equal: %v, old %s, new %s", oldCondition.Status != condition.Status, oldCondition.Status, condition.Status)
	if oldCondition.Status != condition.Status {
		oldCondition.Status = condition.Status
		isUpdate = true
	}
	klog.Infof("after status update: %s", oldCondition.Status)
	klog.Infof("after status update: %s", status.Conditions[0].Status)

	if oldCondition.Reason != condition.Reason {
		oldCondition.Reason = condition.Reason
		isUpdate = true
	}
	if oldCondition.Message != condition.Message {
		oldCondition.Message = condition.Message
		isUpdate = true
	}
	if isUpdate {
		oldCondition.LastTransitionTime = metav1.Now()
	}
	// Return true if one of the fields have changed.
	return isUpdate
}

var _ BackupConditionUpdaterInterface = &realBackupConditionUpdater{}

// FakeBackupConditionUpdater is a fake BackupConditionUpdaterInterface
type FakeBackupConditionUpdater struct {
	BackupLister        listers.BackupLister
	BackupIndexer       cache.Indexer
	updateBackupTracker RequestTracker
}

// NewFakeBackupConditionUpdater returns a FakeBackupConditionUpdater
func NewFakeBackupConditionUpdater(backupInformer informers.BackupInformer) *FakeBackupConditionUpdater {
	return &FakeBackupConditionUpdater{
		backupInformer.Lister(),
		backupInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (c *FakeBackupConditionUpdater) SetUpdateBackupError(err error, after int) {
	c.updateBackupTracker.SetError(err).SetAfter(after)
}

// UpdateBackup updates the Backup
func (c *FakeBackupConditionUpdater) Update(backup *v1alpha1.Backup, _ *v1alpha1.BackupCondition, _ *BackupUpdateStatus) error {
	defer c.updateBackupTracker.Inc()
	if c.updateBackupTracker.ErrorReady() {
		defer c.updateBackupTracker.Reset()
		return c.updateBackupTracker.GetError()
	}

	return c.BackupIndexer.Update(backup)
}

var _ BackupConditionUpdaterInterface = &FakeBackupConditionUpdater{}
