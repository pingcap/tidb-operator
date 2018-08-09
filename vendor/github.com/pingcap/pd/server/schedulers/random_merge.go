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

package schedulers

import (
	"math/rand"

	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/schedule"
)

func init() {
	schedule.RegisterScheduler("random-merge", func(limiter *schedule.Limiter, args []string) (schedule.Scheduler, error) {
		return newRandomMergeScheduler(limiter), nil
	})
}

type randomMergeScheduler struct {
	*baseScheduler
	selector schedule.Selector
}

// newRandomMergeScheduler creates an admin scheduler that shuffles regions
// between stores.
func newRandomMergeScheduler(limiter *schedule.Limiter) schedule.Scheduler {
	filters := []schedule.Filter{
		schedule.NewStateFilter(),
		schedule.NewHealthFilter(),
	}
	base := newBaseScheduler(limiter)
	return &randomMergeScheduler{
		baseScheduler: base,
		selector:      schedule.NewRandomSelector(filters),
	}
}

func (s *randomMergeScheduler) GetName() string {
	return "random-merge-scheduler"
}

func (s *randomMergeScheduler) GetType() string {
	return "random-merge"
}

func (s *randomMergeScheduler) IsScheduleAllowed(cluster schedule.Cluster) bool {
	return s.limiter.OperatorCount(schedule.OpMerge) < cluster.GetMergeScheduleLimit()
}

func (s *randomMergeScheduler) Schedule(cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	schedulerCounter.WithLabelValues(s.GetName(), "schedule").Inc()

	stores := cluster.GetStores()
	store := s.selector.SelectSource(cluster, stores)
	if store == nil {
		schedulerCounter.WithLabelValues(s.GetName(), "no_store").Inc()
		return nil
	}
	region := cluster.RandLeaderRegion(store.GetId(), core.HealthRegion())
	if region == nil {
		schedulerCounter.WithLabelValues(s.GetName(), "no_region").Inc()
		return nil
	}

	target, other := cluster.GetAdjacentRegions(region)
	if (rand.Int()%2 == 0 && other != nil) || target == nil {
		target = other
	}
	if target == nil {
		schedulerCounter.WithLabelValues(s.GetName(), "no_adjacent").Inc()
		return nil
	}

	schedulerCounter.WithLabelValues(s.GetName(), "new_operator").Inc()
	op1, op2, err := schedule.CreateMergeRegionOperator("random-merge", cluster, region, target, schedule.OpAdmin)
	if err != nil {
		return nil
	}
	return []*schedule.Operator{op1, op2}
}
