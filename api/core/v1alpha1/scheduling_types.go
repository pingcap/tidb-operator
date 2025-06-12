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
	SchedulingPortNameClient    = "client"
	DefaultSchedulingPortClient = 3379
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// SchedulingGroupList defines a list of Scheduling groups
type SchedulingGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SchedulingGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=sg
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

// SchedulingGroup defines a group of similar Scheduling instances
type SchedulingGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulingGroupSpec   `json:"spec,omitempty"`
	Status SchedulingGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// SchedulingList defines a list of Scheduling instances
type SchedulingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Scheduling `json:"items"`
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

// Scheduling defines a Scheduling instance
type Scheduling struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulingSpec   `json:"spec,omitempty"`
	Status SchedulingStatus `json:"status,omitempty"`
}

// SchedulingGroupSpec describes the common attributes of aSchedulingGroupup
type SchedulingGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	Replicas *int32         `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template SchedulingTemplate `json:"template"`
}

type SchedulingTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       SchedulingTemplateSpec `json:"spec"`
}

// SchedulingTemplateSpec can only be specified inSchedulingGroupup
// TODO: It's name may need to be changed to distinguish from PodTemplateSpec
type SchedulingTemplateSpec struct {
	Version string `json:"version"`

	// Image is pd's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/pd
	Image *string `json:"image,omitempty"`

	// Server defines server config for Scheduling
	Server SchedulingServer `json:"server,omitempty"`

	Resources ResourceRequirements `json:"resources,omitempty"`

	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// Config defines config file of Scheduling
	// See https://docs.pingcap.com/tidb/stable/Scheduling-configuration-file/
	Config ConfigFile `json:"config,omitempty"`

	// Volumes defines persistent volumes of Scheduling
	// +listType=map
	// +listMapKey=name
	Volumes []Volume `json:"volumes,omitempty"`

	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by Scheduling can be overlayed by this field
	Overlay *Overlay `json:"overlay,omitempty"`
}

type SchedulingServer struct {
	// Ports defines all ports listened by Scheduling
	Ports SchedulingPorts `json:"ports,omitempty"`
}

type SchedulingPorts struct {
	// Client defines port for Scheduling's api service
	Client *Port `json:"client,omitempty"`
}

type SchedulingGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// SchedulingSpec describes the common attributes of a Scheduling instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type SchedulingSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this Scheduling instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported Scheduling dns.
	// A same Scheduling cluster will use a same subdomain
	Subdomain string `json:"subdomain"`

	// SchedulingTemplateSpec embedded some fields managed bySchedulingGroupup
	SchedulingTemplateSpec `json:",inline"`
}

type SchedulingStatus struct {
	CommonStatus `json:",inline"`
}
