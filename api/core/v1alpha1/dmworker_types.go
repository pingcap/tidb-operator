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
	DMWorkerPortName    = "dm-worker"
	DefaultDMWorkerPort = 8262

	DefaultDMWorkerMinReadySeconds = 5

	VolumeMountTypeDMWorkerRelay    VolumeMountType = "relay-dir"
	VolumeMountDMWorkerRelayDefaultPath             = "/var/lib/dm-worker/relay"
)

const (
	DMWorkerGroupCondAvailable   = "Available"
	DMWorkerGroupAvailableReason = "DMWorkerGroupAvailable"
)

// DMGroupReference is a reference to a DMGroup in the same namespace.
type DMGroupReference struct {
	// Name is the name of the DMGroup.
	Name string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// DMWorkerGroupList defines a list of DM worker groups
type DMWorkerGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DMWorkerGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=dmwg
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="DMGroup",type=string,JSONPath=`.spec.dmGroupRef.name`
// +kubebuilder:printcolumn:name="Desired",type=string,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Updated",type=string,JSONPath=`.status.updatedReplicas`
// +kubebuilder:printcolumn:name="UpdateRevision",type=string,JSONPath=`.status.updateRevision`
// +kubebuilder:printcolumn:name="CurrentRevision",type=string,JSONPath=`.status.currentRevision`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DMWorkerGroup defines a group of similar DM worker instances.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 40",message="name must not exceed 40 characters"
type DMWorkerGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DMWorkerGroupSpec   `json:"spec,omitempty"`
	Status DMWorkerGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// DMWorkerList defines a list of DM worker instances
type DMWorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DMWorker `json:"items"`
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

// DMWorker defines a DM worker instance
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 47",message="name must not exceed 47 characters"
type DMWorker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DMWorkerSpec   `json:"spec,omitempty"`
	Status DMWorkerStatus `json:"status,omitempty"`
}

// DMWorkerGroupSpec describes the common attributes of a DMWorkerGroup
type DMWorkerGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// DMGroupRef is a reference to the DMGroup (dm-master cluster) that workers join.
	DMGroupRef DMGroupReference `json:"dmGroupRef"`
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

	Template DMWorkerTemplate `json:"template"`
}

// DMWorkerTemplate defines the template for DM worker instances
type DMWorkerTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       DMWorkerTemplateSpec `json:"spec"`
}

// DMWorkerTemplateSpec can only be specified in DMWorkerGroup
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
type DMWorkerTemplateSpec struct {
	// Version must be a semantic version.
	// It can have a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`
	// Image is the DM worker image.
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/dm
	Image *string `json:"image,omitempty"`
	// Server defines server config for DM worker
	Server    DMWorkerServer       `json:"server,omitempty"`
	Resources ResourceRequirements `json:"resources,omitempty"`
	// UpdateStrategy defines the update strategy for DM worker
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// Config defines the config file of DM worker
	Config ConfigFile `json:"config,omitempty"`

	// Security defines security config
	Security *Security `json:"security,omitempty"`

	// RelayVolume defines the persistent volume for DM worker's relay log storage.
	// When omitted, no relay log PVC is created and dm-worker uses an in-memory relay store.
	// +optional
	RelayVolume *Volume `json:"relayVolume,omitempty"`

	// Volumes defines additional persistent volumes, it is optional.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes,omitempty"`
	// Overlay defines a k8s native resource template patch.
	// All resources (pod, pvcs, ...) managed by DM worker can be overlaid by this field.
	// Use this to mount upstream/downstream TLS cert Secrets into the dm-worker pod.
	Overlay *Overlay `json:"overlay,omitempty"`
}

// DMWorkerServer defines server config for DM worker
type DMWorkerServer struct {
	// Ports defines all ports listened by DM worker
	Ports DMWorkerPorts `json:"ports,omitempty"`
}

// DMWorkerPorts defines the ports for DM worker
type DMWorkerPorts struct {
	// Port defines the main port for DM worker. Default is 8262.
	Port *Port `json:"port,omitempty"`
}

// DMWorkerGroupStatus defines the observed state of a DMWorkerGroup
type DMWorkerGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// DMWorkerSpec describes the attributes of a DM worker instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type DMWorkerSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// DMGroupRef is a reference to the DMGroup (dm-master cluster) that this worker joins.
	// The controller resolves the dm-master client Service address from this reference.
	DMGroupRef DMGroupReference `json:"dmGroupRef"`
	// Features are enabled features
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this DM worker instance.
	// It will be translated into a node affinity config.
	// Topology cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported DM worker DNS.
	// All DM worker instances in the same group use the same subdomain.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// DMWorkerTemplateSpec embeds fields managed by DMWorkerGroup
	DMWorkerTemplateSpec `json:",inline"`
}

// DMWorkerStatus defines the observed state of a DM worker instance
type DMWorkerStatus struct {
	CommonStatus `json:",inline"`
}
