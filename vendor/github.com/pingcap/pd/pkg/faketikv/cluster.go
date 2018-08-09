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

package faketikv

import (
	"context"
	"fmt"
	"math/rand"
	"sort"

	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/pkg/faketikv/cases"
	"github.com/pingcap/pd/pkg/faketikv/simutil"
	"github.com/pingcap/pd/server/core"
)

// ClusterInfo records all cluster information.
type ClusterInfo struct {
	conf *cases.Conf
	*core.RegionsInfo
	Nodes map[uint64]*Node
}

// NewClusterInfo creates the initialized cluster with config.
func NewClusterInfo(pdAddr string, conf *cases.Conf) (*ClusterInfo, error) {
	cluster := &ClusterInfo{
		conf:        conf,
		RegionsInfo: core.NewRegionsInfo(),
		Nodes:       make(map[uint64]*Node),
	}

	for _, store := range conf.Stores {
		node, err := NewNode(store.ID, fmt.Sprintf("mock:://tikv-%d", store.ID), pdAddr)
		if err != nil {
			return nil, errors.Trace(err)
		}
		node.clusterInfo = cluster
		cluster.Nodes[store.ID] = node
	}

	splitKeys := generateKeys(len(conf.Regions) - 1)
	for i, region := range conf.Regions {
		meta := &metapb.Region{
			Id:          region.ID,
			Peers:       region.Peers,
			RegionEpoch: &metapb.RegionEpoch{ConfVer: 1, Version: 1},
		}
		if i > 0 {
			meta.StartKey = []byte(splitKeys[i-1])
		}
		if i < len(conf.Regions)-1 {
			meta.EndKey = []byte(splitKeys[i])
		}
		regionInfo := core.NewRegionInfo(meta, region.Leader)
		regionInfo.ApproximateSize = region.Size
		regionInfo.ApproximateRows = region.Rows
		cluster.RegionsInfo.SetRegion(regionInfo)
	}

	return cluster, nil
}

// GetBootstrapInfo returns a valid bootstrap store and region.
func (c *ClusterInfo) GetBootstrapInfo() (*metapb.Store, *metapb.Region, error) {
	origin := c.RegionsInfo.RandRegion()
	if origin == nil {
		return nil, nil, errors.New("no region found for bootstrap")
	}
	region := origin.Clone()
	if region.Leader == nil {
		return nil, nil, errors.New("bootstrap region has no leader")
	}
	store := c.Nodes[region.Leader.GetStoreId()]
	if store == nil {
		return nil, nil, errors.Errorf("bootstrap store %v not found", region.Leader.GetStoreId())
	}
	region.StartKey, region.EndKey = []byte(""), []byte("")
	region.RegionEpoch = &metapb.RegionEpoch{}
	region.Peers = []*metapb.Peer{region.Leader}
	return store.Store, region.Region, nil
}

func (c *ClusterInfo) nodeHealth(storeID uint64) bool {
	n, ok := c.Nodes[storeID]
	if !ok {
		return false
	}

	return n.GetState() == Up
}

func (c *ClusterInfo) electNewLeader(region *core.RegionInfo) *metapb.Peer {
	var (
		unhealth         int
		newLeaderStoreID uint64
	)
	ids := region.GetStoreIds()
	for id := range ids {
		if c.nodeHealth(id) {
			newLeaderStoreID = id
		} else {
			unhealth++
		}
	}
	if unhealth > len(ids)/2 {
		return nil
	}
	for _, peer := range region.Peers {
		if peer.GetStoreId() == newLeaderStoreID {
			return peer
		}
	}
	return nil
}

func (c *ClusterInfo) stepLeader(region *core.RegionInfo) {
	if region.Leader != nil && c.nodeHealth(region.Leader.GetStoreId()) {
		return
	}
	newLeader := c.electNewLeader(region)
	region.Leader = newLeader
	if newLeader == nil {
		c.SetRegion(region)
		simutil.Logger.Infof("[region %d] no leader", region.GetId())
		return
	}
	simutil.Logger.Infof("[region %d] elect new leader: %+v,old leader: %+v", region.GetId(), newLeader, region.Leader)
	c.SetRegion(region)
	c.reportRegionChange(region.GetId())
}

