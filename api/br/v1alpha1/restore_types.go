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

// Restore represents the restoration of backup of a tidb cluster.
//
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=br
// +kubebuilder:resource:shortName="rt"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="The current status of the restore"
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.status.timeStarted`,description="The time at which the restore was started",priority=1
// +kubebuilder:printcolumn:name="Completed",type=date,JSONPath=`.status.timeCompleted`,description="The time at which the restore was completed",priority=1
// +kubebuilder:printcolumn:name="TimeTaken",type=string,JSONPath=`.status.timeTaken`,description="The time that the restore takes"
// +kubebuilder:printcolumn:name="CommitTS",type=string,JSONPath=`.status.commitTs`,description="The commit ts of tidb cluster restore"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Restore struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec RestoreSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status RestoreStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// RestoreList contains a list of Restore.
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []Restore `json:"items"`
}

// RestoreMode represents the restore mode, such as snapshot or pitr.
// +k8s:openapi-gen=true
type RestoreMode string

const (
	// RestoreModeSnapshot represents restore from a snapshot backup.
	RestoreModeSnapshot RestoreMode = "snapshot"
	// RestoreModePiTR represents PiTR restore which is from a snapshot backup and log backup.
	RestoreModePiTR RestoreMode = "pitr"
	// RestoreModeVolumeSnapshot represents restore from a volume snapshot backup.
	RestoreModeVolumeSnapshot RestoreMode = "volume-snapshot"
)

// RestoreConditionType represents a valid condition of a Restore.
type RestoreConditionType string

const (
	// RestoreScheduled means the restore job has been created to do tidb cluster restore
	RestoreScheduled RestoreConditionType = "Scheduled"
	// RestoreRunning means the Restore is currently being executed.
	RestoreRunning RestoreConditionType = "Running"
	// RestoreVolumeComplete means the Restore has successfully executed part-1 and the
	// backup volumes have been rebuilded from the corresponding snapshot
	RestoreVolumeComplete RestoreConditionType = "VolumeComplete"
	// CleanVolumeComplete means volumes are cleaned successfully if restore volume failed
	CleanVolumeComplete RestoreConditionType = "CleanVolumeComplete"
	// RestoreWarmUpStarted means the Restore has successfully started warm up pods to
	// initialize volumes restored from snapshots
	RestoreWarmUpStarted RestoreConditionType = "WarmUpStarted"
	// RestoreWarmUpComplete means the Restore has successfully warmed up all TiKV volumes
	RestoreWarmUpComplete RestoreConditionType = "WarmUpComplete"
	// RestoreDataComplete means the Restore has successfully executed part-2 and the
	// data in restore volumes has been deal with consistency based on min_resolved_ts
	RestoreDataComplete RestoreConditionType = "DataComplete"
	// RestoreTiKVComplete means in volume restore, all TiKV instances are started and up
	RestoreTiKVComplete RestoreConditionType = "TikvComplete"
	// RestoreComplete means the Restore has successfully executed and the
	// backup data has been loaded into tidb cluster.
	RestoreComplete RestoreConditionType = "Complete"
	// RestoreFailed means the Restore has failed.
	RestoreFailed RestoreConditionType = "Failed"
	// RestoreRetryFailed means this failure can be retried
	RestoreRetryFailed RestoreConditionType = "RetryFailed"
	// RestoreInvalid means invalid restore CR.
	RestoreInvalid RestoreConditionType = "Invalid"
)

// RestoreCondition describes the observed state of a Restore at a certain point.
type RestoreCondition struct {
	Type   RestoreConditionType   `json:"type"`
	Status corev1.ConditionStatus `json:"status"`

	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

// +k8s:openapi-gen=true
// RestoreSpec contains the specification for a restore of a tidb cluster backup.
type RestoreSpec struct {
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
	// To is the tidb cluster that needs to restore.
	To *TiDBAccessConfig `json:"to,omitempty"`
	// Type is the backup type for tidb cluster and only used when Mode = snapshot, such as full, db, table.
	Type BackupType `json:"backupType,omitempty"`
	// Mode is the restore mode. such as snapshot or pitr.
	// +kubebuilder:default=snapshot
	Mode RestoreMode `json:"restoreMode,omitempty"`
	// PitrRestoredTs is the pitr restored ts.
	PitrRestoredTs string `json:"pitrRestoredTs,omitempty"`
	// LogRestoreStartTs is the start timestamp which log restore from.
	// +optional
	LogRestoreStartTs string `json:"logRestoreStartTs,omitempty"`
	// FederalVolumeRestorePhase indicates which phase to execute in federal volume restore
	// +optional
	FederalVolumeRestorePhase FederalVolumeRestorePhase `json:"federalVolumeRestorePhase,omitempty"`
	// VolumeAZ indicates which AZ the volume snapshots restore to.
	// it is only valid for mode of volume-snapshot
	// +optional
	VolumeAZ string `json:"volumeAZ,omitempty"`
	// TikvGCLifeTime is to specify the safe gc life time for restore.
	// The time limit during which data is retained for each GC, in the format of Go Duration.
	// When a GC happens, the current time minus this value is the safe point.
	TikvGCLifeTime *string `json:"tikvGCLifeTime,omitempty"`
	// StorageProvider configures where and how backups should be stored.
	StorageProvider `json:",inline"`
	// PitrFullBackupStorageProvider configures where and how pitr dependent full backup should be stored.
	// +optional
	PitrFullBackupStorageProvider StorageProvider `json:"pitrFullBackupStorageProvider,omitempty"`
	// The storageClassName of the persistent volume for Restore data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize,omitempty"`
	// BR is the configs for BR.
	BR *BRConfig `json:"br,omitempty"`
	// Base tolerations of restore Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Affinity of restore Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Use KMS to decrypt the secrets
	UseKMS bool `json:"useKMS,omitempty"`
	// Specify service account of restore
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// ToolImage specifies the tool image used in `Restore`, which supports BR and TiDB Lightning images.
	// For examples `spec.toolImage: pingcap/br:v4.0.8` or `spec.toolImage: pingcap/tidb-lightning:v4.0.8`
	// For BR image, if it does not contain tag, Pod will use image 'ToolImage:${TiKV_Version}'.
	// +optional
	ToolImage string `json:"toolImage,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// TableFilter means Table filter expression for 'db.table' matching. BR supports this from v4.0.3.
	TableFilter []string `json:"tableFilter,omitempty"`
	// Warmup represents whether to initialize TiKV volumes after volume snapshot restore
	// +optional
	Warmup RestoreWarmupMode `json:"warmup,omitempty"`
	// WarmupImage represents using what image to initialize TiKV volumes
	// +optional
	WarmupImage string `json:"warmupImage,omitempty"`
	// WarmupStrategy
	// +kubebuilder:default=hybrid
	WarmupStrategy RestoreWarmupStrategy `json:"warmupStrategy,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// PriorityClassName of Restore Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Additional volumes of component pod.
	// +optional
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`
	// Additional volume mounts of component pod.
	// +optional
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
	// TolerateSingleTiKVOutage indicates whether to tolerate a single failure of a store without data loss
	// +kubebuilder:default=false
	TolerateSingleTiKVOutage bool `json:"tolerateSingleTiKVOutage,omitempty"`
}

// FederalVolumeRestorePhase represents a phase to execute in federal volume restore
type FederalVolumeRestorePhase string

const (
	// FederalVolumeRestoreVolume means restore volumes of TiKV and start TiKV
	FederalVolumeRestoreVolume FederalVolumeRestorePhase = "restore-volume"
	// FederalVolumeRestoreData means restore data of TiKV to resolved TS
	FederalVolumeRestoreData FederalVolumeRestorePhase = "restore-data"
	// FederalVolumeRestoreFinish means restart TiKV and set recoveryMode true
	FederalVolumeRestoreFinish FederalVolumeRestorePhase = "restore-finish"
)

// RestoreWarmupMode represents when to initialize TiKV volumes
type RestoreWarmupMode string

const (
	// RestoreWarmupModeSync means initialize TiKV volumes before TiKV starts
	RestoreWarmupModeSync RestoreWarmupMode = "sync"
	// RestoreWarmupModeASync means initialize TiKV volumes after restore complete
	RestoreWarmupModeASync RestoreWarmupMode = "async"
)

// RestoreWarmupStrategy represents how to initialize TiKV volumes
type RestoreWarmupStrategy string

const (
	// RestoreWarmupStrategyFio warms up all data block by block. (use fio)
	RestoreWarmupStrategyFio RestoreWarmupStrategy = "fio"
	// RestoreWarmupStrategyHybrid warms up data volume by read sst files one by one, other (e.g. WAL or Raft) will be warmed up via fio.
	RestoreWarmupStrategyHybrid RestoreWarmupStrategy = "hybrid"
	// RestoreWarmupStrategyFsr warms up data volume by enabling Fast Snapshot Restore, other (e.g. WAL or Raft) will be warmed up via fio.
	RestoreWarmupStrategyFsr RestoreWarmupStrategy = "fsr"
	// RestoreWarmupStrategyCheckOnly warm up none data volumes and check wal consistency
	RestoreWarmupStrategyCheckOnly RestoreWarmupStrategy = "check-wal-only"
)

// RestoreStatus represents the current status of a tidb cluster restore.
type RestoreStatus struct {
	// TimeStarted is the time at which the restore was started.
	// +nullable
	TimeStarted metav1.Time `json:"timeStarted,omitempty"`
	// TimeCompleted is the time at which the restore was completed.
	// +nullable
	TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
	// TimeTaken is the time that restore takes, it is TimeCompleted - TimeStarted
	TimeTaken string `json:"timeTaken,omitempty"`
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs string `json:"commitTs,omitempty"`
	// Phase is a user readable state inferred from the underlying Restore conditions
	Phase RestoreConditionType `json:"phase,omitempty"`
	// +nullable
	Conditions []RestoreCondition `json:"conditions,omitempty"`
	// Progresses is the progress of restore.
	// +nullable
	Progresses []Progress `json:"progresses,omitempty"`
}
