// Copyright 2016 PingCAP, Inc.
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
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/namespace"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

type clusterInfo struct {
	sync.RWMutex
	core *schedule.BasicCluster

	id              core.IDAllocator
	kv              *core.KV
	meta            *metapb.Cluster
	activeRegions   int
	opt             *scheduleOption
	regionStats     *regionStatistics
	labelLevelStats *labelLevelStatistics
}

func newClusterInfo(id core.IDAllocator, opt *scheduleOption, kv *core.KV) *clusterInfo {
	return &clusterInfo{
		core:            schedule.NewBasicCluster(),
		id:              id,
		opt:             opt,
		kv:              kv,
		labelLevelStats: newLabelLevelStatistics(),
	}
}

// Return nil if cluster is not bootstrapped.
func loadClusterInfo(id core.IDAllocator, kv *core.KV, opt *scheduleOption) (*clusterInfo, error) {
	c := newClusterInfo(id, opt, kv)

	c.meta = &metapb.Cluster{}
	ok, err := kv.LoadMeta(c.meta)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !ok {
		return nil, nil
	}

	start := time.Now()
	if err := kv.LoadStores(c.core.Stores); err != nil {
		return nil, errors.Trace(err)
	}
	log.Infof("load %v stores cost %v", c.core.Stores.GetStoreCount(), time.Since(start))

	start = time.Now()
	if err := kv.LoadRegions(c.core.Regions); err != nil {
		return nil, errors.Trace(err)
	}
	log.Infof("load %v regions cost %v", c.core.Regions.GetRegionCount(), time.Since(start))

	return c, nil
}

func (c *clusterInfo) allocID() (uint64, error) {
	return c.id.Alloc()
}

// AllocPeer allocs a new peer on a store.
func (c *clusterInfo) AllocPeer(storeID uint64) (*metapb.Peer, error) {
	peerID, err := c.allocID()
	if err != nil {
		log.Errorf("failed to alloc peer: %v", err)
		return nil, errors.Trace(err)
	}
	peer := &metapb.Peer{
		Id:      peerID,
		StoreId: storeID,
	}
	return peer, nil
}

func (c *clusterInfo) getClusterID() uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.meta.GetId()
}

func (c *clusterInfo) getMeta() *metapb.Cluster {
	c.RLock()
	defer c.RUnlock()
	return proto.Clone(c.meta).(*metapb.Cluster)
}

func (c *clusterInfo) putMeta(meta *metapb.Cluster) error {
	c.Lock()
	defer c.Unlock()
	return c.putMetaLocked(proto.Clone(meta).(*metapb.Cluster))
}

func (c *clusterInfo) putMetaLocked(meta *metapb.Cluster) error {
	if c.kv != nil {
		if err := c.kv.SaveMeta(meta); err != nil {
			return errors.Trace(err)
		}
	}
	c.meta = meta
	return nil
}

// GetStore searches for a store by ID.
func (c *clusterInfo) GetStore(storeID uint64) *core.StoreInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.GetStore(storeID)
}

func (c *clusterInfo) putStore(store *core.StoreInfo) error {
	c.Lock()
	defer c.Unlock()
	return c.putStoreLocked(store.Clone())
}

func (c *clusterInfo) putStoreLocked(store *core.StoreInfo) error {
	if c.kv != nil {
		if err := c.kv.SaveStore(store.Store); err != nil {
			return errors.Trace(err)
		}
	}
	return c.core.PutStore(store)
}

// BlockStore stops balancer from selecting the store.
func (c *clusterInfo) BlockStore(storeID uint64) error {
	c.Lock()
	defer c.Unlock()
	return c.core.BlockStore(storeID)
}

// UnblockStore allows balancer to select the store.
func (c *clusterInfo) UnblockStore(storeID uint64) {
	c.Lock()
	defer c.Unlock()
	c.core.UnblockStore(storeID)
}

// GetStores returns all stores in the cluster.
func (c *clusterInfo) GetStores() []*core.StoreInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.GetStores()
}