func (c *ClusterInfo) stepSplit(region *core.RegionInfo) {
	if region.Leader == nil {
		return
	}
	if !c.conf.NeedSplit(region.ApproximateSize, region.ApproximateRows) {
		return
	}
	ids := make([]uint64, 1+len(region.Peers))
	for i := range ids {
		var err error
		ids[i], err = c.allocID(region.Leader.GetStoreId())
		if err != nil {
			simutil.Logger.Infof("alloc id failed: %s", err)
			return
		}
	}

	region.RegionEpoch.Version++
	region.ApproximateSize /= 2
	region.ApproximateRows /= 2

	newRegion := region.Clone()
	newRegion.PendingPeers, newRegion.DownPeers = nil, nil
	for i, peer := range newRegion.Peers {
		peer.Id = ids[i]
	}
	newRegion.Id = ids[len(ids)-1]

	splitKey := generateSplitKey(region.StartKey, region.EndKey)
	newRegion.EndKey, region.StartKey = splitKey, splitKey

	c.SetRegion(region)
	c.SetRegion(newRegion)
	c.reportRegionChange(region.Id)
	c.reportRegionChange(newRegion.Id)
}

func (c *ClusterInfo) reportRegionChange(regionID uint64) {
	region := c.GetRegion(regionID)
	if n, ok := c.Nodes[region.Leader.GetStoreId()]; ok {
		n.reportRegionChange(region.GetId())
	}
}

func (c *ClusterInfo) stepRegions() {
	regions := c.GetRegions()
	for _, region := range regions {
		c.stepLeader(region)
		c.stepSplit(region)
	}
}

func (c *ClusterInfo) updateRegionSize(writtenBytes map[string]int64) {
	for key, size := range writtenBytes {
		region := c.SearchRegion([]byte(key))
		if region == nil {
			simutil.Logger.Errorf("region not found for key %q", key)
			continue
		}
		region.ApproximateSize += size
		c.SetRegion(region)
	}
}

func (c *ClusterInfo) updateRegionWriteBytes(writeBytes map[uint64]int64) {
	for id, bytes := range writeBytes {
		region := c.GetRegion(id)
		if region == nil {
			simutil.Logger.Errorf("region %d not found", id)
			continue
		}
		region.WrittenBytes = uint64(bytes)
		c.SetRegion(region)
	}
}

func (c *ClusterInfo) updateRegionReadBytes(readBytes map[uint64]int64) {
	for id, bytes := range readBytes {
		region := c.GetRegion(id)
		if region == nil {
			simutil.Logger.Errorf("region %d not found", id)
			continue
		}
		region.ReadBytes = uint64(bytes)
		c.SetRegion(region)
	}
}

// AddTask adds task in specify node.
func (c *ClusterInfo) AddTask(task Task) {
	storeID := task.TargetStoreID()
	if n, ok := c.Nodes[storeID]; ok {
		n.AddTask(task)
	}
}

func (c *ClusterInfo) allocID(storeID uint64) (uint64, error) {
	node, ok := c.Nodes[storeID]
	if !ok {
		return 0, errors.Errorf("node %d not found", storeID)
	}
	id, err := node.client.AllocID(context.Background())
	return id, errors.Trace(err)
}

const (
	// 26^10 ~= 1.4e+14, should be enough.
	keyChars = "abcdefghijklmnopqrstuvwxyz"
	keyLen   = 10
)

// generate ordered, unique strings.
func generateKeys(size int) []string {
	m := make(map[string]struct{}, size)
	for len(m) < size {
		k := make([]byte, keyLen)
		for i := range k {
			k[i] = keyChars[rand.Intn(len(keyChars))]
		}
		m[string(k)] = struct{}{}
	}

	v := make([]string, 0, size)
	for k := range m {
		v = append(v, k)
	}
	sort.Sort(sort.StringSlice(v))
	return v
}

func generateSplitKey(start, end []byte) []byte {
	var key []byte
	// lessThanEnd is set as true when the key is already less than end key.
	lessThanEnd := len(end) == 0
	for i, s := range start {
		e := byte('z')
		if !lessThanEnd {
			e = end[i]
		}
		c := (s + e) / 2
		key = append(key, c)
		// case1: s = c < e. Continue with lessThanEnd=true.
		// case2: s < c < e. return key.
		// case3: s = c = e. Continue with lessThanEnd=false.
		lessThanEnd = c < e
		if c > s && c < e {
			return key
		}
	}
	key = append(key, ('a'+'z')/2)
	return key
}
