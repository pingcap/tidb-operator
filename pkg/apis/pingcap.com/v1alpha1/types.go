// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	apps "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

const (
	// AnnotationStorageSize is a storage size annotation key
	AnnotationStorageSize string = "storage.pingcap.com/size"

	// TiDBVolumeName is volume name for TiDB volume
	TiDBVolumeName string = "tidb-volume"

	// TiKVStateUp represents status of Up of TiKV
	TiKVStateUp string = "Up"
)

// MemberType represents member type
type MemberType string

const (
	// PDMemberType is pd container type
	PDMemberType MemberType = "pd"

	// TiDBMemberType is tidb container type
	TiDBMemberType MemberType = "tidb"

	// TiKVMemberType is tikv container type
	TiKVMemberType MemberType = "tikv"

	//PushGatewayMemberType is pushgateway container type
	PushGatewayMemberType MemberType = "pushgateway"

	// UnknownMemberType is unknown container type
	UnknownMemberType MemberType = "unknown"
)

// MemberPhase is the current state of member
type MemberPhase string

const (
	// Normal represents normal state of TiDB cluster.
	Normal MemberPhase = "Normal"
	// Upgrade represents the upgrade state of TiDB cluster.
	Upgrade MemberPhase = "Upgrade"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbCluster is the control script's spec
type TidbCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the behavior of a tidb cluster
	Spec TidbClusterSpec `json:"spec"`

	// Most recently observed status of the tidb cluster
	Status TidbClusterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbClusterList is TidbCluster list
type TidbClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TidbCluster `json:"items"`
}

// TidbClusterSpec describes the attributes that a user creates on a tidb cluster
type TidbClusterSpec struct {
	PD              PDSpec              `json:"pd,omitempty"`
	TiDB            TiDBSpec            `json:"tidb,omitempty"`
	TiKV            TiKVSpec            `json:"tikv,omitempty"`
	TiKVPromGateway TiKVPromGatewaySpec `json:"tikvPromGateway,omitempty"`
	// Services list non-headless services type used in TidbCluster
	Services        []Service                            `json:"services,omitempty"`
	PVReclaimPolicy corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`
	Localtime       bool                                 `json:"localtime,omitempty"`
}

// TidbClusterStatus represents the current status of a tidb cluster.
type TidbClusterStatus struct {
	PD   PDStatus   `json:"pd,omitempty"`
	TiKV TiKVStatus `json:"tikv,omitempty"`
	TiDB TiDBStatus `json:"tidb,omitempty"`
}

// PDSpec contains details of PD member
type PDSpec struct {
	ContainerSpec
	Replicas             int32             `json:"replicas"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
	StorageClassName     string            `json:"storageClassName,omitempty"`
}

// TiDBSpec contains details of PD member
type TiDBSpec struct {
	ContainerSpec
	Replicas             int32             `json:"replicas"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
	StorageClassName     string            `json:"storageClassName,omitempty"`
}

// TiKVSpec contains details of PD member
type TiKVSpec struct {
	ContainerSpec
	Replicas             int32             `json:"replicas"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
	StorageClassName     string            `json:"storageClassName,omitempty"`
}

// TiKVPromGatewaySpec runs as a sidecar with TiKVSpec
type TiKVPromGatewaySpec struct {
	ContainerSpec
}

// ContainerSpec is the container spec of a pod
type ContainerSpec struct {
	Image    string               `json:"image"`
	Requests *ResourceRequirement `json:"requests,omitempty"`
	Limits   *ResourceRequirement `json:"limits,omitempty"`
}

// Service represent service type used in TidbCluster
type Service struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// ResourceRequirement is resource requirements for a pod
type ResourceRequirement struct {
	// CPU is how many cores a pod requires
	CPU string `json:"cpu,omitempty"`
	// Memory is how much memory a pod requires
	Memory string `json:"memory,omitempty"`
	// Storage is storage size a pod requires
	Storage string `json:"storage,omitempty"`
}

// PDStatus is PD status
type PDStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	Members     map[string]PDMember     `json:"members,omitempty"`
}

// PDMember is PD member
type PDMember struct {
	Name string `json:"name"`
	// member id is actually a uint64, but apimachinery's json only treats numbers as int64/float64
	// so uint64 may overflow int64 and thus convert to float64
	ID        string `json:"id"`
	ClientURL string `json:"clientURL"`
	Health    bool   `json:"health"`
}

// TiDBStatus is TiDB status
type TiDBStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	Members     map[string]TiDBMember   `json:"members,omitempty"`
}

// TiDBMember is TiDB member
type TiDBMember struct {
	IP string `json:"ip"`
}

// TiKVStatus is TiKV status
type TiKVStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
	Stores      TiKVStores              `json:"stores,omitempty"`
}

// TiKVStores is all TiKV Stores, contain stores which state are tombstone
type TiKVStores struct {
	CurrentStores   map[string]TiKVStore `json:"currentStores,omitempty"`
	TombStoneStores map[string]TiKVStore `json:"tombStoneStores,omitempty"`
}

// TiKVStores is either Up/Down/Offline/Tombstone
type TiKVStore struct {
	// store id is also uint64, due to the same reason as pd id, we store id as string
	ID                string      `json:"id"`
	PodName           string      `json:"podName"`
	IP                string      `json:"ip"`
	State             string      `json:"state"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
}
