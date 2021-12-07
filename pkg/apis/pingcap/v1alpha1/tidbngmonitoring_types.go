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
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TidbNGMonitoring contains the spec and status of tidb ng monitor
//
// +genclient
// +genclient:noStatus
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName="tngm"
type TidbNGMonitoring struct {
	metav1.TypeMeta `json:",inline"`

	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec contains all spec about tidb ng monitor
	Spec TidbNGMonitoringSpec `json:"spec"`

	// Status is most recently observed status of tidb ng monitor
	//
	// +k8s:openapi-gen=false
	Status TidbNGMonitoringStatus `json:"status,omitempty"`
}

// TiDBNGMonitoringList is TidbNGMonitoring list
//
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TidbNGMonitoringList struct {
	metav1.TypeMeta `json:",inline"`

	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbNGMonitoring `json:"items"`
}

// TidbNGMonitoringSpec is spec of tidb ng monitoring
//
// +k8s:openapi-gen=true
type TidbNGMonitoringSpec struct {
	// ComponentSpec is common spec.
	// NOTE: the same field will be overridden by component's spec.
	ComponentSpec `json:",inline"`

	// Clusters reference TiDB cluster
	//
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:MinItems=1
	Clusters []TidbClusterRef `json:"clusters"`

	// Paused pause controller if it is true
	Paused bool `json:"paused,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by tidb ng monitoring
	//
	// +kubebuilder:default=Retain
	PVReclaimPolicy *corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// ClusterDomain is the Kubernetes Cluster Domain of tidb ng monitoring
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// NGMonitoring is spec of ng monitoring
	NGMonitoring NGMonitoringSpec `json:"ngMonitoring"`
}

// TidbNGMonitoringStatus is status of tidb ng monitoring
type TidbNGMonitoringStatus struct {
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
	BaseImage string `json:"baseImage,omitempty"`

	// StorageClassName is the persistent volume for ng monitoring.
	// Defaults to Kubernetes default storage class.
	StorageClassName *string `json:"storageClassName,omitempty"`

	// StorageVolumes configures additional storage for NG Monitoring pods.
	StorageVolumes []StorageVolume `json:"storageVolumes,omitempty"`

	// Config is the configuration of ng monitoring
	//
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:XPreserveUnknownFields
	Config *config.GenericConfig `json:"config,omitempty"`
}

// NGMonitoringStatus is latest status of ng monitoring
type NGMonitoringStatus struct {
	Synced bool        `json:"synced,omitempty"`
	Phase  MemberPhase `json:"phase,omitempty"`

	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}