func (c *clusterInfo) getMetaStores() []*metapb.Store {
	c.RLock()
	defer c.RUnlock()
	return c.core.Stores.GetMetaStores()
}

func (c *clusterInfo) getStoreCount() int {
	c.RLock()
	defer c.RUnlock()
	return c.core.Stores.GetStoreCount()
}

func (c *clusterInfo) getStoresBytesWriteStat() map[uint64]uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.core.Stores.GetStoresBytesWriteStat()
}

func (c *clusterInfo) getStoresBytesReadStat() map[uint64]uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.core.Stores.GetStoresBytesReadStat()
}

func (c *clusterInfo) getStoresKeysWriteStat() map[uint64]uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.core.Stores.GetStoresKeysWriteStat()
}

func (c *clusterInfo) getStoresKeysReadStat() map[uint64]uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.core.Stores.GetStoresKeysReadStat()
}

// ScanRegions scans region with start key, until number greater than limit.
func (c *clusterInfo) ScanRegions(startKey []byte, limit int) []*core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.ScanRange(startKey, limit)
}

// GetAdjacentRegions returns region's info that is adjacent with specific region
func (c *clusterInfo) GetAdjacentRegions(region *core.RegionInfo) (*core.RegionInfo, *core.RegionInfo) {
	c.RLock()
	defer c.RUnlock()
	return c.core.GetAdjacentRegions(region)
}

// GetRegion searches for a region by ID.
func (c *clusterInfo) GetRegion(regionID uint64) *core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.GetRegion(regionID)
}

// IsRegionHot checks if a region is in hot state.
func (c *clusterInfo) IsRegionHot(id uint64) bool {
	c.RLock()
	defer c.RUnlock()
	return c.core.IsRegionHot(id, c.GetHotRegionLowThreshold())
}

// RandHotRegionFromStore randomly picks a hot region in specified store.
func (c *clusterInfo) RandHotRegionFromStore(store uint64, kind schedule.FlowKind) *core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	r := c.core.HotCache.RandHotRegionFromStore(store, kind, c.GetHotRegionLowThreshold())
	if r == nil {
		return nil
	}
	return c.core.GetRegion(r.RegionID)
}

func (c *clusterInfo) searchRegion(regionKey []byte) *core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.SearchRegion(regionKey)
}

func (c *clusterInfo) putRegion(region *core.RegionInfo) error {
	c.Lock()
	defer c.Unlock()
	return c.putRegionLocked(region.Clone())
}

func (c *clusterInfo) putRegionLocked(region *core.RegionInfo) error {
	if c.kv != nil {
		if err := c.kv.SaveRegion(region.Region); err != nil {
			return errors.Trace(err)
		}
	}
	return c.core.PutRegion(region)
}

func (c *clusterInfo) getRegions() []*core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.GetRegions()
}

func (c *clusterInfo) randomRegion(opts ...core.RegionOption) *core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.RandRegion(opts...)
}

func (c *clusterInfo) getMetaRegions() []*metapb.Region {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.GetMetaRegions()
}

func (c *clusterInfo) getRegionCount() int {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.GetRegionCount()
}

func (c *clusterInfo) getRegionStats(startKey, endKey []byte) *core.RegionStats {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.GetRegionStats(startKey, endKey)
}

func (c *clusterInfo) dropRegion(id uint64) {
	c.Lock()
	defer c.Unlock()
	if region := c.core.GetRegion(id); region != nil {
		c.core.Regions.RemoveRegion(region)
	}
}

func (c *clusterInfo) getStoreRegionCount(storeID uint64) int {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.GetStoreRegionCount(storeID)
}

func (c *clusterInfo) getStoreLeaderCount(storeID uint64) int {
	c.RLock()
	defer c.RUnlock()
	return c.core.Regions.GetStoreLeaderCount(storeID)
}

