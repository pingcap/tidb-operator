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

	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCleanJobName return the clean job name
func (bk *Backup) GetCleanJobName() string {
	return fmt.Sprintf("clean-%s", bk.GetName())
}

// GetBackupJobName return the backup job name
func (bk *Backup) GetBackupJobName() string {
	return fmt.Sprintf("backup-%s", bk.GetName())
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
	condition.LastTransitionTime = metav1.Now()
	// Try to find this Backup condition.
	conditionIndex, oldCondition := GetBackupCondition(status, condition.Type)

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

// IsBackupClean returns true if a Backup has successfully clean
func IsBackupClean(backup *Backup) bool {
	_, condition := GetBackupCondition(&backup.Status, BackupClean)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// ShouldCleanData returns true if a Backup should be cleaned up according to cleanPolicy
func ShouldCleanData(backup *Backup) bool {
	switch backup.Spec.CleanPolicy {
	case CleanPolicyTypeDelete, CleanPolicyTypeOnFailure:
		return true
	default:
		return false
	}
}
