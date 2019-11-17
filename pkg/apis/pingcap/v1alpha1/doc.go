// Copyright 2018 PingCAP, Inc.
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

// +k8s:deepcopy-gen=package,register

// Package v1alpha1 is the v1alpha1 version of the API.
//
// For historical reason, this version are served by kube-apiserver via CRD.
// In favor of backward compatibility, resources in other versions are read and written in v1alpha1
// at storage level, which means v1alpha1 must include all the fields and actually acts as the
// internal version.
//
// By convention, the fields that added for storage but not relevant with the v1alpha1 control logic
// should have a comment to indicates in which version this field is added. For example:
//
// type TidbClusterSpec struct {
//   // For storage, since v1alpha2
//   Config map[string]util.JsonObject `json:"config,omitempty"`
// }
//
// There may be some helper fields introduced to encode the information of other versions, for example,
// a field is changed to a pointer and could be nil, then additional field is required to encode this
// the information "whether this field is set to nil". Due to the nature of
//
// +groupName=pingcap.com
package v1alpha1
