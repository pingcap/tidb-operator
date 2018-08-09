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

package schedulers

import (
	"time"

	"github.com/montanaflynn/stats"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server/cache"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

// scheduleRemovePeer schedules a region to remove the peer.
func scheduleRemovePeer(cluster schedule.Cluster, schedulerName string, s schedule.Selector, filters ...schedule.Filter) (*core.RegionInfo, *metapb.Peer) {
	stores := cluster.GetStores()

	source := s.SelectSource(cluster, stores, filters...)
	if source == nil {
		schedulerCounter.WithLabelValues(schedulerName, "no_store").Inc()
		return nil, nil
	}

	region := cluster.RandFollowerRegion(source.GetId(), core.HealthRegion())
	if region == nil {
		region = cluster.RandLeaderRegion(source.GetId(), core.HealthRegion())
	}
	if region == nil {
		schedulerCounter.WithLabelValues(schedulerName, "no_region").Inc()
		return nil, nil
	}

	return region, region.GetStorePeer(source.GetId())
}

// scheduleAddPeer schedules a new peer.
func scheduleAddPeer(cluster schedule.Cluster, s schedule.Selector, filters ...schedule.Filter) *metapb.Peer {
	stores := cluster.GetStores()

	target := s.SelectTarget(cluster, stores, filters...)
	if target == nil {
		return nil
	}

	newPeer, err := cluster.AllocPeer(target.GetId())
	if err != nil {
		log.Errorf("failed to allocate peer: %v", err)
		return nil
	}

	return newPeer
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func shouldBalance(cluster schedule.Cluster, source, target *core.StoreInfo, region *core.RegionInfo, kind core.ResourceKind, opInfluence schedule.OpInfluence) bool {
	// The reason we use max(regionSize, averageRegionSize) to check is:
	// 1. prevent moving small regions between stores with close scores, leading to unnecessary balance.
	// 2. prevent moving huge regions, leading to over balance.
	regionSize := region.ApproximateSize
	if regionSize < cluster.GetAverageRegionSize() {
		regionSize = cluster.GetAverageRegionSize()
	}

	regionSize = int64(float64(regionSize) * cluster.GetTolerantSizeRatio())
	sourceDelta := opInfluence.GetStoreInfluence(source.GetId()).ResourceSize(kind) - regionSize
	targetDelta := opInfluence.GetStoreInfluence(target.GetId()).ResourceSize(kind) + regionSize

	// Make sure after move, source score is still greater than target score.
	return source.ResourceScore(kind, cluster.GetHighSpaceRatio(), cluster.GetLowSpaceRatio(), sourceDelta) >
		target.ResourceScore(kind, cluster.GetHighSpaceRatio(), cluster.GetLowSpaceRatio(), targetDelta)
}

func adjustBalanceLimit(cluster schedule.Cluster, kind core.ResourceKind) uint64 {
	stores := cluster.GetStores()
	counts := make([]float64, 0, len(stores))
	for _, s := range stores {
		if s.IsUp() {
			counts = append(counts, float64(s.ResourceCount(kind)))
		}
	}
	limit, _ := stats.StandardDeviation(stats.Float64Data(counts))
	return maxUint64(1, uint64(limit))
}

const (
	taintCacheGCInterval = time.Second * 5
	taintCacheTTL        = time.Minute * 5
)

// newTaintCache creates a TTL cache to hold stores that are not able to
// schedule operators.
func newTaintCache() *cache.TTLUint64 {
	return cache.NewIDTTL(taintCacheGCInterval, taintCacheTTL)
}
