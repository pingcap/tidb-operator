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

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/pingcap/tidb-operator/pkg/apiserver/apis/tidb
// +k8s:defaulter-gen=TypeMeta

// Package v1alpha1 the v1alpha1 version of the tidb.pingcap.com api group.
// +groupName=tidb.pingcap.com
package v1alpha1 // import "github.com/pingcap/tidb-operator/pkg/apiserver/apis/tidb/v1alpha1"
