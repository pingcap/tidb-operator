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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CondSuspended is a condition to display whether the group or instance is suspended
	CondSuspended     = "Suspended"
	ReasonSuspended   = "Suspended"
	ReasonSuspending  = "Suspending"
	ReasonUnsuspended = "Unsuspended"
)

const (
	// Ready means all managed resources are ready.
	// NOTE: It does not mean all managed resources are up to date.
	//
	// condition
	CondReady = "Ready"
	// reason for both
	ReasonReady   = "Ready"
	ReasonUnready = "Unready"
	// reason for group
	ReasonNotAllInstancesReady = "NotAllInstancesReady"
	// reason for instance
	ReasonPodNotCreated      = "PodNotCreated"
	ReasonPodNotReady        = "PodNotReady"
	ReasonPodTerminating     = "PodTerminating"
	ReasonInstanceNotHealthy = "InstanceNotHealthy"

	// Synced means all specs of managed resources are up to date and
	// nothing need to do in controller but only status updation.
	CondSynced = "Synced"
	// reason for both
	ReasonSynced   = "Synced"
	ReasonUnsynced = "Unsynced"
	// reason for group
	ReasonNotAllInstancesUpToDate = "NotAllInstancesUpToDate"
	// reason for instance
	ReasonPodNotUpToDate = "PodNotUpToDate"
	ReasonPodNotDeleted  = "PodNotDeleted"
)

// TODO(liubo02): move to meta
const (
	// KeyPrefix defines key prefix of well known labels and annotations
	KeyPrefix = "pingcap.com/"

	// LabelKeyManagedBy means resources are managed by tidb operator
	LabelKeyManagedBy         = KeyPrefix + "managed-by"
	LabelValManagedByOperator = "tidb-operator"

	// LabelKeyCluster means which tidb cluster the resource belongs to
	LabelKeyCluster = KeyPrefix + "cluster"
	// LabelKeyComponent means the component of the resource
	LabelKeyComponent = KeyPrefix + "component"
	// LabelKeyGroup means the component group of the resource
	LabelKeyGroup = KeyPrefix + "group"
	// LabelKeyInstance means the instance of the resource
	LabelKeyInstance = KeyPrefix + "instance"

	// LabelKeyPodSpecHash is the hash of the pod spec.
	LabelKeyPodSpecHash = KeyPrefix + "pod-spec-hash"

	// LabelKeyInstanceRevisionHash is the revision hash of the instance
	LabelKeyInstanceRevisionHash = KeyPrefix + "instance-revision-hash"

	// LabelKeyConfigHash is the hash of the user-specified config (i.e., `.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// Since the tidb operator will overlay the user-specified config with some operator-managed fields,
	// if we hash the overlayed config, with the evolving TiDB Operator, the hash may change,
	// potentially triggering an unexpected rolling update.
	// Instead, we choose to hash the user-specified config,
	// and the worst case is that users expect a reboot but it doesn't happen.
	LabelKeyConfigHash = KeyPrefix + "config-hash"

	// LabelKeyVolumeName is used to distinguish different volumes, e.g. data volumes, log volumes, etc.
	// This label will be added to the PVCs created by the tidb operator.
	LabelKeyVolumeName = KeyPrefix + "volume-name"
)

const (
	// Label value for meta.LabelKeyComponent
	LabelValComponentPD        = "pd"
	LabelValComponentTiDB      = "tidb"
	LabelValComponentTiKV      = "tikv"
	LabelValComponentTiFlash   = "tiflash"
	LabelValComponentTiCDC     = "ticdc"
	LabelValComponentTSO       = "tso"
	LabelValComponentScheduler = "scheduler"
	LabelValComponentTiProxy   = "tiproxy"

	// LabelKeyClusterID is the unique identifier of the cluster.
	// This label is used for backward compatibility with TiDB Operator v1, so it has a different prefix.
	LabelKeyClusterID = "tidb.pingcap.com/cluster-id"
	// LabelKeyMemberID is the unique identifier of a PD member.
	// This label is used for backward compatibility with TiDB Operator v1, so it has a different prefix.
	LabelKeyMemberID = "tidb.pingcap.com/member-id"
	// LabelKeyStoreID is the unique identifier of a TiKV or TiFlash store.
	// This label is used for backward compatibility with TiDB Operator v1, so it has a different prefix.
	LabelKeyStoreID = "tidb.pingcap.com/store-id"
)

