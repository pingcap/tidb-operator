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
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

var (
	_ GroupList         = &TiCDCGroupList{}
	_ Group             = &TiCDCGroup{}
	_ ComponentAccessor = &TiCDC{}
)

const (
	TiCDCPortNameClient    = "client"
	TiCDCPortNamePeer      = "peer"
	DefaultTiCDCPortClient = 2379
	DefaultTiCDCPortPeer   = 2380
)

const (
	// TODO: combine all Health condition
	TiCDCCondHealth   = "Health"
	TiCDCHealthReason = "TiCDCHealth"

	TiCDCCondSuspended = "Suspended"
	TiCDCSuspendReason = "TiCDCSuspend"

	TiCDCGroupCondAvailable   = "Available"
	TiCDCGroupAvailableReason = "TiCDCGroupAvailable"

	TiCDCGroupCondSuspended = "Suspended"
	TiCDCGroupSuspendReason = "TiCDCGroupSuspend"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiCDCGroupList defines a list of TiCDC groups
type TiCDCGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiCDCGroup `json:"items"`
}

func (l *TiCDCGroupList) ToSlice() []Group {
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

// TiCDCGroup defines a group of similar TiCDC instances
type TiCDCGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiCDCGroupSpec   `json:"spec,omitempty"`
	Status TiCDCGroupStatus `json:"status,omitempty"`
}

func (in *TiCDCGroup) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiCDCGroup) GetDesiredReplicas() int32 {
	if in.Spec.Replicas == nil {
		return 0
	}
	return *in.Spec.Replicas
}

func (in *TiCDCGroup) GetDesiredVersion() string {
	return in.Spec.Template.Spec.Version
}

func (in *TiCDCGroup) GetActualVersion() string {
	return in.Status.Version
}

func (in *TiCDCGroup) GetStatus() GroupStatus {
	return in.Status.GroupStatus
}

func (in *TiCDCGroup) ComponentKind() ComponentKind {
	return ComponentKindTiCDC
}

func (in *TiCDCGroup) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiCDCGroup")
}

func (in *TiCDCGroup) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiCDCGroup) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiCDCGroup) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiCDCGroup) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiCDCGroup) IsHealthy() bool {
	// TODO implement me
	return true
}

func (in *TiCDCGroup) GetClientPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Client != nil {
		return in.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return DefaultTiCDCPortClient
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
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=peer
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.status.isOwner`
// +kubebuilder:printcolumn:name="Healthy",type=string,JSONPath=`.status.conditions[?(@.type=="Health")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiCDC defines a TiCDC instance
type TiCDC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiCDCSpec   `json:"spec,omitempty"`
	Status TiCDCStatus `json:"status,omitempty"`
}

func (in *TiCDC) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiCDC) ComponentKind() ComponentKind {
	return ComponentKindTiCDC
}

func (in *TiCDC) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiCDC")
}

func (in *TiCDC) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiCDC) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiCDC) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiCDC) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiCDC) IsHealthy() bool {
	return meta.IsStatusConditionTrue(in.Status.Conditions, TiCDCCondHealth) && in.DeletionTimestamp.IsZero()
}

func (in *TiCDC) GetClientPort() int32 {
	if in.Spec.Server.Ports.Client != nil {
		return in.Spec.Server.Ports.Client.Port
	}
	return DefaultTiCDCPortClient
}

func (in *TiCDC) GetGracefulShutdownTimeout() time.Duration {
	if in.Spec.GracefulShutdownTimeout != nil {
		return in.Spec.GracefulShutdownTimeout.Duration
	}
	return 10 * time.Minute
}

// NOTE: name prefix is used to generate all names of underlying resources of this instance
func (in *TiCDC) NamePrefixAndSuffix() (prefix, suffix string) {
	index := strings.LastIndexByte(in.Name, '-')
	// TODO(liubo02): validate name to avoid '-' is not found
	if index == -1 {
		panic("cannot get name prefix")
	}
	return in.Name[:index], in.Name[index+1:]
}

// This name is not only for pod, but also configMap, hostname and almost all underlying resources
// TODO(liubo02): rename to more reasonable one
func (in *TiCDC) PodName() string {
	prefix, suffix := in.NamePrefixAndSuffix()
	return prefix + "-ticdc-" + suffix
}

// TLSClusterSecretName returns the mTLS secret name for a component.
// TODO(liubo02): move to namer
func (in *TiCDC) TLSClusterSecretName() string {
	prefix, _ := in.NamePrefixAndSuffix()
	return prefix + "-ticdc-cluster-secret"
}

// TiCDCGroupSpec describes the common attributes of a TiCDCGroup
type TiCDCGroupSpec struct {
	Cluster  ClusterReference `json:"cluster"`
	Replicas *int32           `json:"replicas"`

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
	Config ConfigFile `json:"config"`

	Security *TiCDCSecurity `json:"security,omitempty"`

	// Volumes defines persistent volumes of TiCDC, it is optional.
	Volumes []Volume `json:"volumes,omitempty"`
	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by TiCDC can be overlayed by this field
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Overlay *Overlay `json:"overlay,omitempty"`

	// GracefulShutdownTimeout is the timeout of gracefully shutdown a TiCDC pod
	// when scaling in or rolling update.
	// Encoded in the format of Go Duration.
	// Defaults to 10m
	GracefulShutdownTimeout *metav1.Duration `json:"gracefulShutdownTimeout,omitempty"`
}

type TiCDCSecurity struct {
	TLS *TiCDCTLS `json:"tls,omitempty" default:"{}"`
}

type TiCDCTLS struct {
	// SinkTLSSecretNames are the names of secrets that store the
	// client certificate for the sink.
	SinkTLSSecretNames []string `json:"sinkTLSSecretNames,omitempty"`
}

type TiCDCServer struct {
	// Ports defines all ports listened by TiCDC
	Ports TiCDCPorts `json:"ports,omitempty"`
}

type TiCDCPorts struct {
	// Client defines port for TiCDC's api service
	Client *Port `json:"client,omitempty"`
}

type TiCDCGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// TiCDCSpec describes the common attributes of a TiCDC instance
type TiCDCSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`

	// Topology defines the topology domain of this TiCDC instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
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

	// IsOwner indicates whether this TiCDC is the owner
	IsOwner bool `json:"isOwner"`
}
