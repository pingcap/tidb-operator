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

package server

import (
	"fmt"

	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/namespace"
)

type regionStatisticType uint32

const (
	missPeer regionStatisticType = 1 << iota
	extraPeer
	downPeer
	pendingPeer
	offlinePeer
	incorrectNamespace
	learnerPeer
)

type regionStatistics struct {
	opt        *scheduleOption
	classifier namespace.Classifier
	stats      map[regionStatisticType]map[uint64]*core.RegionInfo
	index      map[uint64]regionStatisticType
}

func newRegionStatistics(opt *scheduleOption, classifier namespace.Classifier) *regionStatistics {
	r := &regionStatistics{
		opt:        opt,
		classifier: classifier,
		stats:      make(map[regionStatisticType]map[uint64]*core.RegionInfo),
		index:      make(map[uint64]regionStatisticType),
	}
	r.stats[missPeer] = make(map[uint64]*core.RegionInfo)
	r.stats[extraPeer] = make(map[uint64]*core.RegionInfo)
	r.stats[downPeer] = make(map[uint64]*core.RegionInfo)
	r.stats[pendingPeer] = make(map[uint64]*core.RegionInfo)
	r.stats[offlinePeer] = make(map[uint64]*core.RegionInfo)
	r.stats[incorrectNamespace] = make(map[uint64]*core.RegionInfo)
	r.stats[learnerPeer] = make(map[uint64]*core.RegionInfo)
	return r
}

func (r *regionStatistics) getRegionStatsByType(typ regionStatisticType) []*core.RegionInfo {
	res := make([]*core.RegionInfo, 0, len(r.stats[typ]))
	for _, r := range r.stats[typ] {
		res = append(res, r.Clone())
	}
	return res
}

func (r *regionStatistics) deleteEntry(deleteIndex regionStatisticType, regionID uint64) {
	for typ := regionStatisticType(1); typ <= deleteIndex; typ <<= 1 {
		if deleteIndex&typ != 0 {
			delete(r.stats[typ], regionID)
		}
	}
}

func (r *regionStatistics) Observe(region *core.RegionInfo, stores []*core.StoreInfo) {
	// Region state.
	regionID := region.GetId()
	namespace := r.classifier.GetRegionNamespace(region)
	var (
		peerTypeIndex regionStatisticType
		deleteIndex   regionStatisticType
	)
	if len(region.Peers) < r.opt.GetMaxReplicas(namespace) {
		r.stats[missPeer][regionID] = region
		peerTypeIndex |= missPeer
	} else if len(region.Peers) > r.opt.GetMaxReplicas(namespace) {
		r.stats[extraPeer][regionID] = region
		peerTypeIndex |= extraPeer
	}

	if len(region.DownPeers) > 0 {
		r.stats[downPeer][regionID] = region
		peerTypeIndex |= downPeer
	}

	if len(region.PendingPeers) > 0 {
		r.stats[pendingPeer][regionID] = region
		peerTypeIndex |= pendingPeer
	}

	if len(region.GetLearners()) > 0 {
		r.stats[learnerPeer][regionID] = region
		peerTypeIndex |= learnerPeer
	}

	for _, store := range stores {
		if store.IsOffline() {
			peer := region.GetStorePeer(store.GetId())
			if peer != nil {
				r.stats[offlinePeer][regionID] = region
				peerTypeIndex |= offlinePeer
			}
		}
		ns := r.classifier.GetStoreNamespace(store)
		if ns == namespace {
			continue
		}
		r.stats[incorrectNamespace][regionID] = region
		peerTypeIndex |= incorrectNamespace
		break
	}

	if oldIndex, ok := r.index[regionID]; ok {
		deleteIndex = oldIndex &^ peerTypeIndex
	}
	r.deleteEntry(deleteIndex, regionID)
	r.index[regionID] = peerTypeIndex
}

func (r *regionStatistics) clearDefunctRegion(regionID uint64) {
	if oldIndex, ok := r.index[regionID]; ok {
		r.deleteEntry(oldIndex, regionID)
	}
}

func (r *regionStatistics) Collect() {
	regionStatusGauge.WithLabelValues("miss_peer_region_count").Set(float64(len(r.stats[missPeer])))
	regionStatusGauge.WithLabelValues("extra_peer_region_count").Set(float64(len(r.stats[extraPeer])))
	regionStatusGauge.WithLabelValues("down_peer_region_count").Set(float64(len(r.stats[downPeer])))
	regionStatusGauge.WithLabelValues("pending_peer_region_count").Set(float64(len(r.stats[pendingPeer])))
	regionStatusGauge.WithLabelValues("offline_peer_region_count").Set(float64(len(r.stats[offlinePeer])))
	regionStatusGauge.WithLabelValues("incorrect_namespace_region_count").Set(float64(len(r.stats[incorrectNamespace])))
	regionStatusGauge.WithLabelValues("learner_peer_region_count").Set(float64(len(r.stats[learnerPeer])))
}

type labelLevelStatistics struct {
	regionLabelLevelStats map[uint64]int
	labelLevelCounter     map[int]int
}

func newLabelLevelStatistics() *labelLevelStatistics {
	return &labelLevelStatistics{
		regionLabelLevelStats: make(map[uint64]int),
		labelLevelCounter:     make(map[int]int),
	}
}

func (l *labelLevelStatistics) Observe(region *core.RegionInfo, stores []*core.StoreInfo, labels []string) {
	regionID := region.GetId()
	regionLabelLevel := getRegionLabelIsolationLevel(stores, labels)
	if level, ok := l.regionLabelLevelStats[regionID]; ok {
		if level == regionLabelLevel {
			return
		}
		l.labelLevelCounter[level]--
	}
	l.regionLabelLevelStats[regionID] = regionLabelLevel
	l.labelLevelCounter[regionLabelLevel]++
}

func (l *labelLevelStatistics) Collect() {
	for level, count := range l.labelLevelCounter {
		typ := fmt.Sprintf("level_%d", level)
		regionLabelLevelGauge.WithLabelValues(typ).Set(float64(count))
	}
}

func (l *labelLevelStatistics) clearDefunctRegion(regionID uint64) {
	if level, ok := l.regionLabelLevelStats[regionID]; ok {
		l.labelLevelCounter[level]--
		delete(l.regionLabelLevelStats, regionID)
	}
}

func getRegionLabelIsolationLevel(stores []*core.StoreInfo, labels []string) int {
	if len(stores) == 0 || len(labels) == 0 {
		return 0
	}
	queueStores := [][]*core.StoreInfo{stores}
	for level, label := range labels {
		newQueueStores := make([][]*core.StoreInfo, 0, len(stores))
		for _, stores := range queueStores {
			notIsolatedStores := notIsolatedStoresWithLabel(stores, label)
			if len(notIsolatedStores) > 0 {
				newQueueStores = append(newQueueStores, notIsolatedStores...)
			}
		}
		queueStores = newQueueStores
		if len(queueStores) == 0 {
			return level + 1
		}
	}
	return 0
}

func notIsolatedStoresWithLabel(stores []*core.StoreInfo, label string) [][]*core.StoreInfo {
	m := make(map[string][]*core.StoreInfo)
	for _, s := range stores {
		labelValue := s.GetLabelValue(label)
		if labelValue == "" {
			continue
		}
		m[labelValue] = append(m[labelValue], s)
	}
	var res [][]*core.StoreInfo
	for _, stores := range m {
		if len(stores) > 1 {
			res = append(res, stores)
		}
	}
	return res
}
