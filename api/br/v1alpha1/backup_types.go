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
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=br
// +kubebuilder:resource:shortName="bk"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.backupType`,description="the type of backup, such as full, db, table. Only used when Mode = snapshot."
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.backupMode`,description="the mode of backup, such as snapshot, log."
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="The current status of the backup"
// +kubebuilder:printcolumn:name="BackupPath",type=string,JSONPath=`.status.backupPath`,description="The full path of backup data"
// +kubebuilder:printcolumn:name="BackupSize",type=string,JSONPath=`.status.backupSizeReadable`,description="The data size of the backup"
// +kubebuilder:printcolumn:name="IncrementalBackupSize",type=string,JSONPath=`.status.incrementalBackupSizeReadable`,description="The real size of volume snapshot backup, only valid to volume snapshot backup",priority=10
// +kubebuilder:printcolumn:name="CommitTS",type=string,JSONPath=`.status.commitTs`,description="The commit ts of the backup"
// +kubebuilder:printcolumn:name="LogTruncateUntil",type=string,JSONPath=`.status.logSuccessTruncateUntil`,description="The log backup truncate until ts"
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.status.timeStarted`,description="The time at which the backup was started",priority=1
// +kubebuilder:printcolumn:name="Completed",type=date,JSONPath=`.status.timeCompleted`,description="The time at which the backup was completed",priority=1
// +kubebuilder:printcolumn:name="TimeTaken",type=string,JSONPath=`.status.timeTaken`,description="The time that the backup takes"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Backup is a backup of tidb cluster.
type Backup struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec BackupSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status BackupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// BackupList contains a list of Backup.
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []Backup `json:"items"`
}

// +k8s:openapi-gen=true
// BackupStorageType represents the backend storage type of backup.
type BackupStorageType string

const (
	// BackupStorageTypeS3 represents all storage that compatible with the Amazon S3.
	BackupStorageTypeS3 BackupStorageType = "s3"
	// BackupStorageTypeGcs represents the google cloud storage
	BackupStorageTypeGcs BackupStorageType = "gcs"
	// BackupStorageType represents the azure blob storage
	BackupStorageTypeAzblob BackupStorageType = "azblob"
	// BackupStorageTypeLocal represents local volume storage type
	BackupStorageTypeLocal BackupStorageType = "local"
	// BackupStorageTypeUnknown represents the unknown storage type
	BackupStorageTypeUnknown BackupStorageType = "unknown"
)

// +k8s:openapi-gen=true
// S3StorageProviderType represents the specific storage provider that implements the S3 interface
type S3StorageProviderType string

const (
	// S3StorageProviderTypeCeph represents the S3 compliant storage provider is ceph
	S3StorageProviderTypeCeph S3StorageProviderType = "ceph"
	// S3StorageProviderTypeAWS represents the S3 compliant storage provider is aws
	S3StorageProviderTypeAWS S3StorageProviderType = "aws"
)

// StorageProvider defines the configuration for storing a backup in backend storage.
// +k8s:openapi-gen=true
type StorageProvider struct {
	S3     *S3StorageProvider     `json:"s3,omitempty"`
	Gcs    *GcsStorageProvider    `json:"gcs,omitempty"`
	Azblob *AzblobStorageProvider `json:"azblob,omitempty"`
	Local  *LocalStorageProvider  `json:"local,omitempty"`
}

// LocalStorageProvider defines local storage options, which can be any k8s supported mounted volume
type LocalStorageProvider struct {
	Volume      corev1.Volume      `json:"volume"`
	VolumeMount corev1.VolumeMount `json:"volumeMount"`
	Prefix      string             `json:"prefix,omitempty"`
}

