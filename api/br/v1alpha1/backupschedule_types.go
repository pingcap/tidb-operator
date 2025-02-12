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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupSchedule is a backup schedule of tidb cluster.
//
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=br
// +kubebuilder:resource:shortName="bks"
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`,description="The cron format string used for backup scheduling"
// +kubebuilder:printcolumn:name="MaxBackups",type=integer,JSONPath=`.spec.maxBackups`,description="The max number of backups we want to keep"
// +kubebuilder:printcolumn:name="MaxReservedTime",type=string,JSONPath=`.spec.maxReservedTime`,description="How long backups we want to keep"
// +kubebuilder:printcolumn:name="LastBackup",type=string,JSONPath=`.status.lastBackup`,description="The last backup CR name",priority=1
// +kubebuilder:printcolumn:name="LastBackupTime",type=date,JSONPath=`.status.lastBackupTime`,description="The last time the backup was successfully created",priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type BackupSchedule struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec BackupScheduleSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status BackupScheduleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// BackupScheduleList contains a list of BackupSchedule.
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BackupSchedule `json:"items"`
}

// +k8s:openapi-gen=true
// BackupScheduleSpec contains the backup schedule specification for a tidb cluster.
type BackupScheduleSpec struct {
	// Schedule specifies the cron string used for backup scheduling.
	Schedule string `json:"schedule"`
	// Pause means paused backupSchedule
	Pause bool `json:"pause,omitempty"`
	// MaxBackups is to specify how many backups we want to keep
	// 0 is magic number to indicate un-limited backups.
	// if MaxBackups and MaxReservedTime are set at the same time, MaxReservedTime is preferred
	// and MaxBackups is ignored.
	MaxBackups *int32 `json:"maxBackups,omitempty"`
	// MaxReservedTime is to specify how long backups we want to keep.
	MaxReservedTime *string `json:"maxReservedTime,omitempty"`
	// CompactSpan is to specify how long backups we want to compact.
	CompactSpan *string `json:"compactSpan,omitempty"`
	// BackupTemplate is the specification of the backup structure to get scheduled.
	BackupTemplate BackupSpec `json:"backupTemplate"`
	// LogBackupTemplate is the specification of the log backup structure to get scheduled.
	// +optional
	LogBackupTemplate *BackupSpec `json:"logBackupTemplate"`
	// CompactBackupTemplate is the specification of the compact backup structure to get scheduled.
	// +optional
	CompactBackupTemplate *CompactSpec `json:"compactBackupTemplate"`
	// The storageClassName of the persistent volume for Backup data storage if not storage class name set in BackupSpec.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// BRConfig is the configs for BR
	// +optional
	BR *BRConfig `json:"br,omitempty"`
	// StorageProvider configures where and how backups should be stored.
	// +optional
	StorageProvider `json:",inline"`
}

// BackupScheduleStatus represents the current state of a BackupSchedule.
type BackupScheduleStatus struct {
	// LastBackup represents the last backup.
	LastBackup string `json:"lastBackup,omitempty"`
	// LastCompact represents the last compact
	LastCompact string `json:"lastCompact,omitempty"`
	// logBackup represents the name of log backup.
	LogBackup *string `json:"logBackup,omitempty"`
	// LogBackupStartTs represents the start time of log backup
	LogBackupStartTs *metav1.Time `json:"logBackupStartTs,omitempty"`
	// LastBackupTime represents the last time the backup was successfully created.
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`
	// LastCompactTs represents the endTs of the last compact
	LastCompactTs *metav1.Time `json:"lastCompactTs,omitempty"`
	// NextCompactEndTs represents the scheduled endTs of next compact
	NextCompactEndTs *metav1.Time `json:"nextCompactEndTs,omitempty"`
	// AllBackupCleanTime represents the time when all backup entries are cleaned up
	AllBackupCleanTime *metav1.Time `json:"allBackupCleanTime,omitempty"`
}
