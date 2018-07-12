// Copyright 2017 PingCAP, Inc.
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

package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbCluster is the control script's spec
type TidbCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the behavior of a tidb cluster
	Spec ClusterSpec `json:"spec"`

	// Most recently observed status of the tidb cluster
	Status ClusterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbClusterList is TidbCluster list
type TidbClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TidbCluster `json:"items"`
}

// ClusterSpec describes the attributes that a user creates on a tidb cluster
type ClusterSpec struct {
	PD   PDSpec   `json:"pd,omitempty"`
	TiDB TiDBSpec `json:"tidb,omitempty"`
	TiKV TiKVSpec `json:"tikv,omitempty"`
	// Service type for tidb: ClusterIP | NodePort | LoadBalancer, default: ClusterIP
	// Service is deprecated in favor of Services field
	Service string `json:"service,omitempty"`
	// Config is deprecated in favor of ConfigMap field
	Config map[string]string `json:"config,omitempty"`
	// Monitor can be nil to disable monitor
	// if user want to deploy monitor outside of tidb-operator
	Monitor *MonitorSpec `json:"monitor,omitempty"`
	// PrivilegedTiDB is used for database management on cloud without password
	// this is useful if user forget password or backup database etc
	// this can be disabled if it's nil
	PrivilegedTiDB *PrivilegedTiDBSpec `json:"privilegedTidb,omitempty"`
	// Services list non-headless services type used in TidbCluster
	Services []Service `json:"services,omitempty"`
	// ConfigMap is the ConfigMap name of tidb-cluster config
	ConfigMap string `json:"configMap,omitempty"`
	// Paused represents cluster is paused
	Paused bool `json:"paused,omitempty"`
	// State represents desired cluster state
	State ClusterState `json:"state,omitempty"`
	// RetentionDuration represents the duration this object and all its PVs will be retention after we gracefully delete the cluster
	RetentionDuration string `json:"retentionDuration,omitempty"`
}

// ClusterStatus represents the current status of a tidb cluster.
type ClusterStatus struct {
	PDStatus   PDStatus   `json:"pdStatus,omitempty"`
	TiKVStatus TiKVStatus `json:"tikvStatus,omitempty"`
	TiDBStatus TiDBStatus `json:"tidbStatus,omitempty"`
}

// NormalStatus represents the status of normal
type NormalStatus struct {
	ReallyNormal bool `json:"reallyNormal,omitempty"`
}

// GracefulDeletedStatus represents the status of graceful deleted
type GracefulDeletedStatus struct {
	ReallyGracefulDeleted bool `json:"reallyGracefulDeleted,omitempty"`

	// RetentionTime will be set to now + RetentionDuration if graceful deleted
	RetentionTime *metav1.Time `json:"retentionTime,omitempty"`
}

// RestoreStatus represents the status of restoring
type RestoreStatus struct {
	ReallyRestoring bool         `json:"reallyRestoring,omitempty"`
	BeginTime       *metav1.Time `json:"beginTime,omitempty"`
	EndTime         *metav1.Time `json:"endTime,omitempty"`
}

// ClusterState is a state of TidbCluster
type ClusterState string

const (
	// StateNormal represents normal cluster state
	// Can't be changed to other states manually by user
	StateNormal ClusterState = ""

	// StateGracefulDeleted represents graceful deleted cluster state
	// In this state, tidb-operator will delete all resources belonging to this cluster except this object and all its PD/TiKV PVs
	// This state can only be changed from ReallyNormal() manually by user
	StateGracefulDeleted = "GracefulDeleted"

	// StateRestore represents restore cluster state
	// This state can only be changed from ReallyGracefulDeleted() by user, or be changed to StateNormal by tidb-operator
	StateRestore = "Restore"
)

// PDSpec contains details of PD member
type PDSpec struct {
	ContainerSpec
	Size                 int32             `json:"size"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
}

// TiDBSpec contains details of PD member
type TiDBSpec struct {
	ContainerSpec
	Size                 int32             `json:"size"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
	Binlog               *ContainerSpec    `json:"binlog,omitempty"`
}

// TiKVSpec contains details of PD member
type TiKVSpec struct {
	ContainerSpec
	Size                 int32             `json:"size"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
}

// PrivilegedTiDBSpec is used for database management on cloud without password
// this is useful if user forget password or backup database etc
// this can be disabled if it's nil
type PrivilegedTiDBSpec struct {
	ContainerSpec
	Size                 int32             `json:"size"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
}

