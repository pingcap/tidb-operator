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
	"github.com/pingcap/tidb-operator/pkg/util/config"
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

// `MemberType` represents member type.
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
	// SlowLogTailerMemberType is tidb log tailer container type
	SlowLogTailerMemberType MemberType = "slowlog"
	// UnknownMemberType is unknown container type
	UnknownMemberType MemberType = "unknown"
)

// `MemberPhase` is the current state of member.
type MemberPhase string

const (
	// NormalPhase represents normal state of TiDB cluster.
	NormalPhase MemberPhase = "Normal"
	// UpgradePhase represents the upgrade state of TiDB cluster.
	UpgradePhase MemberPhase = "Upgrade"
)

// `ConfigUpdateStrategy` represents the strategy to update configuration
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

// +k8s:openapi-gen=true
// `TidbCluster` is the control script's spec.
type TidbCluster struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// `Spec` defines the behavior of a TiDB cluster.
	Spec TidbClusterSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the TiDB cluster.
	Status TidbClusterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// `TidbClusterList` is TidbCluster list.
type TidbClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbCluster `json:"items"`
}

// +k8s:openapi-gen=true
// `TidbClusterSpec` describes the attributes that a user creates on a TiDB cluster.
type TidbClusterSpec struct {
	// PD cluster spec
	PD PDSpec `json:"pd"`

	// TiDB cluster spec
	TiDB TiDBSpec `json:"tidb"`

	// TiKV cluster spec
	TiKV TiKVSpec `json:"tikv"`

	// TiFlash cluster spec
	// +optional
	TiFlash *TiFlashSpec `json:"tiflash,omitempty"`

	// Pump cluster spec
	// +optional
	Pump *PumpSpec `json:"pump,omitempty"`

	// Helper spec
	// +optional
	Helper *HelperSpec `json:"helper,omitempty"`

	// Indicates that the TiDB cluster is paused and will not be processed by
	// the controller.
	// +optional
	Paused bool `json:"paused,omitempty"`


	// TODO: remove `optional` after defaulting logic introduced
	
	// TiDB cluster version.
	// +optional
	Version string `json:"version"`

	// `SchedulerName` of TiDB cluster Pods
	// +kubebuilder:default=tidb-scheduler
	SchedulerName string `json:"schedulerName,omitempty"`

	// Refers to the persistent volume reclaim policy applied to the PVs that are consumed by the TiDB cluster.
	// +kubebuilder:default=Recycle
	PVReclaimPolicy corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// `ImagePullPolicy` of TiDB cluster Pods
	// +kubebuilder:default=IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// `ConfigUpdateStrategy` determines how the configuration change is applied to the cluster.<br>
	// `UpdateStrategyInPlace` will update the ConfigMap of configuration in-place, and an extra rolling-update of the
	// cluster component is needed to reload the configuration change.<br>
	// `UpdateStrategyRollingUpdate` will create a new ConfigMap with the new configuration and rolling-update the
	// related components to use the new ConfigMap, that is, the new configuration will be applied automatically.
	// +kubebuilder:validation:Enum=InPlace,RollingUpdate
	// +kubebuilder:default=InPlacne
	ConfigUpdateStrategy ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// Determines whether to enable PVC reclaim for orphan PVC left by statefulset scale-in.<br>
	// Optional: Defaults to `false`
	// +optional
	EnablePVReclaim *bool `json:"enablePVReclaim,omitempty"`

	// Determines whether to enable the TLS connection between TiDB server components.<br>
	// Optional: Defaults to `nil`
	// +optional
	TLSCluster *TLSCluster `json:"tlsCluster,omitempty"`

	// Determines whether Hostnetwork is enabled for TiDB cluster Pods.<br>
	// Optional: Defaults to false
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// `Affinity` of TiDB cluster Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// `PriorityClassName` of TiDB cluster Pods
	// Optional: Defaults to omitted
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Base node selectors of TiDB cluster Pods. Components may add or override selectors upon this respectively.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Base annotations of TiDB cluster Pods. Components may add or override selectors upon this respectively.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Base tolerations of TiDB cluster Pods. Components may add more tolerations upon this respectively.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Time zone of TiDB cluster Pods<br>
	// Optional: Defaults to `UTC`
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// Services list non-headless services type used in TidbCluster.<br>
	// Deprecated
	// +k8s:openapi-gen=false
	Services []Service `json:"services,omitempty"`
}

