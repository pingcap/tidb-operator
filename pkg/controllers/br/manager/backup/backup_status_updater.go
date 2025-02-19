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

package backup

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

// BackupConditionUpdaterInterface enables updating Backup conditions.
type BackupConditionUpdaterInterface interface {
	Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error
}

type realBackupConditionUpdater struct {
	cli      client.Client
	recorder record.EventRecorder
}

// returns a BackupConditionUpdaterInterface that updates the Status of a Backup,
func NewRealBackupConditionUpdater(
	cli client.Client,
	recorder record.EventRecorder,
) BackupConditionUpdaterInterface {
	return &realBackupConditionUpdater{
		cli,
		recorder,
	}
}

func (u *realBackupConditionUpdater) Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error {
	// TODO(ideascf): set reason for all
	if condition != nil {
		if condition.Reason == "" {
			condition.Reason = condition.Type
		}
	}

	ns := backup.GetNamespace()
	backupName := backup.GetName()
	// try best effort to guarantee backup is updated.
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		// Always get the latest backup before update.
		currentBackup := &v1alpha1.Backup{}
		if err := u.cli.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: backupName}, currentBackup); err == nil {
			// make a copy so we don't mutate the shared cache
			*backup = *(currentBackup.DeepCopy()) // TODO(ideascf): do we need to deep copy or just assign?
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %w", ns, backupName, err))
			return err
		}
		isUpdate := false
		// log backup needs update both subcommand status and whole backup status.
		if backup.Spec.Mode == v1alpha1.BackupModeLog {
			isUpdate = updateLogBackupStatus(backup, condition, newStatus)
		} else {
			isUpdate = updateSnapshotBackupStatus(backup, condition, newStatus)
		}
		if isUpdate {
			err := u.cli.Status().Update(context.TODO(), backup)
			if err == nil {
				klog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
				return nil
			}
			klog.Errorf("Failed to update backup [%s/%s], error: %v", ns, backupName, err)
			return err
		}
		return nil
	})
	return err
}

// nolint: gocyclo
// updateBackupStatus updates existing Backup status.
// from the fields in BackupUpdateStatus.
func updateBackupStatus(status *v1alpha1.BackupStatus, newStatus *BackupUpdateStatus) bool {
	if newStatus == nil {
		return false
	}
	isUpdate := false
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
		status.TimeTaken = status.TimeCompleted.Sub(status.TimeStarted.Time).Round(time.Second).String()
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
	if newStatus.ProgressStep != nil {
		progresses, updated := updateBRProgress(status.Progresses, newStatus.ProgressStep, newStatus.Progress, newStatus.ProgressUpdateTime)
		if updated {
			status.Progresses = progresses
			isUpdate = true
		}
	}

	if newStatus.RetryNum != nil || newStatus.RealRetryAt != nil {
		isUpdate = updateBackoffRetryStatus(status, newStatus)
	}

	return isUpdate
}

// updateSnapshotBackupStatus update snapshot mode backup status.
func updateSnapshotBackupStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) bool {
	var isStatusUpdate, isConditionUpdate bool
	isStatusUpdate = updateBackupStatus(&backup.Status, newStatus)
	isConditionUpdate = v1alpha1.UpdateBackupCondition(&backup.Status, &condition.Condition)
	return isStatusUpdate || isConditionUpdate
}

// updateLogBackupStatus update log backup status.
// it will update both the log backup sub command status and the whole log backup status.
func updateLogBackupStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) bool {
	// update whole backup status
	isWholeStatusUpdate := updateWholeLogBackupStatus(backup, condition, newStatus)
	// DeletionTimestamp is not nil when delete and clean backup, no subcommand status needs to be updated
	// LogCheckpointTs is not nil when just update checkpoint ts, no subcommand status needs to be updated
	if backup.DeletionTimestamp != nil || (newStatus != nil && newStatus.LogCheckpointTs != nil) {
		return isWholeStatusUpdate
	}
	// update subcommand status
	isSubCommandStatusUpdate := updateLogSubcommandStatus(backup, condition, newStatus)
	return isSubCommandStatusUpdate || isWholeStatusUpdate
}