// S3StorageProvider represents a S3 compliant storage for storing backups.
// +k8s:openapi-gen=true
type S3StorageProvider struct {
	// Provider represents the specific storage provider that implements the S3 interface
	Provider S3StorageProviderType `json:"provider"`
	// Region in which the S3 compatible bucket is located.
	Region string `json:"region,omitempty"`
	// Path is the full path where the backup is saved.
	// The format of the path must be: "<bucket-name>/<path-to-backup-file>"
	Path string `json:"path,omitempty"`
	// Bucket in which to store the backup data.
	Bucket string `json:"bucket,omitempty"`
	// Endpoint of S3 compatible storage service
	Endpoint string `json:"endpoint,omitempty"`
	// StorageClass represents the storage class
	StorageClass string `json:"storageClass,omitempty"`
	// Acl represents access control permissions for this bucket
	Acl string `json:"acl,omitempty"`
	// SecretName is the name of secret which stores
	// S3 compliant storage access key and secret key.
	SecretName string `json:"secretName,omitempty"`
	// Prefix of the data path.
	Prefix string `json:"prefix,omitempty"`
	// SSE Sever-Side Encryption.
	SSE string `json:"sse,omitempty"`
	// Options Rclone options for backup and restore with dumpling and lightning.
	Options []string `json:"options,omitempty"`
}

// +k8s:openapi-gen=true
// GcsStorageProvider represents the google cloud storage for storing backups.
type GcsStorageProvider struct {
	// ProjectId represents the project that organizes all your Google Cloud Platform resources
	ProjectId string `json:"projectId"`
	// Location in which the gcs bucket is located.
	Location string `json:"location,omitempty"`
	// Path is the full path where the backup is saved.
	// The format of the path must be: "<bucket-name>/<path-to-backup-file>"
	Path string `json:"path,omitempty"`
	// Bucket in which to store the backup data.
	Bucket string `json:"bucket,omitempty"`
	// StorageClass represents the storage class
	StorageClass string `json:"storageClass,omitempty"`
	// ObjectAcl represents the access control list for new objects
	ObjectAcl string `json:"objectAcl,omitempty"`
	// BucketAcl represents the access control list for new buckets
	BucketAcl string `json:"bucketAcl,omitempty"`
	// SecretName is the name of secret which stores the
	// gcs service account credentials JSON.
	SecretName string `json:"secretName,omitempty"`
	// Prefix of the data path.
	Prefix string `json:"prefix,omitempty"`
}

// +k8s:openapi-gen=true
// AzblobStorageProvider represents the azure blob storage for storing backups.
type AzblobStorageProvider struct {
	// Path is the full path where the backup is saved.
	// The format of the path must be: "<container-name>/<path-to-backup-file>"
	Path string `json:"path,omitempty"`
	// Container in which to store the backup data.
	Container string `json:"container,omitempty"`
	// Access tier of the uploaded objects.
	AccessTier string `json:"accessTier,omitempty"`
	// SecretName is the name of secret which stores the
	// azblob service account credentials.
	SecretName string `json:"secretName,omitempty"`
	// StorageAccount is the storage account of the azure blob storage
	// If this field is set, then use this to set backup-manager env
	// Otherwise retrieve the storage account from secret
	StorageAccount string `json:"storageAccount,omitempty"`
	// SasToken is the sas token of the storage account
	SasToken string `json:"sasToken,omitempty"`
	// Prefix of the data path.
	Prefix string `json:"prefix,omitempty"`
}

// BackupType represents the backup type.
// +k8s:openapi-gen=true
type BackupType string

const (
	// BackupTypeFull represents the full backup of tidb cluster.
	BackupTypeFull BackupType = "full"
	// BackupTypeRaw represents the raw backup of tidb cluster.
	BackupTypeRaw BackupType = "raw"
	// BackupTypeDB represents the backup of one DB for the tidb cluster.
	BackupTypeDB BackupType = "db"
	// BackupTypeTable represents the backup of one table for the tidb cluster.
	BackupTypeTable BackupType = "table"
	// BackupTypeTiFlashReplica represents restoring the tiflash replica removed by a failed restore of the older version BR
	BackupTypeTiFlashReplica BackupType = "tiflash-replica"
)

// BackupType represents the backup mode, such as snapshot backup or log backup.
// +k8s:openapi-gen=true
type BackupMode string

const (
	// BackupModeSnapshot represents the snapshot backup of tidb cluster.
	BackupModeSnapshot BackupMode = "snapshot"
	// BackupModeLog represents the log backup of tidb cluster.
	BackupModeLog BackupMode = "log"
	// BackupModeVolumeSnapshot represents volume backup of tidb cluster.
	BackupModeVolumeSnapshot BackupMode = "volume-snapshot"
)

