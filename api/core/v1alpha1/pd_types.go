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
	// VolumeMountTypePDData means data dir of PD
	VolumeMountTypePDData VolumeMountType = "data"

	VolumeMountPDDataDefaultPath = "/var/lib/pd"
)

const (
	PDPortNameClient    = "client"
	PDPortNamePeer      = "peer"
	DefaultPDPortClient = 2379
	DefaultPDPortPeer   = 2380

	// DefaultPDMinReadySeconds is default min ready seconds of pd
	DefaultPDMinReadySeconds = 5
)

const (
	// PDCondInitialized means the operator detects that the PD instance has joined the cluster
	PDCondInitialized = "Initialized"
)

const (
	AnnoKeyInitialClusterNum = "pd.core.pingcap.com/initial-cluster-num"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// PDGroupList defines a list of PD groups
type PDGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PDGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=pdg
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

// PDGroup defines a group of similar PD instances.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 40",message="name must not exceed 40 characters"
type PDGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PDGroupSpec   `json:"spec,omitempty"`
	Status PDGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// PDList defines a list of PD instances
type PDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PD `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=instance
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Leader",type=string,JSONPath=`.status.isLeader`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PD defines a PD instance.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 47",message="name must not exceed 47 characters"
type PD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PDSpec   `json:"spec,omitempty"`
	Status PDStatus `json:"status,omitempty"`
}

// PDGroupSpec describes the common attributes of a PDGroup
// +kubebuilder:validation:XValidation:rule="self.bootstrapped || (self.replicas == oldSelf.replicas)",message="replicas cannot be changed when bootstrapped is false"
type PDGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`

	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

	// Bootstrapped means that pd cluster has been bootstrapped,
	// and there is no need to initialize a new cluster.
	// In other words, this PD group will just join an existing cluster.
	// Normally, this field is automatically changed by operator.
	// If it's true, it cannot be set to false for security
	// +kubebuilder:validation:XValidation:rule="!oldSelf || self",message="bootstrapped cannot be changed from true to false"
	Bootstrapped bool `json:"bootstrapped,omitempty"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template PDTemplate `json:"template"`
}

type PDTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PDTemplateSpec `json:"spec"`
}

// PDTemplateSpec can only be specified in PDGroup
// TODO: It's name may need to be changed to distinguish from PodTemplateSpec
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.mode) && !has(self.mode)) || (has(oldSelf.mode) && has(self.mode))",fieldPath=".mode",message="pd mode can only be set when creating now"
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
// +kubebuilder:validation:XValidation:rule="has(self.volumes) && ('data' in self.volumes.map(v, v.name))",message="data volume must be configured"
type PDTemplateSpec struct {
	// Version must be a semantic version.
	// It can has a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`
	// Image is pd's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/pd
	Image *string `json:"image,omitempty"`

	// Mode of pd, default is normal
	// Now it cannot be changed after creation. It may be supported in the future.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="pd mode is immutable now, it may be supported later"
	Mode PDMode `json:"mode,omitempty"`
	// Server defines server config for PD
	Server         PDServer             `json:"server,omitempty"`
	Resources      ResourceRequirements `json:"resources,omitempty"`
	UpdateStrategy UpdateStrategy       `json:"updateStrategy,omitempty"`
	// Config defines config file of PD
	Config ConfigFile `json:"config,omitempty"`
	// Volumes defines persistent volumes of PD
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes"`
	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by PD can be overlaid by this field
	Overlay *Overlay `json:"overlay,omitempty"`
}

type PDMode string

const (
	// Normal mode of PD, all services are provided by a single component
	PDModeNormal = ""
	// Micro Service Mode of PD, some services(TSO,Scheduler) are separated from PD
	// See https://docs.pingcap.com/tidb/stable/pd-microservices/
	PDModeMS = "ms"
)

type PDServer struct {
	// Ports defines all ports listened by pd
	Ports PDPorts `json:"ports,omitempty"`
}

type PDPorts struct {
	// Client defines port for pd's api service
	Client *Port `json:"client,omitempty"`
	// Peer defines port for peer communication
	Peer *Port `json:"peer,omitempty"`
}

type PDGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// PDSpec describes the common attributes of a PD instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type PDSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`

	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this pd instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported pd dns.
	// A same pd cluster will use a same subdomain
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// PDTemplateSpec embedded some fields managed by PDGroup
	PDTemplateSpec `json:",inline"`
}

type PDStatus struct {
	CommonStatus `json:",inline"`

	// ID is the member id of this pd instance
	ID string `json:"id"`

	// IsLeader indicates whether this pd is the leader
	// NOTE: it's a snapshot from PD, not always up to date
	// TODO(liubo02): change to use condition
	IsLeader bool `json:"isLeader"`
}
