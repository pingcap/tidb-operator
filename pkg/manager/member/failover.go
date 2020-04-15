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

import "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

const (
	unHealthEventReason     = "Unhealthy"
	unHealthEventMsgPattern = "%s pod[%s] is unhealthy, msg:%s"
)

// Failover implements the logic for pd/tikv/tidb's failover and recovery.
type Failover interface {
	Failover(*v1alpha1.TidbCluster) error
	Recover(*v1alpha1.TidbCluster)
}
