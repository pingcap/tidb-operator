// Copyright 2023 PingCAP, Inc.
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: add `kubebuilder:printcolumn` after fileds are defined

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackupFederation is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="vbkf"
// +genclient:noStatus
type VolumeBackupFederation struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec VolumeBackupFederationSpec `json:"spec"`

	// +k8s:openapi-gen=false
	Status VolumeBackupFederationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackupFederationList is VolumeBackupFederation list
// +k8s:openapi-gen=true
type VolumeBackupFederationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeBackupFederation `json:"items"`
}

// VolumeBackupFederationSpec describes the attributes that a user creates on a volume backup federation.
// +k8s:openapi-gen=true
type VolumeBackupFederationSpec struct {
}

// VolumeBackupFederationStatus represents the current status of a volume backup federation.
type VolumeBackupFederationStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackupScheduleFederation is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="vbksf"
// +genclient:noStatus
type VolumeBackupScheduleFederation struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec VolumeBackupScheduleFederationSpec `json:"spec"`

	// +k8s:openapi-gen=false
	Status VolumeBackupScheduleFederationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeBackupScheduleFederationList is VolumeBackupScheduleFederation list
// +k8s:openapi-gen=true
type VolumeBackupScheduleFederationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeBackupScheduleFederation `json:"items"`
}

// VolumeBackupScheduleFederationSpec describes the attributes that a user creates on a volume backup schedule federation.
// +k8s:openapi-gen=true
type VolumeBackupScheduleFederationSpec struct {
}

// VolumeBackupScheduleFederationStatus represents the current status of a volume backup schedule federation.
type VolumeBackupScheduleFederationStatus struct {
}


// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeRestoreFederation is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="vrtf"
// +genclient:noStatus
type VolumeRestoreFederation struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec VolumeRestoreFederationSpec `json:"spec"`

	// +k8s:openapi-gen=false
	Status VolumeRestoreFederationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeRestoreFederationList is VolumeRestoreFederation list
// +k8s:openapi-gen=true
type VolumeRestoreFederationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeRestoreFederation `json:"items"`
}

// VolumeRestoreFederationSpec describes the attributes that a user creates on a volume restore federation.
// +k8s:openapi-gen=true
type VolumeRestoreFederationSpec struct {
}

// VolumeRestoreFederationStatus represents the current status of a volume restore federation.
type VolumeRestoreFederationStatus struct {
}
