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

package data

import (
	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type (
	ClusterPatch                                 func(obj *v1alpha1.Cluster)
	GroupPatch[T client.Object, G runtime.Group] func(obj G)
)

const (
	defaultClusterName      = "tc"
	defaultPDGroupName      = "pdg"
	defaultTiDBGroupName    = "dbg"
	defaultTiKVGroupName    = "kvg"
	defaultTiFlashGroupName = "fg"

	defaultVersion = "v8.1.0"
)

func WithReplicas[T client.Object, G runtime.Group](replicas *int32) GroupPatch[T, G] {
	return func(obj G) {
		obj.SetReplicas(replicas)
	}
}