// RandLeaderRegion returns a random region that has leader on the store.
func (c *clusterInfo) RandLeaderRegion(storeID uint64, opts ...core.RegionOption) *core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.RandLeaderRegion(storeID, opts...)
}

// RandFollowerRegion returns a random region that has a follower on the store.
func (c *clusterInfo) RandFollowerRegion(storeID uint64, opts ...core.RegionOption) *core.RegionInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.RandFollowerRegion(storeID, opts...)
}

// GetAverageRegionSize returns the average region approximate size.
func (c *clusterInfo) GetAverageRegionSize() int64 {
	c.RLock()
	defer c.RUnlock()
	return c.core.GetAverageRegionSize()
}

// GetRegionStores returns all stores that contains the region's peer.
func (c *clusterInfo) GetRegionStores(region *core.RegionInfo) []*core.StoreInfo {
	c.RLock()
	defer c.RUnlock()
	return c.getRegionStoresLocked(region)
}

func (c *clusterInfo) getRegionStoresLocked(region *core.RegionInfo) []*core.StoreInfo {
	var stores []*core.StoreInfo
	for id := range region.GetStoreIds() {
		if store := c.core.Stores.GetStore(id); store != nil {
			stores = append(stores, store)
		}
	}
	return stores
}

// GetLeaderStore returns all stores that contains the region's leader peer.
func (c *clusterInfo) GetLeaderStore(region *core.RegionInfo) *core.StoreInfo {
	c.RLock()
	defer c.RUnlock()
	return c.core.Stores.GetStore(region.Leader.GetStoreId())
}

// GetFollowerStores returns all stores that contains the region's follower peer.
func (c *clusterInfo) GetFollowerStores(region *core.RegionInfo) []*core.StoreInfo {
	c.RLock()
	defer c.RUnlock()
	var stores []*core.StoreInfo
	for id := range region.GetFollowers() {
		if store := c.core.Stores.GetStore(id); store != nil {
			stores = append(stores, store)
		}
	}
	return stores
}

// isPrepared if the cluster information is collected
func (c *clusterInfo) isPrepared() bool {
	c.RLock()
	defer c.RUnlock()
	return float64(c.core.Regions.Length())*collectFactor <= float64(c.activeRegions)
}

// handleStoreHeartbeat updates the store status.
func (c *clusterInfo) handleStoreHeartbeat(stats *pdpb.StoreStats) error {
	c.Lock()
	defer c.Unlock()

	storeID := stats.GetStoreId()
	store := c.core.Stores.GetStore(storeID)
	if store == nil {
		return errors.Trace(core.ErrStoreNotFound(storeID))
	}
	store.Stats = proto.Clone(stats).(*pdpb.StoreStats)
	store.LastHeartbeatTS = time.Now()

	c.core.Stores.SetStore(store)
	return nil
}

func (c *clusterInfo) updateStoreStatusLocked(id uint64) {
	c.core.Stores.SetLeaderCount(id, c.core.Regions.GetStoreLeaderCount(id))
	c.core.Stores.SetRegionCount(id, c.core.Regions.GetStoreRegionCount(id))
	c.core.Stores.SetPendingPeerCount(id, c.core.Regions.GetStorePendingPeerCount(id))
	c.core.Stores.SetLeaderSize(id, c.core.Regions.GetStoreLeaderRegionSize(id))
	c.core.Stores.SetRegionSize(id, c.core.Regions.GetStoreRegionSize(id))
}

