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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TiDBGroup encode the spec and status of a Group of TiDB Instances
type TiDBGroup struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the desired state of TiDBGroup
	Spec TiDBGroupSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the TiDBGroup
	Status TiDBGroupStatus `json:"status"`
}

// +k8s:openapi-gen=true
// TiDBGroupSpec describes the attributes that a user creates on a TiDBGroup
type TiDBGroupSpec struct {
	TiDBSpec `json:",inline"`

	// TidbClusterRef describe the target TidbCluster
	Cluster TidbClusterRef `json:"cluster"`
}

// +k8s:openapi-gen=false
// Most recently observed status of the TiDBGroup
type TiDBGroupStatus struct {
	TiDBStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TiDBGroupList is TiDBGroup list
type TiDBGroupList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TiDBGroup `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TiKVGroup encode the spec and status of a Group of TiKV Instances
type TiKVGroup struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the desired state of TiKVGroup
	Spec TiKVGroupSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the TiKVGroup
	Status TiKVGroupStatus `json:"status"`
}

// +k8s:openapi-gen=true
// TiKVGroupSpec describes the attributes that a user creates on a TiKVGroup
type TiKVGroupSpec struct {
	TiKVSpec `json:",inline"`

	// TidbClusterRef describe the target TidbCluster
	Cluster TidbClusterRef `json:"cluster"`
}

// +k8s:openapi-gen=false
// Most recently observed status of the TiKVGroup
type TiKVGroupStatus struct {
	TiKVStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TKVGroupList is TiKVGroup list
type TKVGroupList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TiKVGroup `json:"items"`
}
