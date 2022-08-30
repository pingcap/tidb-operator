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

package v1alpha1

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	DefaultBatchDeleteOption = BatchDeleteOption{
		BatchConcurrency:   10,
		RoutineConcurrency: 100,
	}

	// defaultCleanOption is default clean option
	defaultCleanOption = CleanOption{
		PageSize:          10000,
		RetryCount:        5,
		BatchDeleteOption: DefaultBatchDeleteOption,
	}
)

// GetCleanJobName return the clean job name
func (bk *Backup) GetCleanJobName() string {
	return fmt.Sprintf("clean-%s", bk.GetName())
}

// GetBackupJobName return the backup job name
func (bk *Backup) GetBackupJobName() string {
	if command := ParseLogBackupSubcommand(bk); command != "" {
		return fmt.Sprintf("backup-%s-%s", bk.GetName(), command)
	}
	return fmt.Sprintf("backup-%s", bk.GetName())
}

// GetAllLogBackupJobName return the all log backup job name
func (bk *Backup) GetAllLogBackupJobName() []string {
	return []string{
		fmt.Sprintf("backup-%s-%s", bk.GetName(), LogStartCommand),
		fmt.Sprintf("backup-%s-%s", bk.GetName(), LogStopCommand),
		fmt.Sprintf("backup-%s-%s", bk.GetName(), LogTruncateCommand),
	}
}

// GetTidbEndpointHash return the hash string base on tidb cluster's host and port
func (bk *Backup) GetTidbEndpointHash() string {
	return HashContents([]byte(bk.Spec.From.GetTidbEndpoint()))
}

// GetBackupPVCName return the backup pvc name
func (bk *Backup) GetBackupPVCName() string {
	return fmt.Sprintf("backup-pvc-%s", bk.GetTidbEndpointHash())
}

// GetInstanceName return the backup instance name
func (bk *Backup) GetInstanceName() string {
	if bk.Labels != nil {
		if v, ok := bk.Labels[label.InstanceLabelKey]; ok {
			return v
		}
	}
	return bk.Name
}

// GetCleanOption return the clean option
func (bk *Backup) GetCleanOption() CleanOption {
	if bk.Spec.CleanOption == nil {
		return defaultCleanOption
	}

	ropt := *bk.Spec.CleanOption
	if ropt.PageSize <= 0 {
		ropt.PageSize = defaultCleanOption.PageSize
	}
	if ropt.RetryCount <= 0 {
		ropt.RetryCount = defaultCleanOption.RetryCount
	}
	if ropt.BatchConcurrency <= 0 {
		ropt.BatchConcurrency = defaultCleanOption.BatchConcurrency
	}
	if ropt.RoutineConcurrency <= 0 {
		ropt.RoutineConcurrency = defaultCleanOption.RoutineConcurrency
	}

	return ropt
}

