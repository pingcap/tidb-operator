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
	"github.com/juju/errors"
	"github.com/pingcap/pd/server/core"
)

var (
	// HotRegionLowThreshold is the low threadshold of hot region
	HotRegionLowThreshold = 3
)

const (
	// RegionHeartBeatReportInterval is the heartbeat report interval of a region
	RegionHeartBeatReportInterval = 60

	statCacheMaxLen              = 1000
	hotWriteRegionMinFlowRate    = 16 * 1024
	hotReadRegionMinFlowRate     = 128 * 1024
	storeHeartBeatReportInterval = 10
	minHotRegionReportInterval   = 3
	hotRegionAntiCount           = 1
)

// BasicCluster provides basic data member and interface for a tikv cluster.
type BasicCluster struct {
	Stores   *core.StoresInfo
	Regions  *core.RegionsInfo
	HotCache *HotSpotCache
}

// NewOpInfluence creates a OpInfluence.
func NewOpInfluence(operators []*Operator, cluster Cluster) OpInfluence {
	influence := OpInfluence{
		storesInfluence:  make(map[uint64]*StoreInfluence),
		regionsInfluence: make(map[uint64]*Operator),
	}

	for _, op := range operators {
		if !op.IsTimeout() && !op.IsFinish() {
			region := cluster.GetRegion(op.RegionID())
			if region != nil {
				op.Influence(influence, region)
			}
		}
		influence.regionsInfluence[op.RegionID()] = op
	}

	return influence
}

// OpInfluence records the influence of the cluster.
type OpInfluence struct {
	storesInfluence  map[uint64]*StoreInfluence
	regionsInfluence map[uint64]*Operator
}

// GetStoreInfluence get storeInfluence of specific store.
func (m OpInfluence) GetStoreInfluence(id uint64) *StoreInfluence {
	storeInfluence, ok := m.storesInfluence[id]
	if !ok {
		storeInfluence = &StoreInfluence{}
		m.storesInfluence[id] = storeInfluence
	}
	return storeInfluence
}

// GetRegionsInfluence gets regionInfluence of specific region.
func (m OpInfluence) GetRegionsInfluence() map[uint64]*Operator {
	return m.regionsInfluence
}

// StoreInfluence records influences that pending operators will make.
type StoreInfluence struct {
	RegionSize  int64
	RegionCount int64
	LeaderSize  int64
	LeaderCount int64
}

// ResourceSize returns delta size of leader/region by influence.
func (s StoreInfluence) ResourceSize(kind core.ResourceKind) int64 {
	switch kind {
	case core.LeaderKind:
		return s.LeaderSize
	case core.RegionKind:
		return s.RegionSize
	default:
		return 0
	}
}

// NewBasicCluster creates a BasicCluster.
func NewBasicCluster() *BasicCluster {
	return &BasicCluster{
		Stores:   core.NewStoresInfo(),
		Regions:  core.NewRegionsInfo(),
		HotCache: newHotSpotCache(),
	}
}

// GetStores returns all Stores in the cluster.
func (bc *BasicCluster) GetStores() []*core.StoreInfo {
	return bc.Stores.GetStores()
}

// GetStore searches for a store by ID.
func (bc *BasicCluster) GetStore(storeID uint64) *core.StoreInfo {
	return bc.Stores.GetStore(storeID)
}

// GetRegion searches for a region by ID.
func (bc *BasicCluster) GetRegion(regionID uint64) *core.RegionInfo {
	return bc.Regions.GetRegion(regionID)
}

// GetRegionStores returns all Stores that contains the region's peer.
func (bc *BasicCluster) GetRegionStores(region *core.RegionInfo) []*core.StoreInfo {
	var Stores []*core.StoreInfo
	for id := range region.GetStoreIds() {
		if store := bc.Stores.GetStore(id); store != nil {
			Stores = append(Stores, store)
		}
	}
	return Stores
}

// GetFollowerStores returns all Stores that contains the region's follower peer.
func (bc *BasicCluster) GetFollowerStores(region *core.RegionInfo) []*core.StoreInfo {
	var Stores []*core.StoreInfo
	for id := range region.GetFollowers() {
		if store := bc.Stores.GetStore(id); store != nil {
			Stores = append(Stores, store)
		}
	}
	return Stores
}

// GetLeaderStore returns all Stores that contains the region's leader peer.
func (bc *BasicCluster) GetLeaderStore(region *core.RegionInfo) *core.StoreInfo {
	return bc.Stores.GetStore(region.Leader.GetStoreId())
}

// GetAdjacentRegions returns region's info that is adjacent with specific region
func (bc *BasicCluster) GetAdjacentRegions(region *core.RegionInfo) (*core.RegionInfo, *core.RegionInfo) {
	return bc.Regions.GetAdjacentRegions(region)
}

// BlockStore stops balancer from selecting the store.
func (bc *BasicCluster) BlockStore(storeID uint64) error {
	return errors.Trace(bc.Stores.BlockStore(storeID))
}

// UnblockStore allows balancer to select the store.
func (bc *BasicCluster) UnblockStore(storeID uint64) {
	bc.Stores.UnblockStore(storeID)
}

// RandFollowerRegion returns a random region that has a follower on the store.
func (bc *BasicCluster) RandFollowerRegion(storeID uint64, opts ...core.RegionOption) *core.RegionInfo {
	return bc.Regions.RandFollowerRegion(storeID, opts...)
}

// RandLeaderRegion returns a random region that has leader on the store.
func (bc *BasicCluster) RandLeaderRegion(storeID uint64, opts ...core.RegionOption) *core.RegionInfo {
	return bc.Regions.RandLeaderRegion(storeID, opts...)
}

// GetAverageRegionSize returns the average region approximate size.
func (bc *BasicCluster) GetAverageRegionSize() int64 {
	return bc.Regions.GetAverageRegionSize()
}

// IsRegionHot checks if a region is in hot state.
func (bc *BasicCluster) IsRegionHot(id uint64, hotThreshold int) bool {
	return bc.HotCache.isRegionHot(id, hotThreshold)
}

// RegionWriteStats returns hot region's write stats.
func (bc *BasicCluster) RegionWriteStats() []*core.RegionStat {
	return bc.HotCache.RegionStats(WriteFlow)
}

// RegionReadStats returns hot region's read stats.
func (bc *BasicCluster) RegionReadStats() []*core.RegionStat {
	return bc.HotCache.RegionStats(ReadFlow)
}

// PutStore put a store
func (bc *BasicCluster) PutStore(store *core.StoreInfo) error {
	bc.Stores.SetStore(store)
	return nil
}

// PutRegion put a region
func (bc *BasicCluster) PutRegion(region *core.RegionInfo) error {
	bc.Regions.SetRegion(region)
	return nil
}

// CheckWriteStatus checks the write status, returns whether need update statistics and item.
func (bc *BasicCluster) CheckWriteStatus(region *core.RegionInfo) (bool, *core.RegionStat) {
	return bc.HotCache.CheckWrite(region, bc.Stores)
}

// CheckReadStatus checks the read status, returns whether need update statistics and item.
func (bc *BasicCluster) CheckReadStatus(region *core.RegionInfo) (bool, *core.RegionStat) {
	return bc.HotCache.CheckRead(region, bc.Stores)
}