// updateLogSubcommandStatus update log backup subcommand status.
func updateLogSubcommandStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) bool {
	// subcommand type should be set in condition, if not, will not update status info according to these condion and status.
	if condition == nil || condition.Command == "" {
		return false
	}

	// init log subcommand status map
	if backup.Status.LogSubCommandStatuses == nil {
		backup.Status.LogSubCommandStatuses = make(map[v1alpha1.LogSubCommandType]v1alpha1.LogSubCommandStatus)
	}
	// update subcommand status
	subStatus, exist := backup.Status.LogSubCommandStatuses[condition.Command]
	if !exist {
		// init subcommand status
		subStatus = v1alpha1.LogSubCommandStatus{
			Command:    condition.Command,
			Conditions: make([]v1alpha1.BackupCondition, 0),
		}
	}

	// update the status info and the condition info
	subcommandStatusUpdate := updateLogSubCommandStatusOnly(&subStatus, newStatus)
	subcomandConditionUpdate := updateLogSubCommandConditionOnly(&subStatus, condition)
	if subcommandStatusUpdate || subcomandConditionUpdate {
		// handle special case for pause and resume, if one is on condition, the other should be updated as repeatable
		if condition.Command == v1alpha1.LogPauseCommand {
			if subStatus, exist := backup.Status.LogSubCommandStatuses[v1alpha1.LogResumeCommand]; exist {
				subStatus.Phase = v1alpha1.BackupRepeatable
				backup.Status.LogSubCommandStatuses[v1alpha1.LogResumeCommand] = subStatus
			}
		}
		if condition.Command == v1alpha1.LogResumeCommand {
			if subStatus, exist := backup.Status.LogSubCommandStatuses[v1alpha1.LogPauseCommand]; exist {
				subStatus.Phase = v1alpha1.BackupRepeatable
				backup.Status.LogSubCommandStatuses[v1alpha1.LogPauseCommand] = subStatus
			}
		}

		backup.Status.LogSubCommandStatuses[condition.Command] = subStatus
		return true
	}
	return false
}