// GetBackupCondition get the specify type's BackupCondition from the given BackupStatus
func GetBackupCondition(status *BackupStatus, conditionType BackupConditionType) (int, *BackupCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// UpdateBackupCondition updates existing Backup condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if Backup condition has changed or has been added.
func UpdateBackupCondition(status *BackupStatus, condition *BackupCondition) bool {
	if condition == nil {
		return false
	}
	condition.LastTransitionTime = metav1.Now()
	// Try to find this Backup condition.
	conditionIndex, oldCondition := GetBackupCondition(status, condition.Type)

	status.Phase = condition.Type

	if oldCondition == nil {
		// We are adding new Backup condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isUpdate := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isUpdate
}

// IsBackupComplete returns true if a Backup has successfully completed
func IsBackupComplete(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupComplete)
	return condition != nil && condition.Status == corev1.ConditionTrue
	// isComplete := condition != nil && condition.Status == corev1.ConditionTrue
	// if isComplete && backup.Spec.Mode == BackupModeLog {
	// 	// stop complete when set stopped
	// 	if backup.Status.LogStopped {
	// 		return true
	// 	} else {
	// 		if !backup.Spec.LogStop {
	// 			// truncate complete when Spec.LogTruncateUntil equals Status.LogTruncateUntil
	// 			return backup.Spec.LogTruncateUntil == backup.Status.LogTruncateUntil
	// 		} else {
	// 			return false
	// 		}
	// 	}
	// }
	// return isComplete
}

// IsBackupInvalid returns true if a Backup has invalid condition set
func IsBackupInvalid(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupInvalid)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsBackupFailed returns true if a Backup has failed
func IsBackupFailed(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupFailed)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsBackupScheduled returns true if a Backup has successfully scheduled
func IsBackupScheduled(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupScheduled)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsBackupRunning returns true if a Backup is Running.
func IsBackupRunning(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupRunning)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsBackupPrepared returns true if a Backup is Prepare.
func IsBackupPrepared(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupPrepare)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsBackupClean returns true if a Backup has been successfully cleaned up
func IsBackupClean(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupClean)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsCleanCandidate returns true if a Backup should be added to clean candidate according to cleanPolicy
func IsCleanCandidate(backup *Backup) bool {
	switch backup.Spec.CleanPolicy {
	case CleanPolicyTypeDelete, CleanPolicyTypeOnFailure:
		return true
	default:
		return false
	}
}

// NeedNotClean returns true if a Backup need not to be cleaned up according to cleanPolicy
func NeedNotClean(backup *Backup) bool {
	return backup.Spec.CleanPolicy == CleanPolicyTypeOnFailure && !IsBackupFailed(backup)
}

// // IsBackupFailed returns true if a Backup has failed
// func IsLogBackupStartComplete(backup *Backup) bool {
// 	_, condition := GetBackupCondition(&backup.Status, LogBackupStartComplete)
// 	return condition != nil && condition.Status == corev1.ConditionTrue
// }

// // IsBackupFailed returns true if a Backup has failed
// func IsLogBackupTruncateComplete(backup *Backup) bool {
// 	_, condition := GetBackupCondition(&backup.Status, LogBackupTruncateComplete)
// 	return condition != nil && condition.Status == corev1.ConditionTrue
// }

// // IsBackupClean returns true if a Backup has been successfully cleaned up
// func IsBackupHandling(backup *Backup) bool {
// 	_, condition := GetBackupCondition(&backup.Status, BackupHandling)
// 	return condition != nil && condition.Status == corev1.ConditionTrue
// }

// ParseLogBackupSubCommand parse the log backup subcommand from cr.
func ParseLogBackupSubcommand(backup *Backup) LogSubCommandType {
	if backup.Spec.Mode != BackupModeLog {
		return ""
	}
	if backup.Spec.LogStop {
		return LogStopCommand
	}
	if backup.Spec.LogTruncateUntil != "" {
		return LogTruncateCommand
	}
	return LogStartCommand
}

// IsLogBackupSubCommandComplete return whether the log backup's subcommand is complete.
func IsLogBackupSubCommandComplete(backup *Backup) bool {
	command := ParseLogBackupSubcommand(backup)
	switch command {
	case LogStartCommand:
		// CommitTs is Recorded means start is already complete
		return backup.Status.CommitTs != ""
	case LogTruncateCommand:
		// Spec.LogTruncateUntil <= Status.CommitTs or Status.LogTruncateUntil
		var (
			err                               error
			newTsUint, oldTsUnit, startTsUnit uint64
		)
		newTsUint, err = config.ParseTSString(backup.Spec.LogTruncateUntil)
		// format err should be handled by sync
		if err != nil {
			return false
		}
		oldTsUnit, _ = config.ParseTSString(backup.Status.LogTruncateUntil)
		startTsUnit, _ = config.ParseTSString(backup.Status.CommitTs)

		return newTsUint <= startTsUnit || newTsUint <= oldTsUnit
	case LogStopCommand:
		return backup.Status.Phase == BackupComplete
	default:
		return true
	}
}

// IsLogBackupSubCommandRunning return whether the log backup's subcommand is running.
func IsLogBackupSubCommandRunning(backup *Backup) bool {
	command := ParseLogBackupSubcommand(backup)
	switch command {
	case LogStartCommand, LogStopCommand:
		// CommitTs is Recorded means start is already complete
		if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
			return subStatus.Phase == BackupScheduled || subStatus.Phase == BackupPrepare || subStatus.Phase == BackupRunning
		}
	case LogTruncateCommand:
		// Spec.LogTruncateUntil <= Status.CommitTs or Status.LogTruncateUntil
		if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
			return subStatus.LogTruncateUntil == backup.Spec.LogTruncateUntil &&
				(subStatus.Phase == BackupScheduled || subStatus.Phase == BackupPrepare || subStatus.Phase == BackupRunning)
		}
	default:
		return false
	}
	return false
}

// IsLogBackupSubCommandFailed return whether the log backup's subcommand is failed.
func IsLogBackupSubCommandFailed(backup *Backup) bool {
	command := ParseLogBackupSubcommand(backup)
	switch command {
	case LogStartCommand, LogStopCommand:
		// CommitTs is Recorded means start is already complete
		if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
			return subStatus.Phase == BackupFailed
		}
	case LogTruncateCommand:
		// Spec.LogTruncateUntil <= Status.CommitTs or Status.LogTruncateUntil
		if subStatus, ok := backup.Status.LogSubCommandStatuses[LogTruncateCommand]; ok {
			return subStatus.LogTruncateUntil == backup.Spec.LogTruncateUntil && subStatus.Phase == BackupFailed
		}
	default:
		return false
	}
	return false
}