// `TidbClusterStatus` represents the current status of a TiDB cluster.
type TidbClusterStatus struct {
	ClusterID string        `json:"clusterID,omitempty"`
	PD        PDStatus      `json:"pd,omitempty"`
	TiKV      TiKVStatus    `json:"tikv,omitempty"`
	TiDB      TiDBStatus    `json:"tidb,omitempty"`
	Pump      PumpStatus    `josn:"pump,omitempty"`
	TiFlash   TiFlashStatus `json:"tiflash,omitempty"`
}

// +k8s:openapi-gen=true
// `PDSpec` contains details of PD members.
type PDSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// Base image of the component. Image tag is now allowed during validation.
	// +kubebuilder:default=pingcap/pd
	// +optional
	BaseImage string `json:"baseImage"`

	// Defines a Kubernetes service of PD cluster.<br>
	// Optional: Defaults to `.spec.services` for backward compatibility
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// Limits the max number of replicas that could be added in failover. 0 means no failover.<br>
	// Optional: Defaults to `3`
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The `storageClassName` of the persistent volume for PD data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// The Configuration of pd-servers
	// +optional
	Config *PDConfig `json:"config,omitempty"`
}

// +k8s:openapi-gen=true
// `TiKVSpec` contains details of TiKV members.
type TiKVSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specifies a Service Account for TiKV.
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// Base image of the component. Image tag is now allowed during validation.
	// +kubebuilder:default=pingcap/tikv
	// +optional
	BaseImage string `json:"baseImage"`

	// Determines whether to create the TiKV container in privileged mode. It is highly discouraged to enable this in
	// critical environment.
	// Optional: defaults to `false`
	// +optional
	Privileged *bool `json:"privileged,omitempty"`

	// Limits the max number of replicas that could be added in failover. 0 means no failover.
	// Optional: Defaults to `3`
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The `storageClassName` of the persistent volume for TiKV data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// The Configuration of tikv-servers
	// +optional
	Config *TiKVConfig `json:"config,omitempty"`
}

// `TiFlashSpec` contains details of TiFlash members.
// +k8s:openapi-gen=true
type TiFlashSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specifies a Service Account for TiFlash.
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// Base image of the component. Image tag is now allowed during validation.
	// +kubebuilder:default=pingcap/tiflash
	// +optional
	BaseImage string `json:"baseImage"`

	// Determines whether to create the TiFlash container in privileged mode. It is highly discouraged to enable this in
	// critical environment.
	// Optional: defaults to `false`
	// +optional
	Privileged *bool `json:"privileged,omitempty"`

	// Limits the max number of replicas that could be added in failover. 0 means no failover.
	// Optional: Defaults to `3`
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The persistent volume claims of the TiFlash data storages.<br>
	// TiFlash supports multiple disks.
	StorageClaims []StorageClaim `json:"storageClaims"`

	// The Configuration of TiFlash
	// +optional
	Config *TiFlashConfig `json:"config,omitempty"`

	// The configurations of the log tailers for TiFlash
	// +optional
	LogTailer *LogTailerSpec `json:"logTailer,omitempty"`
}

// +k8s:openapi-gen=true
// `LogTailerSpec` represents an optional log tailer sidecar container.
type LogTailerSpec struct {
	corev1.ResourceRequirements `json:",inline"`
}

// +k8s:openapi-gen=true
// `StorageClaim` contains details of TiFlash storages.
type StorageClaim struct {
	// Represents the minimum resources the volume should have.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Name of the StorageClass required by the claim.
	// More info: <https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1>
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// +k8s:openapi-gen=true
// `TiDBSpec` contains details of TiDB members.
type TiDBSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component. Image tag is now allowed during validation.
	// +kubebuilder:default=pingcap/tidb
	// +optional
	BaseImage string `json:"baseImage"`

	// Defines a Kubernetes service of TiDB cluster.<br>
	// Optional: No kubernetes service will be created by default.
	// +optional
	Service *TiDBServiceSpec `json:"service,omitempty"`

	// Determines whether to enable TiDB Binlog. It is encouraged to not set this field and rely on the default behavior.
	// Optional: Defaults to `true` if PumpSpec is non-nil, otherwise `false`.
	// +optional
	BinlogEnabled *bool `json:"binlogEnabled,omitempty"`

	// Limits the max number of replicas that could be added in failover. 0 means no failover.
	// Optional: Defaults to `3`
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// Determines whether to output the slow log in an separate sidecar container.
	// Optional: Defaults to `true`
	// +optional
	SeparateSlowLog *bool `json:"separateSlowLog,omitempty"`

	// Determines whether to enable the TLS connection between the SQL client and TiDB server.
	// Optional: Defaults to `nil`
	// +optional
	TLSClient *TiDBTLSClient `json:"tlsClient,omitempty"`

	// The spec of the slow log tailer sidecar
	// +optional
	SlowLogTailer *TiDBSlowLogTailerSpec `json:"slowLogTailer,omitempty"`

	// A list of plugins that are loaded by TiDB server. Empty means the plugin is disabled.
	// +optional
	Plugins []string `json:"plugins,omitempty"`

	// The Configuration of tidb-servers
	// +optional
	Config *TiDBConfig `json:"config,omitempty"`
}

