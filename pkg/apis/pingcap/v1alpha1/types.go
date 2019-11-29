// Copyright 2018 PingCAP, Inc.
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
	"github.com/pingcap/tidb-operator/pkg/util/json"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// TiKVStateUp represents status of Up of TiKV
	TiKVStateUp string = "Up"
	// TiKVStateDown represents status of Down of TiKV
	TiKVStateDown string = "Down"
	// TiKVStateOffline represents status of Offline of TiKV
	TiKVStateOffline string = "Offline"
	// TiKVStateTombstone represents status of Tombstone of TiKV
	TiKVStateTombstone string = "Tombstone"
)

// MemberType represents member type
type MemberType string

const (
	// PDMemberType is pd container type
	PDMemberType MemberType = "pd"
	// TiDBMemberType is tidb container type
	TiDBMemberType MemberType = "tidb"
	// TiKVMemberType is tikv container type
	TiKVMemberType MemberType = "tikv"
	// SlowLogTailerMemberType is tidb log tailer container type
	SlowLogTailerMemberType MemberType = "slowlog"
	// UnknownMemberType is unknown container type
	UnknownMemberType MemberType = "unknown"
)

// MemberPhase is the current state of member
type MemberPhase string

const (
	// NormalPhase represents normal state of TiDB cluster.
	NormalPhase MemberPhase = "Normal"
	// UpgradePhase represents the upgrade state of TiDB cluster.
	UpgradePhase MemberPhase = "Upgrade"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TidbCluster is the control script's spec
type TidbCluster struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the behavior of a tidb cluster
	Spec TidbClusterSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the tidb cluster
	Status TidbClusterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TidbClusterList is TidbCluster list
type TidbClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbCluster `json:"items"`
}