// IsLogBackupSubCommandInvalid return whether the log backup's subcommand is invalid.
func IsLogBackupSubCommandInvalid(backup *Backup) bool {
	command := ParseLogBackupSubcommand(backup)
	switch command {
	case LogStartCommand, LogStopCommand:
		// CommitTs is Recorded means start is already complete
		if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
			return subStatus.Phase == BackupInvalid
		}
	case LogTruncateCommand:
		// Spec.LogTruncateUntil <= Status.CommitTs or Status.LogTruncateUntil
		if subStatus, ok := backup.Status.LogSubCommandStatuses[LogTruncateCommand]; ok {
			return subStatus.LogTruncateUntil == backup.Spec.LogTruncateUntil && subStatus.Phase == BackupInvalid
		}
	default:
		return false
	}
	return false
}

func isInLogTruncate(new, start, old, running string) bool {
	var (
		err                                              error
		newTsUint, oldTsUnit, runningTsUnit, startTsUnit uint64
	)
	newTsUint, err = config.ParseTSString(new)
	// format err should be handled by sync
	if err != nil {
		return false
	}
	oldTsUnit, _ = config.ParseTSString(old)
	runningTsUnit, _ = config.ParseTSString(running)
	startTsUnit, _ = config.ParseTSString(start)
	return newTsUint > startTsUnit && newTsUint > oldTsUnit && newTsUint <= runningTsUnit
}

func GetLogSumcommandConditionInfo(backup *Backup) (reason, message string) {
	command := ParseLogBackupSubcommand(backup)
	if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
		reason = subStatus.Conditions[0].Reason
		message = subStatus.Conditions[0].Message
	}
	return
}

func CanLogSubcommandRun(backup *Backup) bool {
	command := ParseLogBackupSubcommand(backup)
	switch command {
	case LogStartCommand:
		// start can directly run
		return true
	case LogTruncateCommand, LogStopCommand:
		// truncate and stop should wait start complete
		return backup.Status.CommitTs != ""
	default:
		return false
	}
}

// CanLogBackSumcommandFailedRetry return whether log backup can retry, it can be used after make sure failed.
func CanLogBackSumcommandFailedRetry(backup *Backup) bool {
	// now just retry
	return true
	// command := ParseLogBackupSubcommand(backup)
	// subStatus, ok := backup.Status.LogSubCommandStatuses[command]
	// if !ok {
	// 	// should not happen.
	// 	return false
	// }
	// switch command {
	// case LogStartCommand:
	// 	// start can retry
	// 	return true
	// case LogTruncateCommand:
	// 	// Spec.LogTruncateUntil <= Status.CommitTs or Status.LogTruncateUntil
	// 	if condition, ok := backup.Status.LogSubCommandStatuses[LogTruncateCommand]; ok {
	// 		return condition.LogTruncateUntil == backup.Spec.LogTruncateUntil && condition.Phase == BackupInvalid
	// 	}
	// default:
	// 	return false
	// }

}

// func CanLogBackupSubCommandRun(backup *Backup) bool {
// 	if IsLogBackupSubCommandComplete(backup) {
// 		return false
// 	}
// 	if IsLogBackupSubCommandFailed(backup) {
// 		return false
// 	}
// 	if IsBackupRunning(backup) {
// 		return false
// 	}
// 	command := ParseLogBackupSubcommand(backup)
// 	switch command {
// 	case LogStartCommand:
// 		// start command need no precondition
// 		return true
// 	case LogTruncateCommand, LogStopCommand:
// 		// need start success
// 		return backup.Status.CommitTs == ""
// 	default:
// 		return false
// 	}
// }

// // 需要互斥
// func CanLogBackupSubCommandRequeue(backup *Backup) bool {
// 	command := ParseLogBackupSubcommand(backup)
// 	if command == "" {
// 		return false
// 	}
// 	isFailed := false
// 	// start failed, commands are failed.
// 	if condition, ok := backup.Status.LogSubCommandStatuses[LogStartCommand]; ok {
// 		if condition.Phase == BackupFailed {
// 			return true
// 		}
// 	}

// 	switch command {
// 	case LogStartCommand:
// 		return false
// 	case LogTruncateCommand:
// 		// Spec.LogTruncateUntil <= Status.CommitTs or Status.LogTruncateUntil
// 		if condition, ok := backup.Status.LogSubCommandStatuses[LogTruncateCommand]; ok {
// 			return condition.LogTruncateUntil == backup.Spec.LogTruncateUntil &&
// 				(condition.Phase == BackupScheduled || condition.Phase == BackupPrepare || condition.Phase == BackupRunning)
// 		}
// 	default:
// 		return false
// 	}
// 	return false

// 	command := ParseLogBackupSubcommand(backup)
// 	isFailed := false
// 	// start --> start/truncate/stop failed --> failed
// 	// truncate --> start failed --> failed
// 	//              truncate failed --> 是这次 --> failed
// 	//              stop failed --> 不影响

// 	if backup.Status.LogSubCommandConditions.Command == command {
// 		phase := backup.Status.LogSubCommandCondition.Phase
// 		isFailed = phase == BackupFailed
// 	}
// 	if isFailed && command == LogTruncateCommand {
// 		return backup.Spec.LogTruncateUntil == backup.Status.LogSubCommandCondition.LogTruncateUntil
// 	}

// 	return isRunning

// }
