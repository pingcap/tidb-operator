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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

// GetRestoreJobName return the restore job name
func (rs *Restore) GetRestoreJobName() string {
	return fmt.Sprintf("restore-%s", rs.GetName())
}

// GetInstanceName return the restore instance name
func (rs *Restore) GetInstanceName() string {
	if rs.Labels != nil {
		if v, ok := rs.Labels[metav1alpha1.InstanceLabelKey]; ok {
			return v
		}
	}
	return rs.Name
}

// // GetTidbEndpointHash return the hash string base on tidb cluster's host and port
// func (rs *Restore) GetTidbEndpointHash() string {
// 	return HashContents([]byte(rs.Spec.To.GetTidbEndpoint()))
// }

// // GetRestorePVCName return the restore pvc name
// func (rs *Restore) GetRestorePVCName() string {
// 	return fmt.Sprintf("restore-pvc-%s", rs.GetTidbEndpointHash())
// }

// GetRestoreCondition get the specify type's RestoreCondition from the given RestoreStatus
func GetRestoreCondition(status *RestoreStatus, conditionType RestoreConditionType) (int, *metav1.Condition) {
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

// UpdateRestoreCondition updates existing Restore condition or creates a new
// one. Sets LastTransitionTime to now if the status has changed.
// Returns true if Restore condition has changed or has been added.
func UpdateRestoreCondition(status *RestoreStatus, condition *metav1.Condition) bool {
	if condition == nil {
		return false
	}
	condition.LastTransitionTime = metav1.Now()
	status.Phase = RestoreConditionType(condition.Type)
	// Try to find this Restore condition.
	conditionIndex, oldCondition := GetRestoreCondition(status, RestoreConditionType(condition.Type))

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

// IsRestoreInvalid returns true if a Restore has invalid condition set
func IsRestoreInvalid(restore *Restore) bool {
	_, condition := GetRestoreCondition(&restore.Status, RestoreInvalid)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsRestoreComplete returns true if a Restore has successfully completed
func IsRestoreComplete(restore *Restore) bool {
	_, condition := GetRestoreCondition(&restore.Status, RestoreComplete)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsRestoreScheduled returns true if a Restore has successfully scheduled
func IsRestoreScheduled(restore *Restore) bool {
	_, condition := GetRestoreCondition(&restore.Status, RestoreScheduled)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsRestoreRunning returns true if a Restore is Running
func IsRestoreRunning(restore *Restore) bool {
	_, condition := GetRestoreCondition(&restore.Status, RestoreRunning)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsRestoreFailed returns true if a Restore is Failed
func IsRestoreFailed(restore *Restore) bool {
	_, condition := GetRestoreCondition(&restore.Status, RestoreFailed)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsRestoreTiKVComplete returns true if all TiKVs run successfully during volume restore
func IsRestoreTiKVComplete(restore *Restore) bool {
	_, condition := GetRestoreCondition(&restore.Status, RestoreTiKVComplete)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsRestoreDataComplete returns true if a Restore for data consistency has successfully completed
func IsRestoreDataComplete(restore *Restore) bool {
	_, condition := GetRestoreCondition(&restore.Status, RestoreDataComplete)
	return condition != nil && condition.Status == metav1.ConditionTrue
}
