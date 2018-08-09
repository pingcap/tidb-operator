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

func newHotRead() *Conf {
	var conf Conf
	// Initialize the cluster
	for i := 1; i <= 5; i++ {
		conf.Stores = append(conf.Stores, Store{
			ID:        uint64(i),
			Status:    metapb.StoreState_Up,
			Capacity:  10 * gb,
			Available: 9 * gb,
		})
	}
	var id idAllocator
	id.setMaxID(5)
	for i := 0; i < 500; i++ {
		storeIDs := rand.Perm(5)
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
	// select 20 reigons on store 1 as hot read regions.
	readFlow := make(map[uint64]int64, 20)
	for _, r := range conf.Regions {
		if r.Leader.GetStoreId() == 1 {
			readFlow[r.ID] = 128 * mb
			if len(readFlow) == 20 {
				break
			}
		}
	}
	e := &ReadFlowOnRegionInner{}
	e.Step = func(tick int64) map[uint64]int64 {
		return readFlow
	}
	conf.Events = []EventInner{e}
	// Checker description
	conf.Checker = func(regions *core.RegionsInfo) bool {
		var leaderCount [5]int
		for id := range readFlow {
			leaderStore := regions.GetRegion(id).Leader.GetStoreId()
			leaderCount[int(leaderStore-1)]++
		}
		simutil.Logger.Infof("hot region count: %v", leaderCount)

		// check count diff < 2.
		var min, max int
		for i := range leaderCount {
			if leaderCount[i] > leaderCount[max] {
				max = i
			}
			if leaderCount[i] < leaderCount[min] {
				min = i
			}
		}
		return leaderCount[max]-leaderCount[min] < 2
	}

	return &conf
}
