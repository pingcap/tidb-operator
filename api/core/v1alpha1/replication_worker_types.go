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
	ReplicationWorkerPortNameGRPC    = "grpc"
	DefaultReplicationWorkerPortGRPC = 19160

	VolumeMountTypeReplicationWorkerData        VolumeMountType = "data"
	VolumeMountReplicationWorkerDataDefaultPath                 = "/var/lib/replication-worker"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ReplicationWorkerGroupList defines a list of ReplicationWorker groups
type ReplicationWorkerGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ReplicationWorkerGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=rwg
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

// ReplicationWorkerGroup defines a group of similar ReplicationWorker instances
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 30",message="name must not exceed 30 characters"
type ReplicationWorkerGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicationWorkerGroupSpec   `json:"spec,omitempty"`
	Status ReplicationWorkerGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ReplicationWorkerList defines a list of ReplicationWorker instances
type ReplicationWorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ReplicationWorker `json:"items"`
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

// ReplicationWorker defines a ReplicationWorker instance
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 37",message="name must not exceed 37 characters"
type ReplicationWorker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicationWorkerSpec   `json:"spec,omitempty"`
	Status ReplicationWorkerStatus `json:"status,omitempty"`
}

// ReplicationWorkerGroupSpec describes the common attributes of a ReplicationWorkerGroup
type ReplicationWorkerGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Currently, ReplicationWorkerGroup only supports one replicas.
	// +kubebuilder:validation:Maximum=1
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template ReplicationWorkerTemplate `json:"template"`
}

type ReplicationWorkerTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       ReplicationWorkerTemplateSpec `json:"spec"`
}

// ReplicationWorkerTemplateSpec can only be specified in ReplicationWorkerGroup
// TODO: It's name may need to be changed to distinguish from PodTemplateSpec
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
// +kubebuilder:validation:XValidation:rule="has(self.volumes) && ('data' in self.volumes.map(v, v.name))",message="data volume must be configured"
type ReplicationWorkerTemplateSpec struct {
	// Version must be a semantic version.
	// It can have a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`

	// Image is ReplicationWorker's image.
	// If tag is omitted, version will be used as the image tag.
	Image *string `json:"image,omitempty"`

	// Server defines server config for ReplicationWorker
	Server ReplicationWorkerServer `json:"server,omitempty"`

	Resources ResourceRequirements `json:"resources,omitempty"`

	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// Config defines config file of ReplicationWorker
	// See https://docs.pingcap.com/tidb/stable/ReplicationWorker-configuration-file/
	Config ConfigFile `json:"config,omitempty"`

	// Volumes defines persistent volumes of ReplicationWorker
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes"`

	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by ReplicationWorker can be overlayed by this field
	Overlay *Overlay `json:"overlay,omitempty"`
}

type ReplicationWorkerServer struct {
	// Ports defines all ports listened by ReplicationWorker
	Ports ReplicationWorkerPorts `json:"ports,omitempty"`
}

type ReplicationWorkerPorts struct {
	GRPC *Port `json:"grpc,omitempty"`
}

type ReplicationWorkerGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// ReplicationWorkerSpec describes the common attributes of a ReplicationWorker instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type ReplicationWorkerSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this ReplicationWorker instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported ReplicationWorker dns.
	// A same ReplicationWorker cluster will use a same subdomain
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// ReplicationWorkerTemplateSpec embedded some fields managed by ReplicationWorkerGroup
	ReplicationWorkerTemplateSpec `json:",inline"`
}

type ReplicationWorkerStatus struct {
	CommonStatus `json:",inline"`
}
