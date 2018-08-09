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
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/schedule"
)

func init() {
	schedule.RegisterScheduler("shuffle-leader", func(limiter *schedule.Limiter, args []string) (schedule.Scheduler, error) {
		return newShuffleLeaderScheduler(limiter), nil
	})
}

type shuffleLeaderScheduler struct {
	*baseScheduler
	selector schedule.Selector
}

// newShuffleLeaderScheduler creates an admin scheduler that shuffles leaders
// between stores.
func newShuffleLeaderScheduler(limiter *schedule.Limiter) schedule.Scheduler {
	filters := []schedule.Filter{
		schedule.NewBlockFilter(),
		schedule.NewStateFilter(),
		schedule.NewHealthFilter(),
		schedule.NewDisconnectFilter(),
		schedule.NewRejectLeaderFilter(),
	}
	base := newBaseScheduler(limiter)
	return &shuffleLeaderScheduler{
		baseScheduler: base,
		selector:      schedule.NewRandomSelector(filters),
	}
}

func (s *shuffleLeaderScheduler) GetName() string {
	return "shuffle-leader-scheduler"
}

func (s *shuffleLeaderScheduler) GetType() string {
	return "shuffle-leader"
}

func (s *shuffleLeaderScheduler) IsScheduleAllowed(cluster schedule.Cluster) bool {
	return s.limiter.OperatorCount(schedule.OpLeader) < cluster.GetLeaderScheduleLimit()
}

func (s *shuffleLeaderScheduler) Schedule(cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	// We shuffle leaders between stores by:
	// 1. random select a valid store.
	// 2. transfer a leader to the store.
	schedulerCounter.WithLabelValues(s.GetName(), "schedule").Inc()
	stores := cluster.GetStores()
	targetStore := s.selector.SelectTarget(cluster, stores)
	if targetStore == nil {
		schedulerCounter.WithLabelValues(s.GetName(), "no_target_store").Inc()
		return nil
	}
	region := cluster.RandFollowerRegion(targetStore.GetId(), core.HealthRegion())
	if region == nil {
		schedulerCounter.WithLabelValues(s.GetName(), "no_follower").Inc()
		return nil
	}
	schedulerCounter.WithLabelValues(s.GetName(), "new_operator").Inc()
	step := schedule.TransferLeader{FromStore: region.Leader.GetStoreId(), ToStore: targetStore.GetId()}
	op := schedule.NewOperator("shuffleLeader", region.GetId(), region.GetRegionEpoch(), schedule.OpAdmin|schedule.OpLeader, step)
	op.SetPriorityLevel(core.HighPriority)
	return []*schedule.Operator{op}
}
