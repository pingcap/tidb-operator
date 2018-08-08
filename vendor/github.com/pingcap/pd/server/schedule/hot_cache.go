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

package schedule

import (
	"math/rand"
	"time"

	"github.com/pingcap/pd/server/cache"
	"github.com/pingcap/pd/server/core"
)

// FlowKind is a identify Flow types.
type FlowKind uint32

// Flags for flow.
const (
	WriteFlow FlowKind = iota
	ReadFlow
)

// HotSpotCache is a cache hold hot regions.
type HotSpotCache struct {
	writeFlow cache.Cache
	readFlow  cache.Cache
}

func newHotSpotCache() *HotSpotCache {
	return &HotSpotCache{
		writeFlow: cache.NewCache(statCacheMaxLen, cache.TwoQueueCache),
		readFlow:  cache.NewCache(statCacheMaxLen, cache.TwoQueueCache),
	}
}

// CheckWrite checks the write status, returns whether need update statistics and item.
func (w *HotSpotCache) CheckWrite(region *core.RegionInfo, stores *core.StoresInfo) (bool, *core.RegionStat) {
	var (
		WrittenBytesPerSec uint64
		value              *core.RegionStat
	)
	v, isExist := w.writeFlow.Peek(region.GetId())
	if isExist && !Simulating {
		value = v.(*core.RegionStat)
		interval := time.Since(value.LastUpdateTime).Seconds()
		if interval < minHotRegionReportInterval {
			return false, nil
		}
		WrittenBytesPerSec = uint64(float64(region.WrittenBytes) / interval)
	} else {
		WrittenBytesPerSec = uint64(float64(region.WrittenBytes) / float64(RegionHeartBeatReportInterval))
	}

	hotRegionThreshold := calculateWriteHotThreshold(stores)
	return w.isNeedUpdateStatCache(region, WrittenBytesPerSec, hotRegionThreshold, value, WriteFlow)
}

// CheckRead checks the read status, returns whether need update statistics and item.
func (w *HotSpotCache) CheckRead(region *core.RegionInfo, stores *core.StoresInfo) (bool, *core.RegionStat) {
	var (
		ReadBytesPerSec uint64
		value           *core.RegionStat
	)
	v, isExist := w.readFlow.Peek(region.GetId())
	if isExist && !Simulating {
		value = v.(*core.RegionStat)
		interval := time.Since(value.LastUpdateTime).Seconds()
		if interval < minHotRegionReportInterval {
			return false, nil
		}
		ReadBytesPerSec = uint64(float64(region.ReadBytes) / interval)
	} else {
		ReadBytesPerSec = uint64(float64(region.ReadBytes) / float64(RegionHeartBeatReportInterval))
	}
	hotRegionThreshold := calculateReadHotThreshold(stores)
	return w.isNeedUpdateStatCache(region, ReadBytesPerSec, hotRegionThreshold, value, ReadFlow)
}

func (w *HotSpotCache) incMetrics(name string, kind FlowKind) {
	switch kind {
	case WriteFlow:
		hotCacheStatusGauge.WithLabelValues(name, "write").Inc()
	case ReadFlow:
		hotCacheStatusGauge.WithLabelValues(name, "read").Inc()
	}
}

func calculateWriteHotThreshold(stores *core.StoresInfo) uint64 {
	// hotRegionThreshold is use to pick hot region
	// suppose the number of the hot Regions is statCacheMaxLen
	// and we use total written Bytes past storeHeartBeatReportInterval seconds to divide the number of hot Regions
	// divide 2 because the store reports data about two times than the region record write to rocksdb
	divisor := float64(statCacheMaxLen) * 2
	hotRegionThreshold := uint64(float64(stores.TotalBytesWriteRate()) / divisor)

	if hotRegionThreshold < hotWriteRegionMinFlowRate {
		hotRegionThreshold = hotWriteRegionMinFlowRate
	}
	return hotRegionThreshold
}

func calculateReadHotThreshold(stores *core.StoresInfo) uint64 {
	// hotRegionThreshold is use to pick hot region
	// suppose the number of the hot Regions is statLRUMaxLen
	// and we use total Read Bytes past storeHeartBeatReportInterval seconds to divide the number of hot Regions
	divisor := float64(statCacheMaxLen)
	hotRegionThreshold := uint64(float64(stores.TotalBytesReadRate()) / divisor)

	if hotRegionThreshold < hotReadRegionMinFlowRate {
		hotRegionThreshold = hotReadRegionMinFlowRate
	}
	return hotRegionThreshold
}

