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
	_ GroupList         = &PDGroupList{}
	_ Group             = &PDGroup{}
	_ ComponentAccessor = &PD{}
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
)

const (
	// TODO: combine all Health condition
	PDCondHealth   = "Health"
	PDHealthReason = "PDHealth"

	// PDCondInitialized means the operator detects that the PD instance has joined the cluster
	PDCondInitialized = "Initialized"

	PDCondSuspended = "Suspended"
	PDSuspendReason = "PDSuspend"

	PDGroupCondSuspended = "Suspended"
	PDGroupSuspendReason = "PDGroupSuspend"
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

func (l *PDGroupList) ToSlice() []Group {
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

// PDGroup defines a group of similar PD instances
type PDGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PDGroupSpec   `json:"spec,omitempty"`
	Status PDGroupStatus `json:"status,omitempty"`
}

func (in *PDGroup) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *PDGroup) GetDesiredReplicas() int32 {
	if in.Spec.Replicas == nil {
		return 0
	}
	return *in.Spec.Replicas
}

func (in *PDGroup) GetDesiredVersion() string {
	return in.Spec.Version
}

func (in *PDGroup) GetActualVersion() string {
	return in.Status.Version
}

func (in *PDGroup) GetStatus() GroupStatus {
	return in.Status.GroupStatus
}

func (in *PDGroup) ComponentKind() ComponentKind {
	return ComponentKindPD
}

func (in *PDGroup) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("PDGroup")
}

func (in *PDGroup) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *PDGroup) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *PDGroup) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *PDGroup) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *PDGroup) IsHealthy() bool {
	// TODO implement me
	return true
}

func (in *PDGroup) GetClientPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Client != nil {
		return in.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return DefaultPDPortClient
}

func (in *PDGroup) GetPeerPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Peer != nil {
		return in.Spec.Template.Spec.Server.Ports.Peer.Port
	}
	return DefaultPDPortPeer
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
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=peer
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Leader",type=string,JSONPath=`.status.isLeader`
// +kubebuilder:printcolumn:name="Initialized",type=string,JSONPath=`.status.conditions[?(@.type=="Initialized")].status`
// +kubebuilder:printcolumn:name="Healthy",type=string,JSONPath=`.status.conditions[?(@.type=="Health")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PD defines a PD instance
type PD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PDSpec   `json:"spec,omitempty"`
	Status PDStatus `json:"status,omitempty"`
}

func (in *PD) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *PD) ComponentKind() ComponentKind {
	return ComponentKindPD
}

func (in *PD) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("PD")
}

func (in *PD) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *PD) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *PD) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *PD) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *PD) IsHealthy() bool {
	return meta.IsStatusConditionTrue(in.Status.Conditions, PDCondHealth) && in.DeletionTimestamp.IsZero()
}

func (in *PD) GetClientPort() int32 {
	if in.Spec.Server.Ports.Client != nil {
		return in.Spec.Server.Ports.Client.Port
	}
	return DefaultPDPortClient
}

func (in *PD) GetPeerPort() int32 {
	if in.Spec.Server.Ports.Peer != nil {
		return in.Spec.Server.Ports.Peer.Port
	}
	return DefaultPDPortPeer
}

// NOTE: name prefix is used to generate all names of underlying resources of this instance
func (in *PD) NamePrefixAndSuffix() (prefix, suffix string) {
	index := strings.LastIndexByte(in.Name, '-')
	// TODO(liubo02): validate name to avoid '-' is not found
	if index == -1 {
		panic("cannot get name prefix")
	}
	return in.Name[:index], in.Name[index+1:]
}

// This name is not only for pod, but also configMap, hostname and almost all underlying resources
// TODO(liubo02): rename to more reasonable one
func (in *PD) PodName() string {
	prefix, suffix := in.NamePrefixAndSuffix()
	return prefix + "-pd-" + suffix
}

// TLSClusterSecretName returns the mTLS secret name for a component.
// TODO(liubo02): move to namer
func (in *PD) TLSClusterSecretName() string {
	prefix, _ := in.NamePrefixAndSuffix()
	return prefix + "-pd-cluster-secret"
}

// PDGroupSpec describes the common attributes of a PDGroup
type PDGroupSpec struct {
	Cluster  ClusterReference `json:"cluster"`
	Replicas *int32           `json:"replicas"`
	Version  string           `json:"version"`

	// Bootstrapped means that pd cluster has been bootstrapped,
	// and there is no need to initialize a new cluster.
	// In other words, this PD group will just join an existing cluster.
	// Normally, this field is automatically changed by operator.
	// If it's true, it cannot be set to false for security
	Bootstrapped bool `json:"bootstrapped,omitempty"`

	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template PDTemplate `json:"template"`
}

type PDTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PDTemplateSpec `json:"spec"`
}

// PDTemplateSpec can only be specified in PDGroup
// TODO: It's name may need to be changed to distinguish from PodTemplateSpec
type PDTemplateSpec struct {
	// Image is pd's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/pd
	Image *string `json:"image,omitempty"`
	// Server defines server config for PD
	Server         PDServer             `json:"server,omitempty"`
	Resources      ResourceRequirements `json:"resources,omitempty"`
	UpdateStrategy UpdateStrategy       `json:"updateStrategy,omitempty"`
	// Config defines config file of PD
	Config ConfigFile `json:"config"`
	// Volumes defines persistent volumes of PD
	Volumes []Volume `json:"volumes"`
	// Overlay defines a k8s native resource template patch
	// All resources(pod, pvcs, ...) managed by PD can be overlayed by this field
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Overlay *Overlay `json:"overlay,omitempty"`
}

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
type PDSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster ClusterReference `json:"cluster"`

	// Topology defines the topology domain of this pd instance
	// It will be translated into a node affinity config
	// Topology cannot be changed
	Topology Topology `json:"topology,omitempty"`

	// Version specifies the PD version
	Version string `json:"version"`

	// Subdomain means the subdomain of the exported pd dns.
	// A same pd cluster will use a same subdomain
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
	IsLeader bool `json:"isLeader"`
}
