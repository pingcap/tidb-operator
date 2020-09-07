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
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AutoScalerPhase string

const (
	NormalAutoScalerPhase          AutoScalerPhase = "Normal"
	ReadyToScaleOutAutoScalerPhase AutoScalerPhase = "ReadyToScaleOut"
	ReadyToScaleInAutoScalerPhase  AutoScalerPhase = "ReadyToScaleIn"
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

	// We used prometheus to fetch the metrics resources until the pd could provide it.
	// MetricsUrl represents the url to fetch the metrics info
	// Deprecated
	// +optional
	MetricsUrl *string `json:"metricsUrl,omitempty"`

	// TidbMonitorRef describe the target TidbMonitor, when MetricsUrl and Monitor are both set,
	// Operator will use MetricsUrl
	// Deprecated
	// +optional
	Monitor *TidbMonitorRef `json:"monitor,omitempty"`

	// TiKV represents the auto-scaling spec for tikv
	// +optional
	TiKV *TikvAutoScalerSpec `json:"tikv,omitempty"`

	// TiDB represents the auto-scaling spec for tidb
	// +optional
	TiDB *TidbAutoScalerSpec `json:"tidb,omitempty"`

	// Resources represent the resource type definitions that can be used for TiDB/TiKV
	// +optional
	Resources []AutoResource `json:"resources,omitempty"`
}

// +k8s:openapi-gen=true
// AutoResource describes the resource type definitions
type AutoResource struct {
	// ResourceType identifies a specific resource type
	ResourceType string `json:"resource_type,omitempty"`
	// CPU defines the CPU of this resource type
	CPU resource.Quantity `json:"cpu,omitempty"`
	// Memory defines the memory of this resource type
	Memory resource.Quantity `json:"memory,omitempty"`
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
	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale out.
	// It cannot be less than minReplicas.
	// Deprecated
	MaxReplicas int32 `json:"maxReplicas"`

	// minReplicas is the lower limit for the number of replicas to which the autoscaler
	// can scale down.  It defaults to 1 pod. Scaling is active as long as at least one metric value is
	// available.
	// Deprecated
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// Rules defines the rules for auto-scaling with PD API
	Rules map[corev1.ResourceName]AutoRule `json:"rules,omitempty"`

	// ScaleInIntervalSeconds represents the duration seconds between each auto-scaling-in
	// If not set, the default ScaleInIntervalSeconds will be set to 500
	// +optional
	ScaleInIntervalSeconds *int32 `json:"scaleInIntervalSeconds,omitempty"`

	// ScaleOutIntervalSeconds represents the duration seconds between each auto-scaling-out
	// If not set, the default ScaleOutIntervalSeconds will be set to 300
	// +optional
	ScaleOutIntervalSeconds *int32 `json:"scaleOutIntervalSeconds,omitempty"`

	// Deprecated
	// +optional
	Metrics []CustomMetric `json:"metrics,omitempty"`

	// MetricsTimeDuration describes the Time duration to be queried in the Prometheus
	// Deprecated
	// +optional
	MetricsTimeDuration *string `json:"metricsTimeDuration,omitempty"`
	// External makes the auto-scaler controller able to query the external service
	// to fetch the recommended replicas for TiKV/TiDB
	// +optional
	External *ExternalConfig `json:"external,omitempty"`
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

// Deprecated
type CustomMetric struct {
	// metrics contains the specifications for which to use to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used).  The desired replica count is calculated multiplying the
	// ratio between the target value and the current value by the current
	// number of pods.  Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa.  See the individual metric source types for
	// more information about how each type of metric must respond.
	// If not set, the auto-scaling won't happen.
	// +optional
	v2beta2.MetricSpec `json:",inline"`
	// LeastStoragePressurePeriodSeconds is only for the storage auto-scaling case when the resource name in the metricSpec
	// is `Storage`. When the Storage metrics meet the pressure, Operator would wait
	// LeastStoragePressurePeriodSeconds duration then able to scale out.
	// If not set, the default value is `300`
	// +optional
	LeastStoragePressurePeriodSeconds *int64 `json:"leastStoragePressurePeriodSeconds,omitempty"`
	// LeastRemainAvailableStoragePercent indicates the least remaining available storage percent compare to
	// the capacity storage. If the available storage is lower than the capacity storage * LeastRemainAvailableStoragePercent,
	// the storage status will become storage pressure and ready to be scaled out.
	// LeastRemainAvailableStoragePercent should between 5 and 90. If not set, the default value would be 10
	// +optional
	LeastRemainAvailableStoragePercent *int64 `json:"leastRemainAvailableStoragePercent,omitempty"`
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
	// MetricsStatusList describes the metrics status in the last auto-scaling reconciliation
	// +optional
	MetricsStatusList []MetricsStatus `json:"metrics,omitempty"`
	// CurrentReplicas describes the current replicas for the component(tidb/tikv)
	CurrentReplicas int32 `json:"currentReplicas"`
	// RecommendedReplicas describes the calculated replicas in the last auto-scaling reconciliation for the component(tidb/tikv)
	// +optional
	RecommendedReplicas int32 `json:"recommendedReplicas,omitempty"`
	// LastAutoScalingTimestamp describes the last auto-scaling timestamp for the component(tidb/tikv)
	// +optional
	LastAutoScalingTimestamp *metav1.Time `json:"lastAutoScalingTimestamp,omitempty"`
}

// +k8s:openapi-gen=true
// MetricsStatus describe the basic metrics status in the last auto-scaling reconciliation
type MetricsStatus struct {
	// Name indicates the metrics name
	Name string `json:"name"`
	// CurrentValue indicates the value calculated in the last auto-scaling reconciliation
	// +optional
	CurrentValue *string `json:"currentValue,omitempty"`
	// TargetValue indicates the threshold value for this metrics in auto-scaling
	// +optional
	ThresholdValue *string `json:"thresholdValue,omitempty"`
	// +optional
	StorageMetricsStatus `json:",inline"`
}

// +k8s:openapi-gen=true
// StorageMetricsStatus describe the storage metrics status in the last auto-scaling reconciliation
type StorageMetricsStatus struct {
	// StoragePressure indicates whether storage under pressure
	// +optional
	StoragePressure *bool `json:"storagePressure,omitempty"`
	// StoragePressureStartTime indicates the timestamp of the StoragePressure fist become true from false or nil
	// +optional
	StoragePressureStartTime *metav1.Time `json:"storagePressureStartTime,omitempty"`
	// +optional
	AvailableStorage *string `json:"availableStorage,omitempty"`
	// +optional
	CapacityStorage *string `json:"capacityStorage,omitempty"`
	// BaselineAvailableStorage indicates the baseline for available storage size.
	// This is calculated by the capacity storage size * storage auto-scaling baseline percent value
	// If the AvailableStorage is less than the BaselineAvailableStorage, the database is under StoragePressure
	// optional
	BaselineAvailableStorage *string `json:"baselineAvailableStorage,omitempty"`
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
