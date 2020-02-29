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

	// We used prometheus to fetch the metrics resources until the pd could provide it.
	// MetricsUrl represents the url to fetch the metrics info
	// +optional
	MetricsUrl *string `json:"metricsUrl,omitempty"`

	// TidbMonitorRef describe the target TidbMonitor, when MetricsUrl and Monitor are both set,
	// Operator will use MetricsUrl
	// +optional
	Monitor *TidbMonitorRef `json:"monitor,omitempty"`

	// TiKV represents the auto-scaling spec for tikv
	// +optional
	TiKV *TikvAutoScalerSpec `json:"tikv,omitempty"`

	// TiDB represents the auto-scaling spec for tidb
	// +optional
	TiDB *TidbAutoScalerSpec `json:"tidb,omitempty"`
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
	MaxReplicas int32 `json:"maxReplicas"`

	// minReplicas is the lower limit for the number of replicas to which the autoscaler
	// can scale down.  It defaults to 1 pod. Scaling is active as long as at least one metric value is
	// available.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// ScaleInIntervalSeconds represents the duration seconds between each auto-scaling-in
	// If not set, the default ScaleInIntervalSeconds will be set to 500
	// +optional
	ScaleInIntervalSeconds *int32 `json:"scaleInIntervalSeconds,omitempty"`

	// ScaleOutIntervalSeconds represents the duration seconds between each auto-scaling-out
	// If not set, the default ScaleOutIntervalSeconds will be set to 300
	// +optional
	ScaleOutIntervalSeconds *int32 `json:"scaleOutIntervalSeconds,omitempty"`

	// metrics contains the specifications for which to use to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used).  The desired replica count is calculated multiplying the
	// ratio between the target value and the current value by the current
	// number of pods.  Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa.  See the individual metric source types for
	// more information about how each type of metric must respond.
	// If not set, the default metric will be set to 80% average CPU utilization.
	// +optional
	Metrics []v2beta2.MetricSpec `json:"metrics,omitempty"`

	// MetricsTimeDuration describe the Time duration to be queried in the Prometheus
	// +optional
	MetricsTimeDuration *string `json:"metricsTimeDuration,omitempty"`

	// ScaleOutThreshold describe the consecutive threshold for the auto-scaling,
	// if the consecutive counts of the scale-out result in auto-scaling reach this number,
	// the auto-scaling would be performed.
	// If not set, the default value is 3.
	// +optional
	ScaleOutThreshold *int32 `json:"scaleOutThreshold,omitempty"`

	// ScaleInThreshold describe the consecutive threshold for the auto-scaling,
	// if the consecutive counts of the scale-in result in auto-scaling reach this number,
	// the auto-scaling would be performed.
	// If not set, the default value is 5.
	// +optional
	ScaleInThreshold *int32 `json:"scaleInThreshold,omitempty"`
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
}

// TODO: sync status
type TidbClusterAutoSclaerStatus struct {
}
