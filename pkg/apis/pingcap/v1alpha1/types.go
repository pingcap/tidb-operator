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
	"fmt"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
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

type ContainerName string

func (c ContainerName) String() string {
	return string(c)
}

const (
	ContainerSlowLogTailer    ContainerName = "slowlog"
	ContainerRocksDBLogTailer ContainerName = "rocksdblog"
	ContainerRaftLogTailer    ContainerName = "raftlog"
)

// MemberType represents member type
type MemberType string

const (
	// DiscoveryMemberType is discovery member type
	DiscoveryMemberType MemberType = "discovery"
	// PDMemberType is pd member type
	PDMemberType MemberType = "pd"
	// PDMSTSOMemberType is pd microservice tso member type
	PDMSTSOMemberType MemberType = "tso"
	// PDMSSchedulingMemberType is pd microservice scheduling member type
	PDMSSchedulingMemberType MemberType = "scheduling"
	// TiDBMemberType is tidb member type
	TiDBMemberType MemberType = "tidb"
	// TiKVMemberType is tikv member type
	TiKVMemberType MemberType = "tikv"
	// TiFlashMemberType is tiflash member type
	TiFlashMemberType MemberType = "tiflash"
	// TiCDCMemberType is ticdc member type
	TiCDCMemberType MemberType = "ticdc"
	// TiProxyMemberType is ticdc member type
	TiProxyMemberType MemberType = "tiproxy"
	// PumpMemberType is pump member type
	PumpMemberType MemberType = "pump"

	// DMDiscoveryMemberType is discovery member type
	DMDiscoveryMemberType MemberType = "dm-discovery"
	// DMMasterMemberType is dm-master member type
	DMMasterMemberType MemberType = "dm-master"
	// DMWorkerMemberType is dm-worker member type
	DMWorkerMemberType MemberType = "dm-worker"

	// TidbMonitorMemberType is tidbmonitor member type
	TidbMonitorMemberType MemberType = "tidbmonitor"

	// NGMonitoringMemberType is ng monitoring member type
	NGMonitoringMemberType MemberType = "ng-monitoring"

	// TiDBDashboardMemberType is tidb-dashboard member type
	TiDBDashboardMemberType MemberType = "tidb-dashboard"

	// UnknownMemberType is unknown member type
	UnknownMemberType MemberType = "unknown"
)

func PDMSMemberType(name string) MemberType {
	switch name {
	case "tso":
		return PDMSTSOMemberType
	case "scheduling":
		return PDMSSchedulingMemberType
	default:
		panic(fmt.Sprintf("unknown pd ms name %s", name))
	}
}

func IsPDMSMemberType(name MemberType) bool {
	return name == PDMSTSOMemberType || name == PDMSSchedulingMemberType
}

// MemberPhase is the current state of member
type MemberPhase string

