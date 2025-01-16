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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// CondHealth is a condition to display whether the instance is health
	CondHealth = "Health"

	// CondSuspended is a condition to display whether the group or instance is suspended
	CondSuspended     = "Suspended"
	ReasonSuspended   = "Suspended"
	ReasonSuspending  = "Suspending"
	ReasonUnsuspended = "Unsuspended"
)

const (
	// Finalizer is the finalizer used by all resources managed by TiDB Operator.
	Finalizer = "core.pingcap.com/finalizer"

	// LabelKeyPrefix defines key prefix of well known labels
	LabelKeyPrefix = "pingcap.com/"

	// LabelKeyManagedBy means resources are managed by tidb operator
	LabelKeyManagedBy         = LabelKeyPrefix + "managed-by"
	LabelValManagedByOperator = "tidb-operator"

	// LabelKeyCluster means which tidb cluster the resource belongs to
	LabelKeyCluster = LabelKeyPrefix + "cluster"

	// LabelKeyComponent means the component of the resource
	LabelKeyComponent        = LabelKeyPrefix + "component"
	LabelValComponentPD      = "pd"
	LabelValComponentTiDB    = "tidb"
	LabelValComponentTiKV    = "tikv"
	LabelValComponentTiFlash = "tiflash"

	// LabelKeyGroup means the component group of the resource
	LabelKeyGroup = LabelKeyPrefix + "group"
	// LabelKeyInstance means the instance of the resource
	LabelKeyInstance = LabelKeyPrefix + "instance"

	// LabelKeyPodSpecHash is the hash of the pod spec.
	LabelKeyPodSpecHash = LabelKeyPrefix + "pod-spec-hash"

	LabelKeyInstanceRevisionHash = LabelKeyPrefix + "instance-revision-hash"

	// LabelKeyConfigHash is the hash of the user-specified config (i.e., `.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// Since the tidb operator will overlay the user-specified config with some operator-managed fields,
	// if we hash the overlayed config, with the evolving TiDB Operator, the hash may change,
	// potentially triggering an unexpected rolling update.
	// Instead, we choose to hash the user-specified config,
	// and the worst case is that users expect a reboot but it doesn't happen.
	LabelKeyConfigHash = LabelKeyPrefix + "config-hash"
)

const (
	// NamePrefix for "names" in k8s resources
	// Users may overlay some fields in managed resource such as pods. Names with this
	// prefix is preserved to avoid conflicts with fields defined by users.
	NamePrefix = "ti-"

	// VolNamePrefix is defined for custom persistent volume which may have name conflicts
	// with the volumes managed by tidb operator
	VolNamePrefix = NamePrefix + "vol-"
)

const (
	// All volume names
	//
	// VolumeNameConfig defines volume name for main config file
	VolumeNameConfig = NamePrefix + "config"
	// VolumeNamePrestopChecker defines volume name for pre stop checker cmd
	VolumeNamePrestopChecker = NamePrefix + "prestop-checker"

	// All container names
	//
	// Main component containers of the tidb cluster
	ContainerNamePD      = "pd"
	ContainerNameTiKV    = "tikv"
	ContainerNameTiDB    = "tidb"
	ContainerNameTiFlash = "tiflash"
	// An init container to copy pre stop checker cmd to main container
	ContainerNamePrestopChecker = NamePrefix + "prestop-checker"
)

const (
	DirNameConfigPD      = "/etc/pd"
	DirNameConfigTiKV    = "/etc/tikv"
	DirNameConfigTiDB    = "/etc/tidb"
	DirNameConfigTiFlash = "/etc/tiflash"

	// ConfigFileName defines default name of config file
	ConfigFileName = "config.toml"

	// ConfigFileTiFlashProxyName defines default name of tiflash proxy config file
	ConfigFileTiFlashProxyName = "proxy.toml"

	// PrestopDirName defines dir path of pre stop checker cmd
	DirNamePrestop = "/prestop"
)

const (
	DefaultHelperImage = "busybox:1.37.0"
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

type ClusterReference struct {
	Name string `json:"name"`
}

// Topology means the topo for scheduling
// e.g. topology.kubernetes.io/zone: us-west-1a
// It will be translated to pod.spec.nodeSelector
// IMPORTANT: Topology is immutable for an instance
type Topology map[string]string

// Overlay defines some templates of k8s native resources.
// Users can specify this field to overlay the spec of managed resources(pod, pvcs, ...).
type Overlay struct {
	Pod                    *PodOverlay                    `json:"pod,omitempty"`
	PersistentVolumeClaims []PersistentVolumeClaimOverlay `json:"volumeClaims,omitempty"`
}

type PodOverlay struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       *corev1.PodSpec `json:"spec,omitempty"`
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
	// If not specified, the PVC name will be "{component}-{podName}"
	Name string `json:"name,omitempty"`

	// Mounts defines mount infos of this volume
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
	// Mount path of volume, if it's not set, use the default path of this type
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
	// All instances of a group will evenly spread in differnet topologies
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

// GroupStatus defines the common status fields for all component groups.
type GroupStatus struct {
	// Version is the version of all instances in the group.
	// It will be same as the `spec.version` only when all instances are upgraded to the desired version.
	Version string `json:"version,omitempty"`

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
}

// ComponentAccessor is the interface to access details of instances/groups managed by TiDB Operator.
type ComponentAccessor interface {
	GetName() string
	GetNamespace() string
	GetClusterName() string
	ComponentKind() ComponentKind

	// GVK returns the GroupVersionKind of the instance/group.
	GVK() schema.GroupVersionKind
	GetGeneration() int64
	ObservedGeneration() int64
	CurrentRevision() string
	UpdateRevision() string
	CollisionCount() *int32

	IsHealthy() bool
}

func IsUpToDate(a ComponentAccessor) bool {
	return IsReconciled(a) && a.CurrentRevision() == a.UpdateRevision()
}

func StatusChanged(a ComponentAccessor, s CommonStatus) bool {
	return a.CurrentRevision() != s.CurrentRevision || a.UpdateRevision() != s.UpdateRevision || a.CollisionCount() != s.CollisionCount
}

func IsReconciled(a ComponentAccessor) bool {
	return a.ObservedGeneration() == a.GetGeneration()
}

// Instance is the interface for all components.
type Instance interface {
	ComponentAccessor
	*PD | *TiDB | *TiKV | *TiFlash
}

func AllInstancesSynced[T Instance](instances []T, rev string) bool {
	for _, instance := range instances {
		if !IsUpToDate(instance) || instance.CurrentRevision() != rev {
			return false
		}
	}
	return true
}

// Group is the interface for all component groups.
type Group interface {
	ComponentAccessor

	GetDesiredReplicas() int32
	GetDesiredVersion() string
	GetActualVersion() string
	GetStatus() GroupStatus
}

func IsGroupHealthyAndUpToDate(g Group) bool {
	return g.IsHealthy() && IsUpToDate(g) && g.GetStatus().ReadyReplicas == g.GetDesiredReplicas()
}

// GroupType is used for generic functions.
type GroupType interface {
	Group

	*PDGroup | *TiDBGroup | *TiKVGroup | *TiFlashGroup
}

type GroupList interface {
	ToSlice() []Group
}
