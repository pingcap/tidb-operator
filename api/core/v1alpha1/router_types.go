// Copyright 2026 PingCAP, Inc.
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
	RouterPortNameClient    = "client"
	DefaultRouterPortClient = 2379

	// DefaultRouterMinReadySeconds is default min ready seconds of router
	DefaultRouterMinReadySeconds = 5
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// RouterGroupList defines a list of Router groups
type RouterGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RouterGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=rg
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

// RouterGroup defines a group of similar Router instances
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 40",message="name must not exceed 40 characters"
type RouterGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RouterGroupSpec   `json:"spec,omitempty"`
	Status RouterGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// RouterList defines a list of Router instances
type RouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Router `json:"items"`
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

// Router defines a Router instance
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 47",message="name must not exceed 47 characters"
type Router struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RouterSpec   `json:"spec,omitempty"`
	Status RouterStatus `json:"status,omitempty"`
}

// RouterGroupSpec describes the common attributes of a RouterGroup
type RouterGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	// MinReadySeconds specifies the minimum number of seconds for which a newly created pod be ready without any of its containers crashing, for it to be considered available.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MinReadySeconds *int64 `json:"minReadySeconds,omitempty"`

	Template RouterTemplate `json:"template"`
}

type RouterTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       RouterTemplateSpec `json:"spec"`
}

// RouterTemplateSpec can only be specified in RouterGroup
// TODO: It's name may need to be changed to distinguish from PodTemplateSpec
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
type RouterTemplateSpec struct {
	// Version must be a semantic version.
	// It can has a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`

	// Image is pd's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/pd
	Image *string `json:"image,omitempty"`

	// Server defines server config for Router
	Server RouterServer `json:"server,omitempty"`

	Resources ResourceRequirements `json:"resources,omitempty"`

	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// Config defines config file of Router
	Config ConfigFile `json:"config,omitempty"`

	// Security defines security config
	Security *Security `json:"security,omitempty"`

	// Volumes defines persistent volumes of Router
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes,omitempty"`

	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by Router can be overlayed by this field
	Overlay *Overlay `json:"overlay,omitempty"`
}

type RouterServer struct {
	// Ports defines all ports listened by Router
	Ports RouterPorts `json:"ports,omitempty"`
}

type RouterPorts struct {
	// Client defines port for Router's api service
	Client *Port `json:"client,omitempty"`
}

type RouterGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// RouterSpec describes the common attributes of a Router instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type RouterSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this Router instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported Router dns.
	// A same Router cluster will use a same subdomain
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// RouterTemplateSpec embedded some fields managed by RouterGroup
	RouterTemplateSpec `json:",inline"`
}

type RouterStatus struct {
	CommonStatus `json:",inline"`
}
