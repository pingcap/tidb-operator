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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=br
// +kubebuilder:resource:shortName="tibrgc"
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Strategy",type=string,JSONPath=`.spec.gcStrategy.type`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiBRGC is the Schema for the tibrgc API. It allows users to set backup gc strategy and configure resources fo gc workloads.
type TiBRGC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiBRGCSpec   `json:"spec,omitempty"`
	Status TiBRGCStatus `json:"status,omitempty"`
}

// TiBRGCSpec defines the desired state of TiBRGC.
type TiBRGCSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster    v1alpha1.ClusterReference `json:"cluster"`
	GCStrategy TiBRGCStrategy            `json:"gcStrategy,omitempty"`
	// Image is image of br gc, default is pingcap/tikv
	Image   *string       `json:"image,omitempty"`
	Overlay TiBRGCOverlay `json:"overlay,omitempty"`
}

// TiBRGCStatus defines the observed state of TiBRGC.
type TiBRGCStatus struct {
	v1alpha1.CommonStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiBRGCList contains a list of TiBRGC.
type TiBRGCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TiBRGC `json:"items"`
}

type TiBRGCStrategyType string

const (
	TiBRGCStrategyTypeTieredStorage TiBRGCStrategyType = "tiered-storage"
)

type TiBRGCStrategy struct {
	// +kubebuilder:validation:Enum=tiered-storage
	Type TiBRGCStrategyType `json:"type"`
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=2
	TieredStrategies []TieredStorageStrategy `json:"tieredStrategies,omitempty"`
}
type TieredStorageStrategyName string

const (
	TieredStorageStrategyNameToT2Storage TieredStorageStrategyName = "to-t2-storage"
	TieredStorageStrategyNameToT3Storage TieredStorageStrategyName = "to-t3-storage"
)

type TieredStorageStrategy struct {
	// +kubebuilder:validation:Enum=to-t2-storage;to-t3-storage
	Name TieredStorageStrategyName `json:"name"`
	// +kubebuilder:validation:Minimum=1
	TimeThresholdDays uint32 `json:"timeThresholdDays"`
}

type TiBRGCOverlay struct {
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=2
	Pods []TiBRGCPodOverlay `json:"pods,omitempty"`
}

type TiBRGCPodOverlay struct {
	// +kubebuilder:validation:Enum=to-t2-storage;to-t3-storage
	Name    TieredStorageStrategyName `json:"name"`
	Overlay *v1alpha1.Overlay         `json:"overlay,omitempty"`
}
