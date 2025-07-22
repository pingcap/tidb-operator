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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
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
	TiKVCondLeadersEvicted = "LeadersEvicted"
)

const (
	// store state for both TiKV and TiFlash stores

	// StoreStatePreparing means there are no regions on this store.
	// In this state, the store is ready to serve.
	StoreStatePreparing = "Preparing"
	// StoreStateServing means the number of regions reaches a certain proportion.
	// In this state, the store is ready to serve.
	StoreStateServing  = "Serving"
	StoreStateRemoving = "Removing"
	StoreStateRemoved  = "Removed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiKVGroupList defines a list of TiKV groups
type TiKVGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiKVGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=kvg
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Desired",type=string,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Updated",type=string,JSONPath=`.status.updatedReplicas`
// +kubebuilder:printcolumn:name="UpdateRevision",type=string,JSONPath=`.status.updateRevision`
// +kubebuilder:printcolumn:name="CurrentRevision",type=string,JSONPath=`.status.currentRevision`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiKVGroup defines a group of similar TiKV instances.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 30",message="name must not exceed 30 characters"
type TiKVGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiKVGroupSpec   `json:"spec,omitempty"`
	Status TiKVGroupStatus `json:"status,omitempty"`
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
// +kubebuilder:resource:categories=instance
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="StoreID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="StoreState",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Offline",type=boolean,JSONPath=`.spec.offline`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiKV defines a TiKV instance.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 37",message="name must not exceed 37 characters"
type TiKV struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiKVSpec   `json:"spec,omitempty"`
	Status TiKVStatus `json:"status,omitempty"`
}

// TiKVGroupSpec describes the common attributes of a TiKVGroup
type TiKVGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

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
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
// +kubebuilder:validation:XValidation:rule="has(self.volumes) && ('data' in self.volumes.map(v, v.name))",message="data volume must be configured"
type TiKVTemplateSpec struct {
	// Version must be a semantic version.
	// It can has a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
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
	Config         ConfigFile     `json:"config,omitempty"`
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// Volumes defines data volume of TiKV
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes"`

	// PreStop defines preStop config
	PreStop *TiKVPreStop `json:"preStop,omitempty"`
	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by TiKV can be overlayed by this field
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

// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type TiKVSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	// Topology defines the topology domain of this pd instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`
	// Subdomain means the subdomain of the exported tikv dns.
	// A same tikv group will use a same subdomain
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// Offline marks the store as offline in PD to begin data migration.
	// When true, the store will be marked as offline in PD.
	// When false, the store will be marked as online in PD (if possible).
	Offline bool `json:"offline"`

	// TiKVTemplateSpec embedded some fields managed by TiKVGroup
	TiKVTemplateSpec `json:",inline"`
}

type TiKVStatus struct {
	CommonStatus `json:",inline"`
	StoreStatus  `json:",inline"`
}
