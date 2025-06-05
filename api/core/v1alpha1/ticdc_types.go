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
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TiCDCPortName    = "ticdc" // main port
	DefaultTiCDCPort = 8300
)

const (
	TiCDCGroupCondAvailable   = "Available"
	TiCDCGroupAvailableReason = "TiCDCGroupAvailable"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiCDCGroupList defines a list of TiCDC groups
type TiCDCGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiCDCGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=cg
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Desired",type=string,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Updated",type=string,JSONPath=`.status.updatedReplicas`
// +kubebuilder:printcolumn:name="UpdateRevision",type=string,JSONPath=`.status.updateRevision`
// +kubebuilder:printcolumn:name="CurrentRevision",type=string,JSONPath=`.status.currentRevision`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiCDCGroup defines a group of similar TiCDC instances
type TiCDCGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiCDCGroupSpec   `json:"spec,omitempty"`
	Status TiCDCGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiCDCList defines a list of TiCDC instances
type TiCDCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiCDC `json:"items"`
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

// TiCDC defines a TiCDC instance
type TiCDC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiCDCSpec   `json:"spec,omitempty"`
	Status TiCDCStatus `json:"status,omitempty"`
}

// TiCDCGroupSpec describes the common attributes of a TiCDCGroup
type TiCDCGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	Replicas *int32         `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template TiCDCTemplate `json:"template"`
}

type TiCDCTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiCDCTemplateSpec `json:"spec"`
}

// TiCDCTemplateSpec can only be specified in TiCDCGroup
type TiCDCTemplateSpec struct {
	Version string `json:"version"`
	// Image is TiCDC's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/ticdc
	Image *string `json:"image,omitempty"`
	// Server defines server config for TiCDC
	Server         TiCDCServer          `json:"server,omitempty"`
	Resources      ResourceRequirements `json:"resources,omitempty"`
	UpdateStrategy UpdateStrategy       `json:"updateStrategy,omitempty"`
	// Config defines config file of TiCDC
	Config ConfigFile `json:"config,omitempty"`

	// Volumes defines persistent volumes of TiCDC, it is optional.
	// If you want to use ephemeral storage or mount sink TLS certs, you can use "overlay" instead.
	// +listType=map
	// +listMapKey=name
	Volumes []Volume `json:"volumes,omitempty"`
	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by TiCDC can be overlayed by this field
	Overlay *Overlay `json:"overlay,omitempty"`

	// PreStop defines preStop config
	PreStop *TiCDCPreStop `json:"preStop,omitempty"`
}

type TiCDCPreStop struct {
	// Image of pre stop checker
	// Default is pingcap/prestop-checker:latest
	Image *string `json:"image,omitempty"`
}

type TiCDCServer struct {
	// Ports defines all ports listened by TiCDC
	Ports TiCDCPorts `json:"ports,omitempty"`
}

type TiCDCPorts struct {
	// Port defines main port for TiCDC.
	Port *Port `json:"port,omitempty"`
}

type TiCDCGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// TiCDCSpec describes the common attributes of a TiCDC instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type TiCDCSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this TiCDC instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported TiCDC DNS.
	// A same TiCDC cluster will use a same subdomain
	Subdomain string `json:"subdomain"`

	// TiCDCTemplateSpec embedded some fields managed by TiCDCGroup
	TiCDCTemplateSpec `json:",inline"`
}

type TiCDCStatus struct {
	CommonStatus `json:",inline"`

	// ID is the member id of this TiCDC instance
	ID string `json:"id"`

	// should we need to save IsOwner in status?
	// but this value may be changed when scaling in or rolling update
	// TODO(liubo02): change to use condition
	IsOwner bool `json:"isOwner"`
}
