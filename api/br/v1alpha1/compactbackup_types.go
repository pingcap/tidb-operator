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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=br
// +kubebuilder:resource:shortName="cpbk"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`,description="The current status of the compact backup"
// +kubebuilder:printcolumn:name="Progress",type=string,JSONPath=`.status.progress`,description="The progress of the compact backup"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,description="The message of the compact backup"
type CompactBackup struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	Spec CompactSpec `json:"spec"`
	// +k8s:openapi-gen=false
	Status CompactStatus `json:"status,omitempty"`
}

// CompactSpec contains the backup specification for a tidb cluster.
// +k8s:openapi-gen=true
type CompactSpec struct {
	corev1.ResourceRequirements `json:"resources,omitempty"`
	// List of environment variables to set in the container, like v1.Container.Env.
	// Note that the following builtin env vars will be overwritten by values set here
	// - S3_PROVIDER
	// - S3_ENDPOINT
	// - AWS_REGION
	// - AWS_ACL
	// - AWS_STORAGE_CLASS
	// - AWS_DEFAULT_REGION
	// - AWS_ACCESS_KEY_ID
	// - AWS_SECRET_ACCESS_KEY
	// - GCS_PROJECT_ID
	// - GCS_OBJECT_ACL
	// - GCS_BUCKET_ACL
	// - GCS_LOCATION
	// - GCS_STORAGE_CLASS
	// - GCS_SERVICE_ACCOUNT_JSON_KEY
	// - BR_LOG_TO_TERM
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// StorageProvider configures where and how backups should be stored.
	// *** Note: This field should generally not be left empty, unless you are certain the storage provider
	// *** can be obtained from another source, such as a schedule CR.
	StorageProvider `json:",inline"`
	// StartTs is the start ts of the compact backup.
	// Format supports TSO or datetime, e.g. '400036290571534337', '2018-05-11 01:42:23'.
	StartTs string `json:"startTs,omitempty"`
	// EndTs is the end ts of the compact backup.
	// Format supports TSO or datetime, e.g. '400036290571534337', '2018-05-11 01:42:23'.
	// Default is current timestamp.
	// +optional
	EndTs string `json:"endTs,omitempty"`
	// Concurrency is the concurrency of compact backup job
	// +kubebuilder:default=4
	Concurrency int `json:"concurrency,omitempty"`
	// Base tolerations of backup Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// BrImage specifies the br image used in compact `Backup`.
	// For examples `spec.brImage: pingcap/br:v4.0.8`
	// For BR image, if it does not contain tag, Pod will use image 'BrImage:${TiKV_Version}'.
	// +optional
	ToolImage string `json:"toolImage,omitempty"`
	// BRConfig is the configs for BR
	// *** Note: This field should generally not be left empty, unless you are certain the BR config
	// *** can be obtained from another source, such as a schedule CR.
	BR *BRConfig `json:"br,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Affinity of backup Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Use KMS to decrypt the secrets
	UseKMS bool `json:"useKMS,omitempty"`
	// Specify service account of backup
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// PriorityClassName of Backup Job Pods
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// BackoffRetryPolicy the backoff retry policy, currently only valid for snapshot backup
	// +kubebuilder:default=6
	MaxRetryTimes int32 `json:"maxRetryTimes,omitempty"`

	// Additional volumes of component pod.
	// +optional
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`
	// Additional volume mounts of component pod.
	// +optional
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
}

// CompactRetryRecord is the record of compact backoff retry
type CompactRetryRecord struct {
	// RetryNum is the number of retry
	RetryNum int `json:"retryNum,omitempty"`
	// DetectFailedAt is the time when detect failure
	DetectFailedAt metav1.Time `json:"detectFailedAt,omitempty"`
	// Reason is the reason of retry
	RetryReason string `json:"retryReason,omitempty"`
}

type CompactStatus struct {
	// State is the current state of the backup
	State string `json:"state,omitempty"`
	// Progress is the progress of the backup
	Progress string `json:"progress,omitempty"`
	// Message is the error message of the backup
	Message string `json:"message,omitempty"`
	// RetryStatus is status of the backoff retry, it will be used when backup pod or job exited unexpectedly
	RetryStatus []CompactRetryRecord `json:"backoffRetryStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// CompactList contains a list of Compact Backup.
type CompactBackupList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []CompactBackup `json:"items"`
}