// nolint: gocyclo
// updateWholeLogBackupStatus updates the whole log backup status.
func updateWholeLogBackupStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, status *BackupUpdateStatus) bool {
	// call real update interface to update whole status
	doUpdateStatusAndCondition := func(newCondition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) bool {
		isStatusUpdate := updateBackupStatus(&backup.Status, newStatus)
		isConditionUpdate := v1alpha1.UpdateBackupCondition(&backup.Status, &newCondition.Condition)
		return isStatusUpdate || isConditionUpdate
	}

	// restruct update status info according to subcommand's detail info
	restructStatus := func() *BackupUpdateStatus {
		if status == nil {
			return nil
		}
		// copy status, avoid modifying the original value which may reuse in retry
		newStatus := *status
		switch condition.Command {
		case v1alpha1.LogStartCommand:
			// start command, complete condition, should not update TimeCompleted
			// other conditions, can be directly used
			if v1alpha1.BackupConditionType(condition.Type) == v1alpha1.BackupComplete {
				newStatus.TimeCompleted = nil
			}
			return &newStatus
		case v1alpha1.LogStopCommand:
			// stop command, commplete condition, should not update TimeStarted
			// other conditions, no need to be used to update whole status
			if v1alpha1.BackupConditionType(condition.Type) != v1alpha1.BackupComplete {
				return nil
			}
			newStatus.TimeStarted = nil
			return &newStatus
		case v1alpha1.LogTruncateCommand:
			// truncate command, complete condition, shoudld not update TimeStarted, TimeCompleted
			// other conditions, no need to be used to update whole status
			if v1alpha1.BackupConditionType(condition.Type) != v1alpha1.BackupComplete {
				return nil
			}
			newStatus.TimeCompleted = nil
			newStatus.TimeStarted = nil
			return &newStatus
		case v1alpha1.LogPauseCommand:
			// pause command, complete condition, should not update TimeStarted, TimeCompleted
			// other conditions, no need to be used to update whole status
			if v1alpha1.BackupConditionType(condition.Type) != v1alpha1.BackupComplete {
				return nil
			}
			newStatus.TimeCompleted = nil
			newStatus.TimeStarted = nil
			return &newStatus
		case v1alpha1.LogResumeCommand:
			// resume command, complete condition, should not update TimeCompleted, TimeStarted
			// other conditions, no need to be used to update whole status
			if v1alpha1.BackupConditionType(condition.Type) != v1alpha1.BackupComplete {
				return nil
			}
			newStatus.TimeCompleted = nil
			newStatus.TimeStarted = nil
			return &newStatus
		default:
			// should not hanpen
			return nil
		}
	}

	// restruct update condition info according to subcommand's detail info
	restructCondition := func() *v1alpha1.BackupCondition {
		if condition == nil {
			return nil
		}
		newCondition := *condition
		newCondition.Command = ""
		switch condition.Command {
		case v1alpha1.LogStartCommand:
			// start command, complete condition, should not update condition
			// other conditions, can be directly used.
			if v1alpha1.BackupConditionType(condition.Type) == v1alpha1.BackupComplete {
				return nil
			}
			return &newCondition
		case v1alpha1.LogStopCommand:
			// stop command, complete condition, should be updated as stopped
			// other conditions, no need to be used to update whole condition
			if v1alpha1.BackupConditionType(condition.Type) == v1alpha1.BackupComplete {
				newCondition.Type = string(v1alpha1.BackupStopped)
				return &newCondition
			}
			return nil
		case v1alpha1.LogPauseCommand:
			// pause command, complete condition, should be updated as paused
			// other conditions, no need to be used to update whole condition
			if v1alpha1.BackupConditionType(condition.Type) == v1alpha1.BackupComplete {
				newCondition.Type = string(v1alpha1.BackupPaused)
				return &newCondition
			}
			return nil
		case v1alpha1.LogResumeCommand:
			// resume command, complete condition, should be updated as resumed
			// other conditions, no need to be used to update whole condition
			if v1alpha1.BackupConditionType(condition.Type) == v1alpha1.BackupComplete {
				newCondition.Type = string(v1alpha1.BackupRunning)
				return &newCondition
			}
			return nil
		default:
			// truncate command or other, all conditions, no need to be used to update whole condition.
			return nil
		}
	}

	// DeletionTimestamp is not nil when delete and clean backup, condition and status can be directly used to update whole status.
	if backup.DeletionTimestamp != nil {
		return doUpdateStatusAndCondition(condition, status)
	}

	// just update checkpoint ts
	if status != nil && status.LogCheckpointTs != nil {
		return doUpdateStatusAndCondition(nil, status)
	}

	// subcommand type should be set in condition, if not, will not update status info according to these condion and status.
	if condition == nil || condition.Command == "" {
		return false
	}

	// restruct status and condition according to subcommand's detail info, and they wiil be used to update whole log backup status.
	newStatus := restructStatus()
	newCondition := restructCondition()
	return doUpdateStatusAndCondition(newCondition, newStatus)
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

	if status.Phase != v1alpha1.BackupConditionType(condition.Type) {
		status.Phase = v1alpha1.BackupConditionType(condition.Type)
		isUpdate = true
	}

	var oldCondition *v1alpha1.BackupCondition
	// find old Backup condition
	for i, c := range status.Conditions {
		if c.Type == condition.Type {
			oldCondition = &status.Conditions[i]
			break
		}
	}
	// no old condition, make a new one
	if oldCondition == nil {
		oldCondition = &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:               condition.Type,
				Status:             condition.Status,
				Reason:             condition.Reason,
				Message:            condition.Message,
				LastTransitionTime: metav1.Now(),
			},
		}
		status.Conditions = append(status.Conditions, *oldCondition)
		return true
	}

	if oldCondition.Status != condition.Status {
		oldCondition.Status = condition.Status
		isUpdate = true
	}

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

