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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackup is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="vbf"
// +genclient:noStatus
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="The current status of the backup"
// +kubebuilder:printcolumn:name="BackupSize",type=string,JSONPath=`.status.backupSizeReadable`,description="The data size of the backup"
// +kubebuilder:printcolumn:name="CommitTS",type=string,JSONPath=`.status.commitTs`,description="The commit ts of the backup"
// +kubebuilder:printcolumn:name="TimeTaken",type=string,JSONPath=`.status.timeTaken`,description="The time that volume backup federation takes"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
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
	BR                              *BRConfig `json:"br,omitempty"`
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
}

// VolumeBackupStatus represents the current status of a volume backup.
type VolumeBackupStatus struct {
	// Backups are volume backups' information in data plane
	Backups []VolumeBackupMemberStatus `json:"backups,omitempty"`
	// TimeStarted is the time at which the backup was started.
	// +nullable
	TimeStarted metav1.Time `json:"timeStarted,omitempty"`
	// TimeCompleted is the time at which the backup was completed.
	// +nullable
	TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
	// TimeTaken is the time that volume backup federation takes, it is TimeCompleted - TimeStarted
	TimeTaken string `json:"timeTaken,omitempty"`
	// BackupSizeReadable is the data size of the backup.
	// the difference with BackupSize is that its format is human readable
	BackupSizeReadable string `json:"backupSizeReadable,omitempty"`
	// BackupSize is the data size of the backup.
	BackupSize int64 `json:"backupSize,omitempty"`
	// CommitTs is the commit ts of the backup, snapshot ts for full backup or start ts for log backup.
	CommitTs string `json:"commitTs,omitempty"`
	// Phase is a user readable state inferred from the underlying Backup conditions
	Phase VolumeBackupConditionType `json:"phase,omitempty"`
	// +nullable
	Conditions []VolumeBackupCondition `json:"conditions,omitempty"`
}

type VolumeBackupMemberStatus struct {
	// K8sClusterName is the name of the k8s cluster where the tc locates
	K8sClusterName string `json:"k8sClusterName,omitempty"`
	// TCName is the name of the TiDBCluster CR which need to execute volume backup
	TCName string `json:"tcName,omitempty"`
	// TCNamespace is the namespace of the TiDBCluster CR
	TCNamespace string `json:"tcNamespace,omitempty"`
	// BackupName is the name of Backup CR
	BackupName string `json:"backupName"`
	// BackupPath is the location of the backup
	BackupPath string `json:"backupPath,omitempty"`
	// BackupSize is the data size of the backup
	BackupSize int64 `json:"backupSize,omitempty"`
	// CommitTs is the commit ts of the backup
	CommitTs string `json:"commitTs,omitempty"`
}

// VolumeBackupCondition describes the observed state of a VolumeBackup at a certain point.
type VolumeBackupCondition struct {
	Status corev1.ConditionStatus    `json:"status"`
	Type   VolumeBackupConditionType `json:"type"`

	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

// VolumeBackupConditionType represents a valid condition of a VolumeBackup.
type VolumeBackupConditionType string

const (
	// VolumeBackupInvalid means the VolumeBackup is invalid
	VolumeBackupInvalid VolumeBackupConditionType = "Invalid"
	// VolumeBackupRunning means the VolumeBackup is running
	VolumeBackupRunning VolumeBackupConditionType = "Running"
	// VolumeBackupComplete means all the backups in data plane are complete and the VolumeBackup is complete
	VolumeBackupComplete VolumeBackupConditionType = "Complete"
	// VolumeBackupFailed means one of backup in data plane is failed and the VolumeBackup is failed
	VolumeBackupFailed VolumeBackupConditionType = "Failed"
	// VolumeBackupCleaned means all the resources about VolumeBackup have cleaned
	VolumeBackupCleaned VolumeBackupConditionType = "Cleaned"
	// VolumeBackupCleanFailed means the VolumeBackup cleanup is failed
	VolumeBackupCleanFailed VolumeBackupConditionType = "CleanFailed"
)

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
	Clusters []VolumeRestoreMemberCluster `json:"clusters,omitempty"`
	Template VolumeRestoreMemberSpec      `json:"template,omitempty"`
}

// VolumeRestoreMemberCluster contains the TiDB cluster which need to execute volume restore
// +k8s:openapi-gen=true
type VolumeRestoreMemberCluster struct {
	// K8sClusterName is the name of the k8s cluster where the tc locates
	K8sClusterName string `json:"k8sClusterName,omitempty"`
	// TCName is the name of the TiDBCluster CR which need to execute volume backup
	TCName string `json:"tcName,omitempty"`
	// TCNamespace is the namespace of the TiDBCluster CR
	TCNamespace string `json:"tcNamespace,omitempty"`
	// AZName is the available zone which the volume snapshots restore to
	AZName string `json:"azName,omitempty"`
	// Backup is the volume backup information
	Backup VolumeRestoreMemberBackupInfo `json:"backup,omitempty"`
}

// VolumeRestoreMemberSpec contains the restore specification for one tidb cluster
// +k8s:openapi-gen=true
type VolumeRestoreMemberSpec struct {
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
	// RestoredTS is the volume restored ts, it is from CommitTs of volume backup status
	RestoredTs string `json:"restoredTs,omitempty"`
	// BRConfig is the configs for BR
	BR          *BRConfig           `json:"br,omitempty"`
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// ToolImage specifies the tool image used in `Restore`, which supports BR image.
	// For examples `spec.toolImage: pingcap/br:v6.5.0`
	// For BR image, if it does not contain tag, Pod will use image 'ToolImage:${TiKV_Version}'.
	// +optional
	ToolImage string `json:"toolImage,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Specify service account of restore
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// PriorityClassName of Restore Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

type VolumeRestoreMemberBackupInfo struct {
	pingcapv1alpha1.StorageProvider `json:",inline"`
}

// VolumeRestoreStatus represents the current status of a volume restore.
type VolumeRestoreStatus struct {
	// TimeStarted is the time at which the restore was started.
	// +nullable
	TimeStarted metav1.Time `json:"timeStarted,omitempty"`
	// TimeCompleted is the time at which the restore was completed.
	// +nullable
	TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
	// TimeTaken is the time that volume restore federation takes, it is TimeCompleted - TimeStarted
	TimeTaken string `json:"timeTaken,omitempty"`
	// Phase is a user readable state inferred from the underlying Restore conditions
	Phase VolumeRestoreConditionType `json:"phase,omitempty"`
	// +nullable
	Conditions []VolumeRestoreCondition `json:"conditions,omitempty"`
}

// VolumeRestoreCondition describes the observed state of a VolumeRestore at a certain point.
type VolumeRestoreCondition struct {
	Status corev1.ConditionStatus     `json:"status"`
	Type   VolumeRestoreConditionType `json:"type"`

	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

type VolumeRestoreConditionType string

const (
	VolumeRestoreInvalid  VolumeRestoreConditionType = "invalid"
	VolumeRestoreRunning  VolumeRestoreConditionType = "running"
	VolumeRestoreComplete VolumeRestoreConditionType = "complete"
	VolumeRestoreFailed   VolumeRestoreConditionType = "failed"
	VolumeRestoreCleaned  VolumeRestoreConditionType = "Cleaned"
)
