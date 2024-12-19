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
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

var (
	_ GroupList         = &TiFlashGroupList{}
	_ Group             = &TiFlashGroup{}
	_ ComponentAccessor = &TiFlash{}
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
	TiFlashCondHealth   = "Health"
	TiFlashHealthReason = "TiFlashHealth"

	TiFlashCondSuspended = "Suspended"
	TiFlashSuspendReason = "TiFlashSuspend"

	TiFlashGroupCondSuspended = "Suspended"
	TiFlashGroupSuspendReason = "TiFlashGroupSuspend"
)

const (
	TiFlashServerLogContainerName = NamePrefix + "serverlog"
	TiFlashErrorLogContainerName  = NamePrefix + "errorlog"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiFlashGroupList defines a list of TiFlash groups
type TiFlashGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiFlashGroup `json:"items"`
}

func (l *TiFlashGroupList) ToSlice() []Group {
	groups := make([]Group, 0, len(l.Items))
	for i := range l.Items {
		groups = append(groups, &l.Items[i])
	}
	return groups
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=tg
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiFlashGroup defines a group of similar TiFlash instances
type TiFlashGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiFlashGroupSpec   `json:"spec,omitempty"`
	Status TiFlashGroupStatus `json:"status,omitempty"`
}

func (in *TiFlashGroup) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiFlashGroup) GetDesiredReplicas() int32 {
	if in.Spec.Replicas == nil {
		return 0
	}
	return *in.Spec.Replicas
}

func (in *TiFlashGroup) GetDesiredVersion() string {
	return in.Spec.Version
}

func (in *TiFlashGroup) GetActualVersion() string {
	return in.Status.Version
}

func (in *TiFlashGroup) GetStatus() GroupStatus {
	return in.Status.GroupStatus
}

func (in *TiFlashGroup) ComponentKind() ComponentKind {
	return ComponentKindTiFlash
}

func (in *TiFlashGroup) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiFlashGroup")
}

func (in *TiFlashGroup) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiFlashGroup) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiFlashGroup) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiFlashGroup) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiFlashGroup) IsHealthy() bool {
	// TODO implement me
	return true
}

func (in *TiFlashGroup) GetFlashPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Flash != nil {
		return in.Spec.Template.Spec.Server.Ports.Flash.Port
	}
	return DefaultTiFlashPortFlash
}

func (in *TiFlashGroup) GetProxyPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Proxy != nil {
		return in.Spec.Template.Spec.Server.Ports.Proxy.Port
	}
	return DefaultTiFlashPortProxy
}

func (in *TiFlashGroup) GetMetricsPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Metrics != nil {
		return in.Spec.Template.Spec.Server.Ports.Metrics.Port
	}
	return DefaultTiFlashPortMetrics
}

func (in *TiFlashGroup) GetProxyStatusPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.ProxyStatus != nil {
		return in.Spec.Template.Spec.Server.Ports.ProxyStatus.Port
	}
	return DefaultTiFlashPortProxyStatus
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
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=tiflash
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="StoreID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="StoreState",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Healthy",type=string,JSONPath=`.status.conditions[?(@.type=="Health")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiFlash defines a TiFlash instance
type TiFlash struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiFlashSpec   `json:"spec,omitempty"`
	Status TiFlashStatus `json:"status,omitempty"`
}

func (in *TiFlash) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiFlash) ComponentKind() ComponentKind {
	return ComponentKindTiFlash
}

func (in *TiFlash) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiFlash")
}

func (in *TiFlash) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiFlash) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiFlash) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiFlash) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiFlash) IsHealthy() bool {
	return meta.IsStatusConditionTrue(in.Status.Conditions, TiFlashCondHealth) && in.DeletionTimestamp.IsZero()
}

func (in *TiFlash) GetFlashPort() int32 {
	if in.Spec.Server.Ports.Flash != nil {
		return in.Spec.Server.Ports.Flash.Port
	}
	return DefaultTiFlashPortFlash
}

func (in *TiFlash) GetProxyPort() int32 {
	if in.Spec.Server.Ports.Proxy != nil {
		return in.Spec.Server.Ports.Proxy.Port
	}
	return DefaultTiFlashPortProxy
}

func (in *TiFlash) GetMetricsPort() int32 {
	if in.Spec.Server.Ports.Metrics != nil {
		return in.Spec.Server.Ports.Metrics.Port
	}
	return DefaultTiFlashPortMetrics
}

func (in *TiFlash) GetProxyStatusPort() int32 {
	if in.Spec.Server.Ports.ProxyStatus != nil {
		return in.Spec.Server.Ports.ProxyStatus.Port
	}
	return DefaultTiFlashPortProxyStatus
}

// NOTE: name prefix is used to generate all names of underlying resources of this instance
func (in *TiFlash) NamePrefixAndSuffix() (prefix, suffix string) {
	index := strings.LastIndexByte(in.Name, '-')
	// TODO(liubo02): validate name to avoid '-' is not found
	if index == -1 {
		panic("cannot get name prefix")
	}
	return in.Name[:index], in.Name[index+1:]
}

// This name is not only for pod, but also configMap, hostname and almost all underlying resources
// TODO(liubo02): rename to more reasonable one
func (in *TiFlash) PodName() string {
	prefix, suffix := in.NamePrefixAndSuffix()
	return prefix + "-tiflash-" + suffix
}

// TLSClusterSecretName returns the mTLS secret name for a component.
// TODO(liubo02): move to namer
func (in *TiFlash) TLSClusterSecretName() string {
	prefix, _ := in.NamePrefixAndSuffix()
	return prefix + "-tiflash-cluster-secret"
}

type TiFlashGroupSpec struct {
	Cluster  ClusterReference `json:"cluster"`
	Replicas *int32           `json:"replicas"`
	Version  string           `json:"version"`

	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`
	Template         TiFlashTemplate  `json:"template"`
}

type TiFlashTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiFlashTemplateSpec `json:"spec"`
}

type TiFlashTemplateSpec struct {
	// Image is tiflash's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/tiflash
	Image *string `json:"image,omitempty"`
	// Server defines the server config of TiFlash
	Server TiFlashServer `json:"server,omitempty"`
	// Resources defines resource required by TiFlash
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Config defines config file of TiFlash
	Config ConfigFile `json:"config"`

	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// ProxyConfig defines config file of TiFlash proxy
	ProxyConfig ConfigFile `json:"proxyConfig,omitempty"`

	// Volumes defines data volume of TiFlash
	Volumes []Volume `json:"volumes"`

	// LogTailer defines the sidercar log tailer config of TiFlash.
	// We always use sidecar to tail the log of TiFlash now.
	LogTailer *TiFlashLogTailer `json:"logTailer,omitempty"`

	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by TiFlash can be overlayed by this field
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
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

type TiFlashSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`
	// Topology defines the topology domain of this pd instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	Topology Topology `json:"topology,omitempty"`
	// Version specifies the TiFlash version
	Version string `json:"version"`
	// Subdomain means the subdomain of the exported TiFlash dns.
	// A same TiFlash group will use a same subdomain
	Subdomain string `json:"subdomain"`

	// TiFlashTemplateSpec embedded some fields managed by TiFlashGroup
	TiFlashTemplateSpec `json:",inline"`
}

type TiFlashStatus struct {
	CommonStatus `json:",inline"`

	// Store ID
	ID string `json:"id,omitempty"`

	// Store State
	State string `json:"state,omitempty"`
}