// MonitorSpec is the monitor component of TidbCluster
type MonitorSpec struct {
	Prometheus    ContainerSpec `json:"prometheus,omitempty"`
	RetentionDays int32         `json:"retentionDays,omitempty"`
	// Grafana can be nil to disable grafana
	Grafana              *ContainerSpec    `json:"grafana,omitempty"`
	DashboardInstaller   *ContainerSpec    `json:"dashboardInstaller,omitempty"`
	NodeSelector         map[string]string `json:"nodeSelector,omitempty"`
	NodeSelectorRequired bool              `json:"nodeSelectorRequired,omitempty"`
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

const (
	// AnnotationStorageSize is a storage size annotation key
	AnnotationStorageSize string = "storage.pingcap.com/size"

	// TiDBVolumeName is volume name for TiDB volume
	TiDBVolumeName string = "tidb-volume-hostpath"

	// TidbSetResourcePlural is the name plural of TidbSet CRD
	TidbSetResourcePlural string = "tidbsets"

	// TidbReviewResourcePlural is the name plural of TidbReview CRD
	TidbReviewResourcePlural string = "tidbreviews"

	// TiKVStateUp represents status of Up of TiKV
	TiKVStateUp string = "Up"
)

// ContainerType represents container type
type ContainerType string

const (
	// PDContainerType is pd container type
	PDContainerType ContainerType = "pd"

	// TiDBContainerType is tidb container type
	TiDBContainerType ContainerType = "tidb"

	// BinlogContainerType is tidb binlog container type
	BinlogContainerType ContainerType = "tidb-binlog"

	// TiKVContainerType is tikv container type
	TiKVContainerType ContainerType = "tikv"

	//PushGatewayContainerType is pushgateway container type
	PushGatewayContainerType ContainerType = "pushgateway"

	// UnknownContainerType is unknown container type
	UnknownContainerType ContainerType = "unknown"
)

func (ct ContainerType) String() string {
	return string(ct)
}

// PDStatus is PD status
type PDStatus struct {
	StatefulSet        PDStatefulSetStatus `json:"statefulSet,omitempty"`
	Members            map[string]PDMember `json:"members,omitempty"`
	Upgrading          bool                `json:"upgrading,omitempty"`
	LastTransitionTime metav1.Time         `json:"lastTransitionTime,omitempty"`
}

// PDStatefulSetStatus is the status of pd's statefulset
type PDStatefulSetStatus struct {
	Name string `json:"name,omitempty"`
}

// PDMember is PD member
type PDMember struct {
	Name string `json:"name"`
	// member id is actually a uint64, but apimachinery's json only treats numbers as int64/float64
	// so uint64 may overflow int64 and thus convert to float64
	ID string `json:"id"`
	IP string `json:"ip"`
}

// TiDBStatus is TiDB status
type TiDBStatus struct {
	Members            map[string]TiDBMember `json:"members,omitempty"`
	Upgrading          bool                  `json:"upgrading,omitempty"`
	LastTransitionTime metav1.Time           `json:"lastTransitionTime,omitempty"`
}

// TiDBMember is TiDB member
type TiDBMember struct {
	IP string `json:"ip"`
}

// TiKVStatus is TiKV status
type TiKVStatus struct {
	Stores             map[string]TiKVStores `json:"stores,omitempty"`
	Upgrading          bool                  `json:"upgrading,omitempty"`
	LastTransitionTime metav1.Time           `json:"lastTransitionTime,omitempty"`
}

// TiKVStores is either Up/Down/Offline, namely it's in-cluster status
// when status changed from Offline to Tombstone, we delete it
type TiKVStores struct {
	// store id is also uint64, due to the same reason as pd id, we store id as string
	ID                string      `json:"id"`
	IP                string      `json:"ip"`
	State             string      `json:"state"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
}

// GetConfigMapName returns the name of configmap used for TidbCluster
func (tc *TidbCluster) GetConfigMapName() string {
	if tc.Spec.ConfigMap != "" {
		return tc.Spec.ConfigMap
	}
	// if configMap is empty, use a managed configmap
	return fmt.Sprintf("%s-config", tc.GetName())
}