// TiDBAccessConfig defines the configuration for access tidb cluster
// +k8s:openapi-gen=true
type TiDBAccessConfig struct {
	// Host is the tidb cluster access address
	Host string `json:"host"`
	// Port is the port number to use for connecting tidb cluster
	Port int32 `json:"port,omitempty"`
	// User is the user for login tidb cluster
	User string `json:"user,omitempty"`
	// SecretName is the name of secret which stores tidb cluster's password.
	SecretName string `json:"secretName"`
	// TLSClientSecretName is the name of secret which stores tidb server client certificate
	// Optional: Defaults to nil
	// +optional
	TLSClientSecretName *string `json:"tlsClientSecretName,omitempty"`
}

// +k8s:openapi-gen=true
// CleanPolicyType represents the clean policy of backup data in remote storage
type CleanPolicyType string

const (
	// CleanPolicyTypeRetain represents that the backup data in remote storage will be retained when the Backup CR is deleted
	CleanPolicyTypeRetain CleanPolicyType = "Retain"
	// CleanPolicyTypeOnFailure represents that the backup data in remote storage will be cleaned only for the failed backups when the Backup CR is deleted
	CleanPolicyTypeOnFailure CleanPolicyType = "OnFailure"
	// CleanPolicyTypeDelete represents that the backup data in remote storage will be cleaned when the Backup CR is deleted
	CleanPolicyTypeDelete CleanPolicyType = "Delete"
)

// BatchDeleteOption controls the options to delete the objects in batches during the cleanup of backups
//
// +k8s:openapi-gen=true
type BatchDeleteOption struct {
	// DisableBatchConcurrency disables the batch deletions with S3 API and the deletion will be done by goroutines.
	DisableBatchConcurrency bool `json:"disableBatchConcurrency,omitempty"`
	// BatchConcurrency represents the number of batch deletions in parallel.
	// It is used when the storage provider supports the batch delete API, currently, S3 only.
	// default is 10
	BatchConcurrency uint32 `json:"batchConcurrency,omitempty"`
	// RoutineConcurrency represents the number of goroutines that used to delete objects
	// default is 100
	RoutineConcurrency uint32 `json:"routineConcurrency,omitempty"`
}

// CleanOption defines the configuration for cleanup backup
//
// +k8s:openapi-gen=true
type CleanOption struct {
	// PageSize represents the number of objects to clean at a time.
	// default is 10000
	PageSize uint64 `json:"pageSize,omitempty"`
	// RetryCount represents the number of retries in pod when the cleanup fails.
	// +kubebuilder:default=5
	RetryCount int `json:"retryCount,omitempty"`
	// BackoffEnabled represents whether to enable the backoff when a deletion API fails.
	// It is useful when the deletion API is rate limited.
	BackoffEnabled bool `json:"backoffEnabled,omitempty"`

	BatchDeleteOption `json:",inline"`

	// TODO(ideascf): remove this field, EBS volume snapshot backup is deprecated in v2
	// SnapshotsDeleteRatio represents the number of snapshots deleted per second
	// +kubebuilder:default=1
	// SnapshotsDeleteRatio float64 `json:"snapshotsDeleteRatio,omitempty"`
}

