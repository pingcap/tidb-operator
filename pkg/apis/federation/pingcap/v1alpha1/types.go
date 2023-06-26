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

// TODO(federation): add `kubebuilder:printcolumn` after fileds are defined

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
<<<<<<< HEAD
=======
	Clusters []VolumeBackupMemberCluster `json:"clusters,omitempty"`
	Template VolumeBackupMemberSpec      `json:"template,omitempty"`
}

// VolumeBackupMemberCluster contains the TiDB cluster which need to execute volume backup
// +k8s:openapi-gen=true
type VolumeBackupMemberCluster struct {
	// K8sClusterName is the name of the k8s cluster where the tc locates
	K8sClusterName string `json:"k8sClusterName,omitempty"`
	// TCName is the name of the TiDBCluster CR which need to execute volume backup
	TCName string `json:"tcName,omitempty"`
	// TCNamespace is the namespace of the TiDBCluster CR
	TCNamespace string `json:"tcNamespace,omitempty"`
}

// VolumeBackupMemberSpec contains the backup specification for one tidb cluster
// +k8s:openapi-gen=true
type VolumeBackupMemberSpec struct {
	corev1.ResourceRequirements `json:"resources,omitempty"`
	// List of environment variables to set in the container, like v1.Container.Env.
	// Note that the following builtin env vars will be overwritten by values set here
	// - S3_PROVIDER
	// - S3_ENDPOINT
	// - AWS_REGION
	// - AWS_ACL
	// - AWS_STORAGE_CLASS
	// - AWS_DEFAULT_REGION
	// - AWS_ACCESS_KEY_ID
	// - AWS_SECRET_ACCESS_KEY
	// - GCS_PROJECT_ID
	// - GCS_OBJECT_ACL
	// - GCS_BUCKET_ACL
	// - GCS_LOCATION
	// - GCS_STORAGE_CLASS
	// - GCS_SERVICE_ACCOUNT_JSON_KEY
	// - BR_LOG_TO_TERM
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// BRConfig is the configs for BR
	BR *BRConfig `json:"br,omitempty"`
	// StorageProvider configures where and how backups should be stored.
	pingcapv1alpha1.StorageProvider `json:",inline"`
	Tolerations                     []corev1.Toleration `json:"tolerations,omitempty"`
	// ToolImage specifies the tool image used in `Backup`, which supports BR.
	// For examples `spec.toolImage: pingcap/br:v6.5.0`
	// For BR image, if it does not contain tag, Pod will use image 'ToolImage:${TiKV_Version}'.
	// +optional
	ToolImage string `json:"toolImage,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Specify service account of backup
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// CleanPolicy denotes whether to clean backup data when the object is deleted from the cluster, if not set, the backup data will be retained
	CleanPolicy pingcapv1alpha1.CleanPolicyType `json:"cleanPolicy,omitempty"`
	// PriorityClassName of Backup Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// BRConfig contains config for BR
// +k8s:openapi-gen=true
type BRConfig struct {
	// Concurrency is the size of thread pool on each node that execute the backup task
	Concurrency *uint32 `json:"concurrency,omitempty"`
	// CheckRequirements specifies whether to check requirements
	CheckRequirements *bool `json:"checkRequirements,omitempty"`
	// SendCredToTikv specifies whether to send credentials to TiKV
	SendCredToTikv *bool `json:"sendCredToTikv,omitempty"`
	// Options means options for backup data to remote storage with BR. These options has highest priority.
	Options []string `json:"options,omitempty"`
>>>>>>> 9ae7cc6b0 (Fed backup schedule (#5036))
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
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`,description="The cron format string used for backup scheduling"
// +kubebuilder:printcolumn:name="MaxBackups",type=integer,JSONPath=`.spec.maxBackups`,description="The max number of backups we want to keep"
// +kubebuilder:printcolumn:name="MaxReservedTime",type=string,JSONPath=`.spec.maxReservedTime`,description="How long backups we want to keep"
// +kubebuilder:printcolumn:name="LastBackup",type=string,JSONPath=`.status.lastBackup`,description="The last backup CR name",priority=1
// +kubebuilder:printcolumn:name="LastBackupTime",type=date,JSONPath=`.status.lastBackupTime`,description="The last time the backup was successfully created",priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type VolumeBackupSchedule struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec VolumeBackupScheduleSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status VolumeBackupScheduleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// VolumeBackupScheduleList is VolumeBackupSchedule list
type VolumeBackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeBackupSchedule `json:"items"`
}

// +k8s:openapi-gen=true
// VolumeBackupScheduleSpec describes the attributes that a user creates on a volume backup schedule.
type VolumeBackupScheduleSpec struct {
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
	// BackupTemplate is the specification of the volume backup structure to get scheduled.
	BackupTemplate VolumeBackupSpec `json:"backupTemplate"`
}

// VolumeBackupScheduleStatus represents the current status of a volume backup schedule.
type VolumeBackupScheduleStatus struct {
	// LastBackup represents the last backup.
	LastBackup string `json:"lastBackup,omitempty"`
	// LastBackupTime represents the last time the backup was successfully created.
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`
	// AllBackupCleanTime represents the time when all backup entries are cleaned up
	AllBackupCleanTime *metav1.Time `json:"allBackupCleanTime,omitempty"`
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