const (
	// AnnoKeyPrefix defines key prefix of well known annotations
	AnnoKeyPrefix = "core.pingcap.com/"

	// all bool anno will use this val as default
	AnnoValTrue = "true"

	// means the instance is marked as deleted and will be deleted later
	AnnoKeyDeferDelete = AnnoKeyPrefix + "defer-delete"

	// Last instance template is recorded to check whether the pod should be restarted
	AnnoKeyLastInstanceTemplate = AnnoKeyPrefix + "last-instance-template"
)

// ConfigUpdateStrategy represents the strategy to update configuration.
type ConfigUpdateStrategy string

const (
	// ConfigUpdateStrategyHotReload updates config without restarting.
	ConfigUpdateStrategyHotReload ConfigUpdateStrategy = "HotReload"

	// ConfigUpdateStrategyRestart performs a restart to apply changed configs.
	ConfigUpdateStrategyRestart ConfigUpdateStrategy = "Restart"
)

// ObjectMeta is defined for replacing the embedded metav1.ObjectMeta
// Now only labels and annotations are allowed
type ObjectMeta struct {
	// Name must be unique within a namespace. Is required when creating resources, although
	// some resources may allow a client to request the generation of an appropriate name
	// automatically. Name is primarily intended for creation idempotence and configuration
	// definition.
	// Cannot be updated.
	// More info: http://kubernetes.io/docs/user-guide/identifiers#names
	// +optional
	Name string `json:"name,omitempty"`
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ClusterReference is a reference to cluster
// NOTE: namespace may be added into the reference in the future
type ClusterReference struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="cluster name is immutable"
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`
}

// Topology means the topo for scheduling
// e.g. topology.kubernetes.io/zone: us-west-1a
// It will be translated to pod.spec.nodeSelector
// IMPORTANT: Topology is immutable for an instance
// +kubebuilder:validation:MinProperties=1
type Topology map[string]string

// Overlay defines some templates of k8s native resources.
// Users can specify this field to overlay the spec of managed resources(pod, pvcs, ...).
type Overlay struct {
	Pod *PodOverlay `json:"pod,omitempty"`
	// +listType=map
	// +listMapKey=name
	PersistentVolumeClaims []NamedPersistentVolumeClaimOverlay `json:"volumeClaims,omitempty"`
}

type PodOverlay struct {
	// +kubebuilder:validation:XValidation:rule="!has(self.labels) || self.labels.all(key, !key.startsWith('pingcap.com/'))",message="cannot overlay pod labels starting with 'pingcap.com/'"
	ObjectMeta `json:"metadata,omitempty"`
	Spec       *corev1.PodSpec `json:"spec,omitempty"`
}

type NamedPersistentVolumeClaimOverlay struct {
	Name                  string                       `json:"name"`
	PersistentVolumeClaim PersistentVolumeClaimOverlay `json:"volumeClaim"`
}

type PersistentVolumeClaimOverlay struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       *corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

type ConfigFile string

// Volume defines a persistent volume, it will be mounted at a specified root path
// A volume can be mounted for multiple different usages.
// For example, a volume can be mounted for both data and raft log.
type Volume struct {
	// Name is volume name.
	Name string `json:"name"`

	// Mounts defines mount infos of this volume
	// NOTE(liubo02): it cannot be a list map because the key is "type" or "mountPath" which is not supported
	// +listType=atomic
	Mounts []VolumeMount `json:"mounts"`

	// Storage defines the request size of this volume
	Storage resource.Quantity `json:"storage"`

	// StorageClassName means the storage class the volume used.
	// You can modify volumes' attributes by changing the StorageClass
	// when VolumeAttributesClass is not available.
	// Note that only newly created PV will use the new StorageClass.
	StorageClassName *string `json:"storageClassName,omitempty"`

	// VolumeAttributesClassName means the VolumeAttributesClass the volume used.
	// You can modify volumes' attributes by changing it.
	// This feature is introduced since K8s 1.29 as alpha feature and disabled by default.
	// It's only available when the feature is enabled.
	VolumeAttributesClassName *string `json:"volumeAttributesClassName,omitempty"`
}

type VolumeMount struct {
	// Type is a type of the volume mount.
	Type VolumeMountType `json:"type"`
	// Mount path of volume, if it's not set, use the default path of this type.
	// TODO: webhook for empty path if it's not a built-in type.
	MountPath string `json:"mountPath,omitempty"`
	// SubPath is the path of the volume's root path.
	SubPath string `json:"subPath,omitempty"`
}

type VolumeMountType string

// Port defines a listen port
type Port struct {
	Port int32 `json:"port"`
}

// SchedulePolicy defines how instances of the group schedules its pod.
type SchedulePolicy struct {
	Type         SchedulePolicyType          `json:"type"`
	EvenlySpread *SchedulePolicyEvenlySpread `json:"evenlySpread,omitempty"`
}

type SchedulePolicyType string

const (
	// This policy is defined to evenly spread all instances of a group
	// e.g. we may hope tikvs can evenly spread in 3 az
	SchedulePolicyTypeEvenlySpread = "EvenlySpread"
)

type SchedulePolicyEvenlySpread struct {
	// All instances of a group will evenly spread in different topologies
	Topologies []ScheduleTopology `json:"topologies"`
}

type ScheduleTopology struct {
	// Topology means the topo for scheduling
	Topology Topology `json:"topology"`
	// Weight defines how many pods will be scheduled to this topo
	// default is 1
	Weight *int32 `json:"weight,omitempty"`
}

// ResourceRequirements describes the compute resource requirements.
// It's simplified from corev1.ResourceRequirements to fit the most common use cases.
// This field will be translated to requests=limits for all resources.
// If users need to specify more advanced resource requirements, just try to use overlay to override it
type ResourceRequirements struct {
	CPU    *resource.Quantity `json:"cpu,omitempty"`
	Memory *resource.Quantity `json:"memory,omitempty"`
}

// CommonStatus defines common status fields for instances and groups managed by TiDB Operator.
type CommonStatus struct {
	// Conditions contain details of the current state.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// It's used to determine whether the controller has reconciled the latest spec.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// CurrentRevision is the revision of the Controller that created the resource.
	CurrentRevision string `json:"currentRevision,omitempty"`

	// UpdateRevision is the revision of the Controller that should modify the resource.
	UpdateRevision string `json:"updateRevision,omitempty"`

	// CollisionCount is the count of hash collisions. The controller
	// uses this field as a collision avoidance mechanism when it needs to create the name for the
	// newest ControllerRevision.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

// StoreStatus defines the common status fields for all stores.
type StoreStatus struct {
	// ID is the store id.
	ID string `json:"id,omitempty"`

	// State is the store state.
	State string `json:"state,omitempty"`
}

// GroupStatus defines the common status fields for all component groups.
type GroupStatus struct {
	// Version is the version of all instances in the group.
	// It will be same as the `spec.version` only when all instances are upgraded to the desired version.
	Version string `json:"version,omitempty"`

	// Selector is the label selector for scale subresource.
	Selector string `json:"selector"`

	// Replicas is the number of Instance created by the controller.
	Replicas int32 `json:"replicas"`

	// ReadyReplicas is the number of Instances created for this ComponentGroup with a Ready Condition.
	ReadyReplicas int32 `json:"readyReplicas"`

	// CurrentReplicas is the number of Instances created by the Group controller from the Group version
	// indicated by currentRevision.
	CurrentReplicas int32 `json:"currentReplicas"`

	// UpdatedReplicas is the number of Instances created by the Group controller from the Group version
	// indicated by updateRevision.
	UpdatedReplicas int32 `json:"updatedReplicas"`
}

type UpdateStrategy struct {
	// Config determines how the configuration change is applied to the cluster.
	// Valid values are "Restart" (by default) and "HotReload".
	// +kubebuilder:validation:Enum=Restart;HotReload
	// +kubebuilder:default="Restart"
	Config ConfigUpdateStrategy `json:"config,omitempty"`
}

// TLS defines a common tls config for all components
// Now it only support enable or disable.
// TODO(liubo02): add more tls configs
type TLS struct {
	Enabled bool `json:"enabled,omitempty"`

	// SkipCA is used to skip the CA verification for the TLS connection.
	SkipCA bool `json:"skipCA,omitempty"`
}
