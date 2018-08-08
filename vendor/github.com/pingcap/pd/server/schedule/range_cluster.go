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
	"bytes"

	"github.com/pingcap/pd/server/core"
)

// RangeCluster isolates the cluster by range.
type RangeCluster struct {
	Cluster
	regions           *core.RegionsInfo
	tolerantSizeRatio float64
}

const scanLimit = 128

// GenRangeCluster gets a range cluster by specifying start key and end key.
func GenRangeCluster(cluster Cluster, startKey, endKey []byte) *RangeCluster {
	regions := core.NewRegionsInfo()
	scanKey := startKey
	loopEnd := false
	for !loopEnd {
		collect := cluster.ScanRegions(scanKey, scanLimit)
		if len(collect) == 0 {
			break
		}
		for _, r := range collect {
			if bytes.Compare(r.StartKey, endKey) < 0 {
				regions.SetRegion(r)
			} else {
				loopEnd = true
				break
			}
			if string(r.EndKey) == "" {
				loopEnd = true
				break
			}
			scanKey = r.EndKey
		}
	}
	return &RangeCluster{
		Cluster: cluster,
		regions: regions,
	}
}

func (r *RangeCluster) updateStoreInfo(s *core.StoreInfo) {
	id := s.GetId()

	used := float64(s.Stats.GetUsedSize()) / (1 << 20)
	if used == 0 {
		return
	}
	amplification := float64(s.RegionSize) / used
	s.LeaderCount = r.regions.GetStoreLeaderCount(id)
	s.LeaderSize = r.regions.GetStoreLeaderRegionSize(id)
	s.RegionCount = r.regions.GetStoreRegionCount(id)
	s.RegionSize = r.regions.GetStoreRegionSize(id)
	s.PendingPeerCount = r.regions.GetStorePendingPeerCount(id)
	s.Stats.UsedSize = uint64(float64(s.RegionSize)/amplification) * (1 << 20)
	s.Stats.Available = s.Stats.GetCapacity() - s.Stats.GetUsedSize()
}

// GetStore searches for a store by ID.
func (r *RangeCluster) GetStore(id uint64) *core.StoreInfo {
	s := r.Cluster.GetStore(id)
	r.updateStoreInfo(s)
	return s
}

// GetStores returns all Stores in the cluster.
func (r *RangeCluster) GetStores() []*core.StoreInfo {
	stores := r.Cluster.GetStores()
	for _, s := range stores {
		r.updateStoreInfo(s)
	}
	return stores
}

// SetTolerantSizeRatio sets the tolerant size ratio.
func (r *RangeCluster) SetTolerantSizeRatio(ratio float64) {
	r.tolerantSizeRatio = ratio
}

// GetTolerantSizeRatio gets the tolerant size ratio.
func (r *RangeCluster) GetTolerantSizeRatio() float64 {
	if r.tolerantSizeRatio != 0 {
		return r.tolerantSizeRatio
	}
	return r.Cluster.GetTolerantSizeRatio()
}

// RandFollowerRegion returns a random region that has a follower on the store.
func (r *RangeCluster) RandFollowerRegion(storeID uint64, opts ...core.RegionOption) *core.RegionInfo {
	return r.regions.RandFollowerRegion(storeID, opts...)
}

// RandLeaderRegion returns a random region that has leader on the store.
func (r *RangeCluster) RandLeaderRegion(storeID uint64, opts ...core.RegionOption) *core.RegionInfo {
	return r.regions.RandLeaderRegion(storeID, opts...)
}

// GetAverageRegionSize returns the average region approximate size.
func (r *RangeCluster) GetAverageRegionSize() int64 {
	return r.regions.GetAverageRegionSize()
}

// GetRegionStores returns all stores that contains the region's peer.
func (r *RangeCluster) GetRegionStores(region *core.RegionInfo) []*core.StoreInfo {
	stores := r.Cluster.GetRegionStores(region)
	for _, s := range stores {
		r.updateStoreInfo(s)
	}
	return stores
}

// GetFollowerStores returns all stores that contains the region's follower peer.
func (r *RangeCluster) GetFollowerStores(region *core.RegionInfo) []*core.StoreInfo {
	stores := r.Cluster.GetFollowerStores(region)
	for _, s := range stores {
		r.updateStoreInfo(s)
	}
	return stores
}

// GetLeaderStore returns all stores that contains the region's leader peer.
func (r *RangeCluster) GetLeaderStore(region *core.RegionInfo) *core.StoreInfo {
	s := r.Cluster.GetLeaderStore(region)
	r.updateStoreInfo(s)
	return s
}
