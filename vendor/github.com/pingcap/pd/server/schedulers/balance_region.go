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
	"fmt"
	"strconv"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server/cache"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

func init() {
	schedule.RegisterScheduler("balance-region", func(limiter *schedule.Limiter, args []string) (schedule.Scheduler, error) {
		return newBalanceRegionScheduler(limiter), nil
	})
}

// balanceRegionRetryLimit is the limit to retry schedule for selected store.
const balanceRegionRetryLimit = 10

type balanceRegionScheduler struct {
	*baseScheduler
	selector    schedule.Selector
	taintStores *cache.TTLUint64
}

// newBalanceRegionScheduler creates a scheduler that tends to keep regions on
// each store balanced.
func newBalanceRegionScheduler(limiter *schedule.Limiter) schedule.Scheduler {
	taintStores := newTaintCache()
	filters := []schedule.Filter{
		schedule.NewCacheFilter(taintStores),
		schedule.NewStateFilter(),
		schedule.NewHealthFilter(),
		schedule.NewSnapshotCountFilter(),
		schedule.NewPendingPeerCountFilter(),
	}
	base := newBaseScheduler(limiter)
	return &balanceRegionScheduler{
		baseScheduler: base,
		selector:      schedule.NewBalanceSelector(core.RegionKind, filters),
		taintStores:   taintStores,
	}
}

func (s *balanceRegionScheduler) GetName() string {
	return "balance-region-scheduler"
}

func (s *balanceRegionScheduler) GetType() string {
	return "balance-region"
}

func (s *balanceRegionScheduler) IsScheduleAllowed(cluster schedule.Cluster) bool {
	return s.limiter.OperatorCount(schedule.OpRegion) < cluster.GetRegionScheduleLimit()
}

func (s *balanceRegionScheduler) Schedule(cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	schedulerCounter.WithLabelValues(s.GetName(), "schedule").Inc()

	stores := cluster.GetStores()

	// source is the store with highest region score in the list that can be selected as balance source.
	source := s.selector.SelectSource(cluster, stores)
	if source == nil {
		schedulerCounter.WithLabelValues(s.GetName(), "no_store").Inc()
		// Unlike the balanceLeaderScheduler, we don't need to clear the taintCache
		// here. Because normally region score won't change rapidly, and the region
		// balance requires lower sensitivity compare to leader balance.
		return nil
	}

	log.Debugf("[%s] store%d has the max region score", s.GetName(), source.GetId())
	sourceLabel := strconv.FormatUint(source.GetId(), 10)
	balanceRegionCounter.WithLabelValues("source_store", sourceLabel).Inc()

	for i := 0; i < balanceRegionRetryLimit; i++ {
		region := cluster.RandFollowerRegion(source.GetId(), core.HealthRegion())
		if region == nil {
			region = cluster.RandLeaderRegion(source.GetId(), core.HealthRegion())
		}
		if region == nil {
			schedulerCounter.WithLabelValues(s.GetName(), "no_region").Inc()
			continue
		}
		log.Debugf("[%s] select region%d", s.GetName(), region.GetId())

		// We don't schedule region with abnormal number of replicas.
		if len(region.GetPeers()) != cluster.GetMaxReplicas() {
			log.Debugf("[%s] region%d has abnormal replica count", s.GetName(), region.GetId())
			schedulerCounter.WithLabelValues(s.GetName(), "abnormal_replica").Inc()
			continue
		}

		// Skip hot regions.
		if cluster.IsRegionHot(region.GetId()) {
			log.Debugf("[%s] region%d is hot", s.GetName(), region.GetId())
			schedulerCounter.WithLabelValues(s.GetName(), "region_hot").Inc()
			continue
		}

		oldPeer := region.GetStorePeer(source.GetId())
		if op := s.transferPeer(cluster, region, oldPeer, opInfluence); op != nil {
			schedulerCounter.WithLabelValues(s.GetName(), "new_operator").Inc()
			return []*schedule.Operator{op}
		}
	}

	// If no operator can be created for the selected store, ignore it for a while.
	log.Debugf("[%s] no operator created for selected store%d", s.GetName(), source.GetId())
	balanceRegionCounter.WithLabelValues("add_taint", sourceLabel).Inc()
	s.taintStores.Put(source.GetId())
	return nil
}

func (s *balanceRegionScheduler) transferPeer(cluster schedule.Cluster, region *core.RegionInfo, oldPeer *metapb.Peer, opInfluence schedule.OpInfluence) *schedule.Operator {
	// scoreGuard guarantees that the distinct score will not decrease.
	stores := cluster.GetRegionStores(region)
	source := cluster.GetStore(oldPeer.GetStoreId())
	scoreGuard := schedule.NewDistinctScoreFilter(cluster.GetLocationLabels(), stores, source)

	checker := schedule.NewReplicaChecker(cluster, nil)
	storeID, _ := checker.SelectBestReplacementStore(region, oldPeer, scoreGuard)
	if storeID == 0 {
		schedulerCounter.WithLabelValues(s.GetName(), "no_replacement").Inc()
		return nil
	}

	target := cluster.GetStore(storeID)
	log.Debugf("[region %d] source store id is %v, target store id is %v", region.GetId(), source.GetId(), target.GetId())

	if !shouldBalance(cluster, source, target, region, core.RegionKind, opInfluence) {
		log.Debugf(`[%s] skip balance region %d, source %d to target %d ,source size: %v, source score: %v, source influence: %v, 
			target size: %v, target score: %v, target influence: %v, average region size: %v`, s.GetName(), region.GetId(), source.GetId(), target.GetId(),
			source.RegionSize, source.RegionScore(cluster.GetHighSpaceRatio(), cluster.GetLowSpaceRatio(), 0),
			opInfluence.GetStoreInfluence(source.GetId()).ResourceSize(core.RegionKind),
			target.RegionSize, target.RegionScore(cluster.GetHighSpaceRatio(), cluster.GetLowSpaceRatio(), 0),
			opInfluence.GetStoreInfluence(target.GetId()).ResourceSize(core.RegionKind),
			cluster.GetAverageRegionSize())
		schedulerCounter.WithLabelValues(s.GetName(), "skip").Inc()
		return nil
	}

	newPeer, err := cluster.AllocPeer(storeID)
	if err != nil {
		schedulerCounter.WithLabelValues(s.GetName(), "no_peer").Inc()
		return nil
	}
	balanceRegionCounter.WithLabelValues("move_peer", fmt.Sprintf("store%d-out", source.GetId())).Inc()
	balanceRegionCounter.WithLabelValues("move_peer", fmt.Sprintf("store%d-in", target.GetId())).Inc()
	return schedule.CreateMovePeerOperator("balance-region", cluster, region, schedule.OpBalance, oldPeer.GetStoreId(), newPeer.GetStoreId(), newPeer.GetId())
}
