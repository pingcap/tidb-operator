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

package manager

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

// Manager implements the logic for syncing tidbcluster.
type Manager interface {
	// Sync	implements the logic for syncing tidbcluster.
	Sync(*v1alpha1.TidbCluster) error
}

type DMManager interface {
	// Sync implements the logic for syncing dmcluster.
	SyncDM(*v1alpha1.DMCluster) error
}

type TiDBNGMonitoringManager interface {
	Sync(*v1alpha1.TidbNGMonitoring, *v1alpha1.TidbCluster) error
}
