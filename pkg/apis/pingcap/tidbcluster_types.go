// Copyright 2019. PingCAP, Inc.
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

package pingcap

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TidbClusterSpec v1alpha1.TidbClusterSpec
type TidbClusterStatus v1alpha1.TidbClusterStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false
// Internal version of TidbCluster.
//
// TidbCluster is migrated from CRD, so the internal version is v1alpha1 in fact. But
// operating versioned object at storage level violates the convention of apiserver and
// requires additional work, this type definition exists to make things simpler.
type TidbCluster struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the behavior of a tidb cluster
	Spec TidbClusterSpec `json:"spec"`

	// Most recently observed status of the tidb cluster
	Status TidbClusterStatus `json:"status"`
}
