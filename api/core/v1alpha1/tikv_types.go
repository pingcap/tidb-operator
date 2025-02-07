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
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

var (
	_ GroupList         = &TiKVGroupList{}
	_ Group             = &TiKVGroup{}
	_ ComponentAccessor = &TiKV{}
)

const (
	// VolumeMountTypeTiKVData is the main data dir for the tikv
	// The default sub path of this type is ""
	VolumeMountTypeTiKVData VolumeMountType = "data"

	VolumeMountTiKVDataDefaultPath = "/var/lib/tikv"
)

const (
	TiKVPortNameClient    = "client"
	TiKVPortNameStatus    = "status"
	DefaultTiKVPortClient = 20160
	DefaultTiKVPortStatus = 20180
)

const (
	TiKVCondHealth   = CondHealth
	TiKVHealthReason = "TiKVHealth"

	TiKVCondLeadersEvicted = "LeadersEvicted"

	TiKVCondSuspended = CondSuspended
	TiKVSuspendReason = "TiKVSuspend"

	TiKVGroupCondSuspended = CondSuspended
	TiKVGroupSuspendReason = "TiKVGroupSuspend"
)

const (
	// store state for both TiKV and TiFlash stores

	StoreStateUnknown   = "Unknown"
	StoreStatePreparing = "Preparing"
	StoreStateServing   = "Serving"
	StoreStateRemoving  = "Removing"
	StoreStateRemoved   = "Removed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiKVGroupList defines a list of TiKV groups
type TiKVGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiKVGroup `json:"items"`
}

func (l *TiKVGroupList) ToSlice() []Group {
	groups := make([]Group, 0, len(l.Items))
	for i := range l.Items {
		groups = append(groups, &l.Items[i])
	}
	return groups
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=tg
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiKVGroup defines a group of similar TiKV instances
type TiKVGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiKVGroupSpec   `json:"spec,omitempty"`
	Status TiKVGroupStatus `json:"status,omitempty"`
}

func (in *TiKVGroup) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiKVGroup) ComponentKind() ComponentKind {
	return ComponentKindTiKV
}

func (in *TiKVGroup) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiKVGroup")
}

func (in *TiKVGroup) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiKVGroup) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiKVGroup) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiKVGroup) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiKVGroup) IsHealthy() bool {
	// TODO implement me
	return true
}

func (in *TiKVGroup) GetDesiredReplicas() int32 {
	if in.Spec.Replicas == nil {
		return 0
	}
	return *in.Spec.Replicas
}

func (in *TiKVGroup) GetDesiredVersion() string {
	return in.Spec.Template.Spec.Version
}

func (in *TiKVGroup) GetActualVersion() string {
	return in.Status.Version
}

func (in *TiKVGroup) GetStatus() GroupStatus {
	return in.Status.GroupStatus
}

func (in *TiKVGroup) GetClientPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Client != nil {
		return in.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return DefaultTiKVPortClient
}

func (in *TiKVGroup) GetStatusPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Status != nil {
		return in.Spec.Template.Spec.Server.Ports.Status.Port
	}
	return DefaultTiKVPortStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiKVList defines a list of TiKV instances
type TiKVList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiKV `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=tikv
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="StoreID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="StoreState",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Healthy",type=string,JSONPath=`.status.conditions[?(@.type=="Health")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiKV defines a TiKV instance
type TiKV struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiKVSpec   `json:"spec,omitempty"`
	Status TiKVStatus `json:"status,omitempty"`
}

func (in *TiKV) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiKV) ComponentKind() ComponentKind {
	return ComponentKindTiKV
}

func (in *TiKV) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiKV")
}

func (in *TiKV) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiKV) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiKV) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiKV) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiKV) IsHealthy() bool {
	return meta.IsStatusConditionTrue(in.Status.Conditions, TiKVCondHealth) && in.DeletionTimestamp.IsZero()
}

func (in *TiKV) GetClientPort() int32 {
	if in.Spec.Server.Ports.Client != nil {
		return in.Spec.Server.Ports.Client.Port
	}
	return DefaultTiKVPortClient
}

func (in *TiKV) GetStatusPort() int32 {
	if in.Spec.Server.Ports.Status != nil {
		return in.Spec.Server.Ports.Status.Port
	}
	return DefaultTiKVPortStatus
}

// NOTE: name prefix is used to generate all names of underlying resources of this instance
func (in *TiKV) NamePrefixAndSuffix() (prefix, suffix string) {
	index := strings.LastIndexByte(in.Name, '-')
	// TODO(liubo02): validate name to avoid '-' is not found
	if index == -1 {
		panic("cannot get name prefix")
	}
	return in.Name[:index], in.Name[index+1:]
}

// This name is not only for pod, but also configMap, hostname and almost all underlying resources
// TODO(liubo02): rename to more reasonable one
func (in *TiKV) PodName() string {
	prefix, suffix := in.NamePrefixAndSuffix()
	return prefix + "-tikv-" + suffix
}

// TLSClusterSecretName returns the mTLS secret name for a component.
// TODO(liubo02): move to namer
func (in *TiKV) TLSClusterSecretName() string {
	prefix, _ := in.NamePrefixAndSuffix()
	return prefix + "-tikv-cluster-secret"
}

// TiKVGroupSpec describes the common attributes of a TiKVGroup
type TiKVGroupSpec struct {
	Cluster  ClusterReference `json:"cluster"`
	Replicas *int32           `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template TiKVTemplate `json:"template"`
}

type TiKVTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiKVTemplateSpec `json:"spec"`
}

// TiKVTemplateSpec can only be specified in TiKVGroup
// TODO: It's name may need to be changed to distinguish from PodTemplateSpec
type TiKVTemplateSpec struct {
	Version string `json:"version"`
	// Image is tikv's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/tikv
	Image *string `json:"image,omitempty"`
	// Server defines the server config of TiKV
	Server TiKVServer `json:"server,omitempty"`
	// Resources defines resource required by TiKV
	Resources ResourceRequirements `json:"resources,omitempty"`
	// Config defines config file of TiKV
	Config         ConfigFile     `json:"config"`
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// Volumes defines data volume of TiKV
	Volumes []Volume `json:"volumes"`

	// PreStop defines preStop config
	PreStop *TiKVPreStop `json:"preStop,omitempty"`
	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by TiKV can be overlayed by this field
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Overlay *Overlay `json:"overlay,omitempty"`
}

type TiKVPreStop struct {
	// Image of pre stop checker
	// Default is pingcap/prestop-checker:latest
	Image *string `json:"image,omitempty"`
}

type TiKVServer struct {
	// Ports defines all ports listened by tikv
	Ports TiKVPorts `json:"ports,omitempty"`
}

type TiKVPorts struct {
	// Client defines port for tikv's api service
	Client *Port `json:"client,omitempty"`
	// Status defines port for tikv status api
	Status *Port `json:"peer,omitempty"`
}

type TiKVGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

type TiKVSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Topology defines the topology domain of this pd instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	Topology Topology `json:"topology,omitempty"`
	// Subdomain means the subdomain of the exported tikv dns.
	// A same tikv group will use a same subdomain
	Subdomain string `json:"subdomain"`

	// TiKVTemplateSpec embedded some fields managed by TiKVGroup
	TiKVTemplateSpec `json:",inline"`
}

type TiKVStatus struct {
	CommonStatus `json:",inline"`

	// Store ID
	ID string `json:"id,omitempty"`

	// Store State
	State string `json:"state,omitempty"`
}