// handleRegionHeartbeat updates the region information.
func (c *clusterInfo) handleRegionHeartbeat(region *core.RegionInfo) error {
	region = region.Clone()
	c.RLock()
	origin := c.core.Regions.GetRegion(region.GetId())
	isWriteUpdate, writeItem := c.core.CheckWriteStatus(region)
	isReadUpdate, readItem := c.core.CheckReadStatus(region)
	c.RUnlock()

	// Save to KV if meta is updated.
	// Save to cache if meta or leader is updated, or contains any down/pending peer.
	// Mark isNew if the region in cache does not have leader.
	var saveKV, saveCache, isNew bool
	if origin == nil {
		log.Infof("[region %d] Insert new region {%v}", region.GetId(), region)
		saveKV, saveCache, isNew = true, true, true
	} else {
		r := region.GetRegionEpoch()
		o := origin.GetRegionEpoch()
		// Region meta is stale, return an error.
		if r.GetVersion() < o.GetVersion() || r.GetConfVer() < o.GetConfVer() {
			return errors.Trace(ErrRegionIsStale(region.Region, origin.Region))
		}
		if r.GetVersion() > o.GetVersion() {
			log.Infof("[region %d] %s, Version changed from {%d} to {%d}", region.GetId(), core.DiffRegionKeyInfo(origin, region), o.GetVersion(), r.GetVersion())
			saveKV, saveCache = true, true
		}
		if r.GetConfVer() > o.GetConfVer() {
			log.Infof("[region %d] %s, ConfVer changed from {%d} to {%d}", region.GetId(), core.DiffRegionPeersInfo(origin, region), o.GetConfVer(), r.GetConfVer())
			saveKV, saveCache = true, true
		}
		if region.Leader.GetId() != origin.Leader.GetId() {
			log.Infof("[region %d] Leader changed from {%v} to {%v}", region.GetId(), origin.GetPeer(origin.Leader.GetId()), region.GetPeer(region.Leader.GetId()))
			if origin.Leader.GetId() == 0 {
				isNew = true
			}
			saveCache = true
		}
		if len(region.DownPeers) > 0 || len(region.PendingPeers) > 0 {
			saveCache = true
		}
		if len(origin.DownPeers) > 0 || len(origin.PendingPeers) > 0 {
			saveCache = true
		}
		if len(region.GetPeers()) != len(origin.GetPeers()) {
			saveKV, saveCache = true, true
		}
		if region.ApproximateSize != origin.ApproximateSize {
			saveCache = true
		}
		if region.ApproximateRows != origin.ApproximateRows {
			saveCache = true
		}
	}

	if saveKV && c.kv != nil {
		if err := c.kv.SaveRegion(region.Region); err != nil {
			// Not successfully saved to kv is not fatal, it only leads to longer warm-up
			// after restart. Here we only log the error then go on updating cache.
			log.Errorf("[region %d] fail to save region %v: %v", region.GetId(), region, err)
		}
	}
	if !isWriteUpdate && !isReadUpdate && !saveCache && !isNew {
		return nil
	}

	c.Lock()
	defer c.Unlock()
	if isNew {
		c.activeRegions++
	}

	if saveCache {
		overlaps := c.core.Regions.SetRegion(region)
		if c.kv != nil {
			for _, item := range overlaps {
				if err := c.kv.DeleteRegion(item); err != nil {
					log.Errorf("[region %d] fail to delete region %v: %v", item.GetId(), item, err)
				}
			}
		}
		for _, item := range overlaps {
			if c.regionStats != nil {
				c.regionStats.clearDefunctRegion(item.GetId())
			}
			c.labelLevelStats.clearDefunctRegion(item.GetId())
		}

		// Update related stores.
		if origin != nil {
			for _, p := range origin.Peers {
				c.updateStoreStatusLocked(p.GetStoreId())
			}
		}
		for _, p := range region.Peers {
			c.updateStoreStatusLocked(p.GetStoreId())
		}
	}

	if c.regionStats != nil {
		c.regionStats.Observe(region, c.getRegionStoresLocked(region))
	}

	key := region.GetId()
	if isWriteUpdate {
		c.core.HotCache.Update(key, writeItem, schedule.WriteFlow)
	}
	if isReadUpdate {
		c.core.HotCache.Update(key, readItem, schedule.ReadFlow)
	}
	return nil
}

func (c *clusterInfo) updateRegionsLabelLevelStats(regions []*core.RegionInfo) {
	c.Lock()
	defer c.Unlock()
	for _, region := range regions {
		c.labelLevelStats.Observe(region, c.getRegionStoresLocked(region), c.GetLocationLabels())
	}
}

