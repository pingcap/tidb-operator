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
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type (
	ClusterPatch                func(obj *v1alpha1.Cluster)
	GroupPatch[G runtime.Group] func(obj G)
)

const (
	defaultClusterName        = "tc"
	defaultPDGroupName        = "pdg"
	defaultTiDBGroupName      = "dbg"
	defaultTiKVGroupName      = "kvg"
	defaultTiFlashGroupName   = "fg"
	defaultTiCDCGroupName     = "cg"
	defaultTSOGroupName       = "tg"
	defaultSchedulerGroupName = "sg"
	defaultTiProxyGroupName   = "pg"

	defaultVersion        = "v8.5.2"
	defaultTiProxyVersion = "v1.3.0"

	defaultImageRegistry = "gcr.io/pingcap-public/dbaas/"
	defaultHelperImage   = "gcr.io/pingcap-public/dbaas/busybox:1.36.0"
)

const (
	// TODO(liubo02): extract to namer
	DefaultTiDBServiceName    = defaultTiDBGroupName + "-tidb"
	DefaultTiProxyServiceName = defaultTiProxyGroupName + "-tiproxy"
)

func WithReplicas[G runtime.Group](replicas int32) GroupPatch[G] {
	return func(obj G) {
		obj.SetReplicas(replicas)
	}
}

func WithGroupName[G runtime.Group](name string) GroupPatch[G] {
	return func(obj G) {
		obj.SetName(name)
	}
}

func WithGroupVersion[G runtime.Group](version string) GroupPatch[G] {
	return func(obj G) {
		obj.SetVersion(version)
	}
}

func WithGroupCluster[G runtime.Group](cluster string) GroupPatch[G] {
	return func(obj G) {
		obj.SetCluster(cluster)
	}
}
