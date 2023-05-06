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

// TODO: add `kubebuilder:printcolumn` after fileds are defined

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackup is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="vbk"
// +genclient:noStatus
type VolumeBackup struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec VolumeBackupSpec `json:"spec"`

	// +k8s:openapi-gen=false
	Status VolumeBackupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackupList is VolumeBackup list
// +k8s:openapi-gen=true
type VolumeBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeBackup `json:"items"`
}

// VolumeBackupSpec describes the attributes that a user creates on a volume backup.
// +k8s:openapi-gen=true
type VolumeBackupSpec struct {
}

// VolumeBackupStatus represents the current status of a volume backup.
type VolumeBackupStatus struct {
	// +nullable
	Conditions []VolumeBackupCondition `json:"conditions,omitempty"`
}

// VolumeBackupCondition describes the observed state of a VolumeBackup at a certain point.
type VolumeBackupCondition struct {
	Status corev1.ConditionStatus `json:"status"`

	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackupSchedule is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="vbks"
// +genclient:noStatus
type VolumeBackupSchedule struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec VolumeBackupScheduleSpec `json:"spec"`

	// +k8s:openapi-gen=false
	Status VolumeBackupScheduleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackupScheduleList is VolumeBackupSchedule list
// +k8s:openapi-gen=true
type VolumeBackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeBackupSchedule `json:"items"`
}

// VolumeBackupScheduleSpec describes the attributes that a user creates on a volume backup schedule.
// +k8s:openapi-gen=true
type VolumeBackupScheduleSpec struct {
}

// VolumeBackupScheduleStatus represents the current status of a volume backup schedule.
type VolumeBackupScheduleStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeRestore is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="vrt"
// +genclient:noStatus
type VolumeRestore struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec VolumeRestoreSpec `json:"spec"`

	// +k8s:openapi-gen=false
	Status VolumeRestoreStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeRestoreList is VolumeRestore list
// +k8s:openapi-gen=true
type VolumeRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeRestore `json:"items"`
}

// VolumeRestoreSpec describes the attributes that a user creates on a volume restore.
// +k8s:openapi-gen=true
type VolumeRestoreSpec struct {
}

// VolumeRestoreStatus represents the current status of a volume restore.
type VolumeRestoreStatus struct {
	// +nullable
	Conditions []VolumeRestoreCondition `json:"conditions,omitempty"`
}

// VolumeRestoreCondition describes the observed state of a VolumeRestore at a certain point.
type VolumeRestoreCondition struct {
	Status corev1.ConditionStatus `json:"status"`

	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}