func (c *clusterInfo) collectMetrics() {
	if c.regionStats == nil {
		return
	}
	c.RLock()
	defer c.RUnlock()
	c.regionStats.Collect()
	c.labelLevelStats.Collect()
	// collect hot cache metrics
	c.core.HotCache.CollectMetrics(c.core.Stores)
}

func (c *clusterInfo) GetRegionStatsByType(typ regionStatisticType) []*core.RegionInfo {
	if c.regionStats == nil {
		return nil
	}
	c.RLock()
	defer c.RUnlock()
	return c.regionStats.getRegionStatsByType(typ)
}

func (c *clusterInfo) GetOpt() schedule.NamespaceOptions {
	return c.opt
}

func (c *clusterInfo) GetLeaderScheduleLimit() uint64 {
	return c.opt.GetLeaderScheduleLimit(namespace.DefaultNamespace)
}

func (c *clusterInfo) GetRegionScheduleLimit() uint64 {
	return c.opt.GetRegionScheduleLimit(namespace.DefaultNamespace)
}

func (c *clusterInfo) GetReplicaScheduleLimit() uint64 {
	return c.opt.GetReplicaScheduleLimit(namespace.DefaultNamespace)
}

func (c *clusterInfo) GetMergeScheduleLimit() uint64 {
	return c.opt.GetMergeScheduleLimit(namespace.DefaultNamespace)
}

func (c *clusterInfo) GetTolerantSizeRatio() float64 {
	return c.opt.GetTolerantSizeRatio()
}

func (c *clusterInfo) GetLowSpaceRatio() float64 {
	return c.opt.GetLowSpaceRatio()
}

func (c *clusterInfo) GetHighSpaceRatio() float64 {
	return c.opt.GetHighSpaceRatio()
}

func (c *clusterInfo) GetMaxSnapshotCount() uint64 {
	return c.opt.GetMaxSnapshotCount()
}

func (c *clusterInfo) GetMaxPendingPeerCount() uint64 {
	return c.opt.GetMaxPendingPeerCount()
}

func (c *clusterInfo) GetMaxMergeRegionSize() uint64 {
	return c.opt.GetMaxMergeRegionSize()
}

func (c *clusterInfo) GetMaxMergeRegionRows() uint64 {
	return c.opt.GetMaxMergeRegionRows()
}

func (c *clusterInfo) GetSplitMergeInterval() time.Duration {
	return c.opt.GetSplitMergeInterval()
}

func (c *clusterInfo) GetPatrolRegionInterval() time.Duration {
	return c.opt.GetPatrolRegionInterval()
}

func (c *clusterInfo) GetMaxStoreDownTime() time.Duration {
	return c.opt.GetMaxStoreDownTime()
}

func (c *clusterInfo) GetMaxReplicas() int {
	return c.opt.GetMaxReplicas(namespace.DefaultNamespace)
}

func (c *clusterInfo) GetLocationLabels() []string {
	return c.opt.GetLocationLabels()
}

func (c *clusterInfo) GetHotRegionLowThreshold() int {
	return c.opt.GetHotRegionLowThreshold()
}

func (c *clusterInfo) IsRaftLearnerEnabled() bool {
	return c.opt.IsRaftLearnerEnabled()
}

func (c *clusterInfo) CheckLabelProperty(typ string, labels []*metapb.StoreLabel) bool {
	return c.opt.CheckLabelProperty(typ, labels)
}

// RegionReadStats returns hot region's read stats.
func (c *clusterInfo) RegionReadStats() []*core.RegionStat {
	// RegionStats is a thread-safe method
	return c.core.HotCache.RegionStats(schedule.ReadFlow)
}

// RegionWriteStats returns hot region's write stats.
func (c *clusterInfo) RegionWriteStats() []*core.RegionStat {
	// RegionStats is a thread-safe method
	return c.core.HotCache.RegionStats(schedule.WriteFlow)
}