const (
	// NormalPhase represents normal state of TiDB cluster.
	NormalPhase MemberPhase = "Normal"
	// UpgradePhase represents the upgrade state of TiDB cluster.
	UpgradePhase MemberPhase = "Upgrade"
	// ScalePhase represents the scaling state of TiDB cluster.
	ScalePhase MemberPhase = "Scale"
	// SuspendPhase represents the suspend state of TiDB cluster.
	SuspendPhase MemberPhase = "Suspend"
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

type StartScriptVersion string

const (
	StartScriptV1 StartScriptVersion = "v1"
	StartScriptV2 StartScriptVersion = "v2"
)

type StartScriptV2FeatureFlag string

const (
	StartScriptV2FeatureFlagWaitForDnsNameIpMatch          = "WaitForDnsNameIpMatch"
	StartScriptV2FeatureFlagPreferPDAddressesOverDiscovery = "PreferPDAddressesOverDiscovery"
)

type TiProxyCertLayout string

const (
	TiProxyCertLayoutLegacy TiProxyCertLayout = ""
	// TiProxyCertLayoutV1 is a refined version of legacy layout. It's more intuitive and more flexible.
	TiProxyCertLayoutV1 TiProxyCertLayout = "v1"
)

const (
	// AnnoKeySkipFlushLogBackup when set to a `TidbCluster`, during restarting the cluster, log backup tasks won't be flushed.
	AnnoKeySkipFlushLogBackup = "tidb.pingcap.com/tikv-restart-without-flush-log-backup"
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
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.pd.statefulSet.readyReplicas`,description="The ready replicas number of PD cluster"
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

	// PDMS cluster spec
	// +optional
	PDMS []*PDMSSpec `json:"pdms,omitempty"`

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

	// TiProxy cluster spec
	// +optional
	TiProxy *TiProxySpec `json:"tiproxy,omitempty"`

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

	// Whether RecoveryMode is enabled for TiDB cluster to restore
	// Optional: Defaults to false
	// +optional
	RecoveryMode bool `json:"recoveryMode,omitempty"`

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

	// Whether enable PVC replace to recreate the PVC with different specs
	// Optional: Defaults to false
	// +optional
	EnablePVCReplace *bool `json:"enablePVCReplace,omitempty"`

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

	// StartScriptVersion is the version of start script
	// When PD enables microservice mode, pd and pd microservice component will use start script v2.
	//
	// default to "v1"
	// +optional
	// +kubebuilder:validation:Enum:="";"v1";"v2"
	StartScriptVersion StartScriptVersion `json:"startScriptVersion,omitempty"`

	// SuspendAction defines the suspend actions for all component.
	// +optional
	SuspendAction *SuspendAction `json:"suspendAction,omitempty"`

	// PreferIPv6 indicates whether to prefer IPv6 addresses for all components.
	PreferIPv6 bool `json:"preferIPv6,omitempty"`

	// Feature flags used by v2 startup script to enable various features.
	// Examples of supported feature flags:
	// - WaitForDnsNameIpMatch indicates whether PD and TiKV has to wait until local IP address matches the one published to external DNS
	// - PreferPDAddressesOverDiscovery advises start script to use TidbClusterSpec.PDAddresses (if supplied) as argument for pd-server, tikv-server and tidb-server commands
	StartScriptV2FeatureFlags []StartScriptV2FeatureFlag `json:"startScriptV2FeatureFlags,omitempty"`
}

// TidbClusterStatus represents the current status of a tidb cluster.
type TidbClusterStatus struct {
	ClusterID string                 `json:"clusterID,omitempty"`
	PD        PDStatus               `json:"pd,omitempty"`
	PDMS      map[string]*PDMSStatus `json:"pdms,omitempty"`
	TiKV      TiKVStatus             `json:"tikv,omitempty"`
	TiDB      TiDBStatus             `json:"tidb,omitempty"`
	Pump      PumpStatus             `json:"pump,omitempty"`
	TiFlash   TiFlashStatus          `json:"tiflash,omitempty"`
	TiProxy   TiProxyStatus          `json:"tiproxy,omitempty"`
	TiCDC     TiCDCStatus            `json:"ticdc,omitempty"`
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

	// LivenessProbe describes actions that probe the discovery's liveness.
	// the default behavior is like setting type as "tcp"
	// NOTE: only used for TiDB Operator discovery now,
	// for other components, the auto failover feature may be used instead.
	// +optional
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`
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

	// Timeout threshold when pd get started
	// +kubebuilder:default=30
	StartTimeout int `json:"startTimeout,omitempty"`

	// Wait time before pd get started. This wait time is to allow the new DNS record to propagate,
	// ensuring that the PD DNS resolves to the same IP address as the pod.
	// +kubebuilder:default=0
	InitWaitTime int `json:"initWaitTime,omitempty"`

	// Mode is the mode of PD cluster
	// +optional
	// +kubebuilder:validation:Enum:="";"ms"
	Mode string `json:"mode,omitempty"`

	// The default number of spare replicas to scale up when using VolumeReplace feature.
	// In multi-az deployments with topology spread constraints you may need to set this to number of zones to avoid
	// zone skew after volume replace (total replicas always whole multiples of zones).
	// Optional: Defaults to 1
	// +kubebuilder:validation:Minimum=0
	// +optional
	SpareVolReplaceReplicas *int32 `json:"spareVolReplaceReplicas,omitempty"`
}

// +k8s:openapi-gen=true
// PDMSSpec contains details of PD microservice
type PDMSSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Name of the PD microservice
	// +kubebuilder:validation:Enum:="tso";"scheduling"
	Name string `json:"name"`

	// Specify a Service Account for pd ms
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/pd
	// +optional
	BaseImage *string `json:"baseImage"`

	// Service defines a Kubernetes service of PD microservice cluster.
	// Optional: Defaults to `.spec.services` in favor of backward compatibility
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// Config is the configuration of PD microservice servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *PDConfigWraper `json:"config,omitempty"`

	// TLSClientSecretName is the name of secret which stores tidb server client certificate
	// which used by Dashboard.
	// +optional
	TLSClientSecretName *string `json:"tlsClientSecretName,omitempty"`

	// MountClusterClientSecret indicates whether to mount `cluster-client-secret` to the Pod
	// +optional
	MountClusterClientSecret *bool `json:"mountClusterClientSecret,omitempty"`

	// Start up script version
	// +optional
	// +kubebuilder:validation:Enum:="";"v1"
	StartUpScriptVersion string `json:"startUpScriptVersion,omitempty"`

	// The storageClassName of the persistent volume for PD microservice log storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// StorageVolumes configure additional storage for PD microservice pods.
	// +optional
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// Timeout threshold when pd get started
	// +kubebuilder:default=30
	StartTimeout int `json:"startTimeout,omitempty"`
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
	// Defaults to 1500min
	// +optional
	EvictLeaderTimeout *string `json:"evictLeaderTimeout,omitempty"`

	// WaitLeaderTransferBackTimeout indicates the timeout to wait for leader transfer back before
	// the next tikv upgrade.
	//
	// Defaults to 400s
	// +optional
	WaitLeaderTransferBackTimeout *metav1.Duration `json:"waitLeaderTransferBackTimeout,omitempty"`

	// StorageVolumes configure additional storage for TiKV pods.
	// +optional
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// StoreLabels configures additional labels for TiKV stores.
	// +optional
	StoreLabels []string `json:"storeLabels,omitempty"`

	// EnableNamedStatusPort enables status port(20180) in the Pod spec.
	// If you set it to `true` for an existing cluster, the TiKV cluster will be rolling updated.
	EnableNamedStatusPort bool `json:"enableNamedStatusPort,omitempty"`

	// ScalePolicy is the scale configuration for TiKV
	// +optional
	ScalePolicy ScalePolicy `json:"scalePolicy,omitempty"`

	// The default number of spare replicas to scale up when using VolumeReplace feature.
	// In multi-az deployments with topology spread constraints you may need to set this to number of zones to avoid
	// zone skew after volume replace (total replicas always whole multiples of zones).
	// Optional: Defaults to 1
	// +kubebuilder:validation:Minimum=0
	// +optional
	SpareVolReplaceReplicas *int32 `json:"spareVolReplaceReplicas,omitempty"`
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

	// ScalePolicy is the scale configuration for TiFlash
	// +optional
	ScalePolicy ScalePolicy `json:"scalePolicy,omitempty"`
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

	// GracefulShutdownTimeout is the timeout of gracefully shutdown a TiCDC pod.
	// Encoded in the format of Go Duration.
	// Defaults to 10m
	// +optional
	GracefulShutdownTimeout *metav1.Duration `json:"gracefulShutdownTimeout,omitempty"`
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

// TiProxySpec contains details of TiProxy members
// +k8s:openapi-gen=true
type TiProxySpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for TiProxy
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Whether enable SSL connection between tiproxy and TiDB server
	SSLEnableTiDB bool `json:"sslEnableTiDB,omitempty"`

	// TLSClientSecretName is the name of secret which stores tidb server client certificate
	// used by TiProxy to check health status.
	// +optional
	TLSClientSecretName *string `json:"tlsClientSecretName,omitempty"`

	// TiProxyCertLayout is the certificate layout of TiProxy that determines how tidb-operator mount cert secrets
	// and how configure TLS configurations for tiproxy.
	// +optional
	CertLayout TiProxyCertLayout `json:"certLayout,omitempty"`

	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tiproxy
	// +optional
	BaseImage string `json:"baseImage"`

	// Config is the Configuration of tiproxy-servers
	// +optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *TiProxyConfigWraper `json:"config,omitempty"`

	// StorageVolumes configure additional storage for TiProxy pods.
	// +optional
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// The storageClassName of the persistent volume for TiProxy data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// ServerLabels defines the server labels of the TiProxy.
	// Using both this field and config file to manage the labels is an undefined behavior.
	// Note these label keys are managed by TiDB Operator, it will be set automatically and you can not modify them:
	//  - region, topology.kubernetes.io/region
	//  - zone, topology.kubernetes.io/zone
	//  - host
	ServerLabels map[string]string `json:"serverLabels,omitempty"`
}

