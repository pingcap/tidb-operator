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
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ToBRMemberConfig converts BRConfig to BRConfig of data plane
func (bc *BRConfig) ToBRMemberConfig(tcName, tcNamespace string) *pingcapv1alpha1.BRConfig {
	return &pingcapv1alpha1.BRConfig{
		Cluster:           tcName,
		ClusterNamespace:  tcNamespace,
		Concurrency:       bc.Concurrency,
		CheckRequirements: bc.CheckRequirements,
		SendCredToTikv:    bc.SendCredToTikv,
		Options:           bc.Options,
	}
}

// UpdateVolumeBackupCondition adds new condition or update condition if it exists in status
func UpdateVolumeBackupCondition(volumeBackupStatus *VolumeBackupStatus, condition *VolumeBackupCondition) {
	condition.LastTransitionTime = metav1.Now()
	volumeBackupStatus.Phase = condition.Type
	existedCondIndex, existedCondition := GetVolumeBackupCondition(volumeBackupStatus, condition.Type)

	if existedCondIndex == -1 {
		volumeBackupStatus.Conditions = append(volumeBackupStatus.Conditions, *condition)
		return
	}

	if existedCondition.Status == condition.Status {
		condition.LastTransitionTime = existedCondition.LastTransitionTime
	}
	volumeBackupStatus.Conditions[existedCondIndex] = *condition
}

// UpdateVolumeBackupMemberStatus adds new data plane backup status or update it if it exists in status
func UpdateVolumeBackupMemberStatus(volumeBackupStatus *VolumeBackupStatus, k8sClusterName string, backupMember *pingcapv1alpha1.Backup) {
	backupMemberStatus := VolumeBackupMemberStatus{
		K8sClusterName: k8sClusterName,
		TCName:         backupMember.Spec.BR.Cluster,
		TCNamespace:    backupMember.Spec.BR.ClusterNamespace,
		BackupName:     backupMember.Name,
		BackupPath:     backupMember.Status.BackupPath,
		BackupSize:     backupMember.Status.BackupSize,
		CommitTs:       backupMember.Status.CommitTs,
	}

	existedBackupIndex := -1
	for i := range volumeBackupStatus.Backups {
		if volumeBackupStatus.Backups[i].BackupName == backupMember.Name {
			existedBackupIndex = i
			break
		}
	}

	if existedBackupIndex == -1 {
		volumeBackupStatus.Backups = append(volumeBackupStatus.Backups, backupMemberStatus)
		return
	}

	volumeBackupStatus.Backups[existedBackupIndex] = backupMemberStatus
}

// IsVolumeBackupInvalid returns true if VolumeBackup is invalid
func IsVolumeBackupInvalid(volumeBackup *VolumeBackup) bool {
	_, condition := GetVolumeBackupCondition(&volumeBackup.Status, VolumeBackupInvalid)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsVolumeBackupRunning returns true if VolumeBackup is running
func IsVolumeBackupRunning(volumeBackup *VolumeBackup) bool {
	_, condition := GetVolumeBackupCondition(&volumeBackup.Status, VolumeBackupRunning)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsBackupPrepared returns true if VolumeBackup is running
func IsBackupPrepared(volumeBackup *VolumeBackup) bool {
	_, condition := GetVolumeBackupCondition(&volumeBackup.Status, VolumeBackupPrepared)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsVolumeBackupComplete returns true if VolumeBackup is complete
func IsVolumeBackupComplete(volumeBackup *VolumeBackup) bool {
	_, condition := GetVolumeBackupCondition(&volumeBackup.Status, VolumeBackupComplete)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsVolumeBackupFailed returns true if VolumeBackup is failed
func IsVolumeBackupFailed(volumeBackup *VolumeBackup) bool {
	_, condition := GetVolumeBackupCondition(&volumeBackup.Status, VolumeBackupFailed)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// IsVolumeBackupCleaned returns true if all the Backup CRs in data plane has cleaned
func IsVolumeBackupCleaned(volumeBackup *VolumeBackup) bool {
	_, condition := GetVolumeBackupCondition(&volumeBackup.Status, VolumeBackupCleaned)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// GetVolumeBackupCondition gets condition from status, if it doesn't exist, returned index = -1
func GetVolumeBackupCondition(volumeBackupStatus *VolumeBackupStatus, conditionType VolumeBackupConditionType) (index int, condition *VolumeBackupCondition) {
	index = -1
	for i := range volumeBackupStatus.Conditions {
		if volumeBackupStatus.Conditions[i].Type == conditionType {
			index = i
			condition = &volumeBackupStatus.Conditions[i]
			break
		}
	}
	return
}
