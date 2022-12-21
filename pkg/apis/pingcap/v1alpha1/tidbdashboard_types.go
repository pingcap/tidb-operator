// Copyright 2022 PingCAP, Inc.
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

// TidbDashboard contains the spec and status of tidb dashboard.
//
// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName="td"
// +kubebuilder:subresource:status
type TidbDashboard struct {
	metav1.TypeMeta `json:",inline"`

	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec contains all spec about tidb dashboard.
	Spec TidbDashboardSpec `json:"spec"`

	// Status is most recently observed status of tidb dashboard.
	//
	// +k8s:openapi-gen=false
	Status TidbDashboardStatus `json:"status,omitempty"`
}

// TidbDashboardList is a TidbDashboard list.
//
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TidbDashboardList struct {
	metav1.TypeMeta `json:",inline"`

	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbDashboard `json:"items"`
}

// TidbDashboardSpec is spec of tidb dashboard.
//
// +k8s:openapi-gen=true
type TidbDashboardSpec struct {
	// ComponentSpec is common spec.
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Clusters reference TiDB cluster.
	//
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:MinItems=1
	Clusters []TidbClusterRef `json:"clusters"`

	// Persistent volume reclaim policy applied to the PVs that consumed by tidb dashboard.
	//
	// +kubebuilder:default=Retain
	PVReclaimPolicy *corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// Base image of the component (image tag is now allowed during validation).
	//
	// +kubebuilder:default=pingcap/tidb-dashboard
	BaseImage string `json:"baseImage,omitempty"`

	// StorageClassName is the default PVC storage class for tidb dashboard.
	// Defaults to Kubernetes default storage class.
	StorageClassName *string `json:"storageClassName,omitempty"`

	// StorageVolumes configures additional PVC for tidb dashboard.
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// PathPrefix is public URL path prefix for reverse proxies.
	PathPrefix *string `json:"pathPrefix,omitempty"`

	// Service defines a Kubernetes service of tidb dashboard web access.
	Service ServiceSpec `json:"service,omitempty"`

	// Telemetry is whether to enable telemetry.
	// When enabled, usage data will be sent to PingCAP for improving user experience.
	// Optional: Defaults to true
	// +optional
	Telemetry *bool `json:"telemetry,omitempty" default:"true"`

	// Experimental is whether to enable experimental features.
	// When enabled, experimental TiDB Dashboard features will be available.
	// These features are incomplete or not well tested. Suggest not to enable in
	// production.
	// Optional: Defaults to false
	// +optional
	Experimental *bool `json:"experimental,omitempty"`
}

// TidbDashboardStatus is status of tidb dashboard.
type TidbDashboardStatus struct {
	Synced bool        `json:"synced,omitempty"`
	Phase  MemberPhase `json:"phase,omitempty"`

	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}