type Progress struct {
	// Step is the step name of progress
	Step string `json:"step,omitempty"`
	// Progress is the backup progress value
	Progress int `json:"progress,omitempty"` // TODO(ideascf): type changed from float64 to int
	// LastTransitionTime is the update time
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// BackupSpec contains the backup specification for a tidb cluster.
// +k8s:openapi-gen=true
// +kubebuilder:validation:XValidation:rule="has(self.logSubcommand) ? !has(self.logStop) : true",message="Field `logStop` is the old version field, please use `logSubcommand` instead"
// +kubebuilder:validation:XValidation:rule="has(self.logStop) ? !has(self.logSubcommand) : true",message="Field `logStop` is the old version field, please use `logSubcommand` instead"
type BackupSpec struct {
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
	// Type is the backup type for tidb cluster and only used when Mode = snapshot, such as full, db, table.
	Type BackupType `json:"backupType,omitempty"`
	// Mode is the backup mode, such as snapshot backup or log backup.
	// +kubebuilder:default=snapshot
	Mode BackupMode `json:"backupMode,omitempty"`
	// TikvGCLifeTime is to specify the safe gc life time for backup.
	// The time limit during which data is retained for each GC, in the format of Go Duration.
	// When a GC happens, the current time minus this value is the safe point.
	TikvGCLifeTime *string `json:"tikvGCLifeTime,omitempty"`
	// StorageProvider configures where and how backups should be stored.
	// *** Note: This field should generally not be left empty, unless you are certain the storage provider
	// *** can be obtained from another source, such as a schedule CR.
	StorageProvider `json:",inline"`
	// The storageClassName of the persistent volume for Backup data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize,omitempty"`
	// BRConfig is the configs for BR
	// *** Note: This field should generally not be left empty, unless you are certain the BR config
	// *** can be obtained from another source, such as a schedule CR.
	BR *BRConfig `json:"br,omitempty"`
	// CommitTs is the commit ts of the backup, snapshot ts for full backup or start ts for log backup.
	// Format supports TSO or datetime, e.g. '400036290571534337', '2018-05-11 01:42:23'.
	// Default is current timestamp.
	// +optional
	CommitTs string `json:"commitTs,omitempty"`
	// Subcommand is the subcommand for BR, such as start, stop, pause etc.
	// +optional
	// +kubebuilder:validation:Enum:="log-start";"log-stop";"log-pause"
	LogSubcommand LogSubCommandType `json:"logSubcommand,omitempty"`
	// LogTruncateUntil is log backup truncate until timestamp.
	// Format supports TSO or datetime, e.g. '400036290571534337', '2018-05-11 01:42:23'.
	// +optional
	LogTruncateUntil string `json:"logTruncateUntil,omitempty"`
	// LogStop indicates that will stop the log backup.
	// +optional
	LogStop bool `json:"logStop,omitempty"`
	// CalcSizeLevel determines how to size calculation of snapshots for EBS volume snapshot backup
	// +optional
	// +kubebuilder:default="all"
	CalcSizeLevel string `json:"calcSizeLevel,omitempty"`
	// FederalVolumeBackupPhase indicates which phase to execute in federal volume backup
	// +optional
	FederalVolumeBackupPhase FederalVolumeBackupPhase `json:"federalVolumeBackupPhase,omitempty"`
	// ResumeGcSchedule indicates whether resume gc and pd scheduler for EBS volume snapshot backup
	// +optional
	ResumeGcSchedule bool `json:"resumeGcSchedule,omitempty"`
	// TODO(ideascf): remove it in v2
	// DumplingConfig is the configs for dumpling
	Dumpling *DumplingConfig `json:"dumpling,omitempty"`
	// Base tolerations of backup Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// ToolImage specifies the tool image used in `Backup`, which supports BR and Dumpling images.
	// For examples `spec.toolImage: pingcap/br:v4.0.8` or `spec.toolImage: pingcap/dumpling:v4.0.8`
	// For BR image, if it does not contain tag, Pod will use image 'ToolImage:${TiKV_Version}'.
	// +optional
	ToolImage string `json:"toolImage,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// TableFilter means Table filter expression for 'db.table' matching. BR supports this from v4.0.3.
	TableFilter []string `json:"tableFilter,omitempty"`
	// Affinity of backup Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Use KMS to decrypt the secrets
	UseKMS bool `json:"useKMS,omitempty"`
	// Specify service account of backup
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// CleanPolicy denotes whether to clean backup data when the object is deleted from the cluster, if not set, the backup data will be retained
	// +kubebuilder:validation:Enum:=Retain;OnFailure;Delete
	// +kubebuilder:default=Retain
	CleanPolicy CleanPolicyType `json:"cleanPolicy,omitempty"`
	// CleanOption controls the behavior of clean.
	CleanOption *CleanOption `json:"cleanOption,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// PriorityClassName of Backup Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// BackoffRetryPolicy the backoff retry policy, currently only valid for snapshot backup
	BackoffRetryPolicy BackoffRetryPolicy `json:"backoffRetryPolicy,omitempty"`

	// Additional volumes of component pod.
	// +optional
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`
	// Additional volume mounts of component pod.
	// +optional
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
	// VolumeBackupInitJobMaxActiveSeconds represents the deadline (in seconds) of the vbk init job
	// +kubebuilder:default=600
	VolumeBackupInitJobMaxActiveSeconds int `json:"volumeBackupInitJobMaxActiveSeconds,omitempty"`
}

// FederalVolumeBackupPhase represents a phase to execute in federal volume backup
type FederalVolumeBackupPhase string

const (
	// FederalVolumeBackupInitialize means we should stop GC and PD schedule
	FederalVolumeBackupInitialize FederalVolumeBackupPhase = "initialize"
	// FederalVolumeBackupExecute means we should take volume snapshots for TiKV
	FederalVolumeBackupExecute FederalVolumeBackupPhase = "execute"
	// FederalVolumeBackupTeardown means we should resume GC and PD schedule
	FederalVolumeBackupTeardown FederalVolumeBackupPhase = "teardown"
)

// +k8s:openapi-gen=true
// DumplingConfig contains config for dumpling
type DumplingConfig struct {
	// Options means options for backup data to remote storage with dumpling.
	Options []string `json:"options,omitempty"`
	// Deprecated. Please use `Spec.TableFilter` instead. TableFilter means Table filter expression for 'db.table' matching
	TableFilter []string `json:"tableFilter,omitempty"`
}

// +k8s:openapi-gen=true
// BRConfig contains config for BR
type BRConfig struct {
	// ClusterName of backup/restore cluster
	Cluster string `json:"cluster"`
	// Namespace of backup/restore cluster
	ClusterNamespace string `json:"clusterNamespace,omitempty"`
	// Deprecated from BR v4.0.3. Please use `Spec.TableFilter` instead. DB is the specific DB which will be backed-up or restored
	DB string `json:"db,omitempty"`
	// Deprecated from BR v4.0.3. Please use `Spec.TableFilter` instead. Table is the specific table which will be backed-up or restored
	Table string `json:"table,omitempty"`
	// LogLevel is the log level
	LogLevel string `json:"logLevel,omitempty"`
	// StatusAddr is the HTTP listening address for the status report service. Set to empty string to disable
	StatusAddr string `json:"statusAddr,omitempty"`
	// Concurrency is the size of thread pool on each node that execute the backup task
	Concurrency *uint32 `json:"concurrency,omitempty"`
	// RateLimit is the rate limit of the backup task, MB/s per node
	RateLimit *uint `json:"rateLimit,omitempty"`
	// TimeAgo is the history version of the backup task, e.g. 1m, 1h
	TimeAgo string `json:"timeAgo,omitempty"`
	// Checksum specifies whether to run checksum after backup
	Checksum *bool `json:"checksum,omitempty"`
	// CheckRequirements specifies whether to check requirements
	CheckRequirements *bool `json:"checkRequirements,omitempty"`
	// SendCredToTikv specifies whether to send credentials to TiKV
	SendCredToTikv *bool `json:"sendCredToTikv,omitempty"`
	// OnLine specifies whether online during restore
	OnLine *bool `json:"onLine,omitempty"`
	// Options means options for backup data to remote storage with BR. These options has highest priority.
	Options []string `json:"options,omitempty"`
}

// BackoffRetryPolicy is the backoff retry policy, currently only valid for snapshot backup.
// When backup job or pod failed, it will retry in the following way:
// first time: retry after MinRetryDuration
// second time: retry after MinRetryDuration * 2
// third time: retry after MinRetryDuration * 2 * 2
// ...
// as the limit:
// 1. the number of retries can not exceed MaxRetryTimes
// 2. the time from discovery failure can not exceed RetryTimeout
type BackoffRetryPolicy struct {
	// MinRetryDuration is the min retry duration, the retry duration will be MinRetryDuration << (retry num -1)
	// format reference, https://golang.org/pkg/time/#ParseDuration
	// +kubebuilder:default="300s"
	MinRetryDuration string `json:"minRetryDuration,omitempty"`
	// MaxRetryTimes is the max retry times
	// +kubebuilder:default=2
	MaxRetryTimes int `json:"maxRetryTimes,omitempty"`
	// RetryTimeout is the retry timeout
	// format reference, https://golang.org/pkg/time/#ParseDuration
	// +kubebuilder:default="30m"
	RetryTimeout string `json:"retryTimeout,omitempty"`
}

// BackoffRetryRecord is the record of backoff retry
type BackoffRetryRecord struct {
	// RetryNum is the number of retry
	RetryNum int `json:"retryNum,omitempty"`
	// DetectFailedAt is the time when detect failure
	DetectFailedAt *metav1.Time `json:"detectFailedAt,omitempty"`
	// ExpectedRetryAt is the time we calculate and expect retry after it
	ExpectedRetryAt *metav1.Time `json:"expectedRetryAt,omitempty"`
	// RealRetryAt is the time when the retry was actually initiated
	RealRetryAt *metav1.Time `json:"realRetryAt,omitempty"`
	// Reason is the reason of retry
	RetryReason string `json:"retryReason,omitempty"`
	// OriginalReason is the original reason of backup job or pod failed
	OriginalReason string `json:"originalReason,omitempty"`
}

// BackupConditionType represents a valid condition of a Backup.
type BackupConditionType string

const (
	// BackupScheduled means the backup related job has been created
	BackupScheduled BackupConditionType = "Scheduled"
	// BackupRunning means the backup is currently being executed.
	BackupRunning BackupConditionType = "Running"
	// BackupComplete means the backup has successfully executed and the
	// resulting artifact has been stored in backend storage.
	BackupComplete BackupConditionType = "Complete"
	// BackupClean means the clean job has been created to clean backup data
	BackupClean BackupConditionType = "Clean"
	// BackupRepeatable should ONLY be used in log backup
	// It means some log backup sub-command completed and the log backup can be re-run
	BackupRepeatable BackupConditionType = "Repeatable"
	// BackupFailed means the backup has failed.
	BackupFailed BackupConditionType = "Failed"
	// BackupRetryTheFailed means this failure can be retried
	BackupRetryTheFailed BackupConditionType = "RetryFailed"
	// BackupCleanFailed means the clean job has failed
	BackupCleanFailed BackupConditionType = "CleanFailed"
	// BackupInvalid means invalid backup CR
	BackupInvalid BackupConditionType = "Invalid"
	// BackupPrepare means the backup prepare backup process
	BackupPrepare BackupConditionType = "Prepare"
	// BackupPaused means the backup was paused
	BackupPaused BackupConditionType = "Paused"
	// BackupStopped means the backup was stopped, just log backup has this condition
	BackupStopped BackupConditionType = "Stopped"
	// BackupRestart means the backup was restarted, now just support snapshot backup
	BackupRestart BackupConditionType = "Restart"
	// VolumeBackupInitialized means the volume backup has stopped GC and PD scheduler
	VolumeBackupInitialized BackupConditionType = "VolumeBackupInitialized"
	// VolumeBackupInitializeFailed means the volume backup initialize job failed
	VolumeBackupInitializeFailed BackupConditionType = "VolumeBackupInitializeFailed"
	// VolumeBackupSnapshotsCreated means the local volume snapshots created, and they won't be changed
	VolumeBackupSnapshotsCreated BackupConditionType = "VolumeBackupSnapshotsCreated"
	// VolumeBackupInitializeComplete means the volume backup has safely resumed GC and PD scheduler
	VolumeBackupInitializeComplete BackupConditionType = "VolumeBackupInitializeComplete"
	// VolumeBackupComplete means the volume backup has taken volume snapshots successfully
	VolumeBackupComplete BackupConditionType = "VolumeBackupComplete"
	// VolumeBackupFailed means the volume backup take volume snapshots failed
	VolumeBackupFailed BackupConditionType = "VolumeBackupFailed"
)

// BackupCondition describes the observed state of a Backup at a certain point.
type BackupCondition struct {
	Command LogSubCommandType `json:"command,omitempty"`

	metav1.Condition `json:",inline"`
	// TODO(ideascf):  remove these fields
	// Type    BackupConditionType    `json:"type"`
	// Status  corev1.ConditionStatus `json:"status"`
	// // +nullable
	// LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason             string      `json:"reason,omitempty"`
	// Message            string      `json:"message,omitempty"`
}

// LogSubCommandType is the log backup subcommand type.
type LogSubCommandType string

const (
	// LogStartCommand is the start command of log backup.
	LogStartCommand LogSubCommandType = "log-start"
	// LogTruncateCommand is the truncate command of log backup.
	LogTruncateCommand LogSubCommandType = "log-truncate"
	// LogStopCommand is the stop command of log backup.
	LogStopCommand LogSubCommandType = "log-stop"
	// LogPauseCommand is the pause command of log backup.
	LogPauseCommand LogSubCommandType = "log-pause"
	// LogResumeCommand is the resume command of log backup.
	LogResumeCommand LogSubCommandType = "log-resume"
	// LogUnknownCommand is the unknown command of log backup.
	LogUnknownCommand LogSubCommandType = "log-unknown"
)

// LogSubCommandStatus is the log backup subcommand's status.
type LogSubCommandStatus struct {
	// Command is the log backup subcommand.
	Command LogSubCommandType `json:"command,omitempty"`
	// TimeStarted is the time at which the command was started.
	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// +nullable
	TimeStarted metav1.Time `json:"timeStarted,omitempty"`
	// TimeCompleted is the time at which the command was completed.
	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// +nullable
	TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
	// LogTruncatingUntil is log backup truncate until timestamp which is used to mark the truncate command.
	LogTruncatingUntil string `json:"logTruncatingUntil,omitempty"`
	// Phase is the command current phase.
	Phase BackupConditionType `json:"phase,omitempty"`
	// +nullable
	Conditions []BackupCondition `json:"conditions,omitempty"`
}

// BackupStatus represents the current status of a backup.
type BackupStatus struct {
	// BackupPath is the location of the backup.
	BackupPath string `json:"backupPath,omitempty"`
	// TimeStarted is the time at which the backup was started.
	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// +nullable
	TimeStarted metav1.Time `json:"timeStarted,omitempty"`
	// TimeCompleted is the time at which the backup was completed.
	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// +nullable
	TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
	// TimeTaken is the time that backup takes, it is TimeCompleted - TimeStarted
	TimeTaken string `json:"timeTaken,omitempty"`
	// BackupSizeReadable is the data size of the backup.
	// the difference with BackupSize is that its format is human readable
	BackupSizeReadable string `json:"backupSizeReadable,omitempty"`
	// BackupSize is the data size of the backup.
	BackupSize int64 `json:"backupSize,omitempty"`
	// the difference with IncrementalBackupSize is that its format is human readable
	IncrementalBackupSizeReadable string `json:"incrementalBackupSizeReadable,omitempty"`
	// IncrementalBackupSize is the incremental data size of the backup, it is only used for volume snapshot backup
	// it is the real size of volume snapshot backup
	IncrementalBackupSize int64 `json:"incrementalBackupSize,omitempty"`
	// CommitTs is the commit ts of the backup, snapshot ts for full backup or start ts for log backup.
	CommitTs string `json:"commitTs,omitempty"`
	// LogSuccessTruncateUntil is log backup already successfully truncate until timestamp.
	LogSuccessTruncateUntil string `json:"logSuccessTruncateUntil,omitempty"`
	// LogCheckpointTs is the ts of log backup process.
	LogCheckpointTs string `json:"logCheckpointTs,omitempty"`
	// Phase is a user readable state inferred from the underlying Backup conditions
	Phase BackupConditionType `json:"phase,omitempty"`
	// +nullable
	// Conditions []BackupCondition `json:"conditions,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// LogSubCommandStatuses is the detail status of log backup subcommands, record each command separately, but only record the last command.
	LogSubCommandStatuses map[LogSubCommandType]LogSubCommandStatus `json:"logSubCommandStatuses,omitempty"`
	// Progresses is the progress of backup.
	// +nullable
	Progresses []Progress `json:"progresses,omitempty"`
	// BackoffRetryStatus is status of the backoff retry, it will be used when backup pod or job exited unexpectedly
	BackoffRetryStatus []BackoffRetryRecord `json:"backoffRetryStatus,omitempty"`
}
