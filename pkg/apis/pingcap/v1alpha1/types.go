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
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
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

	// DMWorkerStateFree represents status of free of dm-worker
	DMWorkerStateFree string = "free"
	// DMWorkerStateBound represents status of bound of dm-worker
	DMWorkerStateBound string = "bound"
	// DMWorkerStateOffline represents status of offline of dm-worker
	DMWorkerStateOffline string = "offline"

	// PumpStateOnline represents status of online of Pump
	PumpStateOnline string = "online"
	// PumpStateOffline represents status of offline of Pump
	PumpStateOffline string = "offline"
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
	// TiFlashMemberType is tiflash container type
	TiFlashMemberType MemberType = "tiflash"
	// TiCDCMemberType is ticdc container type
	TiCDCMemberType MemberType = "ticdc"
	// PumpMemberType is pump container type
	PumpMemberType MemberType = "pump"
	// DMMasterMemberType is dm-master container type
	DMMasterMemberType MemberType = "dm-master"
	// DMWorkerMemberType is dm-worker container type
	DMWorkerMemberType MemberType = "dm-worker"
	// SlowLogTailerMemberType is tidb slow log tailer container type
	SlowLogTailerMemberType MemberType = "slowlog"
	// RocksDBLogTailerMemberType is tikv rocksdb log tailer container type
	RocksDBLogTailerMemberType MemberType = "rocksdblog"
	// RaftLogTailerMemberType is tikv raft log tailer container type
	RaftLogTailerMemberType MemberType = "raftlog"
	// TidbMonitorMemberType is tidbmonitor type
	TidbMonitorMemberType MemberType = "tidbmonitor"
	// NGMonitoringMemberType is ng monitoring type
	NGMonitoringMemberType MemberType = "ng-monitoring"
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
	// ScalePhase represents the scaling state of TiDB cluster.
	ScalePhase MemberPhase = "Scale"
)

// ConfigUpdateStrategy represents the strategy to update configuration
type ConfigUpdateStrategy string

const (
	// ConfigUpdateStrategyInPlace update the configmap without changing the name
	ConfigUpdateStrategyInPlace ConfigUpdateStrategy = "InPlace"
	// ConfigUpdateStrategyRollingUpdate generate different configmap on configuration update and
	// try to rolling-update the pod controller (e.g. statefulset) to apply updates.
	ConfigUpdateStrategyRollingUpdate ConfigUpdateStrategy = "RollingUpdate"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbCluster is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="tc"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="PD",type=string,JSONPath=`.status.pd.image`,description="The image for PD cluster"
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.pd.requests.storage`,description="The storage size specified for PD node"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.pd.statefulSet.readyReplicas`,description="The desired replicas number of PD cluster"
// +kubebuilder:printcolumn:name="Desire",type=integer,JSONPath=`.spec.pd.replicas`,description="The desired replicas number of PD cluster"
// +kubebuilder:printcolumn:name="TiKV",type=string,JSONPath=`.status.tikv.image`,description="The image for TiKV cluster"
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.tikv.requests.storage`,description="The storage size specified for TiKV node"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.tikv.statefulSet.readyReplicas`,description="The ready replicas number of TiKV cluster"
// +kubebuilder:printcolumn:name="Desire",type=integer,JSONPath=`.spec.tikv.replicas`,description="The desired replicas number of TiKV cluster"
// +kubebuilder:printcolumn:name="TiDB",type=string,JSONPath=`.status.tidb.image`,description="The image for TiDB cluster"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.tidb.statefulSet.readyReplicas`,description="The ready replicas number of TiDB cluster"
// +kubebuilder:printcolumn:name="Desire",type=integer,JSONPath=`.spec.tidb.replicas`,description="The desired replicas number of TiDB cluster"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +genclient:noStatus
type TidbCluster struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the behavior of a tidb cluster
	Spec TidbClusterSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the tidb cluster
	Status TidbClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbClusterList is TidbCluster list
// +k8s:openapi-gen=true
type TidbClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbCluster `json:"items"`
}

