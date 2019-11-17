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

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/pingcap/tidb-operator/pkg/apis/pingcap
// +k8s:defaulter-gen=TypeMeta

// Package v1alpha2 is the v1alpha2 version of the pingcap.com api group.
// Specially, the group version v1alpha1.pingcap.com is served in kube-apiserver via CRD,
// therefore, resources are stored to kube-apiserver in v1alpha1 if the resource also appears in v1alpha1
// to keep the backward compatibility. For example:
//
// tidbcluster.v1alpha2 will be stored to kube-apiserver in the form of tidbcluster.v1alpha1,
// so that:
// - existing tidbcluster.v1alpha1 objects can be read out properly from tidb-apiserver.
// - tidbcluster.v1alpha2 written by tidb-apiserver can be properly consumed by legacy clients, e.g. controller@v1.0
//
// tidbmonitor.v1alpha2 will be stored directly via the generic storage.Store interface, because there is
// no tidbmonitor resource in v1alpha1.
//
// +groupName=pingcap.com
package v1alpha2 // import "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha2"