// updateBRProgress updates progress for backup/restore.
func updateBRProgress(progresses []v1alpha1.Progress, step *string, progress *int, updateTime *metav1.Time) ([]v1alpha1.Progress, bool) {
	var oldProgress *v1alpha1.Progress
	for i, p := range progresses {
		if p.Step == *step {
			oldProgress = &progresses[i]
			break
		}
	}

	makeSureLastProgressOver := func() {
		size := len(progresses)
		if size == 0 || progresses[size-1].Progress >= 100 {
			return
		}
		progresses[size-1].Progress = 100
		progresses[size-1].LastTransitionTime = metav1.Time{Time: time.Now()}
	}

	// no such progress, will new
	if oldProgress == nil {
		makeSureLastProgressOver()
		progresses = append(progresses, v1alpha1.Progress{
			Step:               *step,
			Progress:           *progress,
			LastTransitionTime: *updateTime,
		})
		return progresses, true
	}

	isUpdate := false
	if oldProgress.Progress < *progress {
		oldProgress.Progress = *progress
		isUpdate = true
	}

	if oldProgress.LastTransitionTime != *updateTime {
		oldProgress.LastTransitionTime = *updateTime
		isUpdate = true
	}

	return progresses, isUpdate
}

// nolint: gocyclo
func updateBackoffRetryStatus(status *v1alpha1.BackupStatus, newStatus *BackupUpdateStatus) bool {
	isUpdate := false
	currentRecord := getCurrentBackoffRetryRecord(status, newStatus)

	// no record which is newStatus want to modify in current backup status, we need create a new record
	if currentRecord == nil {
		// no records in BackoffRetryStatus, make record array first
		if len(status.BackoffRetryStatus) == 0 {
			status.BackoffRetryStatus = make([]v1alpha1.BackoffRetryRecord, 0)
		}
		// create a new record
		status.BackoffRetryStatus = append(status.BackoffRetryStatus, v1alpha1.BackoffRetryRecord{})
		currentRecord = &status.BackoffRetryStatus[len(status.BackoffRetryStatus)-1]
		isUpdate = true
	}

	if newStatus.RetryNum != nil && *newStatus.RetryNum != currentRecord.RetryNum {
		currentRecord.RetryNum = *newStatus.RetryNum
		isUpdate = true
	}

	if newStatus.DetectFailedAt != nil && (currentRecord.DetectFailedAt == nil || *newStatus.DetectFailedAt != *currentRecord.DetectFailedAt) {
		currentRecord.DetectFailedAt = newStatus.DetectFailedAt
		isUpdate = true
	}

	if newStatus.ExpectedRetryAt != nil && (currentRecord.ExpectedRetryAt == nil || *newStatus.ExpectedRetryAt != *currentRecord.ExpectedRetryAt) {
		currentRecord.ExpectedRetryAt = newStatus.ExpectedRetryAt
		isUpdate = true
	}

	if newStatus.RealRetryAt != nil && (currentRecord.RealRetryAt == nil || *newStatus.RealRetryAt != *currentRecord.RealRetryAt) {
		currentRecord.RealRetryAt = newStatus.RealRetryAt
		isUpdate = true
	}

	if newStatus.RetryReason != nil && *newStatus.RetryReason != currentRecord.RetryReason {
		currentRecord.RetryReason = *newStatus.RetryReason
		isUpdate = true
	}

	if newStatus.OriginalReason != nil && *newStatus.OriginalReason != currentRecord.OriginalReason {
		currentRecord.OriginalReason = *newStatus.OriginalReason
		isUpdate = true
	}

	return isUpdate
}

func getCurrentBackoffRetryRecord(status *v1alpha1.BackupStatus, newStatus *BackupUpdateStatus) *v1alpha1.BackoffRetryRecord {
	// no record
	if len(status.BackoffRetryStatus) == 0 {
		return nil
	}
	// no RetryNum, means modify latest record
	if newStatus.RetryNum == nil {
		return &status.BackoffRetryStatus[len(status.BackoffRetryStatus)-1]
	}

	if *newStatus.RetryNum <= len(status.BackoffRetryStatus) {
		return &status.BackoffRetryStatus[*newStatus.RetryNum-1]
	}
	return nil
}

var _ BackupConditionUpdaterInterface = &realBackupConditionUpdater{}

/*
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
*/