// TidbClusterSpec describes the attributes that a user creates on a tidb cluster
// +k8s:openapi-gen=true
type TidbClusterSpec struct {
	// Discovery spec
	Discovery DiscoverySpec `json:"discovery,omitempty"`

	// Specify a Service Account
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// PD cluster spec
	// +optional
	PD *PDSpec `json:"pd,omitempty"`

	// TiDB cluster spec
	// +optional
	TiDB *TiDBSpec `json:"tidb,omitempty"`

	// TiKV cluster spec
	// +optional
	TiKV *TiKVSpec `json:"tikv,omitempty"`

	// TiFlash cluster spec
	// +optional
	TiFlash *TiFlashSpec `json:"tiflash,omitempty"`

	// TiCDC cluster spec
	// +optional
	TiCDC *TiCDCSpec `json:"ticdc,omitempty"`

	// Pump cluster spec
	// +optional
	Pump *PumpSpec `json:"pump,omitempty"`

	// Helper spec
	// +optional
	Helper *HelperSpec `json:"helper,omitempty"`

	// Indicates that the tidb cluster is paused and will not be processed by
	// the controller.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// TiDB cluster version
	// +optional
	Version string `json:"version"`
	// TODO: remove optional after defaulting logic introduced

	// SchedulerName of TiDB cluster Pods
	SchedulerName string `json:"schedulerName,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster
	// +kubebuilder:default=Retain
	PVReclaimPolicy *corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// ImagePullPolicy of TiDB cluster Pods
	// +kubebuilder:default=IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
	// UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
	// cluster component is needed to reload the configuration change.
	// UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
	// related components to use the new ConfigMap, that is, the new configuration will be applied automatically.
	ConfigUpdateStrategy ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// Whether enable PVC reclaim for orphan PVC left by statefulset scale-in
	// Optional: Defaults to false
	// +optional
	EnablePVReclaim *bool `json:"enablePVReclaim,omitempty"`

	// Whether enable the TLS connection between TiDB server components
	// Optional: Defaults to nil
	// +optional
	TLSCluster *TLSCluster `json:"tlsCluster,omitempty"`

	// Whether Hostnetwork is enabled for TiDB cluster Pods
	// Optional: Defaults to false
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of TiDB cluster Pods.
	// Will be overwritten by each cluster component's specific affinity setting, e.g. `spec.tidb.affinity`
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of TiDB cluster Pods
	// Optional: Defaults to omitted
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Base node selectors of TiDB cluster Pods, components may add or override selectors upon this respectively
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Base annotations for TiDB cluster, all Pods in the cluster should have these annotations.
	// Can be overrode by annotations in the specific component spec.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Base labels for TiDB cluster, all Pods in the cluster should have these labels.
	// Can be overrode by labels in the specific component spec.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Base tolerations of TiDB cluster Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// DNSConfig Specifies the DNS parameters of a pod.
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`

	// DNSPolicy Specifies the DNSPolicy parameters of a pod.
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Time zone of TiDB cluster Pods
	// Optional: Defaults to UTC
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// (Deprecated) Services list non-headless services type used in TidbCluster
	// +k8s:openapi-gen=false
	Services []Service `json:"services,omitempty"`
	// TODO: really deprecate this in code

	// EnableDynamicConfiguration indicates whether to append `--advertise-status-addr` to the startup parameters of TiKV.
	// +optional
	EnableDynamicConfiguration *bool `json:"enableDynamicConfiguration,omitempty"`
	// TODO: rename this into tikv-specific config name

	// ClusterDomain is the Kubernetes Cluster Domain of TiDB cluster
	// Optional: Defaults to ""
	// +optional
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// AcrossK8s indicates whether deploy TiDB cluster across multiple Kubernetes clusters
	// +optional
	AcrossK8s bool `json:"acrossK8s,omitempty"`

	// Cluster is the external cluster, if configured, the components in this TidbCluster will join to this configured cluster.
	// +optional
	Cluster *TidbClusterRef `json:"cluster,omitempty"`

	// PDAddresses are the external PD addresses, if configured, the PDs in this TidbCluster will join to the configured PD cluster.
	// +optional
	PDAddresses []string `json:"pdAddresses,omitempty"`

	// StatefulSetUpdateStrategy of TiDB cluster StatefulSets
	// +optional
	StatefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType `json:"statefulSetUpdateStrategy,omitempty"`

	// PodManagementPolicy of TiDB cluster StatefulSets
	// +optional
	PodManagementPolicy apps.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// TopologySpreadConstraints describes how a group of pods ought to spread across topology
	// domains. Scheduler will schedule pods in a way which abides by the constraints.
	// This field is is only honored by clusters that enables the EvenPodsSpread feature.
	// All topologySpreadConstraints are ANDed.
	// +optional
	// +listType=map
	// +listMapKey=topologyKey
	TopologySpreadConstraints []TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// TidbClusterStatus represents the current status of a tidb cluster.
type TidbClusterStatus struct {
	ClusterID  string                    `json:"clusterID,omitempty"`
	PD         PDStatus                  `json:"pd,omitempty"`
	TiKV       TiKVStatus                `json:"tikv,omitempty"`
	TiDB       TiDBStatus                `json:"tidb,omitempty"`
	Pump       PumpStatus                `json:"pump,omitempty"`
	TiFlash    TiFlashStatus             `json:"tiflash,omitempty"`
	TiCDC      TiCDCStatus               `json:"ticdc,omitempty"`
	AutoScaler *TidbClusterAutoScalerRef `json:"auto-scaler,omitempty"`
	// Represents the latest available observations of a tidb cluster's state.
	// +optional
	// +nullable
	Conditions []TidbClusterCondition `json:"conditions,omitempty"`
}

// TidbClusterCondition describes the state of a tidb cluster at a certain point.
type TidbClusterCondition struct {
	// Type of the condition.
	Type TidbClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	// +nullable
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// TidbClusterConditionType represents a tidb cluster condition value.
type TidbClusterConditionType string

const (
	// TidbClusterReady indicates that the tidb cluster is ready or not.
	// This is defined as:
	// - All statefulsets are up to date (currentRevision == updateRevision).
	// - All PD members are healthy.
	// - All TiDB pods are healthy.
	// - All TiKV stores are up.
	// - All TiFlash stores are up.
	TidbClusterReady TidbClusterConditionType = "Ready"
)

// The `Type` of the component condition
const (
	// ComponentVolumeResizing indicates that any volume of this component is resizing.
	ComponentVolumeResizing string = "ComponentVolumeResizing"
)

// +k8s:openapi-gen=true
// DiscoverySpec contains details of Discovery members
type DiscoverySpec struct {
	*ComponentSpec              `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`
}

// +k8s:openapi-gen=true
// PDSpec contains details of PD members
type PDSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for pd
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/pd
	// +optional
	BaseImage string `json:"baseImage"`

	// Service defines a Kubernetes service of PD cluster.
	// Optional: Defaults to `.spec.services` in favor of backward compatibility
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The storageClassName of the persistent volume for PD data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// StorageVolumes configure additional storage for PD pods.
	// +optional
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// Subdirectory within the volume to store PD Data. By default, the data
	// is stored in the root directory of volume which is mounted at
	// /var/lib/pd.
	// Specifying this will change the data directory to a subdirectory, e.g.
	// /var/lib/pd/data if you set the value to "data".
	// It's dangerous to change this value for a running cluster as it will
	// upgrade your cluster to use a new storage directory.
	// Defaults to "" (volume's root).
	// +optional
	DataSubDir string `json:"dataSubDir,omitempty"`

	// Config is the Configuration of pd-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *PDConfigWraper `json:"config,omitempty"`

	// TLSClientSecretName is the name of secret which stores tidb server client certificate
	// which used by Dashboard.
	// +optional
	TLSClientSecretName *string `json:"tlsClientSecretName,omitempty"`

	// (Deprecated) EnableDashboardInternalProxy would directly set `internal-proxy` in the `PdConfig`.
	// Note that this is deprecated, we should just set `dashboard.internal-proxy` in `pd.config`.
	// +optional
	EnableDashboardInternalProxy *bool `json:"enableDashboardInternalProxy,omitempty"`

	// MountClusterClientSecret indicates whether to mount `cluster-client-secret` to the Pod
	// +optional
	MountClusterClientSecret *bool `json:"mountClusterClientSecret,omitempty"`
	// Start up script version
	// +optional
	// +kubebuilder:validation:Enum:="";"v1"
	StartUpScriptVersion string `json:"startUpScriptVersion,omitempty"`
}

// TiKVSpec contains details of TiKV members
// +k8s:openapi-gen=true
type TiKVSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for tikv
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tikv
	// +optional
	BaseImage string `json:"baseImage"`

	// Whether create the TiKV container in privileged mode, it is highly discouraged to enable this in
	// critical environment.
	// Optional: defaults to false
	// +optional
	Privileged *bool `json:"privileged,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// Whether output the RocksDB log in a separate sidecar container
	// Optional: Defaults to false
	// +optional
	SeparateRocksDBLog *bool `json:"separateRocksDBLog,omitempty"`

	// Whether output the Raft log in a separate sidecar container
	// Optional: Defaults to false
	// +optional
	SeparateRaftLog *bool `json:"separateRaftLog,omitempty"`

	// Optional volume name configuration for rocksdb log.
	// +optional
	RocksDBLogVolumeName string `json:"rocksDBLogVolumeName,omitempty"`

	// Optional volume name configuration for raft log.
	// +optional
	RaftLogVolumeName string `json:"raftLogVolumeName,omitempty"`

	// LogTailer is the configurations of the log tailers for TiKV
	// +optional
	LogTailer *LogTailerSpec `json:"logTailer,omitempty"`

	// The storageClassName of the persistent volume for TiKV data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Subdirectory within the volume to store TiKV Data. By default, the data
	// is stored in the root directory of volume which is mounted at
	// /var/lib/tikv.
	// Specifying this will change the data directory to a subdirectory, e.g.
	// /var/lib/tikv/data if you set the value to "data".
	// It's dangerous to change this value for a running cluster as it will
	// upgrade your cluster to use a new storage directory.
	// Defaults to "" (volume's root).
	// +optional
	DataSubDir string `json:"dataSubDir,omitempty"`

	// Config is the Configuration of tikv-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *TiKVConfigWraper `json:"config,omitempty"`

	// RecoverFailover indicates that Operator can recover the failed Pods
	// +optional
	RecoverFailover bool `json:"recoverFailover,omitempty"`

	// Failover is the configurations of failover
	// +optional
	Failover *Failover `json:"failover,omitempty"`

	// MountClusterClientSecret indicates whether to mount `cluster-client-secret` to the Pod
	// +optional
	MountClusterClientSecret *bool `json:"mountClusterClientSecret,omitempty"`

	// EvictLeaderTimeout indicates the timeout to evict tikv leader, in the format of Go Duration.
	// Defaults to 10m
	// +optional
	EvictLeaderTimeout *string `json:"evictLeaderTimeout,omitempty"`

	// StorageVolumes configure additional storage for TiKV pods.
	// +optional
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// StoreLabels configures additional labels for TiKV stores.
	// +optional
	StoreLabels []string `json:"storeLabels,omitempty"`

	// EnableNamedStatusPort enables status port(20180) in the Pod spec.
	// If you set it to `true` for an existing cluster, the TiKV cluster will be rolling updated.
	EnableNamedStatusPort bool `json:"enableNamedStatusPort,omitempty"`
}

// TiFlashSpec contains details of TiFlash members
// +k8s:openapi-gen=true
type TiFlashSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for TiFlash
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tiflash
	// +optional
	BaseImage string `json:"baseImage"`

	// Whether create the TiFlash container in privileged mode, it is highly discouraged to enable this in
	// critical environment.
	// Optional: defaults to false
	// +optional
	Privileged *bool `json:"privileged,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The persistent volume claims of the TiFlash data storages.
	// TiFlash supports multiple disks.
	StorageClaims []StorageClaim `json:"storageClaims"`

	// Config is the Configuration of TiFlash
	// +optional
	Config *TiFlashConfigWraper `json:"config,omitempty"`

	// Initializer is the configurations of the init container for TiFlash
	//
	// +optional
	Initializer *InitContainerSpec `json:"initializer,omitempty"`

	// LogTailer is the configurations of the log tailers for TiFlash
	// +optional
	LogTailer *LogTailerSpec `json:"logTailer,omitempty"`

	// RecoverFailover indicates that Operator can recover the failover Pods
	// +optional
	RecoverFailover bool `json:"recoverFailover,omitempty"`

	// Failover is the configurations of failover
	// +optional
	Failover *Failover `json:"failover,omitempty"`
}