// LogTailerSpec represents an optional log tailer sidecar container
// +k8s:openapi-gen=true
type LogTailerSpec struct {
	corev1.ResourceRequirements `json:",inline"`

	// If true, we use native sidecar feature to tail log
	// It requires enable feature gate "SidecarContainers"
	// This feature is introduced at 1.28, default enabled at 1.29, and GA at 1.33
	// See https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/
	// and https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
	// +optional
	UseSidecar bool `json:"useSidecar,omitempty"`
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

	// Whether enable `tidb_auth_token` authentication method. The tidb_auth_token authentication method is used only for the internal operation of TiDB Cloud.
	// Optional: Defaults to false
	// +optional
	TokenBasedAuthEnabled *bool `json:"tokenBasedAuthEnabled,omitempty"`

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

	// Initializer is the init configurations of TiDB
	//
	// +optional
	Initializer *TiDBInitializer `json:"initializer,omitempty"`

	// BootstrapSQLConfigMapName is the name of the ConfigMap which contains the bootstrap SQL file with the key `bootstrap-sql`,
	// which will only be executed when a TiDB cluster bootstrap on the first time.
	// The field should be set ONLY when create a TC, since it only take effect on the first time bootstrap.
	// Only v6.5.1+ supports this feature.
	// +optional
	BootstrapSQLConfigMapName *string `json:"bootstrapSQLConfigMapName,omitempty"`

	// ScalePolicy is the scale configuration for TiDB.
	// +optional
	ScalePolicy ScalePolicy `json:"scalePolicy,omitempty"`

	// CustomizedStartupProbe is the customized startup probe for TiDB.
	// You can provide your own startup probe for TiDB.
	// The image will be an init container, and the tidb-server container will copy the probe binary from it, and execute it.
	// The probe binary in the image should be placed under the root directory, i.e., `/your-probe`.
	// +optional
	CustomizedStartupProbe *CustomizedProbe `json:"customizedStartupProbe,omitempty"`

	// Arguments is the extra command line arguments for TiDB server.
	// +optional
	Arguments []string `json:"arguments,omitempty"`

	// ServerLabels defines the server labels of the TiDB server.
	// Using both this field and config file to manage the labels is an undefined behavior.
	// Note these label keys are managed by TiDB Operator, it will be set automatically and you can not modify them:
	//  - region, topology.kubernetes.io/region
	//  - zone, topology.kubernetes.io/zone
	//  - host
	ServerLabels map[string]string `json:"serverLabels,omitempty"`
}

