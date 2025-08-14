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

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ClusterList defines a list of TiDB clusters
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PD",type="integer",JSONPath=".status.components[?(@.kind==\"PD\")].replicas"
// +kubebuilder:printcolumn:name="TiKV",type="integer",JSONPath=".status.components[?(@.kind==\"TiKV\")].replicas"
// +kubebuilder:printcolumn:name="TiDB",type="integer",JSONPath=".status.components[?(@.kind==\"TiDB\")].replicas"
// +kubebuilder:printcolumn:name="TiFlash",type="integer",JSONPath=".status.components[?(@.kind==\"TiFlash\")].replicas"
// +kubebuilder:printcolumn:name="TiProxy",type="integer",JSONPath=".status.components[?(@.kind==\"TiProxy\")].replicas"
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Cluster defines a TiDB cluster
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 37",message="name must not exceed 37 characters"
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="oldSelf == null || !has(self.bootstrapSQL) || (has(oldSelf.bootstrapSQL) && self.bootstrapSQL.name == oldSelf.bootstrapSQL.name)",message="bootstrapSQL can only be set at creation, can be unset, but cannot be changed to a different value"
// +kubebuilder:validation:XValidation:rule="oldSelf == null || (has(oldSelf.tlsCluster) == has(self.tlsCluster)) && (!has(self.tlsCluster) || self.tlsCluster.enabled == oldSelf.tlsCluster.enabled)",message="tlsCluster is immutable"
type ClusterSpec struct {
	// SuspendAction defines the suspend actions for the cluster.
	SuspendAction *SuspendAction `json:"suspendAction,omitempty"`

	// Whether enable the TLS connection between TiDB cluster components.
	TLSCluster *TLSCluster `json:"tlsCluster,omitempty"`

	// BootstrapSQL refers to a configmap which contains the bootstrap SQL file with the key `bootstrap-sql`,
	// which will only be executed when a TiDB cluster bootstrap on the first time.
	// Only v6.5.1+ supports this feature.
	BootstrapSQL *corev1.LocalObjectReference `json:"bootstrapSQL,omitempty"`

	// UpgradePolicy defines the upgrade policy for the cluster.
	UpgradePolicy UpgradePolicy `json:"upgradePolicy,omitempty"`

	// Paused specifies whether to pause the reconciliation loop for all components of the cluster.
	Paused bool `json:"paused,omitempty"`

	// RevisionHistoryLimit is the maximum number of revisions that will
	// be maintained in each Group's revision history.
	// The revision history consists of all revisions not represented by a currently applied version.
	// The default value is 10.
	// +kubebuilder:validation:Minimum=0
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty"`

	// +kubebuilder:validation:XValidation:rule="oldSelf.exists(fg, fg.name == 'FeatureModification') || self.filter(fg, fg.name != 'FeatureModification') == oldSelf",message="can only enable FeatureModification if it's not enabled"
	// +kubebuilder:validation:XValidation:rule="self.exists(fg, fg.name == 'FeatureModification') || !oldSelf.exists(fg, fg.name == 'FeatureModification')",message="cannot disable FeatureModification"
	// +listType=map
	// +listMapKey=name
	FeatureGates []meta.FeatureGate `json:"featureGates,omitempty"`

	Security *ClusterSecurity `json:"security,omitempty"`
}

type SuspendAction struct {
	// SuspendCompute indicates delete the pods but keep the PVCs.
	SuspendCompute bool `json:"suspendCompute,omitempty"`
}

// TLSCluster is used to enable mutual TLS connection between TiDB cluster components.
// https://docs.pingcap.com/tidb/stable/enable-tls-between-components
type TLSCluster struct {
	// Enable mutual TLS connection between TiDB cluster components.
	// Once enabled, the mutual authentication applies to all components,
	// and it does not support applying to only part of the components.
	// The steps to enable this feature:
	//   1. Generate TiDB cluster components certificates and a client-side certifiacete for them.
	//      There are multiple ways to generate these certificates:
	//        - user-provided certificates: https://docs.pingcap.com/tidb/stable/generate-self-signed-certificates
	//        - use the K8s built-in certificate signing system signed certificates: https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/
	//        - or use cert-manager signed certificates: https://cert-manager.io/
	//   2. Create one secret object for one component group which contains the certificates created above.
	//      The name of this Secret must be: <groupName>-<componentName>-cluster-secret.
	//        For PD: kubectl create secret generic <groupName>-pd-cluster-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        For TiKV: kubectl create secret generic <groupName>-tikv-cluster-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        For TiDB: kubectl create secret generic <groupName>-tidb-cluster-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        For TiBR: kubectl create secret generic <tibrName>-tibr-cluster-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        For Client: kubectl create secret generic <clusterName>-cluster-client-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//        Same for other components.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

type ClusterSecurity struct {
	// SessionTokenSigningCertKeyPair is the name of the K8s secret, where stores certificates for signing the TiDB session token,
	// which is used by TiProxy for session migration.
	// You can generate certificates and create a secret by: kubectl create secret generic <secret-name> --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key>
	// Note: this field will only be used when the feature gate `SessionTokenSigning` is enabled.
	SessionTokenSigningCertKeyPair *corev1.LocalObjectReference `json:"sessionTokenSigningCertKeyPair,omitempty"`
}

type UpgradePolicy string

const (
	// UpgradePolicyDefault means the cluster will be upgraded in the following order:
	// PD, TiProxy, TiFlash, TiKV, TiDB.
	UpgradePolicyDefault UpgradePolicy = "Default"

	// UpgradePolicyNoConstraints means the cluster will be upgraded without any constraints,
	// all components will be upgraded at the same time.
	UpgradePolicyNoConstraints UpgradePolicy = "NoConstraints"
)

type ClusterStatus struct {
	// observedGeneration is the most recent generation observed for this Cluster. It corresponds to the
	// Cluster's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`

	// Components is the status of each component in the cluster.
	// +patchMergeKey=kind
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=kind
	Components []ComponentStatus `json:"components,omitempty" patchStrategy:"merge" patchMergeKey:"kind" protobuf:"bytes,1,rep,name=components"`

	// Conditions contains the current status of the cluster.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// ID is the cluster id.
	ID string `json:"id"`

	// PD means url of the pd service, it's prepared for internal use
	// e.g. https://pd:2379
	PD string `json:"pd,omitempty"`

	// FeatureGates of this cluster
	// +listType=map
	// +listMapKey=name
	FeatureGates []meta.FeatureGateStatus `json:"featureGates,omitempty"`
}

type ComponentKind string

const (
	ComponentKindPD      ComponentKind = "PD"
	ComponentKindTiKV    ComponentKind = "TiKV"
	ComponentKindTiDB    ComponentKind = "TiDB"
	ComponentKindTiFlash ComponentKind = "TiFlash"
	ComponentKindTiCDC   ComponentKind = "TiCDC"
	ComponentKindTiProxy ComponentKind = "TiProxy"
)

// ComponentStatus is the status of a component in the cluster.
type ComponentStatus struct {
	// Kind is the kind of the component, e.g., PD, TiKV, TiDB, TiFlash.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=PD;TiKV;TiDB;TiFlash;TiCDC;TiProxy
	Kind ComponentKind `json:"kind"`

	// Replicas is the number of desired replicas of the component.
	// +kubebuilder:validation:Required
	Replicas int32 `json:"replicas"`
}

const (
	// ClusterCondAvailable means the cluster is available, i.e. the cluster can be used.
	// But it does not mean all members in the cluster are healthy.
	ClusterCondAvailable = "Available"

	// ClusterCondProgressing means the cluster is progressing, i.e. the cluster is being created, updated, scaled, etc.
	ClusterCondProgressing = "Progressing"
	ClusterCreationReason  = "ClusterCreation"
	ClusterDeletionReason  = "ClusterDeletion"
	ClusterAvailableReason = "ClusterAvailable"

	ClusterCondSuspended = "Suspended"
	ClusterSuspendReason = "ClusterSuspend"
)
