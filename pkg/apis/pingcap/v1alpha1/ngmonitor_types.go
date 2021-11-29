// Copyright 2021 PingCAP, Inc.
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
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbMonitor contains the spec and status of tidb ng monitor
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="tnm"
type TiDBNGMonitoring struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec contains all spec about tidb ng monitor
	Spec TiDBNGMonitoringSpec `json:"spec"`

	// Status is most recently observed status of tidb ng monitor
	//
	// +k8s:openapi-gen=false
	Status TiDBNGMonitoringStatus `json:"status,omitempty"`
}

type TiDBNGMonitoringList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TiDBNGMonitoring `json:"items"`
}

// TiDBNGMonitoringSpec is spec of tidb ng monitoring
//
// +k8s:openapi-gen=true
type TiDBNGMonitoringSpec struct {
	// ComponentSpec is common spec.
	// NOTE: the same field will be overridden by component's spec.
	ComponentSpec `json:",inline"`

	// Clusters reference TiDB cluster
	//
	// +kubebuilder:validation:MaxItems=1
	Clusters []TidbClusterRef `json:"clusters"`

	// Paused pause controller if it is true
	Paused bool `json:"paused,omitempty"`
}

// TiDBNGMonitoringStatus is status of tidb ng monitoring
type TiDBNGMonitoringStatus struct {
	// NGMonitoring is status of ng monitoring
	NGMonitoring NGMonitoringStatus `json:"ngMonitoring,omitempty"`
}

// NGMonitoringSpec is spec of ng monitoring
//
// +k8s:openapi-gen=true
type NGMonitoringSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Base image of the component, image tag is now allowed during validation
	//
	// +kubebuilder:default=pingcap/ng-monitoring
	BaseImage string `json:"baseImage"`

	// StorageClassName is the persistent volume for ng monitoring.
	// Defaults to Kubernetes default storage class.
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// NGMonitoringStatus is latest status of ng monitoring
type NGMonitoringStatus struct {
	Synced bool        `json:"synced,omitempty"`
	Phase  MemberPhase `json:"phase,omitempty"`

	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}