const rollingWindowsSize = 5

func (w *HotSpotCache) isNeedUpdateStatCache(region *core.RegionInfo, flowBytes uint64, hotRegionThreshold uint64, oldItem *core.RegionStat, kind FlowKind) (bool, *core.RegionStat) {
	newItem := &core.RegionStat{
		RegionID:       region.GetId(),
		FlowBytes:      flowBytes,
		LastUpdateTime: time.Now(),
		StoreID:        region.Leader.GetStoreId(),
		Version:        region.GetRegionEpoch().GetVersion(),
		AntiCount:      hotRegionAntiCount,
	}

	if oldItem != nil {
		newItem.HotDegree = oldItem.HotDegree + 1
		newItem.Stats = oldItem.Stats
	}
	if flowBytes >= hotRegionThreshold {
		if oldItem == nil {
			w.incMetrics("add_item", kind)
			newItem.Stats = core.NewRollingStats(rollingWindowsSize)
		}
		newItem.Stats.Add(float64(flowBytes))
		return true, newItem
	}
	// smaller than hotReionThreshold
	if oldItem == nil {
		return false, newItem
	}
	if oldItem.AntiCount <= 0 {
		w.incMetrics("remove_item", kind)
		return true, nil
	}
	// eliminate some noise
	newItem.HotDegree = oldItem.HotDegree - 1
	newItem.AntiCount = oldItem.AntiCount - 1
	newItem.Stats.Add(float64(flowBytes))
	return true, newItem
}

// Update updates the cache.
func (w *HotSpotCache) Update(key uint64, item *core.RegionStat, kind FlowKind) {
	switch kind {
	case WriteFlow:
		if item == nil {
			w.writeFlow.Remove(key)
		} else {
			w.writeFlow.Put(key, item)
			w.incMetrics("update_item", kind)
		}
	case ReadFlow:
		if item == nil {
			w.readFlow.Remove(key)
		} else {
			w.readFlow.Put(key, item)
			w.incMetrics("update_item", kind)
		}
	}
}

// RegionStats returns hot items according to kind
func (w *HotSpotCache) RegionStats(kind FlowKind) []*core.RegionStat {
	var elements []*cache.Item
	switch kind {
	case WriteFlow:
		elements = w.writeFlow.Elems()
	case ReadFlow:
		elements = w.readFlow.Elems()
	}
	stats := make([]*core.RegionStat, len(elements))
	for i := range elements {
		stats[i] = elements[i].Value.(*core.RegionStat)
	}
	return stats
}

// RandHotRegionFromStore random picks a hot region in specify store.
func (w *HotSpotCache) RandHotRegionFromStore(storeID uint64, kind FlowKind, hotThreshold int) *core.RegionStat {
	stats := w.RegionStats(kind)
	for _, i := range rand.Perm(len(stats)) {
		if stats[i].HotDegree >= hotThreshold && stats[i].StoreID == storeID {
			return stats[i]
		}
	}
	return nil
}

// CollectMetrics collect the hot cache metrics
func (w *HotSpotCache) CollectMetrics(stores *core.StoresInfo) {
	hotCacheStatusGauge.WithLabelValues("total_length", "write").Set(float64(w.writeFlow.Len()))
	hotCacheStatusGauge.WithLabelValues("total_length", "read").Set(float64(w.readFlow.Len()))
	threshold := calculateWriteHotThreshold(stores)
	hotCacheStatusGauge.WithLabelValues("hotThreshold", "write").Set(float64(threshold))
	threshold = calculateReadHotThreshold(stores)
	hotCacheStatusGauge.WithLabelValues("hotThreshold", "read").Set(float64(threshold))
}

func (w *HotSpotCache) isRegionHot(id uint64, hotThreshold int) bool {
	if stat, ok := w.writeFlow.Peek(id); ok {
		if stat.(*core.RegionStat).HotDegree >= hotThreshold {
			return true
		}
	}
	if stat, ok := w.readFlow.Peek(id); ok {
		return stat.(*core.RegionStat).HotDegree >= hotThreshold
	}
	return false
}
