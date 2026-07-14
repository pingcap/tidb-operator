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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	// PlacementTiKVGroupLabelKey is the normal TiKVGroup placement label key.
	PlacementTiKVGroupLabelKey = "k/tikvgroup"
	// PlacementTiKVGroupExclusiveLabelKey marks stores as exclusive for PD placement rules.
	PlacementTiKVGroupExclusiveLabelKey = "$k/exclusive"
	// PlacementTiKVGroupExclusiveLabelValue marks an exclusive store.
	PlacementTiKVGroupExclusiveLabelValue = "true"

	PlacementPolicyRoleVoter = "voter"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// PlacementPolicyList defines a list of placement policies.
type PlacementPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PlacementPolicy `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=policy,shortName=pp
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PlacementPolicy defines PD placement rules for selected key ranges.
type PlacementPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlacementPolicySpec   `json:"spec,omitempty"`
	Status PlacementPolicyStatus `json:"status,omitempty"`
}

// PlacementPolicySpec defines placement policy target groups and rules.
type PlacementPolicySpec struct {
	Cluster ClusterReference `json:"cluster"`

	// GroupRefs selects TiKV groups that receive all rules in this policy.
	// +listType=map
	// +listMapKey=group
	// +listMapKey=kind
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	GroupRefs []PlacementPolicyGroupRef `json:"groupRefs"`

	// Rules defines key range placement rules applied to every groupRef.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	Rules []PlacementPolicyRule `json:"rules"`
}

// PlacementPolicyGroupRef references a placement-capable group.
type PlacementPolicyGroupRef struct {
	// +kubebuilder:validation:Enum=core.pingcap.com
	Group string `json:"group"`
	// +kubebuilder:validation:Enum=TiKVGroup
	Kind string `json:"kind"`
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// PlacementPolicyRule defines one user-visible placement rule.
type PlacementPolicyRule struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// +kubebuilder:validation:Enum=voter
	Role string `json:"role"`

	// +kubebuilder:validation:Minimum=1
	Count int32 `json:"count"`

	Selector PlacementPolicySelector `json:"selector"`
}

// PlacementPolicySelector selects a key range by friendly identifiers.
// +kubebuilder:validation:XValidation:rule="has(self.keyspace)",message="selector.keyspace is required"
type PlacementPolicySelector struct {
	// Selector to select key range by keyspace
	Keyspace *PlacementPolicyKeyspaceSelector `json:"keyspace,omitempty"`
}

type PlacementPolicyKeyspaceSelector struct {
	// Keyspaces selects PD keyspace IDs. Database and table selectors are reserved
	// for a later version because their key ranges are unstable today.
	// +listType=set
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:Pattern=`^(0|[1-9][0-9]*)$`
	IDs []string `json:"ids"`
}

// PlacementPolicyStatus records synced PD placement bundles.
type PlacementPolicyStatus struct {
	// ObservedGeneration is the most recent generation observed by the controller.
	// It's used to determine whether the controller has reconciled the latest spec.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions contain details of the current state.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
