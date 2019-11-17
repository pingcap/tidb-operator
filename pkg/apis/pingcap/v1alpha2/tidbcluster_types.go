// Copyright 2019. PingCAP, Inc.
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

package v1alpha2

import (
	"github.com/pingcap/tidb-operator/pkg/util/json"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TiKVState is the current state of a TiKV store
type TiKVState string

const (
	// TiKVStateUp represents status of Up of TiKV
	TiKVStateUp TiKVState = "Up"
	// TiKVStateDown represents status of Down of TiKV
	TiKVStateDown TiKVState = "Down"
	// TiKVStateOffline represents status of Offline of TiKV
	TiKVStateOffline TiKVState = "Offline"
	// TiKVStateTombstone represents status of Tombstone of TiKV
	TiKVStateTombstone TiKVState = "Tombstone"
)

// MemberPhase is the current state of member
type MemberPhase string

const (
	// NormalPhase represents normal state of TiDB cluster.
	NormalPhase MemberPhase = "Normal"
	// UpgradePhase represents the upgrade state of TiDB cluster.
	UpgradePhase MemberPhase = "Upgrade"
)

// PVCDeletePolicy defines how to handle orphan PVC left by scale-in of statefulset
type PVCDeletePolicy string

const (
	// DeletePolicyDefer retains the PVC during scale-in, and delete the PVC when
	// a new Pod with the same ordinal is created during scale-out
	DeletePolicyDefer PVCDeletePolicy = "Deferred"
	// DeletePolicyImmediate delete the PVC once the bounding Pod is deleted successfully
	DeletePolicyImmediate PVCDeletePolicy = "Immediate"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbCluster
// +k8s:openapi-gen=true
// +genregister:unversioned=false
// +resource:path=tidbclusters
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

// +genregister:unversioned=false
// TidbClusterSpec describes the attributes that a user creates on a tidb cluster
type TidbClusterSpec struct {
	// Version of TiDB cluster
	Version string `json:"version,omitempty"`

	// PD cluster spec
	PD PDSpec `json:"pd,omitempty"`

	// TiKV cluster spec
	TiKV TiKVSpec `json:"tikv,omitempty"`

	// TiDB cluster spec
	TiDB TiDBSpec `json:"tidb,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster
	PVReclaimPolicy corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// Delete behavior policy of the orphaned PVC created by the statefulset controller
	PVCDeletePolicy PVCDeletePolicy `json:"pvcDeletePolicy,omitempty"`

	// Whether TLS connection between TiDB server components is enabled
	TLSClusterEnabled bool `json:"tLSClusterEnabled,omitempty"`

	// Scheduler name of TiDB cluster Pods
	SchedulerName string `json:"schedulerName,omitempty"`

	// Time zone of TiDB cluster Pods
	Timezone string `json:"timezone,omitempty"`

	// ImagePullPolicy of TiDB cluster Pods
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Whether Hostnetwork is enabled for TiDB cluster Pods
	HostNetwork *bool `json:"hostNetwork,omitempty"`

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

// +genregister:unversioned=false
// PDSpec contains details of PD members
type PDSpec struct {
	ComponentBaseSpec
	Storage StorageSpec `json:"storage,omitempty"`

	// Service type of the non-headless service that proxies Pod pods, nil means no service
	ServiceType *corev1.ServiceType `json:"serviceType,omitempty"`

	// Configuration of PD.
	// TODO: use determined schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +genregister:unversioned=false
// TiKVSpec contains details of TiKV members
type TiKVSpec struct {
	ComponentBaseSpec
	Privileged       bool        `json:"privileged,omitempty"`
	MaxFailoverCount int32       `json:"maxFailoverCount,omitempty"`
	Storage          StorageSpec `json:"storage,omitempty"`

	// Configuration of TiKV.
	// TODO: use determined schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +genregister:unversioned=false
// TiDBSpec contains details of TiDB members
type TiDBSpec struct {
	ComponentBaseSpec
	BinlogEnabled    bool  `json:"binlogEnabled,omitempty"`
	MaxFailoverCount int32 `json:"maxFailoverCount,omitempty"`

	// Whether the output stream of TiDB slow log is separated to STDOUT of a sidecar container
	SeparateSlowLog bool `json:"separateSlowLog,omitempty"`

	// Whether the TLS connection between clients and TiDB servers is enabled
	TLSClientEnabled bool `json:"tlsClientEnabled,omitempty"`

	// Service type of the non-headless service that proxies TiDB pods, nil means no service
	ServiceType *corev1.ServiceType `json:"serviceType,omitempty"`

	// Plugins is a list of plugins that are loaded by TiDB server, empty means plugin disabled
	Plugins []string `json:"plugins,omitempty"`

	// Configuration of TiDB.
	// TODO: use determined schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +genregister:unversioned=false
// PumpSpec contains details of Pump members
type PumpSpec struct {
	ComponentBaseSpec
	Storage StorageSpec `json:"storage,omitempty"`

	// Configuration of Pump.
	// TODO: use determined schema
	Config map[string]json.JsonObject `json:"config,omitempty"`
}

// +genregister:unversioned=false
// ComponentBaseSpec is the base spec of each component
type ComponentBaseSpec struct {
	// Replicas of the component, 0 (if allowed) means do not create this component
	Replicas int32 `json:"replicas"`

	// Base image of the component, e.g. pingcap/tidb, image tag is now allowed during validation
	BaseImage string `json:"baseImage,omitempty"`

	// Version of the component. Override the cluster-level version if non-empty
	Version string `json:"version,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Whether Hostnetwork of the component is enabled. Override the cluster-level setting if present
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of the component. Override the cluster-level affinity if present
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of the component. Override the cluster-level affinity if present
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// NodeSelector of the component. Merged into the cluster-level nodeSelector if non-empty
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Annotations of the component. Merged into the cluster-level annotations if non-empty
	Annotations map[string]string `json:"annotations,omitempty"`

	// Tolerations of the component. Merged into the cluster-level tolerations if non-empty
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Resource requests of the component
	Requests *ResourceRequirement `json:"requests,omitempty"`

	// Resource limits of the component
	Limits *ResourceRequirement `json:"limits,omitempty"`

	// PodSecurityContext of the component
	// TODO: make this configurable at cluster level
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// +genregister:unversioned=false
// StorageSpec is the base spec of component's storage
type StorageSpec struct {
	ClassName string `json:"className,omitempty"`
	Size      string `json:"className,omitempty"`
}

// +genregister:unversioned=false
// ResourceRequirement is resource requirements for a pod
type ResourceRequirement struct {
	// CPU is how many cores a pod requires
	CPU string `json:"cpu,omitempty"`
	// Memory is how much memory a pod requires
	Memory string `json:"memory,omitempty"`
}

// +genregister:unversioned=false
// TidbClusterStatus represents the current status of a tidb cluster.
type TidbClusterStatus struct {
	ClusterID string     `json:"clusterID,omitempty"`
	PD        PDStatus   `json:"pd,omitempty"`
	TiKV      TiKVStatus `json:"tikv,omitempty"`
	TiDB      TiDBStatus `json:"tidb,omitempty"`
}

// +genregister:unversioned=false
// PDStatus is PD status
type PDStatus struct {
	Synced         bool                       `json:"synced,omitempty"`
	Phase          MemberPhase                `json:"phase,omitempty"`
	StatefulSet    *apps.StatefulSetStatus    `json:"statefulSet,omitempty"`
	Members        map[string]PDMember        `json:"members,omitempty"`
	Leader         PDMember                   `json:"leader,omitempty"`
	FailureMembers map[string]PDFailureMember `json:"failureMembers,omitempty"`
}

// +genregister:unversioned=false
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

// +genregister:unversioned=false
// PDFailureMember is the pd failure member information
type PDFailureMember struct {
	PodName       string      `json:"podName,omitempty"`
	MemberID      string      `json:"memberID,omitempty"`
	PVCUID        types.UID   `json:"pvcUID,omitempty"`
	MemberDeleted bool        `json:"memberDeleted,omitempty"`
	CreatedAt     metav1.Time `json:"createdAt,omitempty"`
}

// +genregister:unversioned=false
// TiDBStatus is TiDB status
type TiDBStatus struct {
	Phase                    MemberPhase                  `json:"phase,omitempty"`
	StatefulSet              *apps.StatefulSetStatus      `json:"statefulSet,omitempty"`
	Members                  map[string]TiDBMember        `json:"members,omitempty"`
	FailureMembers           map[string]TiDBFailureMember `json:"failureMembers,omitempty"`
	ResignDDLOwnerRetryCount int32                        `json:"resignDDLOwnerRetryCount,omitempty"`
}

// +genregister:unversioned=false
// TiDBMember is TiDB member
type TiDBMember struct {
	Name   string `json:"name"`
	Health bool   `json:"health"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Node hosting pod of this TiDB member.
	NodeName string `json:"node,omitempty"`
}

// +genregister:unversioned=false
// TiDBFailureMember is the tidb failure member information
type TiDBFailureMember struct {
	PodName   string      `json:"podName,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// +genregister:unversioned=false
// TiKVStatus is TiKV status
type TiKVStatus struct {
	Synced          bool                        `json:"synced,omitempty"`
	Phase           MemberPhase                 `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore        `json:"stores,omitempty"`
	TombstoneStores map[string]TiKVStore        `json:"tombstoneStores,omitempty"`
	FailureStores   map[string]TiKVFailureStore `json:"failureStores,omitempty"`
}

// +genregister:unversioned=false
// TiKVStores is either Up/Down/Offline/Tombstone
type TiKVStore struct {
	// store id is also uint64, due to the same reason as pd id, we store id as string
	ID                string      `json:"id"`
	PodName           string      `json:"podName"`
	IP                string      `json:"ip"`
	LeaderCount       int32       `json:"leaderCount"`
	State             TiKVState   `json:"state"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// +genregister:unversioned=false
// TiKVFailureStore is the tikv failure store information
type TiKVFailureStore struct {
	PodName   string      `json:"podName,omitempty"`
	StoreID   string      `json:"storeID,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}
