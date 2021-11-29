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
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TidbMonitor contains the spec and status of tidb ng monitor
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="tnm"
type TiDBNGMonitor struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec contains all spec about tidb ng monitor
	Spec TiDBNGMonitorSpec `json:"spec"`

	// Status is most recently observed status of tidb ng monitor
	//
	// +k8s:openapi-gen=false
	Status TiDBNGMonitorStatus `json:"status,omitempty"`
}

type TiDBNGMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TiDBNGMonitor `json:"items"`
}

// +k8s:openapi-gen=true
type TiDBNGMonitorSpec struct {
	// ComponentSpec is common spec.
	// NOTE: the same field will be overridden by component's spec.
	ComponentSpec `json:",inline"`

	// Clusters reference TiDB cluster
	Clusters []TidbClusterRef `json:"clusters"`
}

type TiDBNGMonitorStatus struct {
	Monitor MonitorStatus `json:"monitor,omitempty"`
}

// MonitorSpec is spec of ng monitor
//
// +k8s:openapi-gen=true
type MonitorSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	//
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	Replicas int32 `json:"replicas"`

	// Base image of the component, image tag is now allowed during validation
	//
	// +kubebuilder:default=pingcap/ng-monitor
	BaseImage string `json:"baseImage"`

	// StorageClassName is the persistent volume for ng monitor.
	// Defaults to Kubernetes default storage class.
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// MonitorStatus is latest status of ng monitor
type MonitorStatus struct {
	Synced bool        `json:"synced,omitempty"`
	Phase  MemberPhase `json:"phase,omitempty"`

	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}
