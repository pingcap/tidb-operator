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

package member

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	apps "k8s.io/api/apps/v1"
)

// Upgrader implements the logic for upgrading the tidb cluster.
type Upgrader interface {
	// Upgrade upgrade the cluster
	Upgrade(*v1alpha1.TidbCluster, *apps.StatefulSet, *apps.StatefulSet) error
}

type TiKVGroupUpgrader interface {
	// Upgrade upgrade the tikvgroup
	Upgrade(*v1alpha1.TiKVGroup, *v1alpha1.TidbCluster, *apps.StatefulSet, *apps.StatefulSet) error
}
