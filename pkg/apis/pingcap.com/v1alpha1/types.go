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
	SchedulerName string   `json:"schedulerName,omitempty"`
	PD            PDSpec   `json:"pd,omitempty"`
	TiDB          TiDBSpec `json:"tidb,omitempty"`
	TiKV          TiKVSpec `json:"tikv,omitempty"`
	// +k8s:openapi-gen=false
	TiKVPromGateway TiKVPromGatewaySpec `json:"tikvPromGateway,omitempty"`
	// Services list non-headless services type used in TidbCluster
	Services        []Service                            `json:"services,omitempty"`
	PVReclaimPolicy corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`
	Timezone        string                               `json:"timezone,omitempty"`
	// Enable TLS connection between TiDB server compoments
	EnableTLSCluster bool `json:"enableTLSCluster,omitempty"`
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
	ContainerSpec
	// +k8s:openapi-gen=false
	PodAttributesSpec
	Replicas         int32  `json:"replicas"`
	StorageClassName string `json:"storageClassName,omitempty"`
}

// +k8s:openapi-gen=true
// TiDBSpec contains details of TiDB members
type TiDBSpec struct {
	// +k8s:openapi-gen=false
	ContainerSpec
	// +k8s:openapi-gen=false
	PodAttributesSpec
	Replicas         int32                 `json:"replicas"`
	StorageClassName string                `json:"storageClassName,omitempty"`
	BinlogEnabled    bool                  `json:"binlogEnabled,omitempty"`
	MaxFailoverCount int32                 `json:"maxFailoverCount,omitempty"`
	SeparateSlowLog  bool                  `json:"separateSlowLog,omitempty"`
	SlowLogTailer    TiDBSlowLogTailerSpec `json:"slowLogTailer,omitempty"`
	EnableTLSClient  bool                  `json:"enableTLSClient,omitempty"`
}

// +k8s:openapi-gen=true
// TiDBSlowLogTailerSpec represents an optional log tailer sidecar with TiDB
type TiDBSlowLogTailerSpec struct {
	// +k8s:openapi-gen=false
	ContainerSpec
}

// +k8s:openapi-gen=true
// TiKVSpec contains details of TiKV members
type TiKVSpec struct {
	// +k8s:openapi-gen=false
	ContainerSpec
	// +k8s:openapi-gen=false
	PodAttributesSpec
	Replicas         int32  `json:"replicas"`
	Privileged       bool   `json:"privileged,omitempty"`
	StorageClassName string `json:"storageClassName,omitempty"`
	MaxFailoverCount int32  `json:"maxFailoverCount,omitempty"`
}

// +k8s:openapi-gen=false
// TiKVPromGatewaySpec runs as a sidecar with TiKVSpec
type TiKVPromGatewaySpec struct {
	ContainerSpec
}

// +k8s:openapi-gen=false
// ContainerSpec is the container spec of a pod
type ContainerSpec struct {
	Image           string               `json:"image"`
	ImagePullPolicy corev1.PullPolicy    `json:"imagePullPolicy,omitempty"`
	Requests        *ResourceRequirement `json:"requests,omitempty"`
	Limits          *ResourceRequirement `json:"limits,omitempty"`
}

// +k8s:openapi-gen=false
// PodAttributesControlSpec is a spec of some general attributes of TiKV, TiDB and PD Pods
type PodAttributesSpec struct {
	Affinity           *corev1.Affinity           `json:"affinity,omitempty"`
	NodeSelector       map[string]string          `json:"nodeSelector,omitempty"`
	Tolerations        []corev1.Toleration        `json:"tolerations,omitempty"`
	Annotations        map[string]string          `json:"annotations,omitempty"`
	HostNetwork        bool                       `json:"hostNetwork,omitempty"`
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	PriorityClassName  string                     `json:"priorityClassName,omitempty"`
}

// +k8s:openapi-gen=true
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
	// BackupStorageTypeCeph represents the backend storage type is ceph.
	BackupStorageTypeCeph BackupStorageType = "ceph"
)

// +k8s:openapi-gen=true
// StorageProvider defines the configuration for storing a backup in backend storage.
type StorageProvider struct {
	Ceph *CephStorageProvider `json:"ceph"`
}

// +k8s:openapi-gen=true
// cephStorageProvider represents an ceph compatible bucket for storing backups.
type CephStorageProvider struct {
	// Region in which the ceph bucket is located.
	Region string `json:"region"`
	// Bucket in which to store the Backup.
	Bucket string `json:"bucket"`
	// Endpoint is the access address of the ceph object storage.
	Endpoint string `json:"endpoint"`
	// SecretName is the name of secret which stores
	// ceph object store access key and secret key.
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
	Type BackupType `json:"backupType"`
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
	// MaxBackups is to specify how many backups we want to keep
	// 0 is magic number to indicate un-limited backups.
	MaxBackups *int32 `json:"maxBackups"`
	// MaxReservedTime is to specify how long backups we want to keep.
	MaxReservedTime *string `json:"maxReservedTime"`
	// BackupTemplate is the specification of the backup structure to get scheduled.
	BackupTemplate BackupSpec `json:"backupTemplate"`
	// StorageClassName is the storage class for backup job's PV.
	StorageClassName string `json:"storageClassName"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize"`
}

// BackupScheduleStatus represents the current state of a BackupSchedule.
type BackupScheduleStatus struct {
	// LastBackup represents the last backup.
	LastBackup string `json:"lastBackup"`
	// LastBackupTime represents the last time the backup was successfully created.
	LastBackupTime *metav1.Time `json:"lastBackupTime"`
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