// TiCDCSpec contains details of TiCDC members
// +k8s:openapi-gen=true
type TiCDCSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for TiCDC
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// TLSClientSecretNames are the names of secrets that store the
	// client certificates for the downstream.
	// +optional
	TLSClientSecretNames []string `json:"tlsClientSecretNames,omitempty"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/ticdc
	// +optional
	BaseImage string `json:"baseImage"`

	// Config is the Configuration of tidbcdc servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *CDCConfigWraper `json:"config,omitempty"`

	// StorageVolumes configure additional storage for TiCDC pods.
	// +optional
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// The storageClassName of the persistent volume for TiCDC data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// TiCDCConfig is the configuration of tidbcdc
// ref https://github.com/pingcap/ticdc/blob/a28d9e43532edc4a0380f0ef87314631bf18d866/pkg/config/config.go#L176
// +k8s:openapi-gen=true
type TiCDCConfig struct {
	// Time zone of TiCDC
	// Optional: Defaults to UTC
	// +optional
	Timezone *string `toml:"tz,omitempty" json:"timezone,omitempty"`

	// CDC GC safepoint TTL duration, specified in seconds
	// Optional: Defaults to 86400
	// +optional
	GCTTL *int32 `toml:"gc-ttl,omitempty" json:"gcTTL,omitempty"`

	// LogLevel is the log level
	// Optional: Defaults to info
	// +optional
	LogLevel *string `toml:"log-level,omitempty" json:"logLevel,omitempty"`

	// LogFile is the log file
	// Optional: Defaults to /dev/stderr
	// +optional
	LogFile *string `toml:"log-file,omitempty" json:"logFile,omitempty"`
}

// LogTailerSpec represents an optional log tailer sidecar container
// +k8s:openapi-gen=true
type LogTailerSpec struct {
	corev1.ResourceRequirements `json:",inline"`
}

// InitContainerSpec contains basic spec about a init container
//
// +k8s:openapi-gen=true
type InitContainerSpec struct {
	corev1.ResourceRequirements `json:",inline"`
}

// StorageClaim contains details of TiFlash storages
// +k8s:openapi-gen=true
type StorageClaim struct {
	// Resources represents the minimum resources the volume should have.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Name of the StorageClass required by the claim.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// TiDBSpec contains details of TiDB members
// +k8s:openapi-gen=true
type TiDBSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for tidb
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tidb
	// +optional
	BaseImage string `json:"baseImage"`

	// Service defines a Kubernetes service of TiDB cluster.
	// Optional: No kubernetes service will be created by default.
	// +optional
	Service *TiDBServiceSpec `json:"service,omitempty"`

	// Whether enable TiDB Binlog, it is encouraged to not set this field and rely on the default behavior
	// Optional: Defaults to true if PumpSpec is non-nil, otherwise false
	// +optional
	BinlogEnabled *bool `json:"binlogEnabled,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// Whether output the slow log in an separate sidecar container
	// Optional: Defaults to true
	// +optional
	SeparateSlowLog *bool `json:"separateSlowLog,omitempty"`

	// Optional volume name configuration for slow query log.
	// +optional
	SlowLogVolumeName string `json:"slowLogVolumeName,omitempty"`

	// The specification of the slow log tailer sidecar
	// +optional
	SlowLogTailer *TiDBSlowLogTailerSpec `json:"slowLogTailer,omitempty"`

	// Whether enable the TLS connection between the SQL client and TiDB server
	// Optional: Defaults to nil
	// +optional
	TLSClient *TiDBTLSClient `json:"tlsClient,omitempty"`

	// Plugins is a list of plugins that are loaded by TiDB server, empty means plugin disabled
	// +optional
	Plugins []string `json:"plugins,omitempty"`
	// TODO: additional volumes should be used to hold .so plugin binaries.
	// Because this is not a complete implementation, maybe we can change this without backward compatibility.

	// Config is the Configuration of tidb-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *TiDBConfigWraper `json:"config,omitempty"`

	// Lifecycle describes actions that the management system should take in response to container lifecycle
	// events. For the PostStart and PreStop lifecycle handlers, management of the container blocks
	// until the action is complete, unless the container process fails, in which case the handler is aborted.
	// +optional
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty"`

	// StorageVolumes configure additional storage for TiDB pods.
	// +optional
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// The storageClassName of the persistent volume for TiDB data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// ReadinessProbe describes actions that probe the tidb's readiness.
	// the default behavior is like setting type as "tcp"
	// +optional
	ReadinessProbe *TiDBProbe `json:"readinessProbe,omitempty"`

	// Initializer is the init configurations of TiDB
	//
	// +optional
	Initializer *TiDBInitializer `json:"initializer,omitempty"`
}

type TiDBInitializer struct {
	CreatePassword bool `json:"createPassword,omitempty"`
}

const (
	// TCPProbeType represents the readiness prob method with TCP
	TCPProbeType string = "tcp"
	// CommandProbeType represents the readiness prob method with arbitrary unix `exec` call format commands
	CommandProbeType string = "command"
)

