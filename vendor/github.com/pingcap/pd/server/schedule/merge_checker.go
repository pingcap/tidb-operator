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

	"github.com/pingcap/pd/server/cache"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/namespace"
	log "github.com/sirupsen/logrus"
)

// As region split history is not persisted. We put a special marker into
// splitCache to prevent merging any regions when server is recently started.
const mergeBlockMarker = 0

// MergeChecker ensures region to merge with adjacent region when size is small
type MergeChecker struct {
	cluster    Cluster
	classifier namespace.Classifier
	splitCache *cache.TTLUint64
}

// NewMergeChecker creates a merge checker.
func NewMergeChecker(cluster Cluster, classifier namespace.Classifier) *MergeChecker {
	splitCache := cache.NewIDTTL(time.Minute, cluster.GetSplitMergeInterval())
	splitCache.Put(mergeBlockMarker)
	return &MergeChecker{
		cluster:    cluster,
		classifier: classifier,
		splitCache: splitCache,
	}
}

// RecordRegionSplit put the recently splitted region into cache. MergeChecker
// will skip check it for a while.
func (m *MergeChecker) RecordRegionSplit(regionID uint64) {
	m.splitCache.PutWithTTL(regionID, nil, m.cluster.GetSplitMergeInterval())
}

// Check verifies a region's replicas, creating an Operator if need.
func (m *MergeChecker) Check(region *core.RegionInfo) (*Operator, *Operator) {
	if m.splitCache.Exists(mergeBlockMarker) {
		checkerCounter.WithLabelValues("merge_checker", "recently_start").Inc()
		return nil, nil
	}

	if m.splitCache.Exists(region.GetId()) {
		checkerCounter.WithLabelValues("merge_checker", "recently_split").Inc()
		return nil, nil
	}

	checkerCounter.WithLabelValues("merge_checker", "check").Inc()

	// when pd just started, it will load region meta from etcd
	// but the size for these loaded region info is 0
	// pd don't know the real size of one region until the first heartbeat of the region
	// thus here when size is 0, just skip.
	if region.ApproximateSize == 0 {
		checkerCounter.WithLabelValues("merge_checker", "skip").Inc()
		return nil, nil
	}

	// region is not small enough
	if region.ApproximateSize > int64(m.cluster.GetMaxMergeRegionSize()) ||
		region.ApproximateRows > int64(m.cluster.GetMaxMergeRegionRows()) {
		checkerCounter.WithLabelValues("merge_checker", "no_need").Inc()
		return nil, nil
	}

	// skip region has down peers or pending peers or learner peers
	if len(region.DownPeers) > 0 || len(region.PendingPeers) > 0 || len(region.Learners) > 0 {
		checkerCounter.WithLabelValues("merge_checker", "special_peer").Inc()
		return nil, nil
	}

	if len(region.Region.GetPeers()) != m.cluster.GetMaxReplicas() {
		checkerCounter.WithLabelValues("merge_checker", "abnormal_replica").Inc()
		return nil, nil
	}

	// skip hot region
	if m.cluster.IsRegionHot(region.GetId()) {
		checkerCounter.WithLabelValues("merge_checker", "hot_region").Inc()
		return nil, nil
	}

	var target *core.RegionInfo
	prev, next := m.cluster.GetAdjacentRegions(region)

	target = m.checkTarget(region, prev, target)
	target = m.checkTarget(region, next, target)

	if target == nil {
		checkerCounter.WithLabelValues("merge_checker", "no_target").Inc()
		return nil, nil
	}

	checkerCounter.WithLabelValues("merge_checker", "new_operator").Inc()
	log.Debugf("try to merge region {%v} into region {%v}", region, target)
	op1, op2, err := CreateMergeRegionOperator("merge-region", m.cluster, region, target, OpMerge)
	if err != nil {
		return nil, nil
	}
	return op1, op2
}

func (m *MergeChecker) checkTarget(region, adjacent, target *core.RegionInfo) *core.RegionInfo {
	// if is not hot region and under same namesapce
	if adjacent != nil && !m.cluster.IsRegionHot(adjacent.GetId()) &&
		m.classifier.AllowMerge(region, adjacent) &&
		len(adjacent.DownPeers) == 0 && len(adjacent.PendingPeers) == 0 && len(adjacent.Learners) == 0 {
		// if both region is not hot, prefer the one with smaller size
		if target == nil || target.ApproximateSize > adjacent.ApproximateSize {
			// peer count should equal
			if len(adjacent.Region.GetPeers()) == m.cluster.GetMaxReplicas() {
				target = adjacent
			}
		}
	}
	return target
}
