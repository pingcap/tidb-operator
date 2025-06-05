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
	TSOPortNameClient    = "client"
	DefaultTSOPortClient = 3379
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TSOGroupList defines a list of TSO groups
type TSOGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TSOGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=tc;group,shortName=tg
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

// TSOGroup defines a group of similar TSO instances
type TSOGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TSOGroupSpec   `json:"spec,omitempty"`
	Status TSOGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TSOList defines a list of TSO instances
type TSOList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TSO `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=tc;instance
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TSO defines a TSO instance
type TSO struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TSOSpec   `json:"spec,omitempty"`
	Status TSOStatus `json:"status,omitempty"`
}

// TSOGroupSpec describes the common attributes of a TSOGroup
type TSOGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	Replicas *int32         `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template TSOTemplate `json:"template"`
}

type TSOTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TSOTemplateSpec `json:"spec"`
}

// TSOTemplateSpec can only be specified in TSOGroup
// TODO: It's name may need to be changed to distinguish from PodTemplateSpec
type TSOTemplateSpec struct {
	Version string `json:"version"`

	// Image is pd's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/pd
	Image *string `json:"image,omitempty"`

	// Server defines server config for TSO
	Server TSOServer `json:"server,omitempty"`

	Resources ResourceRequirements `json:"resources,omitempty"`

	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// Config defines config file of TSO
	// See https://docs.pingcap.com/tidb/stable/tso-configuration-file/
	Config ConfigFile `json:"config,omitempty"`

	// Volumes defines persistent volumes of TSO
	// +listType=map
	// +listMapKey=name
	Volumes []Volume `json:"volumes,omitempty"`

	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by TSO can be overlayed by this field
	Overlay *Overlay `json:"overlay,omitempty"`
}

type TSOServer struct {
	// Ports defines all ports listened by tso
	Ports TSOPorts `json:"ports,omitempty"`
}

type TSOPorts struct {
	// Client defines port for tso's api service
	Client *Port `json:"client,omitempty"`
}

type TSOGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// TSOSpec describes the common attributes of a TSO instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type TSOSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this tso instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported tso dns.
	// A same tso cluster will use a same subdomain
	Subdomain string `json:"subdomain"`

	// TSOTemplateSpec embedded some fields managed by TSOGroup
	TSOTemplateSpec `json:",inline"`
}

type TSOStatus struct {
	CommonStatus `json:",inline"`
}
