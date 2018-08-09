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

	"github.com/pingcap/pd/server/cache"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

func init() {
	schedule.RegisterScheduler("balance-leader", func(limiter *schedule.Limiter, args []string) (schedule.Scheduler, error) {
		return newBalanceLeaderScheduler(limiter), nil
	})
}

// balanceLeaderRetryLimit is the limit to retry schedule for selected source store and target store.
const balanceLeaderRetryLimit = 10

type balanceLeaderScheduler struct {
	*baseScheduler
	selector    schedule.Selector
	taintStores *cache.TTLUint64
}

// newBalanceLeaderScheduler creates a scheduler that tends to keep leaders on
// each store balanced.
func newBalanceLeaderScheduler(limiter *schedule.Limiter) schedule.Scheduler {
	taintStores := newTaintCache()
	filters := []schedule.Filter{
		schedule.NewBlockFilter(),
		schedule.NewStateFilter(),
		schedule.NewHealthFilter(),
		schedule.NewDisconnectFilter(),
		schedule.NewRejectLeaderFilter(),
		schedule.NewCacheFilter(taintStores),
	}
	base := newBaseScheduler(limiter)
	return &balanceLeaderScheduler{
		baseScheduler: base,
		selector:      schedule.NewBalanceSelector(core.LeaderKind, filters),
		taintStores:   taintStores,
	}
}

func (l *balanceLeaderScheduler) GetName() string {
	return "balance-leader-scheduler"
}

func (l *balanceLeaderScheduler) GetType() string {
	return "balance-leader"
}

func (l *balanceLeaderScheduler) IsScheduleAllowed(cluster schedule.Cluster) bool {
	return l.limiter.OperatorCount(schedule.OpLeader) < cluster.GetLeaderScheduleLimit()
}

func (l *balanceLeaderScheduler) Schedule(cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	schedulerCounter.WithLabelValues(l.GetName(), "schedule").Inc()

	stores := cluster.GetStores()

	// source/target is the store with highest/lowest leader score in the list that
	// can be selected as balance source/target.
	source := l.selector.SelectSource(cluster, stores)
	target := l.selector.SelectTarget(cluster, stores)

	// No store can be selected as source or target.
	if source == nil || target == nil {
		schedulerCounter.WithLabelValues(l.GetName(), "no_store").Inc()
		// When the cluster is balanced, all stores will be added to the cache once
		// all of them have been selected. This will cause the scheduler to not adapt
		// to sudden change of a store's leader. Here we clear the taint cache and
		// re-iterate.
		l.taintStores.Clear()
		return nil
	}

	log.Debugf("[%s] store%d has the max leader score, store%d has the min leader score", l.GetName(), source.GetId(), target.GetId())
	sourceStoreLabel := strconv.FormatUint(source.GetId(), 10)
	targetStoreLabel := strconv.FormatUint(target.GetId(), 10)
	balanceLeaderCounter.WithLabelValues("high_score", sourceStoreLabel).Inc()
	balanceLeaderCounter.WithLabelValues("low_score", targetStoreLabel).Inc()

	for i := 0; i < balanceLeaderRetryLimit; i++ {
		if op := l.transferLeaderOut(source, cluster, opInfluence); op != nil {
			balanceLeaderCounter.WithLabelValues("transfer_out", sourceStoreLabel).Inc()
			return op
		}
		if op := l.transferLeaderIn(target, cluster, opInfluence); op != nil {
			balanceLeaderCounter.WithLabelValues("transfer_in", targetStoreLabel).Inc()
			return op
		}
	}

	// If no operator can be created for the selected stores, ignore them for a while.
	log.Debugf("[%s] no operator created for selected store%d and store%d", l.GetName(), source.GetId(), target.GetId())
	balanceLeaderCounter.WithLabelValues("add_taint", strconv.FormatUint(source.GetId(), 10)).Inc()
	l.taintStores.Put(source.GetId())
	balanceLeaderCounter.WithLabelValues("add_taint", strconv.FormatUint(target.GetId(), 10)).Inc()
	l.taintStores.Put(target.GetId())
	return nil
}

func (l *balanceLeaderScheduler) transferLeaderOut(source *core.StoreInfo, cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	region := cluster.RandLeaderRegion(source.GetId(), core.HealthRegion())
	if region == nil {
		log.Debugf("[%s] store%d has no leader", l.GetName(), source.GetId())
		schedulerCounter.WithLabelValues(l.GetName(), "no_leader_region").Inc()
		return nil
	}
	target := l.selector.SelectTarget(cluster, cluster.GetFollowerStores(region))
	if target == nil {
		log.Debugf("[%s] region %d has no target store", l.GetName(), region.GetId())
		schedulerCounter.WithLabelValues(l.GetName(), "no_target_store").Inc()
		return nil
	}
	return l.createOperator(region, source, target, cluster, opInfluence)
}

func (l *balanceLeaderScheduler) transferLeaderIn(target *core.StoreInfo, cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	region := cluster.RandFollowerRegion(target.GetId(), core.HealthRegion())
	if region == nil {
		log.Debugf("[%s] store%d has no follower", l.GetName(), target.GetId())
		schedulerCounter.WithLabelValues(l.GetName(), "no_follower_region").Inc()
		return nil
	}
	source := cluster.GetStore(region.Leader.GetStoreId())
	if source == nil {
		log.Debugf("[%s] region %d has no leader", l.GetName(), region.GetId())
		schedulerCounter.WithLabelValues(l.GetName(), "no_leader").Inc()
		return nil
	}
	return l.createOperator(region, source, target, cluster, opInfluence)
}

func (l *balanceLeaderScheduler) createOperator(region *core.RegionInfo, source, target *core.StoreInfo, cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	if cluster.IsRegionHot(region.GetId()) {
		log.Debugf("[%s] region %d is hot region, ignore it", l.GetName(), region.GetId())
		schedulerCounter.WithLabelValues(l.GetName(), "region_hot").Inc()
		return nil
	}

	if !shouldBalance(cluster, source, target, region, core.LeaderKind, opInfluence) {
		log.Debugf(`[%s] skip balance region %d, source %d to target %d, source size: %v, source score: %v, source influence: %v,
			target size: %v, target score: %v, target influence: %v, average region size: %v`, l.GetName(), region.GetId(), source.GetId(), target.GetId(),
			source.LeaderSize, source.LeaderScore(0), opInfluence.GetStoreInfluence(source.GetId()).ResourceSize(core.LeaderKind),
			target.LeaderSize, target.LeaderScore(0), opInfluence.GetStoreInfluence(target.GetId()).ResourceSize(core.LeaderKind),
			cluster.GetAverageRegionSize())
		schedulerCounter.WithLabelValues(l.GetName(), "skip").Inc()
		return nil
	}

	schedulerCounter.WithLabelValues(l.GetName(), "new_operator").Inc()
	balanceLeaderCounter.WithLabelValues("move_leader", fmt.Sprintf("store%d-out", source.GetId())).Inc()
	balanceLeaderCounter.WithLabelValues("move_leader", fmt.Sprintf("store%d-in", target.GetId())).Inc()
	step := schedule.TransferLeader{FromStore: region.Leader.GetStoreId(), ToStore: target.GetId()}
	op := schedule.NewOperator("balance-leader", region.GetId(), region.GetRegionEpoch(), schedule.OpBalance|schedule.OpLeader, step)
	return []*schedule.Operator{op}
}
