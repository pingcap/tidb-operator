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

package restore

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
)

// GetRestoreCondition get the specify type's RestoreCondition from the given RestoreStatus
func GetRestoreCondition(status *v1alpha1.RestoreStatus, conditionType v1alpha1.RestoreConditionType) (int, *v1alpha1.RestoreCondition) {
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

// UpdateRestoreCondition updates existing Restore condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if Restore condition has changed or has been added.
func UpdateRestoreCondition(status *v1alpha1.RestoreStatus, condition *v1alpha1.RestoreCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this Restore condition.
	conditionIndex, oldCondition := GetRestoreCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new Restore condition.
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

// IsRestoreComplete returns true if a Restore has successfully completed
func IsRestoreComplete(backup *v1alpha1.Restore) bool {
	_, condition := GetRestoreCondition(&backup.Status, v1alpha1.RestoreComplete)
	return condition != nil && condition.Status == corev1.ConditionTrue
}