// TiDBProbe contains details of probing tidb.
// +k8s:openapi-gen=true
// default probe by TCPPort on 4000.
type TiDBProbe struct {
	// "tcp" will use TCP socket to connetct port 4000
	//
	// "command" will probe the status api of tidb.
	// This will use curl command to request tidb, before v4.0.9 there is no curl in the image,
	// So do not use this before v4.0.9.
	// +kubebuilder:validation:Enum=tcp;command
	// +optional
	Type *string `json:"type,omitempty"` // tcp or command
}

// PumpSpec contains details of Pump members
// +k8s:openapi-gen=true
type PumpSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for pump
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tidb-binlog
	// +optional
	BaseImage string `json:"baseImage"`

	// The storageClassName of the persistent volume for Pump data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// The configuration of Pump cluster.
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *config.GenericConfig `json:"config,omitempty"`

	// +k8s:openapi-gen=false
	// For backward compatibility with helm chart
	SetTimeZone *bool `json:"setTimeZone,omitempty"`
}

// HelperSpec contains details of helper component
// +k8s:openapi-gen=true
type HelperSpec struct {
	// Image used to tail slow log and set kernel parameters if necessary, must have `tail` and `sysctl` installed
	// Optional: Defaults to busybox:1.26.2. Recommended to set to 1.34.1 for new installations.
	// +optional
	Image *string `json:"image,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	// Optional: Defaults to the cluster-level setting
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// TiDBSlowLogTailerSpec represents an optional log tailer sidecar with TiDB
// +k8s:openapi-gen=true
type TiDBSlowLogTailerSpec struct {
	corev1.ResourceRequirements `json:",inline"`

	// (Deprecated) Image used for slowlog tailer.
	// Use `spec.helper.image` instead
	// +k8s:openapi-gen=false
	Image *string `json:"image,omitempty"`

	// (Deprecated) ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	// Use `spec.helper.imagePullPolicy` instead
	// +k8s:openapi-gen=false
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// ComponentSpec is the base spec of each component, the fields should always accessed by the Basic<Component>Spec() method to respect the cluster-level properties
// +k8s:openapi-gen=true
type ComponentSpec struct {
	// (Deprecated) Image of the component
	// Use `baseImage` and `version` instead
	// +k8s:openapi-gen=false
	Image string `json:"image,omitempty"`

	// Version of the component. Override the cluster-level version if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Version *string `json:"version,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	// Optional: Defaults to cluster-level setting
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Whether Hostnetwork of the component is enabled. Override the cluster-level setting if present
	// Optional: Defaults to cluster-level setting
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of the component. Override the cluster-level setting if present.
	// Optional: Defaults to cluster-level setting
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of the component. Override the cluster-level one if present
	// Optional: Defaults to cluster-level setting
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// SchedulerName of the component. Override the cluster-level one if present
	// Optional: Defaults to cluster-level setting
	// +optional
	SchedulerName *string `json:"schedulerName,omitempty"`

	// NodeSelector of the component. Merged into the cluster-level nodeSelector if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Annotations for the component. Merge into the cluster-level annotations if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels for the component. Merge into the cluster-level labels if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Tolerations of the component. Override the cluster-level tolerations if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// ConfigUpdateStrategy of the component. Override the cluster-level updateStrategy if present
	// Optional: Defaults to cluster-level setting
	// +optional
	ConfigUpdateStrategy *ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// List of environment variables to set in the container, like v1.Container.Env.
	// Note that the following env names cannot be used and will be overridden by TiDB Operator builtin envs
	// - NAMESPACE
	// - TZ
	// - SERVICE_NAME
	// - PEER_SERVICE_NAME
	// - HEADLESS_SERVICE_NAME
	// - SET_NAME
	// - HOSTNAME
	// - CLUSTER_NAME
	// - POD_NAME
	// - BINLOG_ENABLED
	// - SLOW_LOG_FILE
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Extend the use scenarios for env
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Init containers of the components
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// Additional containers of the component.
	// +optional
	AdditionalContainers []corev1.Container `json:"additionalContainers,omitempty"`

	// Additional volumes of component pod.
	// +optional
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`

	// Additional volume mounts of component pod.
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`

	// DNSConfig Specifies the DNS parameters of a pod.
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`

	// DNSPolicy Specifies the DNSPolicy parameters of a pod.
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request.
	// Value must be non-negative integer. The value zero indicates delete immediately.
	// If this value is nil, the default grace period will be used instead.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// Defaults to 30 seconds.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// StatefulSetUpdateStrategy indicates the StatefulSetUpdateStrategy that will be
	// employed to update Pods in the StatefulSet when a revision is made to
	// Template.
	// +optional
	StatefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType `json:"statefulSetUpdateStrategy,omitempty"`

	// PodManagementPolicy of TiDB cluster StatefulSets
	// +optional
	PodManagementPolicy apps.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`

	// TopologySpreadConstraints describes how a group of pods ought to spread across topology
	// domains. Scheduler will schedule pods in a way which abides by the constraints.
	// This field is is only honored by clusters that enables the EvenPodsSpread feature.
	// All topologySpreadConstraints are ANDed.
	// +optional
	// +listType=map
	// +listMapKey=topologyKey
	TopologySpreadConstraints []TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// ServiceSpec specifies the service object in k8s
// +k8s:openapi-gen=true
type ServiceSpec struct {
	// Type of the real kubernetes service
	Type corev1.ServiceType `json:"type,omitempty"`

	// Additional annotations for the service
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Additional labels for the service
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// LoadBalancerIP is the loadBalancerIP of service
	// Optional: Defaults to omitted
	// +optional
	LoadBalancerIP *string `json:"loadBalancerIP,omitempty"`

	// ClusterIP is the clusterIP of service
	// +optional
	ClusterIP *string `json:"clusterIP,omitempty"`

	// PortName is the name of service port
	// +optional
	PortName *string `json:"portName,omitempty"`

	// The port that will be exposed by this service.
	//
	// NOTE: only used for TiDB
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`

	// LoadBalancerSourceRanges is the loadBalancerSourceRanges of service
	// If specified and supported by the platform, this will restrict traffic through the cloud-provider
	// load-balancer will be restricted to the specified client IPs. This field will be ignored if the
	// cloud-provider does not support the feature."
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#aws-nlb-support
	// Optional: Defaults to omitted
	// +optional
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`
}

// TiDBServiceSpec defines `.tidb.service` field of `TidbCluster.spec`.
// +k8s:openapi-gen=true
type TiDBServiceSpec struct {
	// +k8s:openapi-gen=false
	ServiceSpec `json:",inline"`

	// ExternalTrafficPolicy of the service
	// Optional: Defaults to omitted
	// +optional
	ExternalTrafficPolicy *corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`

	// Whether expose the status port
	// Optional: Defaults to true
	// +optional
	ExposeStatus *bool `json:"exposeStatus,omitempty"`

	// Expose the tidb cluster mysql port to MySQLNodePort
	// Optional: Defaults to 0
	// +optional
	MySQLNodePort *int `json:"mysqlNodePort,omitempty"`

	// Expose the tidb status node port to StatusNodePort
	// Optional: Defaults to 0
	// +optional
	StatusNodePort *int `json:"statusNodePort,omitempty"`

	// Expose additional ports for TiDB
	// Optional: Defaults to omitted
	// +optional
	AdditionalPorts []corev1.ServicePort `json:"additionalPorts,omitempty"`
}

// (Deprecated) Service represent service type used in TidbCluster
// +k8s:openapi-gen=false
type Service struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// PDStatus is PD status
type PDStatus struct {
	// +optional
	Synced      bool                    `json:"synced"`
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	// Members contains PDs in current TidbCluster
	Members map[string]PDMember `json:"members,omitempty"`
	// PeerMembers contains PDs NOT in current TidbCluster
	PeerMembers     map[string]PDMember        `json:"peerMembers,omitempty"`
	Leader          PDMember                   `json:"leader,omitempty"`
	FailureMembers  map[string]PDFailureMember `json:"failureMembers,omitempty"`
	UnjoinedMembers map[string]UnjoinedMember  `json:"unjoinedMembers,omitempty"`
	Image           string                     `json:"image,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
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
	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// EmptyStruct is defined to delight controller-gen tools
// Only named struct is allowed by controller-gen
type EmptyStruct struct{}

// PDFailureMember is the pd failure member information
type PDFailureMember struct {
	PodName       string                    `json:"podName,omitempty"`
	MemberID      string                    `json:"memberID,omitempty"`
	PVCUID        types.UID                 `json:"pvcUID,omitempty"`
	PVCUIDSet     map[types.UID]EmptyStruct `json:"pvcUIDSet,omitempty"`
	MemberDeleted bool                      `json:"memberDeleted,omitempty"`
	// +nullable
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// UnjoinedMember is the pd unjoin cluster member information
type UnjoinedMember struct {
	PodName   string                    `json:"podName,omitempty"`
	PVCUID    types.UID                 `json:"pvcUID,omitempty"`
	PVCUIDSet map[types.UID]EmptyStruct `json:"pvcUIDSet,omitempty"`
	// +nullable
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// TiDBStatus is TiDB status
type TiDBStatus struct {
	Phase                    MemberPhase                  `json:"phase,omitempty"`
	StatefulSet              *apps.StatefulSetStatus      `json:"statefulSet,omitempty"`
	Members                  map[string]TiDBMember        `json:"members,omitempty"`
	FailureMembers           map[string]TiDBFailureMember `json:"failureMembers,omitempty"`
	ResignDDLOwnerRetryCount int32                        `json:"resignDDLOwnerRetryCount,omitempty"`
	Image                    string                       `json:"image,omitempty"`
	PasswordInitialized      *bool                        `json:"passwordInitialized,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TiDBMember is TiDB member
type TiDBMember struct {
	Name   string `json:"name"`
	Health bool   `json:"health"`
	// Last time the health transitioned from one to another.
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Node hosting pod of this TiDB member.
	NodeName string `json:"node,omitempty"`
}

// TiDBFailureMember is the tidb failure member information
type TiDBFailureMember struct {
	PodName string `json:"podName,omitempty"`
	// +nullable
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

var (
	EvictLeaderAnnKeys = []string{EvictLeaderAnnKey, EvictLeaderAnnKeyForResize}
)

const (
	// EvictLeaderAnnKey is the annotation key to evict leader used by user.
	EvictLeaderAnnKey = "tidb.pingcap.com/evict-leader"
	// EvictLeaderAnnKeyForResize is the annotation key to evict leader user by pvc resizer.
	EvictLeaderAnnKeyForResize = "tidb.pingcap.com/evict-leader-for-resize"
)

// The `Value` of annotation controls the behavior when the leader count drops to zero, the valid value is one of:
//
// - `none`: doing nothing.
// - `delete-pod`: delete pod and remove the evict-leader scheduler from PD.
const (
	EvictLeaderValueNone      = "none"
	EvictLeaderValueDeletePod = "delete-pod"
)

type EvictLeaderStatus struct {
	PodCreateTime metav1.Time `json:"podCreateTime,omitempty"`
	Value         string      `json:"value,omitempty"`
}

// TiKVStatus is TiKV status
type TiKVStatus struct {
	Synced          bool                          `json:"synced,omitempty"`
	Phase           MemberPhase                   `json:"phase,omitempty"`
	BootStrapped    bool                          `json:"bootStrapped,omitempty"`
	StatefulSet     *apps.StatefulSetStatus       `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore          `json:"stores,omitempty"`
	PeerStores      map[string]TiKVStore          `json:"peerStores,omitempty"`
	TombstoneStores map[string]TiKVStore          `json:"tombstoneStores,omitempty"`
	FailureStores   map[string]TiKVFailureStore   `json:"failureStores,omitempty"`
	FailoverUID     types.UID                     `json:"failoverUID,omitempty"`
	Image           string                        `json:"image,omitempty"`
	EvictLeader     map[string]*EvictLeaderStatus `json:"evictLeader,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TiFlashStatus is TiFlash status
type TiFlashStatus struct {
	Synced          bool                        `json:"synced,omitempty"`
	Phase           MemberPhase                 `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore        `json:"stores,omitempty"`
	PeerStores      map[string]TiKVStore        `json:"peerStores,omitempty"`
	TombstoneStores map[string]TiKVStore        `json:"tombstoneStores,omitempty"`
	FailureStores   map[string]TiKVFailureStore `json:"failureStores,omitempty"`
	FailoverUID     types.UID                   `json:"failoverUID,omitempty"`
	Image           string                      `json:"image,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TiCDCStatus is TiCDC status
type TiCDCStatus struct {
	Synced      bool                    `json:"synced,omitempty"`
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	Captures    map[string]TiCDCCapture `json:"captures,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TiCDCCapture is TiCDC Capture status
type TiCDCCapture struct {
	PodName string `json:"podName,omitempty"`
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	IsOwner bool   `json:"isOwner,omitempty"`
	Ready   bool   `json:"ready,omitempty"`
}

// TiKVStores is either Up/Down/Offline/Tombstone
type TiKVStore struct {
	// store id is also uint64, due to the same reason as pd id, we store id as string
	ID          string `json:"id"`
	PodName     string `json:"podName"`
	IP          string `json:"ip"`
	LeaderCount int32  `json:"leaderCount"`
	State       string `json:"state"`
	// Last time the health transitioned from one to another.
	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// TiKVFailureStore is the tikv failure store information
type TiKVFailureStore struct {
	PodName string `json:"podName,omitempty"`
	StoreID string `json:"storeID,omitempty"`
	// +nullable
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// PumpNodeStatus represents the status saved in etcd.
type PumpNodeStatus struct {
	NodeID string `json:"nodeId"`
	Host   string `json:"host"`
	State  string `json:"state"`

	// NB: Currently we save the whole `PumpNodeStatus` in the status of the CR.
	// However, the following fields will be updated continuously.
	// To avoid CR being updated and re-synced continuously, we exclude these fields.
	// MaxCommitTS int64  `json:"maxCommitTS"`
	// UpdateTS    int64  `json:"updateTS"`
}

// PumpStatus is Pump status
type PumpStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	Members     []*PumpNodeStatus       `json:"members,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TiDBTLSClient can enable TLS connection between TiDB server and MySQL client
// +k8s:openapi-gen=true
type TiDBTLSClient struct {
	// When enabled, TiDB will accept TLS encrypted connections from MySQL client
	// The steps to enable this feature:
	//   1. Generate a TiDB server-side certificate and a client-side certifiacete for the TiDB cluster.
	//      There are multiple ways to generate certificates:
	//        - user-provided certificates: https://pingcap.com/docs/stable/how-to/secure/enable-tls-clients/
	//        - use the K8s built-in certificate signing system signed certificates: https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/
	//        - or use cert-manager signed certificates: https://cert-manager.io/
	//   2. Create a K8s Secret object which contains the TiDB server-side certificate created above.
	//      The name of this Secret must be: <clusterName>-tidb-server-secret.
	//        kubectl create secret generic <clusterName>-tidb-server-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//   3. Create a K8s Secret object which contains the TiDB client-side certificate created above which will be used by TiDB Operator.
	//      The name of this Secret must be: <clusterName>-tidb-client-secret.
	//        kubectl create secret generic <clusterName>-tidb-client-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//   4. Set Enabled to `true`.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// DisableClientAuthn will skip client's certificate validation from the TiDB server.
	// Optional: defaults to false
	// +optional
	DisableClientAuthn bool `json:"disableClientAuthn,omitempty"`

	// SkipInternalClientCA will skip TiDB server's certificate validation for internal components like Initializer, Dashboard, etc.
	// Optional: defaults to false
	// +optional
	SkipInternalClientCA bool `json:"skipInternalClientCA,omitempty"`
}

// TLSCluster can enable mutual TLS connection between TiDB cluster components
// https://pingcap.com/docs/stable/how-to/secure/enable-tls-between-components/
type TLSCluster struct {
	// Enable mutual TLS connection between TiDB cluster components
	// Once enabled, the mutual authentication applies to all components,
	// and it does not support applying to only part of the components.
	// The steps to enable this feature:
	//   1. Generate TiDB cluster components certificates and a client-side certifiacete for them.
	//      There are multiple ways to generate these certificates:
	//        - user-provided certificates: https://pingcap.com/docs/stable/how-to/secure/generate-self-signed-certificates/
	//        - use the K8s built-in certificate signing system signed certificates: https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/
	//        - or use cert-manager signed certificates: https://cert-manager.io/
	//   2. Create one secret object for one component which contains the certificates created above.
	//      The name of this Secret must be: <clusterName>-<componentName>-cluster-secret.
	//        For PD: kubectl create secret generic <clusterName>-pd-cluster-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        For TiKV: kubectl create secret generic <clusterName>-tikv-cluster-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        For TiDB: kubectl create secret generic <clusterName>-tidb-cluster-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        For Client: kubectl create secret generic <clusterName>-cluster-client-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        Same for other components.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Backup is a backup of tidb cluster.
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="bk"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="The current status of the backup"
// +kubebuilder:printcolumn:name="BackupPath",type=string,JSONPath=`.status.backupPath`,description="The full path of backup data"
// +kubebuilder:printcolumn:name="BackupSize",type=string,JSONPath=`.status.backupSizeReadable`,description="The data size of the backup"
// +kubebuilder:printcolumn:name="CommitTS",type=string,JSONPath=`.status.commitTs`,description="The commit ts of tidb cluster dump"
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.status.timeStarted`,description="The time at which the backup was started",priority=1
// +kubebuilder:printcolumn:name="Completed",type=date,JSONPath=`.status.timeCompleted`,description="The time at which the backup was completed",priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
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

	BatchDeleteOption `json:",inline"`
}

// BackupSpec contains the backup specification for a tidb cluster.
// +k8s:openapi-gen=true
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
	// From is the tidb cluster that needs to backup.
	From *TiDBAccessConfig `json:"from,omitempty"`
	// Type is the backup type for tidb cluster.
	Type BackupType `json:"backupType,omitempty"`
	// TikvGCLifeTime is to specify the safe gc life time for backup.
	// The time limit during which data is retained for each GC, in the format of Go Duration.
	// When a GC happens, the current time minus this value is the safe point.
	TikvGCLifeTime *string `json:"tikvGCLifeTime,omitempty"`
	// StorageProvider configures where and how backups should be stored.
	StorageProvider `json:",inline"`
	// The storageClassName of the persistent volume for Backup data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize,omitempty"`
	// BRConfig is the configs for BR
	BR *BRConfig `json:"br,omitempty"`
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
	CleanPolicy CleanPolicyType `json:"cleanPolicy,omitempty"`
	// CleanOption controls the behavior of clean.
	CleanOption *CleanOption `json:"cleanOption,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// PriorityClassName of Backup Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

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
	// SendCredToTikv specifies whether to send credentials to TiKV
	SendCredToTikv *bool `json:"sendCredToTikv,omitempty"`
	// OnLine specifies whether online during restore
	OnLine *bool `json:"onLine,omitempty"`
	// Options means options for backup data to remote storage with BR. These options has highest priority.
	Options []string `json:"options,omitempty"`
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
	// BackupInvalid means invalid backup CR
	BackupInvalid BackupConditionType = "Invalid"
	// BackupPrepare means the backup prepare backup process
	BackupPrepare BackupConditionType = "Prepare"
)

// BackupCondition describes the observed state of a Backup at a certain point.
type BackupCondition struct {
	Type   BackupConditionType    `json:"type"`
	Status corev1.ConditionStatus `json:"status"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
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
	// BackupSizeReadable is the data size of the backup.
	// the difference with BackupSize is that its format is human readable
	BackupSizeReadable string `json:"backupSizeReadable,omitempty"`
	// BackupSize is the data size of the backup.
	BackupSize int64 `json:"backupSize,omitempty"`
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs string `json:"commitTs,omitempty"`
	// Phase is a user readable state inferred from the underlying Backup conditions
	Phase BackupConditionType `json:"phase,omitempty"`
	// +nullable
	Conditions []BackupCondition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupSchedule is a backup schedule of tidb cluster.
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="bks"
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`,description="The cron format string used for backup scheduling"
// +kubebuilder:printcolumn:name="MaxBackups",type=integer,JSONPath=`.spec.maxBackups`,description="The max number of backups we want to keep"
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
	// BackupTemplate is the specification of the backup structure to get scheduled.
	BackupTemplate BackupSpec `json:"backupTemplate"`
	// The storageClassName of the persistent volume for Backup data storage if not storage class name set in BackupSpec.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// StorageSize is the request storage size for backup job
	StorageSize string `json:"storageSize,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// BackupScheduleStatus represents the current state of a BackupSchedule.
type BackupScheduleStatus struct {
	// LastBackup represents the last backup.
	LastBackup string `json:"lastBackup,omitempty"`
	// LastBackupTime represents the last time the backup was successfully created.
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`
	// AllBackupCleanTime represents the time when all backup entries are cleaned up
	AllBackupCleanTime *metav1.Time `json:"allBackupCleanTime,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Restore represents the restoration of backup of a tidb cluster.
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="rt"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`,description="The current status of the restore"
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.status.timeStarted`,description="The time at which the restore was started",priority=1
// +kubebuilder:printcolumn:name="Completed",type=date,JSONPath=`.status.timeCompleted`,description="The time at which the restore was completed",priority=1
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
	// Type is the backup type for tidb cluster.
	Type BackupType `json:"backupType,omitempty"`
	// TikvGCLifeTime is to specify the safe gc life time for restore.
	// The time limit during which data is retained for each GC, in the format of Go Duration.
	// When a GC happens, the current time minus this value is the safe point.
	TikvGCLifeTime *string `json:"tikvGCLifeTime,omitempty"`
	// StorageProvider configures where and how backups should be stored.
	StorageProvider `json:",inline"`
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

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// PriorityClassName of Restore Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// RestoreStatus represents the current status of a tidb cluster restore.
type RestoreStatus struct {
	// TimeStarted is the time at which the restore was started.
	// +nullable
	TimeStarted metav1.Time `json:"timeStarted,omitempty"`
	// TimeCompleted is the time at which the restore was completed.
	// +nullable
	TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs string `json:"commitTs,omitempty"`
	// Phase is a user readable state inferred from the underlying Restore conditions
	Phase RestoreConditionType `json:"phase,omitempty"`
	// +nullable
	Conditions []RestoreCondition `json:"conditions,omitempty"`
}

// +k8s:openapi-gen=true
// IngressSpec describe the ingress desired state for the target component
type IngressSpec struct {
	// Hosts describe the hosts for the ingress
	Hosts []string `json:"hosts"`
	// Annotations describe the desired annotations for the ingress
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// TLS configuration. Currently the Ingress only supports a single TLS
	// port, 443. If multiple members of this list specify different hosts, they
	// will be multiplexed on the same port according to the hostname specified
	// through the SNI TLS extension, if the ingress controller fulfilling the
	// ingress supports SNI.
	// +optional
	TLS []networkingv1.IngressTLS `json:"tls,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DMCluster is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="dc"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Master",type=string,JSONPath=`.status.master.image`,description="The image for dm-master cluster"
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.master.storageSize`,description="The storage size specified for dm-master node"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.master.statefulSet.readyReplicas`,description="The ready replicas number of dm-master cluster"
// +kubebuilder:printcolumn:name="Desire",type=integer,JSONPath=`.spec.master.replicas`,description="The desired replicas number of dm-master cluster"
// +kubebuilder:printcolumn:name="Worker",type=string,JSONPath=`.status.worker.image`,description="The image for dm-master cluster"
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.worker.storageSize`,description="The storage size specified for dm-worker node"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.worker.statefulSet.readyReplicas`,description="The ready replicas number of dm-worker cluster"
// +kubebuilder:printcolumn:name="Desire",type=integer,JSONPath=`.spec.worker.replicas`,description="The desired replicas number of dm-worker cluster"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type DMCluster struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the behavior of a dm cluster
	Spec DMClusterSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the dm cluster
	Status DMClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// DMClusterList is DMCluster list
type DMClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []DMCluster `json:"items"`
}

// +k8s:openapi-gen=true
// DMDiscoverySpec contains details of Discovery members for dm
type DMDiscoverySpec struct {
	*ComponentSpec              `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// (Deprecated) Address indicates the existed TiDB discovery address
	// +k8s:openapi-gen=false
	Address string `json:"address,omitempty"`
}

// +k8s:openapi-gen=true
// DMClusterSpec describes the attributes that a user creates on a dm cluster
type DMClusterSpec struct {
	// Discovery spec
	Discovery DMDiscoverySpec `json:"discovery,omitempty"`

	// dm-master cluster spec
	// +optional
	Master MasterSpec `json:"master"`

	// dm-worker cluster spec
	// +optional
	Worker *WorkerSpec `json:"worker,omitempty"`

	// Indicates that the dm cluster is paused and will not be processed by
	// the controller.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// dm cluster version
	// +optional
	Version string `json:"version"`
	// TODO: remove optional after defaulting logic introduced

	// SchedulerName of DM cluster Pods
	SchedulerName string `json:"schedulerName,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by DM cluster
	// +kubebuilder:default=Retain
	PVReclaimPolicy *corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// ImagePullPolicy of DM cluster Pods
	// +kubebuilder:default=IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
	// UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
	// cluster component is needed to reload the configuration change.
	// UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
	// related components to use the new ConfigMap, that is, the new configuration will be applied automatically.
	ConfigUpdateStrategy ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// Whether enable PVC reclaim for orphan PVC left by statefulset scale-in
	// Optional: Defaults to false
	// +optional
	EnablePVReclaim *bool `json:"enablePVReclaim,omitempty"`

	// Whether enable the TLS connection between DM server components
	// Optional: Defaults to nil
	// +optional
	TLSCluster *TLSCluster `json:"tlsCluster,omitempty"`

	// TLSClientSecretNames are the names of secrets which stores mysql/tidb server client certificates
	// that used by dm-master and dm-worker.
	// +optional
	TLSClientSecretNames []string `json:"tlsClientSecretNames,omitempty"`

	// Whether Hostnetwork is enabled for DM cluster Pods
	// Optional: Defaults to false
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of DM cluster Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of DM cluster Pods
	// Optional: Defaults to omitted
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Base node selectors of DM cluster Pods, components may add or override selectors upon this respectively
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Additional annotations for the dm cluster
	// Can be overrode by annotations in master spec or worker spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Additional labels for the dm cluster
	// Can be overrode by labels in master spec or worker spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Time zone of DM cluster Pods
	// Optional: Defaults to UTC
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// Base tolerations of DM cluster Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// DNSConfig Specifies the DNS parameters of a pod.
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`

	// DNSPolicy Specifies the DNSPolicy parameters of a pod.
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// StatefulSetUpdateStrategy of DM cluster StatefulSets
	// +optional
	StatefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType `json:"statefulSetUpdateStrategy,omitempty"`

	// PodManagementPolicy of DM cluster StatefulSets
	// +optional
	PodManagementPolicy apps.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`

	// TopologySpreadConstraints describes how a group of pods ought to spread across topology
	// domains. Scheduler will schedule pods in a way which abides by the constraints.
	// This field is is only honored by clusters that enables the EvenPodsSpread feature.
	// All topologySpreadConstraints are ANDed.
	// +optional
	// +listType=map
	// +listMapKey=topologyKey
	TopologySpreadConstraints []TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// DMClusterStatus represents the current status of a dm cluster.
type DMClusterStatus struct {
	Master MasterStatus `json:"master,omitempty"`
	Worker WorkerStatus `json:"worker,omitempty"`

	// Represents the latest available observations of a dm cluster's state.
	// +optional
	// +nullable
	Conditions []DMClusterCondition `json:"conditions,omitempty"`
}

// +k8s:openapi-gen=true
// MasterSpec contains details of dm-master members
type MasterSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/dm
	// +optional
	BaseImage string `json:"baseImage,omitempty"`

	// Service defines a Kubernetes service of Master cluster.
	// Optional: Defaults to `.spec.services` in favor of backward compatibility
	// +optional
	Service *MasterServiceSpec `json:"service,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The storageClassName of the persistent volume for dm-master data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// StorageSize is the request storage size for dm-master.
	// Defaults to "10Gi".
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// Subdirectory within the volume to store dm-master Data. By default, the data
	// is stored in the root directory of volume which is mounted at
	// /var/lib/dm-master.
	// Specifying this will change the data directory to a subdirectory, e.g.
	// /var/lib/dm-master/data if you set the value to "data".
	// It's dangerous to change this value for a running cluster as it will
	// upgrade your cluster to use a new storage directory.
	// Defaults to "" (volume's root).
	// +optional
	DataSubDir string `json:"dataSubDir,omitempty"`

	// Config is the Configuration of dm-master-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *MasterConfigWraper `json:"config,omitempty"`
}

type MasterServiceSpec struct {
	ServiceSpec `json:",inline"`

	// ExternalTrafficPolicy of the service
	// Optional: Defaults to omitted
	// +optional
	ExternalTrafficPolicy *corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"` // Expose the tidb cluster mysql port to MySQLNodePort

	// Optional: Defaults to 0
	// +optional
	MasterNodePort *int `json:"masterNodePort,omitempty"`
}

// +k8s:openapi-gen=true
// WorkerSpec contains details of dm-worker members
type WorkerSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/dm
	// +optional
	BaseImage string `json:"baseImage,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The storageClassName of the persistent volume for dm-worker data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// StorageSize is the request storage size for dm-worker.
	// Defaults to "10Gi".
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// Subdirectory within the volume to store dm-worker Data. By default, the data
	// is stored in the root directory of volume which is mounted at
	// /var/lib/dm-worker.
	// Specifying this will change the data directory to a subdirectory, e.g.
	// /var/lib/dm-worker/data if you set the value to "data".
	// It's dangerous to change this value for a running cluster as it will
	// upgrade your cluster to use a new storage directory.
	// Defaults to "" (volume's root).
	// +optional
	DataSubDir string `json:"dataSubDir,omitempty"`

	// Config is the Configuration of dm-worker-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *WorkerConfigWraper `json:"config,omitempty"`

	// RecoverFailover indicates that Operator can recover the failover Pods
	// +optional
	RecoverFailover bool `json:"recoverFailover,omitempty"`

	// Failover is the configurations of failover
	// +optional
	Failover *Failover `json:"failover,omitempty"`
}

// DMClusterCondition is dm cluster condition
type DMClusterCondition struct {
	// Type of the condition.
	Type DMClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	// +nullable
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +nullable
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// DMClusterConditionType represents a dm cluster condition value.
type DMClusterConditionType string

const (
	// DMClusterReady indicates that the dm cluster is ready or not.
	// This is defined as:
	// - All statefulsets are up to date (currentRevision == updateRevision).
	// - All Master members are healthy.
	// - All Worker pods are up.
	DMClusterReady DMClusterConditionType = "Ready"
)

// MasterStatus is dm-master status
type MasterStatus struct {
	Synced          bool                           `json:"synced,omitempty"`
	Phase           MemberPhase                    `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus        `json:"statefulSet,omitempty"`
	Members         map[string]MasterMember        `json:"members,omitempty"`
	Leader          MasterMember                   `json:"leader,omitempty"`
	FailureMembers  map[string]MasterFailureMember `json:"failureMembers,omitempty"`
	UnjoinedMembers map[string]UnjoinedMember      `json:"unjoinedMembers,omitempty"`
	Image           string                         `json:"image,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// MasterMember is dm-master member status
type MasterMember struct {
	Name string `json:"name"`
	// member id is actually a uint64, but apimachinery's json only treats numbers as int64/float64
	// so uint64 may overflow int64 and thus convert to float64
	ID        string `json:"id"`
	ClientURL string `json:"clientURL"`
	Health    bool   `json:"health"`
	// Last time the health transitioned from one to another.
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// MasterFailureMember is the dm-master failure member information
type MasterFailureMember struct {
	PodName       string    `json:"podName,omitempty"`
	MemberID      string    `json:"memberID,omitempty"`
	PVCUID        types.UID `json:"pvcUID,omitempty"`
	MemberDeleted bool      `json:"memberDeleted,omitempty"`
	// +nullable
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// WorkerStatus is dm-worker status
type WorkerStatus struct {
	Synced         bool                           `json:"synced,omitempty"`
	Phase          MemberPhase                    `json:"phase,omitempty"`
	StatefulSet    *apps.StatefulSetStatus        `json:"statefulSet,omitempty"`
	Members        map[string]WorkerMember        `json:"members,omitempty"`
	FailureMembers map[string]WorkerFailureMember `json:"failureMembers,omitempty"`
	FailoverUID    types.UID                      `json:"failoverUID,omitempty"`
	Image          string                         `json:"image,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Represents the latest available observations of a component's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// WorkerMember is dm-worker member status
type WorkerMember struct {
	Name  string `json:"name,omitempty"`
	Addr  string `json:"addr,omitempty"`
	Stage string `json:"stage"`
	// Last time the health transitioned from one to another.
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// WorkerFailureMember is the dm-worker failure member information
type WorkerFailureMember struct {
	PodName string `json:"podName,omitempty"`
	// +nullable
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// StorageVolume configures additional PVC template for StatefulSets and volumeMount for pods that mount this PVC.
// Note:
// If `MountPath` is not set, volumeMount will not be generated. (You may not want to set this field when you inject volumeMount
// in somewhere else such as Mutating Admission Webhook)
// If `StorageClassName` is not set, default to the `spec.${component}.storageClassName`
type StorageVolume struct {
	Name             string  `json:"name"`
	StorageClassName *string `json:"storageClassName,omitempty"`
	StorageSize      string  `json:"storageSize"`
	MountPath        string  `json:"mountPath,omitempty"`
}

type ObservedStorageVolumeStatus struct {
	// BoundCount is the count of bound volumes.
	// +optional
	BoundCount int `json:"boundCount"`
	// CurrentCount is the count of volumes whose capacity is equal to `currentCapacity`.
	// +optional
	CurrentCount int `json:"currentCount"`
	// ResizedCount is the count of volumes whose capacity is equal to `resizedCapacity`.
	// +optional
	ResizedCount int `json:"resizedCount"`
	// CurrentCapacity is the current capacity of the volume.
	// If any volume is resizing, it is the capacity before resizing.
	// If all volumes are resized, it is the resized capacity and same as desired capacity.
	CurrentCapacity resource.Quantity `json:"currentCapacity"`
	// ResizedCapacity is the desired capacity of the volume.
	ResizedCapacity resource.Quantity `json:"resizedCapacity"`
}

// StorageVolumeName is the volume name which is same as `volumes.name` in Pod spec.
type StorageVolumeName string

// StorageVolumeStatus is the actual status for a storage
type StorageVolumeStatus struct {
	ObservedStorageVolumeStatus `json:",inline"`
	// Name is the volume name which is same as `volumes.name` in Pod spec.
	Name StorageVolumeName `json:"name"`
}

// TopologySpreadConstraint specifies how to spread matching pods among the given topology.
// It is a minimal version of corev1.TopologySpreadConstraint to avoid to add too many fields of API
// Refer to https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints
type TopologySpreadConstraint struct {
	// TopologyKey is the key of node labels. Nodes that have a label with this key
	// and identical values are considered to be in the same topology.
	// We consider each <key, value> as a "bucket", and try to put balanced number
	// of pods into each bucket.
	// MaxSkew is default set to 1
	// WhenUnsatisfiable is default set to DoNotSchedule
	// LabelSelector is generated by component type
	// See pkg/apis/pingcap/v1alpha1/tidbcluster_component.go#TopologySpreadConstraints()
	TopologyKey string `json:"topologyKey"`
}

// Failover contains the failover specification.
// +k8s:openapi-gen=true
type Failover struct {
	// RecoverByUID indicates that TiDB Operator will recover the failover by this UID,
	// it takes effect only when set `spec.recoverFailover=false`
	// +optional
	RecoverByUID types.UID `json:"recoverByUID,omitempty"`
}