// +k8s:openapi-gen=true
// `PumpSpec` contains details of Pump members.
type PumpSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// Base image of the component. Image tag is now allowed during validation.
	// +kubebuilder:default=pingcap/tidb-binlog
	// +optional
	BaseImage string `json:"baseImage"`

	// The `storageClassName` of the persistent volume for Pump data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// TODO: add schema

	// The configuration of Pump cluster.
	// +optional
	config.GenericConfig `json:",inline"`

	// +k8s:openapi-gen=false
	// For backward compatibility with helm chart
	SetTimeZone *bool `json:"setTimeZone,omitempty"`
}

// +k8s:openapi-gen=true
// `HelperSpec` contains details of the helper component.
type HelperSpec struct {
	// Image used to tail slow log and set kernel parameters if necessary. `tail` and `sysctl` must be installed.
	// Optional: Defaults to `busybox:1.26.2`
	// +optional
	Image *string `json:"image,omitempty"`

	// `ImagePullPolicy` of the component. Override the cluster-level `imagePullPolicy` if present.
	// Optional: Defaults to the cluster-level setting
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// +k8s:openapi-gen=true
// `TiDBSlowLogTailerSpec` represents an optional log tailer sidecar with TiDB.
type TiDBSlowLogTailerSpec struct {
	corev1.ResourceRequirements `json:",inline"`

	// Image used for slowlog tailer<br>
	// Deprecated. Use `TidbCluster.HelperImage` instead.
	// +k8s:openapi-gen=false
	Image *string `json:"image,omitempty"`

	// `ImagePullPolicy` of the component. Override the cluster-level `imagePullPolicy` if present.
	// Deprecated. Use `TidbCluster.HelperImagePullPolicy` instead.
	// +k8s:openapi-gen=false
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// +k8s:openapi-gen=true
// `ComponentSpec` is the base spec of each component. The fields should always be accessed by the Basic&lt;Component&gt;Spec() method to respect the cluster-level properties.
type ComponentSpec struct {
	// Image of the component. Override baseImage and version if present.<br>
	// Deprecated
	// +k8s:openapi-gen=false
	Image string `json:"image,omitempty"`

	// Version of the component. Override the cluster-level version if non-empty<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	Version *string `json:"version,omitempty"`

	// `ImagePullPolicy` of the component. Override the cluster-level `imagePullPolicy` if present.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Determines whether Hostnetwork of the component is enabled. Override the cluster-level setting if present.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// `Affinity` of the component. Override the cluster-level one if present.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// `PriorityClassName` of the component. Override the cluster-level one if present.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// `SchedulerName` of the component. Override the cluster-level one if present.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	SchedulerName *string `json:"schedulerName,omitempty"`

	// `NodeSelector` of the component. Merged into the cluster-level `nodeSelector` if non-empty.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// `Annotations` of the component. Merged into the cluster-level `annotations` if non-empty.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// `Tolerations` of the component. Override the cluster-level `tolerations` if non-empty.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// `PodSecurityContext` of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// `ConfigUpdateStrategy` of the component. Override the cluster-level `updateStrategy` if present.<br>
	// Optional: Defaults to the cluster-level setting
	// +optional
	ConfigUpdateStrategy *ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// List of environment variables to set in the container, like
	// v1.Container.Env.<br>
	// Note that following env names cannot be used and may be overridden by
	// tidb-operator built envs.
	// - `NAMESPACE`
	// - `TZ`
	// - `SERVICE_NAME`
	// - `PEER_SERVICE_NAME`
	// - `HEADLESS_SERVICE_NAME`
	// - `SET_NAME`
	// - `HOSTNAME`
	// - `CLUSTER_NAME`
	// - `POD_NAME`
	// - `BINLOG_ENABLED`
	// - `SLOW_LOG_FILE`
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// +k8s:openapi-gen=true
type ServiceSpec struct {
	// Type of the real kubernetes service
	Type corev1.ServiceType `json:"type,omitempty"`

	// Additional annotations of the kubernetes service object
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// The loadBalancerIP of service<br>
	// Optional: Defaults to `omitted`
	// +optional
	LoadBalancerIP *string `json:"loadBalancerIP,omitempty"`

	// The clusterIP of service
	// +optional
	ClusterIP *string `json:"clusterIP,omitempty"`

	// The name of service port
	// +optional
	PortName *string `json:"portName,omitempty"`
}

// +k8s:openapi-gen=true
type TiDBServiceSpec struct {
	// +k8s:openapi-gen=false
	ServiceSpec

	// `ExternalTrafficPolicy` of the service<br>
	// Optional: Defaults to `omitted`
	// +optional
	ExternalTrafficPolicy *corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`

	// Determines whether to expose the status port.<br>
	// Optional: Defaults to `true`
	// +optional
	ExposeStatus *bool `json:"exposeStatus,omitempty"`
}

// +k8s:openapi-gen=false
// Deprecated<br>
// `Service` represent service type used in TidbCluster.
type Service struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// `PDStatus` is PD status.
type PDStatus struct {
	Synced          bool                       `json:"synced,omitempty"`
	Phase           MemberPhase                `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus    `json:"statefulSet,omitempty"`
	Members         map[string]PDMember        `json:"members,omitempty"`
	Leader          PDMember                   `json:"leader,omitempty"`
	FailureMembers  map[string]PDFailureMember `json:"failureMembers,omitempty"`
	UnjoinedMembers map[string]UnjoinedMember  `json:"unjoinedMembers,omitempty"`
	Image           string                     `json:"image,omitempty"`
}

// `PDMember` is PD member.
type PDMember struct {
	Name string `json:"name"`
	// member id is actually a uint64, but apimachinery's json only treats numbers as int64/float64,
	// so uint64 may overflow int64 and thus convert to float64.
	ID        string `json:"id"`
	ClientURL string `json:"clientURL"`
	Health    bool   `json:"health"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// `PDFailureMember` is the pd failure member information.
type PDFailureMember struct {
	PodName       string      `json:"podName,omitempty"`
	MemberID      string      `json:"memberID,omitempty"`
	PVCUID        types.UID   `json:"pvcUID,omitempty"`
	MemberDeleted bool        `json:"memberDeleted,omitempty"`
	CreatedAt     metav1.Time `json:"createdAt,omitempty"`
}

// `UnjoinedMember` is the pd unjoin cluster member information.
type UnjoinedMember struct {
	PodName   string      `json:"podName,omitempty"`
	PVCUID    types.UID   `json:"pvcUID,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// `TiDBStatus` is TiDB status.
type TiDBStatus struct {
	Phase                    MemberPhase                  `json:"phase,omitempty"`
	StatefulSet              *apps.StatefulSetStatus      `json:"statefulSet,omitempty"`
	Members                  map[string]TiDBMember        `json:"members,omitempty"`
	FailureMembers           map[string]TiDBFailureMember `json:"failureMembers,omitempty"`
	ResignDDLOwnerRetryCount int32                        `json:"resignDDLOwnerRetryCount,omitempty"`
	Image                    string                       `json:"image,omitempty"`
}

// `TiDBMember` is TiDB member.
type TiDBMember struct {
	Name   string `json:"name"`
	Health bool   `json:"health"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Node hosting pod of this TiDB member.
	NodeName string `json:"node,omitempty"`
}

// `TiDBFailureMember` is the TiDB failure member information.
type TiDBFailureMember struct {
	PodName   string      `json:"podName,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// `TiKVStatus` is TiKV status.
type TiKVStatus struct {
	Synced          bool                        `json:"synced,omitempty"`
	Phase           MemberPhase                 `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore        `json:"stores,omitempty"`
	TombstoneStores map[string]TiKVStore        `json:"tombstoneStores,omitempty"`
	FailureStores   map[string]TiKVFailureStore `json:"failureStores,omitempty"`
	Image           string                      `json:"image,omitempty"`
}

// `TiFlashStatus` is TiFlash status.
type TiFlashStatus struct {
	Synced          bool                        `json:"synced,omitempty"`
	Phase           MemberPhase                 `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore        `json:"stores,omitempty"`
	TombstoneStores map[string]TiKVStore        `json:"tombstoneStores,omitempty"`
	FailureStores   map[string]TiKVFailureStore `json:"failureStores,omitempty"`
	Image           string                      `json:"image,omitempty"`
}

// `TiKVStores` is either Up/Down/Offline/Tombstone.
type TiKVStore struct {
	// store id is also uint64. For the same reason as pd id, we store id as string.
	ID                string      `json:"id"`
	PodName           string      `json:"podName"`
	IP                string      `json:"ip"`
	LeaderCount       int32       `json:"leaderCount"`
	State             string      `json:"state"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// `TiKVFailureStore` is the tikv failure store information.
type TiKVFailureStore struct {
	PodName   string      `json:"podName,omitempty"`
	StoreID   string      `json:"storeID,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// `PumpStatus` is Pump status.
type PumpStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}

// `TiDBTLSClient` can enable TLS connection between TiDB server and MySQL client.
type TiDBTLSClient struct {
	// When enabled, TiDB will accept TLS encrypted connections from MySQL client.<br><br>
	// The steps to enable this feature:<br>
	// &nbsp;&nbsp;1. Generate a TiDB server-side certificate and a client-side certificate for the TiDB cluster.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;There are multiple ways to generate certificates:<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- <a href="https://pingcap.com/docs/stable/how-to/secure/enable-tls-clients/">user-provided certificates</a><br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- use the <a href="https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/">K8s built-in certificate signing system signed certificates</a><br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- or use <a href="https://cert-manager.io/">cert-manager signed certificates</a><br>
	// &nbsp;&nbsp;2. Create a K8s Secret object which contains the TiDB server-side certificate created above.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;The name of this Secret must be: ${clusterName}-tidb-server-secret.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;kubectl create secret generic ${clusterName}-tidb-server-secret --namespace=${namespace} --from-file=tls.crt=&lt;path/to/tls.crt&gt; --from-file=tls.key=&lt;path/to/tls.key&gt; --from-file=ca.crt=&lt;path/to/ca.crt&gt;<br>
	// &nbsp;&nbsp;3. Create a K8s Secret object which contains the TiDB client-side certificate created above which will be used by TiDB Operator.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;The name of this Secret must be: ${clusterName}-tidb-client-secret.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;kubectl create secret generic ${clusterName}-tidb-client-secret --namespace=${namespace} --from-file=tls.crt=&lt;path/to/tls.crt&gt; --from-file=tls.key=&lt;path/to/tls.key&gt; --from-file=ca.crt=&lt;path/to/ca.crt&gt;<br>
	// &nbsp;&nbsp;4. Set Enabled to `true`.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// Specifies a secret of client cert for backup/restore<br>
	// Optional: Defaults to &lt;cluster&gt;-tidb-client-secret<br><br>
	// +optional
	// If you want to specify a secret for backup/restore, generate a Secret Object according to the third step of the above procedure. The difference is that the Secret Name can be freely defined, and you can then copy the Secret Name to TLSSecret.<br>
	// This field only works in backup/restore process.
	TLSSecret string `json:"tlsSecret,omitempty"`
}

// TLSCluster can enable TLS connection between TiDB server components.
// For details, see <a href="https://pingcap.com/docs/stable/how-to/secure/enable-tls-between-components/">Enable TLS Authentication</a>.
type TLSCluster struct {
	// Enables mutual TLS authentication among TiDB components.<br>
	// Once enabled, the mutual authentication applies to all components,
	// and it does not support applying to only part of the components.<br><br>
	// The steps to enable this feature:<br>
	// &nbsp;&nbsp;1. Generate TiDB server components certificates and a client-side certificate for them.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;There are multiple ways to generate these certificates:<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- <a href="https://pingcap.com/docs/stable/how-to/secure/enable-tls-clients/">user-provided certificates</a><br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- use the <a href="https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/">K8s built-in certificate signing system signed certificates</a><br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- or use <a href="https://cert-manager.io/">cert-manager signed certificates</a><br>
	// &nbsp;&nbsp;2. Create one secret object for one component which contains the certificates created above.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;The name of this Secret must be: ${clusterName}-&lt;componentName&gt;-cluster-secret.<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;For PD: kubectl create secret generic ${clusterName}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=&lt;path/to/tls.crt&gt; --from-file=tls.key=&lt;path/to/tls.key&gt; --from-file=ca.crt=&lt;path/to/ca.crt&gt;<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;For TiKV: kubectl create secret generic ${clusterName}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt><br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;For TiDB: kubectl create secret generic ${clusterName}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=<path/to/&lt;tls.crt&gt; --from-file=tls.key=&lt;path/to/tls.key&gt; --from-file=ca.crt=&lt;path/to/ca.crt&gt;<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;For Client: kubectl create secret generic ${clusterName}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=&lt;path/to/tls.crt&gt; --from-file=tls.key=&lt;path/to/tls.key&gt; --from-file=ca.crt=&lt;path/to/ca.crt&gt;<br>
	// &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Same for other components.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// `Backup` is a backup of the TiDB cluster.
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
// `BackupList` contains a list of Backup.
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []Backup `json:"items"`
}

// +k8s:openapi-gen=true
// `BackupStorageType` represents the backend storage type of backup.
type BackupStorageType string

const (
	// `BackupStorageTypeS3` represents all storage that is compatible with the Amazon S3.
	BackupStorageTypeS3 BackupStorageType = "s3"
	// `BackupStorageTypeGcs` represents the Google Cloud storage.
	BackupStorageTypeGcs BackupStorageType = "gcs"
	// `BackupStorageTypeUnknown` represents the unknown storage type.
	BackupStorageTypeUnknown BackupStorageType = "unknown"
)

// +k8s:openapi-gen=true
// `S3StorageProviderType` represents the specific storage provider that implements the S3 interface.
type S3StorageProviderType string

const (
	// `S3StorageProviderTypeCeph` means that the S3 compliant storage provider is ceph.
	S3StorageProviderTypeCeph S3StorageProviderType = "ceph"
	// S3StorageProviderTypeAWS means that the S3 compliant storage provider is AWS.
	S3StorageProviderTypeAWS S3StorageProviderType = "aws"
)

// +k8s:openapi-gen=true
// `StorageProvider` defines the configuration for storing a backup in backend storage.
type StorageProvider struct {
	S3  *S3StorageProvider  `json:"s3,omitempty"`
	Gcs *GcsStorageProvider `json:"gcs,omitempty"`
}

// +k8s:openapi-gen=true
// `S3StorageProvider` represents a S3 compliant storage for storing backups.
type S3StorageProvider struct {
	// Represents the specific storage provider that implements the S3 interface.
	Provider S3StorageProviderType `json:"provider"`
	// Region in which the S3 compatible bucket is located.
	Region string `json:"region,omitempty"`
	// The full path where the backup is saved<br>
	// The format of the path must be: "&lt;bucket-name&gt;/&lt;path-to-backup-file&gt;"
	Path string `json:"path,omitempty"`
	// Bucket in which the backup data is stored
	Bucket string `json:"bucket,omitempty"`
	// Endpoint of S3 compatible storage service
	Endpoint string `json:"endpoint,omitempty"`
	// Represents the storage class.
	StorageClass string `json:"storageClass,omitempty"`
	// Represents access control permissions for this bucket.
	Acl string `json:"acl,omitempty"`
	// SecretName is the name of secret which stores
	// S3 compliant storage access key and secret key.
	SecretName string `json:"secretName,omitempty"`
	// Prefix for the keys.
	Prefix string `json:"prefix,omitempty"`
	// SSE Sever-Side Encryption.
	SSE string `json:"sse,omitempty"`
}

// +k8s:openapi-gen=true
// `GcsStorageProvider` represents the Google Cloud storage for storing backups.
type GcsStorageProvider struct {
	// Represents the project that organizes all your Google Cloud Platform resources.
	ProjectId string `json:"projectId"`
	// Location in which the gcs bucket is located.
	Location string `json:"location,omitempty"`
	// Path is the full path where the backup is saved.<br>
	// The format of the path must be: "&lt;bucket-name&gt;/&lt;path-to-backup-file&gt;"
	Path string `json:"path,omitempty"`
	// Bucket in which to store the backup data.
	Bucket string `json:"bucket,omitempty"`
	// Represents the storage class.
	StorageClass string `json:"storageClass,omitempty"`
	// Represents the access control list for new objects.
	ObjectAcl string `json:"objectAcl,omitempty"`
	// Represents the access control list for new buckets.
	BucketAcl string `json:"bucketAcl,omitempty"`
	// The name of secret which stores the
	// gcs service account credentials JSON.
	SecretName string `json:"secretName"`
}

// +k8s:openapi-gen=true
// `BackupType` represents the backup type.
type BackupType string

const (
	// Represents the full backup of the TiDB cluster.
	BackupTypeFull BackupType = "full"
	// Represents the incremental backup of the TiDB cluster.
	BackupTypeInc BackupType = "incremental"
	// Represents the backup of one DB for the TiDB cluster.
	BackupTypeDB BackupType = "db"
	// Represents the backup of one table for the TiDB cluster.
	BackupTypeTable BackupType = "table"
)

// +k8s:openapi-gen=true
// `TiDBAccessConfig` defines the configuration for accessing the TiDB cluster.
type TiDBAccessConfig struct {
	// The TiDB cluster access address.
	Host string `json:"host"`
	// The port number to use for connecting the TiDB cluster.
	Port int32 `json:"port,omitempty"`
	// The user for login the TiDB cluster.
	User string `json:"user,omitempty"`
	// The name of secret which stores the TiDB cluster's password.
	SecretName string `json:"secretName"`
	// Determines whether to enable the TLS connection between the SQL client and TiDB server.<br>
	// Optional: Defaults to `nil`
	// +optional
	TLSClient *TiDBTLSClient `json:"tlsClient,omitempty"`
}

// +k8s:openapi-gen=true
// `BackupSpec` contains the backup specification for a TiDB cluster.
type BackupSpec struct {
	// `From` is the TiDB cluster that needs to backup.
	From TiDBAccessConfig `json:"from,omitempty"`
	// `Type` is the backup type for TiDB cluster.
	Type BackupType `json:"backupType,omitempty"`
	// `TikvGCLifeTime` is to specify the safe GC life time for backup.<br>
	// The time limit during which data is retained for each GC, in the format of Go Duration.<br>
	// When a GC happens, the current time minus this value is the safe point.
	TikvGCLifeTime *string `json:"tikvGCLifeTime,omitempty"`
	// `StorageProvider` configures where and how backups should be stored.
	StorageProvider `json:",inline"`
	// The storageClassName of the persistent volume for Backup data storage.<br>
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// `StorageSize` is the request storage size for backup job.
	StorageSize string `json:"storageSize,omitempty"`
	// `BRConfig` is the configs for BR.
	BR *BRConfig `json:"br,omitempty"`
	// Base tolerations of backup Pods. Components may add more tolerations upon this respectively.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Affinity of backup Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Use KMS to decrypt the secrets.
	UseKMS bool `json:"useKMS,omitempty"`
	// Specifies service account of backup.
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// +k8s:openapi-gen=true
// `BRConfig` contains configuration for BR.
type BRConfig struct {
	// `ClusterName` of backup/restore cluster
	Cluster string `json:"cluster"`
	// `Namespace` of backup/restore cluster
	ClusterNamespace string `json:"clusterNamespace,omitempty"`
	// `DB` is the specific database which will be backed-up or restored.
	DB string `json:"db,omitempty"`
	// `Table` is the specific table which will be backed-up or restored.
	Table string `json:"table,omitempty"`
	// `LogLevel` is the log level.
	LogLevel string `json:"logLevel,omitempty"`
	// `StatusAddr` is the HTTP listening address for the status report service.<br>To disable the service, set it to empty string.
	StatusAddr string `json:"statusAddr,omitempty"`
	// `Concurrency` is the size of thread pool on each node that execute the backup task.
	Concurrency *uint32 `json:"concurrency,omitempty"`
	// `RateLimit` is the rate limit of the backup task, MB/s per node.
	RateLimit *uint `json:"rateLimit,omitempty"`
	// `TimeAgo` is the history version of the backup task, e.g. 1m, 1h.
	TimeAgo string `json:"timeAgo,omitempty"`
	// `Checksum` specifies whether to run checksum after backup.
	Checksum *bool `json:"checksum,omitempty"`
	// `SendCredToTikv` specifies whether to send credentials to TiKV.
	SendCredToTikv *bool `json:"sendCredToTikv,omitempty"`
	// `OnLine` specifies whether online during restore.
	OnLine *bool `json:"onLine,omitempty"`
}

// `BackupConditionType` represents a valid condition of a Backup.
type BackupConditionType string

const (
	// `BackupScheduled` means the backup related job has been created.
	BackupScheduled BackupConditionType = "Scheduled"
	// `BackupRunning` means the backup is currently being executed.
	BackupRunning BackupConditionType = "Running"
	// `BackupComplete` means the backup has successfully executed and the
	// resulting artifact has been stored in backend storage.
	BackupComplete BackupConditionType = "Complete"
	// `BackupClean` means the clean job has been created to clean backup data.
	BackupClean BackupConditionType = "Clean"
	// `BackupFailed` means the backup has failed.
	BackupFailed BackupConditionType = "Failed"
	// `BackupRetryFailed` means this failure can be retried.
	BackupRetryFailed BackupConditionType = "RetryFailed"
	// `BackupInvalid` means invalid backup CR.
	BackupInvalid BackupConditionType = "Invalid"
)

// `BackupCondition` describes the observed state of a Backup at a certain point.
type BackupCondition struct {
	Type               BackupConditionType    `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
}

// `BackupStatus` represents the current status of a backup.
type BackupStatus struct {
	// `BackupPath` is the location of the backup.
	BackupPath string `json:"backupPath"`
	// `TimeStarted` is the time at which the backup was started.
	TimeStarted metav1.Time `json:"timeStarted"`
	// `TimeCompleted` is the time at which the backup was completed.
	TimeCompleted metav1.Time `json:"timeCompleted"`
	// `BackupSize` is the data size of the backup.
	BackupSize int64 `json:"backupSize"`
	// `CommitTs` is the snapshot time point of the TiDB cluster.
	CommitTs   string            `json:"commitTs"`
	Conditions []BackupCondition `json:"conditions"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// `BackupSchedule` is a backup schedule of the TiDB cluster.
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
// `BackupScheduleList` contains a list of BackupSchedule.
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BackupSchedule `json:"items"`
}

// +k8s:openapi-gen=true
// `BackupScheduleSpec` contains the backup schedule specification for a TiDB cluster.
type BackupScheduleSpec struct {
	// `Schedule` specifies the cron string used for backup scheduling.
	Schedule string `json:"schedule"`
	// `Pause` means paused backupSchedule.
	Pause bool `json:"pause,omitempty"`
	// `MaxBackups` is to specify how many backups we want to keep.<br>
	// 0 is a magic number to indicate un-limited backups.
	MaxBackups *int32 `json:"maxBackups,omitempty"`
	// `MaxReservedTime` is to specify how long we want to keep the backups.
	MaxReservedTime *string `json:"maxReservedTime,omitempty"`
	// `BackupTemplate` is the specification of the backup structure to get scheduled.
	BackupTemplate BackupSpec `json:"backupTemplate"`
	// The `storageClassName` of the persistent volume for Backup data storage if not storage class name set in BackupSpec.<br>
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// `StorageSize` is the request storage size for the backup job.
	StorageSize string `json:"storageSize,omitempty"`
}

// `BackupScheduleStatus` represents the current state of a BackupSchedule.
type BackupScheduleStatus struct {
	// `LastBackup` represents the last backup.
	LastBackup string `json:"lastBackup"`
	// `LastBackupTime` represents the last time the backup was successfully created.
	LastBackupTime *metav1.Time `json:"lastBackupTime"`
	// `AllBackupCleanTime` represents the time when all backup entries are cleaned up.
	AllBackupCleanTime *metav1.Time `json:"allBackupCleanTime"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// `Restore` represents the restoration of backup of a TiDB cluster.
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
// `RestoreList` contains a list of Restore.
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []Restore `json:"items"`
}

// `RestoreConditionType` represents a valid condition of a Restore.
type RestoreConditionType string

const (
	// `RestoreScheduled` means the restore job has been created to restore the TiDB cluster.
	RestoreScheduled RestoreConditionType = "Scheduled"
	// `RestoreRunning` means the restore is currently being executed.
	RestoreRunning RestoreConditionType = "Running"
	// `RestoreComplete` means the Restore has successfully executed, and the
	// backup data has been loaded into the TiDB cluster.
	RestoreComplete RestoreConditionType = "Complete"
	// `RestoreFailed` means the Restore has failed.
	RestoreFailed RestoreConditionType = "Failed"
	// `RestoreRetryFailed` means this failure can be retried.
	RestoreRetryFailed RestoreConditionType = "RetryFailed"
	// `RestoreInvalid` means invalid restore CR.
	RestoreInvalid RestoreConditionType = "Invalid"
)

// `RestoreCondition` describes the observed state of a Restore at a certain point.
type RestoreCondition struct {
	Type               RestoreConditionType   `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
}

// +k8s:openapi-gen=true
// `RestoreSpec` contains the specification for a restore of a TiDB cluster backup.
type RestoreSpec struct {
	// `To` is the TiDB cluster that needs to restore.
	To TiDBAccessConfig `json:"to,omitempty"`
	// `Type` is the backup type for the TiDB cluster.
	Type BackupType `json:"backupType,omitempty"`
	// `TikvGCLifeTime` specifies the safe GC life time for restore.<br>
	// The time limit during which data is retained for each GC, in the format of Go Duration.<br>
	// When a GC happens, the current time minus this value is the safe point.
	TikvGCLifeTime *string `json:"tikvGCLifeTime,omitempty"`
	// `StorageProvider` configures where and how backups should be stored.
	StorageProvider `json:",inline"`
	// The `storageClassName` of the persistent volume for Restore data storage.<br>
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// `StorageSize` is the request storage size for backup job.
	StorageSize string `json:"storageSize,omitempty"`
	// `BR` is the configs for BR.
	BR *BRConfig `json:"br,omitempty"`
	// Base tolerations of restore Pods. Components may add more tolerations upon this respectively.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Affinity of restore Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Use KMS to decrypt the secrets.
	UseKMS bool `json:"useKMS,omitempty"`
	// Specifies the service account of restore.
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// `RestoreStatus` represents the current status of a TiDB cluster restore.
type RestoreStatus struct {
	// `TimeStarted` is the time at which the restore was started.
	TimeStarted metav1.Time `json:"timeStarted"`
	// `TimeCompleted` is the time at which the restore was completed.
	TimeCompleted metav1.Time        `json:"timeCompleted"`
	Conditions    []RestoreCondition `json:"conditions"`
}
