// Copyright 2020 PingCAP, Inc.
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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TidbClusterAutoScaler is the control script's spec
type TidbClusterAutoScaler struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec describes the state of the TidbClusterAutoScaler
	Spec TidbClusterAutoScalerSpec `json:"spec"`

	// Status describe the status of the TidbClusterAutoScaler
	Status TidbClusterAutoSclaerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TidbClusterAutoScalerList is TidbClusterAutoScaler list
type TidbClusterAutoScalerList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbClusterAutoScaler `json:"items"`
}

// +k8s:openapi-gen=true
// TidbAutoScalerSpec describes the state of the TidbClusterAutoScaler
type TidbClusterAutoScalerSpec struct {
	// TidbClusterRef describe the target TidbCluster
	Cluster TidbClusterRef `json:"cluster"`

	// TiKV represents the auto-scaling spec for tikv
	// +optional
	TiKV *TikvAutoScalerSpec `json:"tikv,omitempty"`

	// TiDB represents the auto-scaling spec for tidb
	// +optional
	TiDB *TidbAutoScalerSpec `json:"tidb,omitempty"`
}

// +k8s:openapi-gen=true
// AutoResource describes the resource type definitions
type AutoResource struct {
	// CPU defines the CPU of this resource type
	CPU resource.Quantity `json:"cpu"`
	// Memory defines the memory of this resource type
	Memory resource.Quantity `json:"memory"`
	// Storage defines the storage of this resource type
	Storage resource.Quantity `json:"storage,omitempty"`
	// Count defines the max availabel count of this resource type
	Count *int32 `json:"count,omitempty"`
}

// +k8s:openapi-gen=true
// AutoRule describes the rules for auto-scaling with PD API
type AutoRule struct {
	// MaxThreshold defines the threshold to scale out
	MaxThreshold float64 `json:"max_threshold"`
	// MinThreshold defines the threshold to scale in, not applicable to `storage` rule
	MinThreshold *float64 `json:"min_threshold,omitempty"`
	// ResourceTypes defines the resource types that can be used for scaling
	ResourceTypes []string `json:"resource_types,omitempty"`
}

// +k8s:openapi-gen=true
// TikvAutoScalerSpec describes the spec for tikv auto-scaling
type TikvAutoScalerSpec struct {
	BasicAutoScalerSpec `json:",inline"`
}

// +k8s:openapi-gen=true
// TidbAutoScalerSpec describes the spec for tidb auto-scaling
type TidbAutoScalerSpec struct {
	BasicAutoScalerSpec `json:",inline"`
}

// +k8s:openapi-gen=true
// BasicAutoScalerSpec describes the basic spec for auto-scaling
type BasicAutoScalerSpec struct {
	// Rules defines the rules for auto-scaling with PD API
	Rules map[corev1.ResourceName]AutoRule `json:"rules,omitempty"`

	// External makes the auto-scaler controller able to query the external service
	// to fetch the recommended replicas for TiKV/TiDB
	// +optional
	External *ExternalConfig `json:"external,omitempty"`

	// ScaleInIntervalSeconds represents the duration seconds between each auto-scaling-in
	// If not set, the default ScaleInIntervalSeconds will be set to 500
	// +optional
	ScaleInIntervalSeconds *int32 `json:"scaleInIntervalSeconds,omitempty"`

	// ScaleOutIntervalSeconds represents the duration seconds between each auto-scaling-out
	// If not set, the default ScaleOutIntervalSeconds will be set to 300
	// +optional
	ScaleOutIntervalSeconds *int32 `json:"scaleOutIntervalSeconds,omitempty"`

	// Resources represent the resource type definitions that can be used for TiDB/TiKV
	// The key is resource_type name of the resource
	// +optional
	Resources map[string]AutoResource `json:"resources,omitempty"`
}

// +k8s:openapi-gen=true
// ExternalConfig represents the external config.
type ExternalConfig struct {
	// ExternalEndpoint makes the auto-scaler controller able to query the
	// external service to fetch the recommended replicas for TiKV/TiDB
	// +optional
	Endpoint ExternalEndpoint `json:"endpoint"`
	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale out.
	MaxReplicas int32 `json:"maxReplicas"`
}

// +k8s:openapi-gen=true
// TidbMonitorRef reference to a TidbMonitor
type TidbMonitorRef struct {
	// Namespace is the namespace that TidbMonitor object locates,
	// default to the same namespace with TidbClusterAutoScaler
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name is the name of TidbMonitor object
	Name string `json:"name"`

	// GrafanaEnabled indicate whether the grafana is enabled for this target tidbmonitor
	// +optional
	GrafanaEnabled bool `json:"grafanaEnabled,omitempty"`
}

// +k8s:openapi-gen=true
// TidbClusterAutoSclaerStatus describe the whole status
type TidbClusterAutoSclaerStatus struct {
	// Tikv describes the status for the tikv in the last auto-scaling reconciliation
	// +optional
	TiKV *TikvAutoScalerStatus `json:"tikv,omitempty"`
	// Tidb describes the status for the tidb in the last auto-scaling reconciliation
	// +optional
	TiDB *TidbAutoScalerStatus `json:"tidb,omitempty"`
}

// +k8s:openapi-gen=true
// TidbAutoScalerStatus describe the auto-scaling status of tidb
type TidbAutoScalerStatus struct {
	BasicAutoScalerStatus `json:",inline"`
}

// +k8s:openapi-gen=true
// TikvAutoScalerStatus describe the auto-scaling status of tikv
type TikvAutoScalerStatus struct {
	BasicAutoScalerStatus `json:",inline"`
}

// +k8s:openapi-gen=true
// BasicAutoScalerStatus describe the basic auto-scaling status
type BasicAutoScalerStatus struct {
	// LastAutoScalingTimestamp describes the last auto-scaling timestamp for the component(tidb/tikv)
	// +optional
	LastAutoScalingTimestamp *metav1.Time `json:"lastAutoScalingTimestamp,omitempty"`
}

// +k8s:openapi-gen=true
// ExternalEndpoint describes the external service endpoint
// which provides the ability to get the tikv/tidb auto-scaling recommended replicas
type ExternalEndpoint struct {
	// Host indicates the external service's host
	Host string `json:"host"`
	// Port indicates the external service's port
	Port int32 `json:"port"`
	// Path indicates the external service's path
	Path string `json:"path"`
	// TLSSecret indicates the Secret which stores the TLS configuration. If set, the operator will use https
	// to communicate to the external service
	// +optional
	TLSSecret *SecretRef `json:"tlsSecret,omitempty"`
}

// +k8s:openapi-gen=true
// SecretRef indicates to secret ref
type SecretRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// +k8s:openapi-gen=true
// TidbClusterAutoScalerRef indicates to the target auto-scaler ref
type TidbClusterAutoScalerRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
