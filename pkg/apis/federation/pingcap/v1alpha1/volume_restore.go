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

package v1alpha1

import (
	"time"

	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsVolumeRestoreRunning(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreRunning)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func IsVolumeRestoreVolumeComplete(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreVolumeComplete)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func IsVolumeRestoreTiKVComplete(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreTiKVComplete)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func IsVolumeRestoreDataComplete(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreDataComplete)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func IsVolumeRestoreComplete(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreComplete)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func IsVolumeRestoreFailed(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreFailed)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func IsVolumeRestoreCleaned(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreCleaned)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// UpdateVolumeRestoreCondition adds new condition or update condition if it exists in status
func UpdateVolumeRestoreCondition(volumeRestoreStatus *VolumeRestoreStatus, condition *VolumeRestoreCondition) {
	condition.LastTransitionTime = metav1.Now()
	volumeRestoreStatus.Phase = condition.Type
	existedCondIndex, existedCondition := GetVolumeRestoreCondition(volumeRestoreStatus, condition.Type)

	if existedCondIndex == -1 {
		volumeRestoreStatus.Conditions = append(volumeRestoreStatus.Conditions, *condition)
		return
	}

	if existedCondition.Status == condition.Status {
		condition.LastTransitionTime = existedCondition.LastTransitionTime
	}
	volumeRestoreStatus.Conditions[existedCondIndex] = *condition
}

func GetVolumeRestoreCondition(volumeRestoreStatus *VolumeRestoreStatus, conditionType VolumeRestoreConditionType) (index int, condition *VolumeRestoreCondition) {
	index = -1
	for i := range volumeRestoreStatus.Conditions {
		if volumeRestoreStatus.Conditions[i].Type == conditionType {
			index = i
			condition = &volumeRestoreStatus.Conditions[i]
			break
		}
	}
	return
}

func UpdateVolumeRestoreMemberStatus(volumeRestoreStatus *VolumeRestoreStatus, k8sClusterName string, restoreMember *pingcapv1alpha1.Restore) {
	restoreMemberStatus := VolumeRestoreMemberStatus{
		K8sClusterName: k8sClusterName,
		TCNamespace:    restoreMember.Spec.BR.ClusterNamespace,
		TCName:         restoreMember.Spec.BR.Cluster,
		RestoreName:    restoreMember.Name,
		CommitTs:       restoreMember.Status.CommitTs,
		Phase:          restoreMember.Status.Phase,
	}
	if restoreMember.Status.Phase == pingcapv1alpha1.RestoreFailed {
		var failedCondition *pingcapv1alpha1.RestoreCondition
		for i := range restoreMember.Status.Conditions {
			restoreCondition := restoreMember.Status.Conditions[i]
			if restoreCondition.Type == pingcapv1alpha1.RestoreFailed {
				failedCondition = &restoreCondition
				break
			}
		}
		if failedCondition != nil {
			restoreMemberStatus.Reason = failedCondition.Reason
			restoreMemberStatus.Message = failedCondition.Message
		}
	}

	for i := range volumeRestoreStatus.Restores {
		if volumeRestoreStatus.Restores[i].RestoreName == restoreMemberStatus.RestoreName {
			volumeRestoreStatus.Restores[i] = restoreMemberStatus
			return
		}
	}

	volumeRestoreStatus.Restores = append(volumeRestoreStatus.Restores, restoreMemberStatus)
}

func StartVolumeRestoreStep(volumeRestoreStatus *VolumeRestoreStatus, stepName VolumeRestoreStepType) {
	for i := range volumeRestoreStatus.Steps {
		if volumeRestoreStatus.Steps[i].StepName == stepName {
			// the step is already existed, do nothing
			return
		}
	}

	volumeRestoreStatus.Steps = append(volumeRestoreStatus.Steps, VolumeRestoreStep{
		StepName:    stepName,
		TimeStarted: metav1.Now(),
	})
}

func FinishVolumeRestoreStep(volumeRestoreStatus *VolumeRestoreStatus, stepName VolumeRestoreStepType) {
	for i := range volumeRestoreStatus.Steps {
		if volumeRestoreStatus.Steps[i].StepName == stepName {
			if volumeRestoreStatus.Steps[i].TimeCompleted.Unix() > 0 {
				return
			}

			volumeRestoreStatus.Steps[i].TimeCompleted = metav1.Now()
			volumeRestoreStatus.Steps[i].TimeTaken = volumeRestoreStatus.Steps[i].TimeCompleted.
				Sub(volumeRestoreStatus.Steps[i].TimeStarted.Time).Round(time.Second).String()
		}
	}
}
