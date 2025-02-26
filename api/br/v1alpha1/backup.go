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

package v1alpha1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
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

// GetCleanJobName return the clean job name for log backup
func (bk *Backup) GetStopLogBackupJobName() string {
	return fmt.Sprintf("stop-%s", bk.GetName())
}

// GetBackupJobName return the backup job name
func (bk *Backup) GetBackupJobName() string {
	if command := ParseLogBackupSubcommand(bk); command != "" {
		return fmt.Sprintf("backup-%s-%s", bk.GetName(), command)
	}
	return fmt.Sprintf("backup-%s", bk.GetName())
}

func (bk *Backup) GetVolumeBackupInitializeJobName() string {
	backupJobName := bk.GetBackupJobName()
	return fmt.Sprintf("%s-init", backupJobName)
}

// GetAllLogBackupJobName return the all log backup job name
func (bk *Backup) GetAllLogBackupJobName() []string {
	return []string{
		fmt.Sprintf("backup-%s-%s", bk.GetName(), LogStartCommand),
		fmt.Sprintf("backup-%s-%s", bk.GetName(), LogStopCommand),
		fmt.Sprintf("backup-%s-%s", bk.GetName(), LogTruncateCommand),
	}
}

// GetInstanceName return the backup instance name
func (bk *Backup) GetInstanceName() string {
	if bk.Labels != nil {
		if v, ok := bk.Labels[metav1alpha1.InstanceLabelKey]; ok {
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

var (
	// BackupControllerKind contains the schema.GroupVersionKind for backup controller type.
	BackupControllerKind         = SchemeGroupVersion.WithKind("Backup")
	backupScheduleControllerKind = SchemeGroupVersion.WithKind("BackupSchedule")
	RestoreControllerKind        = SchemeGroupVersion.WithKind("Restore")
)

// GetBackupOwnerRef returns Backup's OwnerReference
func GetBackupOwnerRef(backup *Backup) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         BackupControllerKind.GroupVersion().String(),
		Kind:               BackupControllerKind.Kind,
		Name:               backup.GetName(),
		UID:                backup.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetBackupCondition get the specify type's BackupCondition from the given BackupStatus
func GetBackupCondition(status *BackupStatus, conditionType BackupConditionType) (int, *metav1.Condition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == string(conditionType) {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// GetRestoreOwnerRef returns Restore's OwnerReference
func GetRestoreOwnerRef(restore *Restore) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         RestoreControllerKind.GroupVersion().String(),
		Kind:               RestoreControllerKind.Kind,
		Name:               restore.GetName(),
		UID:                restore.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetBackupScheduleOwnerRef returns BackupSchedule's OwnerReference
func GetBackupScheduleOwnerRef(bs *BackupSchedule) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         backupScheduleControllerKind.GroupVersion().String(),
		Kind:               backupScheduleControllerKind.Kind,
		Name:               bs.GetName(),
		UID:                bs.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// UpdateBackupCondition updates existing Backup condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if Backup condition has changed or has been added.
func UpdateBackupCondition(status *BackupStatus, condition *metav1.Condition) bool {
	if condition == nil {
		return false
	}
	tp := BackupConditionType(condition.Type)
	condition.LastTransitionTime = metav1.Now()
	// Try to find this Backup condition.
	conditionIndex, oldCondition := GetBackupCondition(status, tp)

	isDiffPhase := status.Phase != tp

	// restart condition no need to update to phase
	if isDiffPhase && tp != BackupRestart {
		status.Phase = tp
	}

	if oldCondition == nil {
		// We are adding new Backup condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}

	// if phase is diff, we need update condition
	if isDiffPhase {
		status.Conditions[conditionIndex] = *condition
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
	if backup.Spec.Mode == BackupModeLog {
		return IsLogBackupSubCommandOntheCondition(backup, BackupComplete)
	}
	_, condition := GetBackupCondition(&backup.Status, BackupComplete)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsBackupInvalid returns true if a Backup has invalid condition set
func IsBackupInvalid(backup *Backup) bool {
	if backup.Spec.Mode == BackupModeLog {
		return IsLogBackupSubCommandOntheCondition(backup, BackupInvalid)
	}
	_, condition := GetBackupCondition(&backup.Status, BackupInvalid)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsBackupFailed returns true if a Backup has failed
func IsBackupFailed(backup *Backup) bool {
	if backup.Spec.Mode == BackupModeLog {
		return IsLogBackupSubCommandOntheCondition(backup, BackupFailed)
	}
	_, condition := GetBackupCondition(&backup.Status, BackupFailed)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsBackupScheduled returns true if a Backup has successfully scheduled
func IsBackupScheduled(backup *Backup) bool {
	if backup.Spec.Mode == BackupModeLog {
		return IsLogBackupSubCommandOntheCondition(backup, BackupScheduled)
	}
	_, condition := GetBackupCondition(&backup.Status, BackupScheduled)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// HaveTruncateUntil returns true if a Backup has truncate until set
func HaveTruncateUntil(backup *Backup) bool {
	if backup.Spec.Mode != BackupModeLog {
		return false
	}
	return backup.Spec.LogTruncateUntil != ""
}

// IsBackupRunning returns true if a Backup is Running.
func IsBackupRunning(backup *Backup) bool {
	if backup.Spec.Mode == BackupModeLog {
		return IsLogBackupSubCommandOntheCondition(backup, BackupRunning)
	}
	_, condition := GetBackupCondition(&backup.Status, BackupRunning)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsBackupRestart returns true if a Backup was restarted.
func IsBackupRestart(backup *Backup) bool {
	_, hasRestartCondition := GetBackupCondition(&backup.Status, BackupRestart)
	return hasRestartCondition != nil
}

// IsBackupPrepared returns true if a Backup is Prepare.
func IsBackupPrepared(backup *Backup) bool {
	if backup.Spec.Mode == BackupModeLog {
		return IsLogBackupSubCommandOntheCondition(backup, BackupPrepare)
	}
	_, condition := GetBackupCondition(&backup.Status, BackupPrepare)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsBackupClean returns true if a Backup has been successfully cleaned up
func IsBackupClean(backup *Backup) bool {
	// TODO: now we don't handle fault state, maybe we should consider it in the future
	if backup.Spec.Mode == BackupModeLog && IsLogBackupOnTrack(backup) {
		return false
	}
	if NeedRetainData(backup) {
		return true
	}
	_, condition := GetBackupCondition(&backup.Status, BackupClean)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsBackupCleanFailed returns true if a Backup has failed to clean up
func IsBackupCleanFailed(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupCleanFailed)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsCleanCandidate returns true if a Backup should be added to clean candidate according to cleanPolicy
func IsCleanCandidate(backup *Backup) bool {
	return backup.Spec.Mode == BackupModeLog ||
		backup.Spec.CleanPolicy == CleanPolicyTypeDelete ||
		backup.Spec.CleanPolicy == CleanPolicyTypeOnFailure
}

// NeedRetainData returns true if a Backup need not to be cleaned up according to cleanPolicy
func NeedRetainData(backup *Backup) bool {
	return backup.Spec.CleanPolicy == CleanPolicyTypeRetain ||
		(backup.Spec.CleanPolicy == CleanPolicyTypeOnFailure && !IsBackupFailed(backup))
}

// ParseLogBackupSubcommand parse the log backup subcommand from cr.
func ParseLogBackupSubcommand(backup *Backup) LogSubCommandType {
	if backup.Spec.Mode != BackupModeLog {
		return ""
	}

	var subCommand LogSubCommandType

	switch backup.Spec.LogSubcommand {
	// Users can omit the LogSubcommand field and use the `LogStop` field to stop log backups as in older version.
	case "":
		if backup.Spec.LogStop || IsLogBackupAlreadyStop(backup) {
			subCommand = LogStopCommand
		} else {
			subCommand = LogStartCommand
		}
	case LogStartCommand:
		if IsLogBackupAlreadyPaused(backup) {
			subCommand = LogResumeCommand
		} else {
			subCommand = LogStartCommand
		}
	case LogStopCommand:
		subCommand = LogStopCommand
	case LogPauseCommand:
		subCommand = LogPauseCommand
	default:
		return LogUnknownCommand
	}

	// If the selected subcommand is already sync and logTruncateUntil is set, switch to LogTruncateCommand
	if IsLogSubcommandAlreadySync(backup, subCommand) && backup.Spec.LogTruncateUntil != "" && backup.Spec.LogTruncateUntil != backup.Status.LogSuccessTruncateUntil {
		return LogTruncateCommand
	}

	return subCommand
}

// IsLogSubcommandAlreadySync return whether the log subcommand already sync.
// It only check start/stop/pause subcommand. Truncate subcommand need to check the `logTruncateUntil` separately.
func IsLogSubcommandAlreadySync(backup *Backup, subCommand LogSubCommandType) bool {
	switch subCommand {
	case LogStartCommand:
		return IsLogBackupAlreadyStart(backup)
	case LogStopCommand:
		return IsLogBackupAlreadyStop(backup)
	case LogPauseCommand:
		return IsLogBackupAlreadyPaused(backup)
	case LogResumeCommand:
		return IsLogBackupAlreadyRunning(backup)
	default:
		return false
	}
}

// IsLogBackupSubCommandOntheCondition return whether the log subcommand on the condition.
func IsLogBackupSubCommandOntheCondition(backup *Backup, conditionType BackupConditionType) bool {
	command := ParseLogBackupSubcommand(backup)
	switch command {
	case LogStartCommand, LogStopCommand, LogPauseCommand, LogResumeCommand:
		if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
			return subStatus.Phase == conditionType
		}
	case LogTruncateCommand:
		// truncate Command's truncating until is the spec truncate until means the truncate is in progress.
		if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
			return subStatus.LogTruncatingUntil == backup.Spec.LogTruncateUntil && subStatus.Phase == conditionType
		}
	default:
		return false
	}
	return false
}

// GetLogSubcommandConditionInfo gets log subcommand current phase's reason and message
func GetLogSubcommandConditionInfo(backup *Backup) (reason, message string) {
	command := ParseLogBackupSubcommand(backup)
	if subStatus, ok := backup.Status.LogSubCommandStatuses[command]; ok {
		for _, condition := range subStatus.Conditions {
			if string(subStatus.Phase) == string(condition.Type) {
				reason = condition.Reason
				message = condition.Message
				break
			}
		}
	}
	return
}

// IsLogBackupAlreadyStart return whether log backup has already started.
func IsLogBackupAlreadyStart(backup *Backup) bool {
	return backup.Spec.Mode == BackupModeLog && backup.Status.CommitTs != ""
}

// IsLogBackupAlreadyTruncate return whether log backup has already truncated.
func IsLogBackupAlreadyTruncate(backup *Backup) bool {
	if backup.Spec.Mode != BackupModeLog {
		return false
	}
	// spec truncate Until TS <= start commit TS or success truncate until means log backup has been truncated.
	var specTS, successedTS, startCommitTS uint64
	var err error

	specTS, err = ParseTSString(backup.Spec.LogTruncateUntil)
	if err != nil {
		return false
	}
	successedTS, _ = ParseTSString(backup.Status.LogSuccessTruncateUntil)
	startCommitTS, _ = ParseTSString(backup.Status.CommitTs)

	return specTS <= startCommitTS || specTS <= successedTS
}

// IsLogBackupAlreadyStop return whether log backup has already stopped.
func IsLogBackupAlreadyStop(backup *Backup) bool {
	return backup.Spec.Mode == BackupModeLog && backup.Status.Phase == BackupStopped
}

// IsLogBackupOnTrack returns whether log backup is on track.
func IsLogBackupOnTrack(backup *Backup) bool {
	if backup.Spec.Mode != BackupModeLog {
		return false
	}

	switch backup.Status.Phase {
	case BackupScheduled, BackupPrepare, BackupRunning, BackupPaused:
		return true
	default:
		return false
	}
}

// IsLogBackupAlreadyPaused return whether log backup has already paused.
func IsLogBackupAlreadyPaused(backup *Backup) bool {
	return backup.Spec.Mode == BackupModeLog && backup.Status.Phase == BackupPaused
}

// IsLogBackupAlreadyRunning return whether log backup has already resumed.
func IsLogBackupAlreadyRunning(backup *Backup) bool {
	return backup.Spec.Mode == BackupModeLog && backup.Status.Phase == BackupRunning
}

func (r BackoffRetryRecord) IsTriggered() bool {
	return r.RealRetryAt != nil
}

func (b *Backup) LastRetryRecord() (BackoffRetryRecord, bool) {
	if len(b.Status.BackoffRetryStatus) == 0 {
		return BackoffRetryRecord{}, false
	}
	return b.Status.BackoffRetryStatus[len(b.Status.BackoffRetryStatus)-1], true
}

// TODO(ideascf): extract this to a common function
// Retry creates and records the next retry time and reason, the real retry will happen at record.ExpectedRetryAt
func (b *Backup) Retry(reason, originalReason string, now time.Time) (nextRunAt *time.Time, _ error) {
	lastRetry, ok := b.LastRetryRecord()
	if ok && lastRetry.RealRetryAt != nil {
		return nil, fmt.Errorf("retry num %d already triggered", lastRetry.RetryNum)
	}

	// check whether the retry is exceed limit
	exceedReason, err := isBackoffRetryExeedLimit(&b.Spec.BackoffRetryPolicy, b.Status.BackoffRetryStatus, now)
	if err != nil {
		return nil, fmt.Errorf("check whether the retry is exceed limit for backup %s/%s: %s", b.Namespace, b.Name, err)
	}
	if exceedReason != "" {
		UpdateBackupCondition(&b.Status, &metav1.Condition{
			Type:    string(BackupFailed),
			Status:  metav1.ConditionTrue,
			Reason:  "ExceedRetryLimit",
			Message: exceedReason,
		})
		return nil, nil
	}

	// calculate the next retry time
	retryNum := len(b.Status.BackoffRetryStatus) + 1
	nextRetryTime, err := calcNextRetryTime(&b.Spec.BackoffRetryPolicy, b.Status.BackoffRetryStatus, retryNum, now)
	if err != nil {
		return nil, fmt.Errorf("calculate the next retry time for backup %s/%s: %s", b.Namespace, b.Name, err)
	}

	// record the next retry record
	record := BackoffRetryRecord{
		RetryNum:        retryNum,
		DetectFailedAt:  &metav1.Time{Time: now},
		ExpectedRetryAt: &nextRetryTime,
		RetryReason:     reason,
		OriginalReason:  originalReason,
	}
	b.Status.BackoffRetryStatus = append(b.Status.BackoffRetryStatus, record)
	UpdateBackupCondition(&b.Status, &metav1.Condition{
		Type:    string(BackupRetryTheFailed),
		Status:  metav1.ConditionTrue,
		Reason:  "RetryFailedBackup",
		Message: fmt.Sprintf("reason %s, original reason %s", record.RetryReason, record.OriginalReason),
	})
	return &nextRetryTime.Time, nil
}

// PostRetry records the retry result, in other words, the retry is triggered
func (b *Backup) PostRetry(now time.Time) error {
	if len(b.Status.BackoffRetryStatus) == 0 {
		return fmt.Errorf("no retry status found")
	}
	lastRetry := &b.Status.BackoffRetryStatus[len(b.Status.BackoffRetryStatus)-1]
	if lastRetry.RealRetryAt != nil {
		// is already triggered, ignore
		return nil
	}

	lastRetry.RealRetryAt = &metav1.Time{Time: now}
	UpdateBackupCondition(&b.Status, &metav1.Condition{
		Type:    string(BackupRestart),
		Status:  metav1.ConditionTrue,
		Reason:  "RetryTriggered",
		Message: fmt.Sprintf("retry triggered at %s", now),
	})
	return nil
}

// isBackoffRetryExeedLimit checks whether the `newRetry` is exceed limit.
// The `exceedReason` will be empty if the `newRetry` is not exceed limit.
func isBackoffRetryExeedLimit(retryPolicy *BackoffRetryPolicy, retryRecords []BackoffRetryRecord, now time.Time) (exceedReason string, _ error) {
	failedReason := ""
	// check is exceed retry limit
	isExceedRetryTimes := isExceedRetryTimes(retryPolicy, retryRecords)
	if isExceedRetryTimes {
		failedReason = fmt.Sprintf("exceed retry times, max is %d. ", retryPolicy.MaxRetryTimes)
	}
	isRetryTimeout, err := isRetryTimeout(retryPolicy, retryRecords, now)
	if err != nil {
		klog.Errorf("fail to check whether the retry is timeout")
		return "", err
	}
	if isRetryTimeout {
		failedReason = fmt.Sprintf("retry timeout, max is %s.", retryPolicy.RetryTimeout)
	}
	if isExceedRetryTimes || isRetryTimeout {
		return failedReason, nil
	}
	return "", nil
}

func isExceedRetryTimes(policy *BackoffRetryPolicy, records []BackoffRetryRecord) bool {
	if len(records) == 0 {
		return false
	}

	currentRetryNum := records[len(records)-1].RetryNum
	return currentRetryNum >= policy.MaxRetryTimes
}

func isRetryTimeout(policy *BackoffRetryPolicy, records []BackoffRetryRecord, now time.Time) (bool, error) {
	if len(records) == 0 {
		return false, nil
	}
	firstDetectAt := records[0].DetectFailedAt
	retryTimeout, err := time.ParseDuration(policy.RetryTimeout)
	if err != nil {
		klog.Errorf("fail to parse retryTimeout(%s): %s", policy.RetryTimeout, err)
		return false, err
	}
	return now.Unix()-firstDetectAt.Unix() > int64(retryTimeout)/int64(time.Second), nil
}

func calcNextRetryTime(policy *BackoffRetryPolicy, records []BackoffRetryRecord, retryNum int, now time.Time) (metav1.Time, error) {
	minDuration, err := time.ParseDuration(policy.MinRetryDuration)
	if err != nil {
		return metav1.Time{}, fmt.Errorf("fail to parse minRetryDuration %s: %s", policy.MinRetryDuration, err)
	}
	duration := time.Duration(minDuration.Nanoseconds() << (retryNum - 1))
	return metav1.NewTime(now.Add(duration)), nil
}