type CustomizedProbe struct {
	// Image is the image of the probe binary.
	// +required
	Image string `json:"image"`
	// BinaryName is the name of the probe binary.
	// +required
	BinaryName string `json:"binaryName"`
	// Args is the arguments of the probe binary.
	// +optional
	Args []string `json:"args"`
	// Number of seconds after the container has started before liveness probes are initiated.
	// +optional
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`
	// Number of seconds after which the probe times out.
	// Defaults to 1 second. Minimum value is 1.
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
	// How often (in seconds) to perform the probe.
	// Default to 10 seconds. Minimum value is 1.
	// +optional
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.
	// +optional
	SuccessThreshold *int32 `json:"successThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Defaults to 3. Minimum value is 1.
	// +optional
	FailureThreshold *int32 `json:"failureThreshold,omitempty"`
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

// Probe contains details of probing tidb.
// +k8s:openapi-gen=true
// default probe by TCPPort on tidb 4000 / tikv 20160 / pd 2349.
type Probe struct {
	// "tcp" will use TCP socket to connect component port.
	//
	// "command" will probe the status api of tidb.
	// This will use curl command to request tidb, before v4.0.9 there is no curl in the image,
	// So do not use this before v4.0.9.
	// +kubebuilder:validation:Enum=tcp;command
	// +optional
	Type *string `json:"type,omitempty"` // tcp or command
	// Number of seconds after the container has started before liveness probes are initiated.
	// Default to 10 seconds.
	// +kubebuilder:validation:Minimum=0
	// +optional
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`
	// How often (in seconds) to perform the probe.
	// Default to Kubernetes default (10 seconds). Minimum value is 1.
	// +kubebuilder:validation:Minimum=1
	// +optional
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`
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

	// If true, we use native sidecar feature to tail log
	// It requires enable feature gate "SidecarContainers"
	// This feature is introduced at 1.28, default enabled at 1.29, and GA at 1.33
	// See https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/
	// and https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
	// +optional
	UseSidecar bool `json:"useSidecar,omitempty"`
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
	// If the container names in this field match with the ones generated by
	// TiDB Operator, the container configurations will be merged into the
	// containers generated by TiDB Operator via strategic merge patch.
	// If the container names in this field do not match with the ones
	// generated by TiDB Operator, the container configurations will be
	// appended to the Pod container spec directly.
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

	// SuspendAction defines the suspend actions for all component.
	// +optional
	SuspendAction *SuspendAction `json:"suspendAction,omitempty"`

	// ReadinessProbe describes actions that probe the components' readiness.
	// the default behavior is like setting type as "tcp"
	// +optional
	ReadinessProbe *Probe `json:"readinessProbe,omitempty"`
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

	// loadBalancerClass is the class of the load balancer implementation this Service belongs to.
	// If specified, the value of this field must be a label-style identifier, with an optional prefix,
	// e.g. "internal-vip" or "example.com/internal-vip". Unprefixed names are reserved for end-users.
	// This field can only be set when the Service type is 'LoadBalancer'. If not set, the default load
	// balancer implementation is used, today this is typically done through the cloud provider integration,
	// but should apply for any default implementation. If set, it is assumed that a load balancer
	// implementation is watching for Services with a matching class. Any default load balancer
	// implementation (e.g. cloud providers) should ignore Services that set this field.
	// This field can only be set when creating or updating a Service to type 'LoadBalancer'.
	// Once set, it can not be changed. This field will be wiped when a service is updated to a non 'LoadBalancer' type.
	// +optional
	LoadBalancerClass *string `json:"loadBalancerClass,omitempty" protobuf:"bytes,21,opt,name=loadBalancerClass"`
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

// SuspendAction defines the suspend actions for a component.
//
// +k8s:openapi-gen=true
type SuspendAction struct {
	SuspendStatefulSet bool `json:"suspendStatefulSet,omitempty"`
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
	// Indicates that a Volume replace using VolumeReplacing feature is in progress.
	VolReplaceInProgress bool `json:"volReplaceInProgress,omitempty"`
}

// PDMSStatus is PD microservice status
type PDMSStatus struct {
	Name string `json:"name,omitempty"`
	// +optional
	Synced      bool                    `json:"synced"`
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	// Volumes contains the status of all volumes.
	Volumes map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
	// Members contains other service in current TidbCluster
	Members []string `json:"members,omitempty"`
	Image   string   `json:"image,omitempty"`
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
	HostDown      bool                      `json:"hostDown,omitempty"`
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
	// Indicates that a Volume replace using VolumeReplacing feature is in progress.
	VolReplaceInProgress bool `json:"volReplaceInProgress,omitempty"`
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

var EvictLeaderAnnKeys = []string{EvictLeaderAnnKey, EvictLeaderAnnKeyForResize}

const (
	// EvictLeaderAnnKey is the annotation key to evict leader used by user.
	EvictLeaderAnnKey = "tidb.pingcap.com/evict-leader"
	// EvictLeaderAnnKeyForResize is the annotation key to evict leader user by pvc resizer.
	EvictLeaderAnnKeyForResize = "tidb.pingcap.com/evict-leader-for-resize"
	// PDLeaderTransferAnnKey is the annotation key to transfer PD leader used by user.
	PDLeaderTransferAnnKey = "tidb.pingcap.com/pd-transfer-leader"
	// TiDBGracefulShutdownAnnKey is the annotation key to graceful shutdown tidb pod by user.
	TiDBGracefulShutdownAnnKey = "tidb.pingcap.com/tidb-graceful-shutdown"
	// TiKVEvictLeaderExpirationTimeAnnKey is the annotation key to expire evict leader annotation. Type: time.RFC3339.
	TiKVEvictLeaderExpirationTimeAnnKey = "tidb.pingcap.com/tikv-evict-leader-expiration-time"
	// PDLeaderTransferExpirationTimeAnnKey is the annotation key to expire transfer leader annotation. Type: time.RFC3339.
	PDLeaderTransferExpirationTimeAnnKey = "tidb.pingcap.com/pd-evict-leader-expiration-time"
	// ReplaceVolumeAnnKey is the annotation key to replace disks used by pod.
	ReplaceVolumeAnnKey = "tidb.pingcap.com/replace-volume"
)

// The `Value` of annotation controls the behavior when the leader count drops to zero, the valid value is one of:
//
// - `none`: doing nothing.
// - `delete-pod`: delete pod and remove the evict-leader scheduler from PD.
const (
	EvictLeaderValueNone      = "none"
	EvictLeaderValueDeletePod = "delete-pod"
)

// The `Value` of PD leader transfer annotation controls the behavior when the leader is transferred to another member, the valid value is one of:
//
// - `none`: doing nothing.
// - `delete-pod`: delete pod.
const (
	TransferLeaderValueNone      = "none"
	TransferLeaderValueDeletePod = "delete-pod"
)

// The `Value` of TiDB deletion annotation controls the behavior when the tidb pod got deleted, the valid value is one of:
//
// - `none`: doing nothing.
// - `delete-pod`: delete pod.
const (
	TiDBPodDeletionValueNone = "none"
	TiDBPodDeletionDeletePod = "delete-pod"
)

// Only supported value for ReplaceVolume Annotation.
const (
	ReplaceVolumeValueTrue = "true"
)

type EvictLeaderStatus struct {
	PodCreateTime metav1.Time `json:"podCreateTime,omitempty"`
	BeginTime     metav1.Time `json:"beginTime,omitempty"`
	Value         string      `json:"value,omitempty"`
}

const (
	// It means whether some pods are evicting leader
	// This condition is used to avoid too many pods evict leader at same time
	// Normally we only allow one pod evicts leader.
	// TODO: set this condition before all leader eviction behavior
	ConditionTypeLeaderEvicting = "LeaderEvicting"
)

// TiKVStatus is TiKV status
type TiKVStatus struct {
	Synced          bool                          `json:"synced,omitempty"`
	Phase           MemberPhase                   `json:"phase,omitempty"`
	BootStrapped    bool                          `json:"bootStrapped,omitempty"`
	StatefulSet     *apps.StatefulSetStatus       `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore          `json:"stores,omitempty"` // key: store id
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
	// Indicates that a Volume replace using VolumeReplacing feature is in progress.
	VolReplaceInProgress bool `json:"volReplaceInProgress,omitempty"`
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
	// Indicates that a Volume replace using VolumeReplacing feature is in progress.
	VolReplaceInProgress bool `json:"volReplaceInProgress,omitempty"`
}

// TiProxyMember is TiProxy member
type TiProxyMember struct {
	Name   string `json:"name"`
	Health bool   `json:"health"`
	// Additional healthinfo if it is healthy.
	// +optional
	Info string `json:"info"`
	// Last time the health transitioned from one to another.
	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// TiProxyStatus is TiProxy status
type TiProxyStatus struct {
	Synced      bool                                       `json:"synced,omitempty"`
	Phase       MemberPhase                                `json:"phase,omitempty"`
	Members     map[string]TiProxyMember                   `json:"members,omitempty"`
	StatefulSet *apps.StatefulSetStatus                    `json:"statefulSet,omitempty"`
	Volumes     map[StorageVolumeName]*StorageVolumeStatus `json:"volumes,omitempty"`
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
	// LeaderCountBeforeUpgrade records the leader count before upgrade.
	//
	// It is set when evicting leader and used to wait for most leaders to transfer back after upgrade.
	// It is unset after leader transfer is completed.
	LeaderCountBeforeUpgrade *int32 `json:"leaderCountBeforeUpgrade,omitempty"`
}

// TiKVFailureStore is the tikv failure store information
type TiKVFailureStore struct {
	PodName      string                    `json:"podName,omitempty"`
	StoreID      string                    `json:"storeID,omitempty"`
	PVCUIDSet    map[types.UID]EmptyStruct `json:"pvcUIDSet,omitempty"`
	StoreDeleted bool                      `json:"storeDeleted,omitempty"`
	HostDown     bool                      `json:"hostDown,omitempty"`
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
	// ForcePathStyle for the backup and restore to connect s3 with path style(true) or virtual host(false).
	ForcePathStyle *bool `json:"forcePathStyle,omitempty"`
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

	// SnapshotsDeleteRatio represents the number of snapshots deleted per second
	// +kubebuilder:default=1
	SnapshotsDeleteRatio float64 `json:"snapshotsDeleteRatio,omitempty"`
}

type Progress struct {
	// Step is the step name of progress
	Step string `json:"step,omitempty"`
	// Progress is the backup progress value
	Progress float64 `json:"progress,omitempty"`
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
	// From is the tidb cluster that needs to backup.
	From *TiDBAccessConfig `json:"from,omitempty"`
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
	Command LogSubCommandType      `json:"command,omitempty"`
	Type    BackupConditionType    `json:"type"`
	Status  corev1.ConditionStatus `json:"status"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
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
	Conditions []BackupCondition `json:"conditions,omitempty"`
	// LogSubCommandStatuses is the detail status of log backup subcommands, record each command separately, but only record the last command.
	LogSubCommandStatuses map[LogSubCommandType]LogSubCommandStatus `json:"logSubCommandStatuses,omitempty"`
	// Progresses is the progress of backup.
	// +nullable
	Progresses []Progress `json:"progresses,omitempty"`
	// BackoffRetryStatus is status of the backoff retry, it will be used when backup pod or job exited unexpectedly
	BackoffRetryStatus []BackoffRetryRecord `json:"backoffRetryStatus,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupSchedule is a backup schedule of tidb cluster.
//
// +k8s:openapi-gen=true
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
	// CompactInterval is to specify how long backups we want to compact.
	CompactInterval *string `json:"compactInterval,omitempty"`
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
	// LastCompactProgress represents the endTs of the last compact
	LastCompactProgress *metav1.Time `json:"lastCompactProgress,omitempty"`
	// LastCompactExecutionTs represents the execution time of the last compact
	LastCompactExecutionTs *metav1.Time `json:"lastCompactExecutionTs,omitempty"`
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

// PruneType represents the prune type for restore.
// +k8s:openapi-gen=true
type PruneType string

const (
	// PruneTypeAfterFailed represents prune after failed.
	PruneTypeAfterFailed PruneType = "afterFailed"
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
	// RestorePruneScheduled means a prune job has been scheduled after restore failure.
	RestorePruneScheduled RestoreConditionType = "PruneScheduled"
	// RestorePruneRunning means the prune job is currently running.
	RestorePruneRunning RestoreConditionType = "PruneRunning"
	// RestorePruneComplete means the prune job has successfully completed.
	RestorePruneComplete RestoreConditionType = "PruneComplete"
	// RestorePruneFailed means the prune job has failed.
	RestorePruneFailed RestoreConditionType = "PruneFailed"
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
	// +optional
	PitrRestoredTs string `json:"pitrRestoredTs,omitempty"`
	// Prune is the prune type for restore, it is optional and can only have two valid values: afterFailed/alreadyFailed
	// +optional
	// +kubebuilder:validation:Enum:=afterFailed
	Prune PruneType `json:"prune,omitempty"`
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
	// +kubebuilder:default=0
	BackoffLimit int32 `json:"backoffLimit,omitempty"`
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

	// LivenessProbe describes actions that probe the discovery's liveness.
	// the default behavior is like setting type as "tcp"
	// NOTE: only used for TiDB Operator discovery now,
	// for other components, the auto failover feature may be used instead.
	// +optional
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`

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

	// SuspendAction defines the suspend actions for all component.
	// +optional
	SuspendAction *SuspendAction `json:"suspendAction,omitempty"`

	// PreferIPv6 indicates whether to prefer IPv6 addresses for all components.
	PreferIPv6 bool `json:"preferIPv6,omitempty"`
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

	// Start up script version
	// +optional
	// +kubebuilder:validation:Enum:="";"v1"
	StartUpScriptVersion string `json:"startUpScriptVersion,omitempty"`
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
	// ModifiedCount is the count of modified volumes.
	// +optional
	ModifiedCount int `json:"modifiedCount"`
	// CurrentCapacity is the current capacity of the volume.
	// If any volume is resizing, it is the capacity before resizing.
	// If all volumes are resized, it is the resized capacity and same as desired capacity.
	// +optional
	CurrentCapacity resource.Quantity `json:"currentCapacity"`
	// ModifiedCapacity is the modified capacity of the volume.
	// +optional
	ModifiedCapacity resource.Quantity `json:"modifiedCapacity"`
	// CurrentStorageClass is the modified capacity of the volume.
	// +optional
	CurrentStorageClass string `json:"currentStorageClass"`
	// ModifiedStorageClass is the modified storage calss of the volume.
	// +optional
	ModifiedStorageClass string `json:"modifiedStorageClass"`

	// (Deprecated) ResizedCapacity is the desired capacity of the volume.
	// +optional
	ResizedCapacity resource.Quantity `json:"resizedCapacity"`
	// (Deprecated) ResizedCount is the count of volumes whose capacity is equal to `resizedCapacity`.
	// +optional
	ResizedCount int `json:"resizedCount"`
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
	// WhenUnsatisfiable is default set to DoNotSchedule
	// LabelSelector is generated by component type
	// See pkg/apis/pingcap/v1alpha1/tidbcluster_component.go#TopologySpreadConstraints()
	TopologyKey string `json:"topologyKey"`

	// MaxSkew describes the degree to which pods may be unevenly distributed.
	// When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference
	// between the number of matching pods in the target topology and the global minimum.
	// The global minimum is the minimum number of matching pods in an eligible domain
	// or zero if the number of eligible domains is less than MinDomains.
	// For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same
	// labelSelector spread as 2/2/1:
	// In this case, the global minimum is 1.
	// +-------+-------+-------+
	// | zone1 | zone2 | zone3 |
	// +-------+-------+-------+
	// |  P P  |  P P  |   P   |
	// +-------+-------+-------+
	// - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 2/2/2;
	// scheduling it onto zone1(zone2) would make the ActualSkew(3-1) on zone1(zone2)
	// violate MaxSkew(1).
	// - if MaxSkew is 2, incoming pod can be scheduled onto any zone.
	// When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence
	// to topologies that satisfy it.
	// Default value is 1.
	// +kubebuilder:default=1
	// +optional
	MaxSkew int32 `json:"maxSkew" protobuf:"varint,1,opt,name=maxSkew"`

	// MinDomains indicates a minimum number of eligible domains.
	// When the number of eligible domains with matching topology keys is less than minDomains,
	// Pod Topology Spread treats "global minimum" as 0, and then the calculation of Skew is performed.
	// And when the number of eligible domains with matching topology keys equals or greater than minDomains,
	// this value has no effect on scheduling.
	// As a result, when the number of eligible domains is less than minDomains,
	// scheduler won't schedule more than maxSkew Pods to those domains.
	// If value is nil, the constraint behaves as if MinDomains is equal to 1.
	// Valid values are integers greater than 0.
	// When value is not nil, WhenUnsatisfiable must be DoNotSchedule.
	//
	// For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and pods with the same
	// labelSelector spread as 2/2/2:
	// +-------+-------+-------+
	// | zone1 | zone2 | zone3 |
	// +-------+-------+-------+
	// |  P P  |  P P  |  P P  |
	// +-------+-------+-------+
	// The number of domains is less than 5(MinDomains), so "global minimum" is treated as 0.
	// In this situation, new pod with the same labelSelector cannot be scheduled,
	// because computed skew will be 3(3 - 0) if new Pod is scheduled to any of the three zones,
	// it will violate MaxSkew.
	// +optional
	MinDomains *int32 `json:"minDomains,omitempty" protobuf:"varint,5,opt,name=minDomains"`

	// NodeAffinityPolicy indicates how we will treat Pod's nodeAffinity/nodeSelector
	// when calculating pod topology spread skew. Options are:
	// - Honor: only nodes matching nodeAffinity/nodeSelector are included in the calculations.
	// - Ignore: nodeAffinity/nodeSelector are ignored. All nodes are included in the calculations.
	//
	// If this value is nil, the behavior is equivalent to the Honor policy.
	// +optional
	NodeAffinityPolicy *corev1.NodeInclusionPolicy `json:"nodeAffinityPolicy,omitempty" protobuf:"bytes,6,opt,name=nodeAffinityPolicy"`

	// MatchLabels is used to overwrite generated corev1.TopologySpreadConstraints.LabelSelector
	// corev1.TopologySpreadConstraint generated in component_spec.go will set a
	// LabelSelector automatically with some KV.
	// Historically, it is l["comp"] = "" for component tiproxy. And we will use
	// MatchLabels to keep l["comp"] = "" for old clusters with tiproxy
	// +optional
	MatchLabels label.Label `json:"matchLabels"`
}

// Failover contains the failover specification.
// +k8s:openapi-gen=true
type Failover struct {
	// RecoverByUID indicates that TiDB Operator will recover the failover by this UID,
	// it takes effect only when set `spec.recoverFailover=false`
	// +optional
	RecoverByUID types.UID `json:"recoverByUID,omitempty"`
}

type ScalePolicy struct {
	// ScaleInParallelism configures max scale in replicas for TiKV stores.
	// +kubebuilder:default=1
	// +optional
	ScaleInParallelism *int32 `json:"scaleInParallelism,omitempty"`

	// ScaleOutParallelism configures max scale out replicas for TiKV stores.
	// +kubebuilder:default=1
	// +optional
	ScaleOutParallelism *int32 `json:"scaleOutParallelism,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="cpbk"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`,description="The current status of the compact backup"
// +kubebuilder:printcolumn:name="EndTs",type=string,JSONPath=`.status.endTs`,description="The endTs of the compact backup"
// +kubebuilder:printcolumn:name="Progress",type=string,JSONPath=`.status.progress`,description="The detailed progress of a running compact backup"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="The message of the compact backup"
type CompactBackup struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec CompactSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status CompactStatus `json:"status,omitempty"`
}

// CompactSpec contains the backup specification for a tidb cluster.
// +k8s:openapi-gen=true
type CompactSpec struct {
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
	// StorageProvider configures where and how backups should be stored.
	// *** Note: This field should generally not be left empty, unless you are certain the storage provider
	// *** can be obtained from another source, such as a schedule CR.
	StorageProvider `json:",inline"`
	// StartTs is the start ts of the compact backup.
	// Format supports TSO or datetime, e.g. '400036290571534337', '2018-05-11 01:42:23'.
	StartTs string `json:"startTs,omitempty"`
	// EndTs is the end ts of the compact backup.
	// Format supports TSO or datetime, e.g. '400036290571534337', '2018-05-11 01:42:23'.
	// Default is current timestamp.
	// +optional
	EndTs string `json:"endTs,omitempty"`
	// Concurrency is the concurrency of compact backup job
	// +kubebuilder:default=4
	Concurrency int `json:"concurrency,omitempty"`
	// Base tolerations of backup Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// ToolImage specifies the br image used in compact `Backup`.
	// For examples `spec.toolImage: pingcap/br:v4.0.8`
	// For BR image, if it does not contain tag, Pod will use the same version in tc 'BrImage:${TiKV_Version}'.
	// +optional
	ToolImage string `json:"toolImage,omitempty"`
	// TikvImage specifies the tikv image used in compact `Backup`.
	// For examples `spec.tikvImage: pingcap/tikv:v9.0.0`
	// For TiKV image, if it does not contain tag, Pod will use the same version in tc 'TiKVImage:${TiKV_Version}'.
	// +optional
	TiKVImage string `json:"tikvImage,omitempty"`
	// BRConfig is the configs for BR
	// *** Note: This field should generally not be left empty, unless you are certain the BR config
	// *** can be obtained from another source, such as a schedule CR.
	BR *BRConfig `json:"br,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Affinity of backup Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Use KMS to decrypt the secrets
	UseKMS bool `json:"useKMS,omitempty"`
	// Specify service account of backup
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// PriorityClassName of Backup Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// BackoffRetryPolicy the backoff retry policy, currently only valid for snapshot backup
	// +kubebuilder:default=6
	MaxRetryTimes int32 `json:"maxRetryTimes,omitempty"`

	// Additional volumes of component pod.
	// +optional
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`
	// Additional volume mounts of component pod.
	// +optional
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
}

// CompactRetryRecord is the record of compact backoff retry
type CompactRetryRecord struct {
	// RetryNum is the number of retry
	RetryNum int `json:"retryNum,omitempty"`
	// DetectFailedAt is the time when detect failure
	DetectFailedAt metav1.Time `json:"detectFailedAt,omitempty"`
	// Reason is the reason of retry
	RetryReason string `json:"retryReason,omitempty"`
}

type CompactStatus struct {
	// State is the current state of the backup
	State string `json:"state,omitempty"`
	// Progress is the detailed progress of a running backup
	Progress string `json:"progress,omitempty"`
	// Message is the error message of the backup
	Message string `json:"message,omitempty"`
	// endTs is the real endTs processed by the compact backup
	EndTs string `json:"endTs,omitempty"`
	// RetryStatus is status of the backoff retry, it will be used when backup pod or job exited unexpectedly
	RetryStatus []CompactRetryRecord `json:"backoffRetryStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// CompactList contains a list of Compact Backup.
type CompactBackupList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []CompactBackup `json:"items"`
}
