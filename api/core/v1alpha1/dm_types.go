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
	DMPortName     = "dm-master"
	DefaultDMPort  = 8261
	DMPeerPortName = "dm-master-peer"
	DefaultDMPeerPort = 8291

	DefaultDMMinReadySeconds = 5
)

const (
	DMGroupCondAvailable   = "Available"
	DMGroupAvailableReason = "DMGroupAvailable"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// DMGroupList defines a list of DM master groups
type DMGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DMGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=dmg
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Desired",type=string,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Updated",type=string,JSONPath=`.status.updatedReplicas`
// +kubebuilder:printcolumn:name="UpdateRevision",type=string,JSONPath=`.status.updateRevision`
// +kubebuilder:printcolumn:name="CurrentRevision",type=string,JSONPath=`.status.currentRevision`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DMGroup defines a group of similar DM master instances.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 40",message="name must not exceed 40 characters"
type DMGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DMGroupSpec   `json:"spec,omitempty"`
	Status DMGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// DMList defines a list of DM master instances
type DMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DM `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=instance
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DM defines a DM master instance
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 47",message="name must not exceed 47 characters"
type DM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DMSpec   `json:"spec,omitempty"`
	Status DMStatus `json:"status,omitempty"`
}

// DMGroupSpec describes the common attributes of a DMGroup
type DMGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled features
	Features []meta.Feature `json:"features,omitempty"`

	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	// MinReadySeconds specifies the minimum number of seconds for which a newly created pod
	// must be ready without any containers crashing for it to be considered available.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MinReadySeconds *int64 `json:"minReadySeconds,omitempty"`

	Template DMTemplate `json:"template"`
}

// DMTemplate defines the template for DM master instances
type DMTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       DMTemplateSpec `json:"spec"`
}

// DMTemplateSpec can only be specified in DMGroup
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
type DMTemplateSpec struct {
	// Version must be a semantic version.
	// It can have a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`
	// Image is the DM master image.
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/dm
	Image *string `json:"image,omitempty"`
	// Server defines server config for DM master
	Server    DMServer             `json:"server,omitempty"`
	Resources ResourceRequirements `json:"resources,omitempty"`
	// UpdateStrategy defines the update strategy for DM master
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// Config defines the config file of DM master
	Config ConfigFile `json:"config,omitempty"`

	// Security defines security config
	Security *Security `json:"security,omitempty"`

	// DataVolume defines the persistent volume for DM master's embedded etcd data.
	// This volume is required for dm-master to persist cluster state.
	DataVolume Volume `json:"dataVolume"`

	// Volumes defines additional persistent volumes, it is optional.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes,omitempty"`
	// Overlay defines a k8s native resource template patch.
	// All resources (pod, pvcs, ...) managed by DM master can be overlaid by this field.
	Overlay *Overlay `json:"overlay,omitempty"`
}

// DMServer defines server config for DM master
type DMServer struct {
	// Ports defines all ports listened by DM master
	Ports DMPorts `json:"ports,omitempty"`
}

// DMPorts defines the ports for DM master
type DMPorts struct {
	// Port defines the main API/gRPC port for DM master. Default is 8261.
	Port *Port `json:"port,omitempty"`
	// PeerPort defines the etcd peer port for DM master. Default is 8291.
	PeerPort *Port `json:"peerPort,omitempty"`
}

// DMGroupStatus defines the observed state of a DMGroup
type DMGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// DMSpec describes the attributes of a DM master instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type DMSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled features
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this DM master instance.
	// It will be translated into a node affinity config.
	// Topology cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported DM master DNS.
	// All DM master instances in the same group use the same subdomain.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// DMTemplateSpec embeds fields managed by DMGroup
	DMTemplateSpec `json:",inline"`
}

// DMStatus defines the observed state of a DM master instance
type DMStatus struct {
	CommonStatus `json:",inline"`

	// MemberID is the etcd member ID of this DM master instance
	MemberID string `json:"memberID,omitempty"`
}
