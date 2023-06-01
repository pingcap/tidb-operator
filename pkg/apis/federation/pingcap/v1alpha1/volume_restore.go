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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsVolumeRestoreRunning(volumeRestore *VolumeRestore) bool {
	_, condition := GetVolumeRestoreCondition(&volumeRestore.Status, VolumeRestoreRunning)
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
