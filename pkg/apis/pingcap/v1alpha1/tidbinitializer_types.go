// Copyright 2019 PingCAP, Inc.
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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InitializePhase string

const (
	// InitializePhasePending indicates that the initialization is still pending waiting the cluster to appear
	InitializePhasePending InitializePhase = "Pending"
	// InitializePhaseRunning indicates the the initialization is in progress
	InitializePhaseRunning InitializePhase = "Running"
	// InitializePhaseCompleted indicates the initialization is completed,
	// that is, the target tidb-cluster is fully initialized
	InitializePhaseCompleted InitializePhase = "Completed"
	// InitializePhaseFailed indicates the initialization is failed and need manual intervention
	InitializePhaseFailed InitializePhase = "Failed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TidbInitializer is a TiDB cluster initializing job
type TidbInitializer struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the desired state of TidbInitializer
	Spec TidbInitializerSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the TidbInitializer
	Status TidbInitializerStatus `json:"status"`
}

// +k8s:openapi-gen=true
// TidbInitializer spec encode the desired state of tidb initializer Job
type TidbInitializerSpec struct {
	Image string `json:"image"`

	Clusters TidbClusterRef `json:"cluster"`

	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// permitHost is the host which will only be allowed to connect to the TiDB.
	// +optional
	PermitHost *string `json:"permitHost,omitempty"`

	// InitSql is the SQL statements executed after the TiDB cluster is bootstrapped.
	// +optional
	InitSql *string `json:"initSql,omitempty"`

	// InitSqlConfigMapName reference a configmap that provide init-sql, take high precedence than initSql if set
	// +optional
	InitSqlConfigMap *string `json:"initSqlConfigMap,omitempty"`

	// +optional
	PasswordSecret *string `json:"passwordSecret,omitempty"`

	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Time zone of TiDB initializer Pods
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// TLSClientSecretName is the name of secret which stores tidb server client certificate
	// Optional: Defaults to nil
	// +optional
	TLSClientSecretName *string `json:"tlsClientSecretName,omitempty"`
}

// +k8s:openapi-gen=true
type TidbInitializerStatus struct {
	batchv1.JobStatus `json:",inline"`

	// Phase is a user readable state inferred from the underlying Job status and TidbCluster status
	Phase InitializePhase `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
type TidbInitializerList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbInitializer `json:"items"`
}
