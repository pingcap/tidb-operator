// Copyright 2017 PingCAP, Inc.
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

package schedule

import (
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
)

// Simulating is an option to overpass the impact of accelerated time. Should
// only turned on by the simulator.
var Simulating bool

// Options for schedulers.
type Options interface {
	GetLeaderScheduleLimit() uint64
	GetRegionScheduleLimit() uint64
	GetReplicaScheduleLimit() uint64
	GetMergeScheduleLimit() uint64

	GetMaxSnapshotCount() uint64
	GetMaxPendingPeerCount() uint64
	GetMaxStoreDownTime() time.Duration
	GetMaxMergeRegionSize() uint64
	GetMaxMergeRegionRows() uint64
	GetSplitMergeInterval() time.Duration

	GetMaxReplicas() int
	GetLocationLabels() []string

	GetHotRegionLowThreshold() int
	GetTolerantSizeRatio() float64
	GetLowSpaceRatio() float64
	GetHighSpaceRatio() float64

	IsRaftLearnerEnabled() bool
	CheckLabelProperty(typ string, labels []*metapb.StoreLabel) bool
}

// NamespaceOptions for namespace cluster.
type NamespaceOptions interface {
	GetLeaderScheduleLimit(name string) uint64
	GetRegionScheduleLimit(name string) uint64
	GetReplicaScheduleLimit(name string) uint64
	GetMergeScheduleLimit(name string) uint64
	GetMaxReplicas(name string) int
}

const (
	// RejectLeader is the label property type that sugguests a store should not
	// have any region leaders.
	RejectLeader = "reject-leader"
)
