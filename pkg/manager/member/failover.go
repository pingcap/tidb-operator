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
)

// TODO: move this to a centralized place
// Since the "Unhealthy" is a very universal event reason string, which could apply to all the TiDB/DM cluster components,
// we should make a global event module, and put event related constants there.
const (
	unHealthEventReason     = "Unhealthy"
	unHealthEventMsgPattern = "%s pod[%s] is unhealthy, msg:%s"
	FailedSetStoreLabels    = "FailedSetStoreLabels"
	recoveryEventReason     = "Recovery"
)

// Failover implements the logic for pd/tikv/tidb's failover and recovery.
type Failover interface {
	Failover(*v1alpha1.TidbCluster) error
	Recover(*v1alpha1.TidbCluster)
	RemoveUndesiredFailures(*v1alpha1.TidbCluster)
}

// DMFailover implements the logic for dm-master/dm-worker's failover and recovery.
type DMFailover interface {
	Failover(*v1alpha1.DMCluster) error
	Recover(*v1alpha1.DMCluster)
	RemoveUndesiredFailures(*v1alpha1.DMCluster)
}
