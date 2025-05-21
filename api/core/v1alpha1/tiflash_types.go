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
)

const (
	TiFlashPortNameFlash       = "tiflash"
	TiFlashPortNameProxy       = "proxy"
	TiFlashPortNameMetrics     = "metrics"
	TiFlashPortNameProxyStatus = "proxy-metrics" // both used for metrics and status, same name as v1

	DefaultTiFlashPortFlash       = 3930
	DefaultTiFlashPortProxy       = 20170
	DefaultTiFlashPortMetrics     = 8234
	DefaultTiFlashPortProxyStatus = 20292
)

const (
	// VolumeMountTypeTiFlashData is the main data dir for the tiflash
	VolumeMountTypeTiFlashData VolumeMountType = "data"

	VolumeMountTiFlashDataDefaultPath = "/var/lib/tiflash"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiFlashGroupList defines a list of TiFlash groups
type TiFlashGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiFlashGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=tc;group,shortName=fg
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

// TiFlashGroup defines a group of similar TiFlash instances
type TiFlashGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiFlashGroupSpec   `json:"spec,omitempty"`
	Status TiFlashGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiFlashList defines a list of TiFlash instances
type TiFlashList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiFlash `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=tc;instance
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="StoreID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="StoreState",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiFlash defines a TiFlash instance
type TiFlash struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiFlashSpec   `json:"spec,omitempty"`
	Status TiFlashStatus `json:"status,omitempty"`
}

type TiFlashGroupSpec struct {
	Cluster  ClusterReference `json:"cluster"`
	Replicas *int32           `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`
	Template         TiFlashTemplate  `json:"template"`
}

type TiFlashTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiFlashTemplateSpec `json:"spec"`
}

type TiFlashTemplateSpec struct {
	Version string `json:"version"`
	// Image is tiflash's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/tiflash
	Image *string `json:"image,omitempty"`
	// Server defines the server config of TiFlash
	Server TiFlashServer `json:"server,omitempty"`
	// Resources defines resource required by TiFlash
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Config defines config file of TiFlash
	Config ConfigFile `json:"config,omitempty"`

	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// ProxyConfig defines config file of TiFlash proxy
	ProxyConfig ConfigFile `json:"proxyConfig,omitempty"`

	// Volumes defines data volume of TiFlash
	// +listType=map
	// +listMapKey=name
	Volumes []Volume `json:"volumes"`

	// LogTailer defines the sidercar log tailer config of TiFlash.
	// We always use sidecar to tail the log of TiFlash now.
	LogTailer *TiFlashLogTailer `json:"logTailer,omitempty"`

	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by TiFlash can be overlayed by this field
	Overlay *Overlay `json:"overlay,omitempty"`
}

type TiFlashServer struct {
	// Ports defines all ports listened by tiflash
	Ports TiFlashPorts `json:"ports,omitempty"`
}

type TiFlashPorts struct {
	Flash   *Port `json:"flash,omitempty"`
	Metrics *Port `json:"metrics,omitempty"`

	Proxy       *Port `json:"proxy,omitempty"`
	ProxyStatus *Port `json:"proxyStatus,omitempty"`
}

type TiFlashLogTailer struct {
	// Image to tail log to stdout
	// Default is busybox:1.37.0
	Image *string `json:"image,omitempty"`

	// ResourceRequirements defines the resource requirements for the log sidecar.
	Resources ResourceRequirements `json:"resources,omitempty"`
}

type TiFlashGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type TiFlashSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Topology defines the topology domain of this pd instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`
	// Subdomain means the subdomain of the exported TiFlash dns.
	// A same TiFlash group will use a same subdomain
	Subdomain string `json:"subdomain"`

	// TiFlashTemplateSpec embedded some fields managed by TiFlashGroup
	TiFlashTemplateSpec `json:",inline"`
}

type TiFlashStatus struct {
	CommonStatus `json:",inline"`
	StoreStatus  `json:",inline"`
}