// +k8s:openapi-gen=true
// TidbClusterSpec describes the attributes that a user creates on a tidb cluster
type TidbClusterSpec struct {

	// PD cluster spec
	PD PDSpec `json:"pd,omitempty"`

	// TiDB cluster spec
	TiDB TiDBSpec `json:"tidb,omitempty"`

	// TiKV cluster spec
	TiKV TiKVSpec `json:"tikv,omitempty"`

	// Pump cluster spec
	Pump *PumpSpec `json:"pump,omitempty"`

	// Helper spec
	Helper HelperSpec `json:"helper,omitempty"`

	// Services list non-headless services type used in TidbCluster
	// Deprecated
	Services []Service `json:"services,omitempty"`

	// TiDB cluster version
	Version string `json:"version,omitempty"`

	// SchedulerName of TiDB cluster Pods
	SchedulerName string `json:"schedulerName,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster
	PVReclaimPolicy corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// Whether enable PVC reclaim for orphan PVC left by statefulset scale-in
	EnablePVReclaim bool `json:"enablePVReclaim,omitempty"`

	// Enable TLS connection between TiDB server components
	EnableTLSCluster bool `json:"enableTLSCluster,omitempty"`

	// Time zone of TiDB cluster Pods
	Timezone string `json:"timezone,omitempty"`

	// ImagePullPolicy of TiDB cluster Pods
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Whether Hostnetwork is enabled for TiDB cluster Pods
	HostNetwork bool `json:"hostNetwork,omitempty"`

	// Affinity of TiDB cluster Pods
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of TiDB cluster Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Base node selectors of TiDB cluster Pods, components may add or override selectors upon this respectively
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Base annotations of TiDB cluster Pods, components may add or override selectors upon this respectively
	Annotations map[string]string `json:"annotations,omitempty"`

	// Base tolerations of TiDB cluster Pods, components may add more tolreations upon this respectively
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// TidbClusterStatus represents the current status of a tidb cluster.
type TidbClusterStatus struct {
	ClusterID string     `json:"clusterID,omitempty"`
	PD        PDStatus   `json:"pd,omitempty"`
	TiKV      TiKVStatus `json:"tikv,omitempty"`
	TiDB      TiDBStatus `json:"tidb,omitempty"`
}

// +k8s:openapi-gen=true
// PDSpec contains details of PD members
type PDSpec struct {
	// +k8s:openapi-gen=false
	ComponentSpec
	// +k8s:openapi-gen=false
	Resources
	Replicas int32 `json:"replicas"`
	// +k8s:openapi-gen=false
	Service          *ServiceSpec `json:"service,omitempty"`
	StorageClassName string       `json:"storageClassName,omitempty"`

	// +k8s:openapi-gen=false
	// TODO: add schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +k8s:openapi-gen=true
// TiKVSpec contains details of TiKV members
type TiKVSpec struct {
	// +k8s:openapi-gen=false
	ComponentSpec
	// +k8s:openapi-gen=false
	Resources
	Replicas int32 `json:"replicas"`
	// +k8s:openapi-gen=false
	Service          *ServiceSpec `json:"service,omitempty"`
	Privileged       bool         `json:"privileged,omitempty"`
	StorageClassName string       `json:"storageClassName,omitempty"`
	MaxFailoverCount int32        `json:"maxFailoverCount,omitempty"`

	// +k8s:openapi-gen=false
	// TODO: add schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +k8s:openapi-gen=true
// TiDBSpec contains details of TiDB members
type TiDBSpec struct {
	// +k8s:openapi-gen=false
	ComponentSpec
	// +k8s:openapi-gen=false
	Resources
	Replicas int32 `json:"replicas"`
	// +k8s:openapi-gen=false
	Service          *TiDBServiceSpec `json:"service,omitempty"`
	BinlogEnabled    bool             `json:"binlogEnabled,omitempty"`
	MaxFailoverCount int32            `json:"maxFailoverCount,omitempty"`
	SeparateSlowLog  bool             `json:"separateSlowLog,omitempty"`
	StorageClassName string           `json:"storageClassName,omitempty"`
	// +k8s:openapi-gen=false
	SlowLogTailer   TiDBSlowLogTailerSpec `json:"slowLogTailer,omitempty"`
	EnableTLSClient bool                  `json:"enableTLSClient,omitempty"`

	// Plugins is a list of plugins that are loaded by TiDB server, empty means plugin disabled
	Plugins []string `json:"plugins,omitempty"`

	// +k8s:openapi-gen=false
	// TODO: add schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +k8s:openapi-gen=true
// PumpSpec contains details of Pump members
type PumpSpec struct {
	// +k8s:openapi-gen=false
	ComponentSpec
	// +k8s:openapi-gen=false
	Resources

	StorageClassName string `json:"storageClassName,omitempty"`
	Replicas         int32  `json:"replicas"`

	// +k8s:openapi-gen=false
	// TODO: add schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +k8s:openapi-gen=true
// HelperSpec contains details of helper component
type HelperSpec struct {
	// Image used to tail slow log and set kernel parameters if necessary, must have `tail` and `sysctl` installed
	Image string `json:"image,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// +k8s:openapi-gen=true
// TiDBSlowLogTailerSpec represents an optional log tailer sidecar with TiDB
type TiDBSlowLogTailerSpec struct {
	// +k8s:openapi-gen=false
	Resources

	// Image used for slowlog tailer
	// Deprecated, use TidbCluster.HelperImage instead
	Image string `json:"image,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	// Deprecated, use TidbCluster.HelperImagePullPolicy instead
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// +k8s:openapi-gen=true
type Resources struct {
	// Resource requests of the component
	Requests *ResourceRequirement `json:"requests,omitempty"`

	// Resource limits of the component
	Limits *ResourceRequirement `json:"limits,omitempty"`
}

// +k8s:openapi-gen=true
// ComponentSpec is the base spec of each component, the fields should always accessed by the Basic<Component>Spec() method to respect the cluster-level properties
type ComponentSpec struct {
	// Image of the component, override baseImage and version if present
	// Deprecated
	Image string `json:"image,omitempty"`

	// Base image of the component, e.g. pingcap/tidb, image tag is now allowed during validation
	BaseImage string `json:"baseImage,omitempty"`

	// Version of the component. Override the cluster-level version if non-empty
	Version string `json:"version,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Whether Hostnetwork of the component is enabled. Override the cluster-level setting if present
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of the component. Override the cluster-level one if present
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of the component. Override the cluster-level one if present
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// SchedulerName of the component. Override the cluster-level one if present
	SchedulerName string `json:"schedulerName,omitempty"`

	// NodeSelector of the component. Merged into the cluster-level nodeSelector if non-empty
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Annotations of the component. Merged into the cluster-level annotations if non-empty
	Annotations map[string]string `json:"annotations,omitempty"`

	// Tolerations of the component. Override the cluster-level tolerations if non-empty
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PodSecurityContext of the component
	// TODO: make this configurable at cluster level
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// +k8s:openapi-gen=true
type ServiceSpec struct {
	// Type of the real kubernetes service, e.g. ClusterIP
	Type corev1.ServiceType `json:"type,omitempty"`

	// Additional annotations of the kubernetes service object
	Annotations map[string]string `json:"annotations,omitempty"`

	// LoadBalancerIP is the loadBalancerIP of service
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
}

// +k8s:openapi-gen=true
type TiDBServiceSpec struct {
	// +k8s:openapi-gen=false
	ServiceSpec

	// ExternalTrafficPolicy of the service
	ExternalTrafficPolicy corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`

	// Whether expose the status port
	ExposeStatus bool `json:"exposeStatus,omitempty"`
}

// +k8s:openapi-gen=true
// Deprecated
// Service represent service type used in TidbCluster
type Service struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// +k8s:openapi-gen=true
// ResourceRequirement is resource requirements for a pod
type ResourceRequirement struct {
	// CPU is how many cores a pod requires
	CPU string `json:"cpu,omitempty"`
	// Memory is how much memory a pod requires
	Memory string `json:"memory,omitempty"`
	// Storage is storage size a pod requires
	Storage string `json:"storage,omitempty"`
}

// PDStatus is PD status
type PDStatus struct {
	Synced         bool                       `json:"synced,omitempty"`
	Phase          MemberPhase                `json:"phase,omitempty"`
	StatefulSet    *apps.StatefulSetStatus    `json:"statefulSet,omitempty"`
	Members        map[string]PDMember        `json:"members,omitempty"`
	Leader         PDMember                   `json:"leader,omitempty"`
	FailureMembers map[string]PDFailureMember `json:"failureMembers,omitempty"`
}

// PDMember is PD member
type PDMember struct {
	Name string `json:"name"`
	// member id is actually a uint64, but apimachinery's json only treats numbers as int64/float64
	// so uint64 may overflow int64 and thus convert to float64
	ID        string `json:"id"`
	ClientURL string `json:"clientURL"`
	Health    bool   `json:"health"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// PDFailureMember is the pd failure member information
type PDFailureMember struct {
	PodName       string      `json:"podName,omitempty"`
	MemberID      string      `json:"memberID,omitempty"`
	PVCUID        types.UID   `json:"pvcUID,omitempty"`
	MemberDeleted bool        `json:"memberDeleted,omitempty"`
	CreatedAt     metav1.Time `json:"createdAt,omitempty"`
}

// TiDBStatus is TiDB status
type TiDBStatus struct {
	Phase                    MemberPhase                  `json:"phase,omitempty"`
	StatefulSet              *apps.StatefulSetStatus      `json:"statefulSet,omitempty"`
	Members                  map[string]TiDBMember        `json:"members,omitempty"`
	FailureMembers           map[string]TiDBFailureMember `json:"failureMembers,omitempty"`
	ResignDDLOwnerRetryCount int32                        `json:"resignDDLOwnerRetryCount,omitempty"`
}

// TiDBMember is TiDB member
type TiDBMember struct {
	Name   string `json:"name"`
	Health bool   `json:"health"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Node hosting pod of this TiDB member.
	NodeName string `json:"node,omitempty"`
}

// TiDBFailureMember is the tidb failure member information
type TiDBFailureMember struct {
	PodName   string      `json:"podName,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// TiKVStatus is TiKV status
type TiKVStatus struct {
	Synced          bool                        `json:"synced,omitempty"`
	Phase           MemberPhase                 `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore        `json:"stores,omitempty"`
	TombstoneStores map[string]TiKVStore        `json:"tombstoneStores,omitempty"`
	FailureStores   map[string]TiKVFailureStore `json:"failureStores,omitempty"`
}

// TiKVStores is either Up/Down/Offline/Tombstone
type TiKVStore struct {
	// store id is also uint64, due to the same reason as pd id, we store id as string
	ID                string      `json:"id"`
	PodName           string      `json:"podName"`
	IP                string      `json:"ip"`
	LeaderCount       int32       `json:"leaderCount"`
	State             string      `json:"state"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// TiKVFailureStore is the tikv failure store information
type TiKVFailureStore struct {
	PodName   string      `json:"podName,omitempty"`
	StoreID   string      `json:"storeID,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// Backup is a backup of tidb cluster.
type Backup struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec BackupSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status BackupStatus `json:"status"`
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

// +k8s:openapi-gen=true
// StorageProvider defines the configuration for storing a backup in backend storage.
type StorageProvider struct {
	S3  *S3StorageProvider  `json:"s3,omitempty"`
	Gcs *GcsStorageProvider `json:"gcs,omitempty"`
}

// +k8s:openapi-gen=true
// S3StorageProvider represents a S3 compliant storage for storing backups.
type S3StorageProvider struct {
	// Provider represents the specific storage provider that implements the S3 interface
	Provider S3StorageProviderType `json:"provider"`
	// Region in which the S3 compatible bucket is located.
	Region string `json:"region,omitempty"`
	// Bucket in which to store the Backup.
	Bucket string `json:"bucket,omitempty"`
	// Endpoint of S3 compatible storage service
	Endpoint string `json:"endpoint,omitempty"`
	// StorageClass represents the storage class
	StorageClass string `json:"storageClass,omitempty"`
	// Acl represents access control permissions for this bucket
	Acl string `json:"acl,omitempty"`
	// SecretName is the name of secret which stores
	// S3 compliant storage access key and secret key.
	SecretName string `json:"secretName"`
}

// +k8s:openapi-gen=true
// GcsStorageProvider represents the google cloud storage for storing backups.
type GcsStorageProvider struct {
	// ProjectId represents the project that organizes all your Google Cloud Platform resources
	ProjectId string `json:"projectId"`
	// Location in which the gcs bucket is located.
	Location string `json:"location,omitempty"`
	// Bucket in which to store the Backup.
	Bucket string `json:"bucket,omitempty"`
	// StorageClass represents the storage class
	StorageClass string `json:"storageClass,omitempty"`
	// ObjectAcl represents the access control list for new objects
	ObjectAcl string `json:"objectAcl,omitempty"`
	// BucketAcl represents the access control list for new buckets
	BucketAcl string `json:"bucketAcl,omitempty"`
	// SecretName is the name of secret which stores the
	// gcs service account credentials JSON .
	SecretName string `json:"secretName"`
}

// +k8s:openapi-gen=true
// BackupType represents the backup type.
type BackupType string

const (
	// BackupTypeFull represents the full backup of tidb cluster.
	BackupTypeFull BackupType = "full"
	// BackupTypeInc represents the incremental backup of tidb cluster.
	BackupTypeInc BackupType = "incremental"
)

// +k8s:openapi-gen=true
// BackupSpec contains the backup specification for a tidb cluster.
type BackupSpec struct {
	// Cluster is the Cluster to backup.
	Cluster string `json:"cluster"`
	// TidbSecretName is the name of secret which stores
	// tidb cluster's username and password.
	TidbSecretName string `json:"tidbSecretName"`
	// Type is the backup type for tidb cluster.
	Type BackupType `json:"backupType,omitempty"`
	// StorageType is the backup storage type.
	StorageType BackupStorageType `json:"storageType"`
	// StorageProvider configures where and how backups should be stored.
	StorageProvider `json:",inline"`
	// StorageClassName is the storage class for backup job's PV.
	StorageClassName string `json:"storageClassName"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize"`
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
	// BackupFailed means the backup has failed.
	BackupFailed BackupConditionType = "Failed"
	// BackupRetryFailed means this failure can be retried
	BackupRetryFailed BackupConditionType = "RetryFailed"
)

// BackupCondition describes the observed state of a Backup at a certain point.
type BackupCondition struct {
	Type               BackupConditionType    `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
}

// BackupStatus represents the current status of a backup.
type BackupStatus struct {
	// BackupPath is the location of the backup.
	BackupPath string `json:"backupPath"`
	// TimeStarted is the time at which the backup was started.
	TimeStarted metav1.Time `json:"timeStarted"`
	// TimeCompleted is the time at which the backup was completed.
	TimeCompleted metav1.Time `json:"timeCompleted"`
	// BackupSize is the data size of the backup.
	BackupSize int64 `json:"backupSize"`
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs   string            `json:"commitTs"`
	Conditions []BackupCondition `json:"conditions"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// BackupSchedule is a backup schedule of tidb cluster.
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
	MaxBackups *int32 `json:"maxBackups,omitempty"`
	// MaxReservedTime is to specify how long backups we want to keep.
	MaxReservedTime *string `json:"maxReservedTime,omitempty"`
	// BackupTemplate is the specification of the backup structure to get scheduled.
	BackupTemplate BackupSpec `json:"backupTemplate"`
	// StorageClassName is the storage class for backup job's PV.
	StorageClassName string `json:"storageClassName,omitempty"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize,omitempty"`
}

// BackupScheduleStatus represents the current state of a BackupSchedule.
type BackupScheduleStatus struct {
	// LastBackup represents the last backup.
	LastBackup string `json:"lastBackup"`
	// LastBackupTime represents the last time the backup was successfully created.
	LastBackupTime *metav1.Time `json:"lastBackupTime"`
	// AllBackupCleanTime represents the time when all backup entries are cleaned up
	AllBackupCleanTime *metav1.Time `json:"allBackupCleanTime"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// Restore represents the restoration of backup of a tidb cluster.
type Restore struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec RestoreSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status RestoreStatus `json:"status"`
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

// RestoreConditionType represents a valid condition of a Restore.
type RestoreConditionType string

const (
	// RestoreScheduled means the restore job has been created to do tidb cluster restore
	RestoreScheduled RestoreConditionType = "Scheduled"
	// RestoreRunning means the Restore is currently being executed.
	RestoreRunning RestoreConditionType = "Running"
	// RestoreComplete means the Restore has successfully executed and the
	// backup data has been loaded into tidb cluster.
	RestoreComplete RestoreConditionType = "Complete"
	// RestoreFailed means the Restore has failed.
	RestoreFailed RestoreConditionType = "Failed"
	// RestoreRetryFailed means this failure can be retried
	RestoreRetryFailed RestoreConditionType = "RetryFailed"
)

// RestoreCondition describes the observed state of a Restore at a certain point.
type RestoreCondition struct {
	Type               RestoreConditionType   `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
}

// +k8s:openapi-gen=true
// RestoreSpec contains the specification for a restore of a tidb cluster backup.
type RestoreSpec struct {
	// Cluster represents the tidb cluster to be restored.
	Cluster string `json:"cluster"`
	// Backup represents the backup object to be restored.
	Backup string `json:"backup"`
	// Namespace is the namespace of the backup.
	BackupNamespace string `json:"backupNamespace"`
	// SecretName is the name of the secret which stores
	// tidb cluster's username and password.
	TidbSecretName string `json:"tidbSecretName"`
	// StorageClassName is the storage class for restore job's PV.
	StorageClassName string `json:"storageClassName"`
	// StorageSize is the request storage size for restore job
	StorageSize string `json:"storageSize"`
}

// RestoreStatus represents the current status of a tidb cluster restore.
type RestoreStatus struct {
	// TimeStarted is the time at which the restore was started.
	TimeStarted metav1.Time `json:"timeStarted"`
	// TimeCompleted is the time at which the restore was completed.
	TimeCompleted metav1.Time        `json:"timeCompleted"`
	Conditions    []RestoreCondition `json:"conditions"`
}
