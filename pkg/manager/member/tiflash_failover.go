// Copyright 2020 PingCAP, Inc.
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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

type tiflashFailover struct {
	tiflashFailoverPeriod time.Duration
}

// NewTiFlashFailover returns a tiflash Failover
func NewTiFlashFailover(tiflashFailoverPeriod time.Duration) Failover {
	return &tiflashFailover{tiflashFailoverPeriod}
}

// TODO: Finish the failover logic
func (tff *tiflashFailover) Failover(tc *v1alpha1.TidbCluster) error {
	return nil
}

func (tff *tiflashFailover) Recover(_ *v1alpha1.TidbCluster) {
	// Do nothing now
}

type fakeTiFlashFailover struct{}

// NewFakeTiFlashFailover returns a fake Failover
func NewFakeTiFlashFailover() Failover {
	return &fakeTiFlashFailover{}
}

func (ftff *fakeTiFlashFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (ftff *fakeTiFlashFailover) Recover(_ *v1alpha1.TidbCluster) {
	return
}
