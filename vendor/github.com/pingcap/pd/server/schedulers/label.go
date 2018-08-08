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
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

func init() {
	schedule.RegisterScheduler("label", func(limiter *schedule.Limiter, args []string) (schedule.Scheduler, error) {
		return newLabelScheduler(limiter), nil
	})
}

type labelScheduler struct {
	*baseScheduler
	selector schedule.Selector
}

func newLabelScheduler(limiter *schedule.Limiter) schedule.Scheduler {
	filters := []schedule.Filter{
		schedule.NewBlockFilter(),
		schedule.NewStateFilter(),
		schedule.NewHealthFilter(),
		schedule.NewDisconnectFilter(),
		schedule.NewRejectLeaderFilter(),
	}
	return &labelScheduler{
		baseScheduler: newBaseScheduler(limiter),
		selector:      schedule.NewBalanceSelector(core.LeaderKind, filters),
	}
}

func (s *labelScheduler) GetName() string {
	return "label-scheduler"
}

func (s *labelScheduler) GetType() string {
	return "label"
}

func (s *labelScheduler) IsScheduleAllowed(cluster schedule.Cluster) bool {
	return s.limiter.OperatorCount(schedule.OpLeader) < cluster.GetLeaderScheduleLimit()
}

func (s *labelScheduler) Schedule(cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	schedulerCounter.WithLabelValues(s.GetName(), "schedule").Inc()
	stores := cluster.GetStores()
	rejectLeaderStores := make(map[uint64]struct{})
	for _, s := range stores {
		if cluster.CheckLabelProperty(schedule.RejectLeader, s.Labels) {
			rejectLeaderStores[s.GetId()] = struct{}{}
		}
	}
	if len(rejectLeaderStores) == 0 {
		schedulerCounter.WithLabelValues(s.GetName(), "skip").Inc()
		return nil
	}
	log.Debugf("label scheduler reject leader store list: %v", rejectLeaderStores)
	for id := range rejectLeaderStores {
		if region := cluster.RandLeaderRegion(id); region != nil {
			log.Debugf("label scheduler selects region %d to transfer leader", region.GetId())
			excludeStores := make(map[uint64]struct{})
			for _, p := range region.DownPeers {
				excludeStores[p.GetPeer().GetStoreId()] = struct{}{}
			}
			for _, p := range region.PendingPeers {
				excludeStores[p.GetStoreId()] = struct{}{}
			}
			filter := schedule.NewExcludedFilter(nil, excludeStores)
			target := s.selector.SelectTarget(cluster, cluster.GetFollowerStores(region), filter)
			if target == nil {
				log.Debugf("label scheduler no target found for region %d", region.GetId())
				schedulerCounter.WithLabelValues(s.GetName(), "no_target").Inc()
				continue
			}

			schedulerCounter.WithLabelValues(s.GetName(), "new_operator").Inc()
			step := schedule.TransferLeader{FromStore: id, ToStore: target.GetId()}
			op := schedule.NewOperator("label-reject-leader", region.GetId(), region.GetRegionEpoch(), schedule.OpLeader, step)
			return []*schedule.Operator{op}
		}
	}
	schedulerCounter.WithLabelValues(s.GetName(), "no_region").Inc()
	return nil
}
