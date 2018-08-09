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

package cases

import (
	"math/rand"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/pkg/faketikv/simutil"
	"github.com/pingcap/pd/server/core"
)

func newHotWrite() *Conf {
	var conf Conf
	// Initialize the cluster
	for i := 1; i <= 10; i++ {
		conf.Stores = append(conf.Stores, Store{
			ID:        uint64(i),
			Status:    metapb.StoreState_Up,
			Capacity:  10 * gb,
			Available: 9 * gb,
		})
	}
	var id idAllocator
	id.setMaxID(10)
	for i := 0; i < 500; i++ {
		storeIDs := rand.Perm(10)
		peers := []*metapb.Peer{
			{Id: id.nextID(), StoreId: uint64(storeIDs[0] + 1)},
			{Id: id.nextID(), StoreId: uint64(storeIDs[1] + 1)},
			{Id: id.nextID(), StoreId: uint64(storeIDs[2] + 1)},
		}
		conf.Regions = append(conf.Regions, Region{
			ID:     id.nextID(),
			Peers:  peers,
			Leader: peers[0],
			Size:   96 * mb,
			Rows:   960000,
		})
	}
	conf.MaxID = id.maxID

	// Events description
	// select 5 reigons on store 1 as hot write regions.
	writeFlow := make(map[uint64]int64, 5)
	for _, r := range conf.Regions {
		if r.Leader.GetStoreId() == 1 {
			writeFlow[r.ID] = 2 * mb
			if len(writeFlow) == 5 {
				break
			}
		}
	}
	e := &WriteFlowOnRegionInner{}
	e.Step = func(tick int64) map[uint64]int64 {
		return writeFlow
	}

	conf.Events = []EventInner{e}

	// Checker description
	conf.Checker = func(regions *core.RegionsInfo) bool {
		var leaderCount, peerCount [10]int
		for id := range writeFlow {
			region := regions.GetRegion(id)
			leaderCount[int(region.Leader.GetStoreId()-1)]++
			for _, p := range region.Peers {
				peerCount[int(p.GetStoreId()-1)]++
			}
		}
		simutil.Logger.Infof("hot region leader count: %v, peer count: %v", leaderCount, peerCount)

		// check count diff <= 2.
		var minLeader, maxLeader, minPeer, maxPeer int
		for i := range leaderCount {
			if leaderCount[i] > leaderCount[maxLeader] {
				maxLeader = i
			}
			if leaderCount[i] < leaderCount[minLeader] {
				minLeader = i
			}
			if peerCount[i] > peerCount[maxPeer] {
				maxPeer = i
			}
			if peerCount[i] < peerCount[minPeer] {
				minPeer = i
			}
		}
		return leaderCount[maxLeader]-leaderCount[minLeader] <= 2 && peerCount[maxPeer]-peerCount[minPeer] <= 2
	}

	return &conf
}
