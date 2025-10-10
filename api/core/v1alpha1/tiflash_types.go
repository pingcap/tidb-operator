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
// +kubebuilder:resource:categories=group,shortName=fg
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
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 40",message="name must not exceed 40 characters"
type TiFlashGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiFlashGroupSpec   `json:"spec,omitempty"`
	Status TiFlashGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiFlashList defines a list of TiFlash instances.
type TiFlashList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiFlash `json:"items"`
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

// TiFlash defines a TiFlash instance.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 47",message="name must not exceed 47 characters"
type TiFlash struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiFlashSpec   `json:"spec,omitempty"`
	Status TiFlashStatus `json:"status,omitempty"`
}

type TiFlashGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`
	Template         TiFlashTemplate  `json:"template"`
}

type TiFlashTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiFlashTemplateSpec `json:"spec"`
}

// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
// +kubebuilder:validation:XValidation:rule="has(self.volumes) && ('data' in self.volumes.map(v, v.name))",message="data volume must be configured"
type TiFlashTemplateSpec struct {
	// Version must be a semantic version.
	// It can has a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
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

	// Security defines security config
	Security *Security `json:"security,omitempty"`

	// Volumes defines data volume of TiFlash
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
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
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	// Topology defines the topology domain of this pd instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`
	// Subdomain means the subdomain of the exported TiFlash dns.
	// A same TiFlash group will use a same subdomain
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// Offline marks the store as offline in PD to begin data migration.
	// When true, the store will be marked as offline in PD.
	// When false, the store will be marked as online in PD (if possible).
	// +optional
	Offline *bool `json:"offline,omitempty"`

	// TiFlashTemplateSpec embedded some fields managed by TiFlashGroup
	TiFlashTemplateSpec `json:",inline"`
}

type TiFlashStatus struct {
	CommonStatus `json:",inline"`
	StoreStatus  `json:",inline"`
}
