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

package cases

import (
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/pkg/faketikv/simutil"
	"github.com/pingcap/pd/server/core"
)

func newBalanceLeader() *Conf {
	var conf Conf

	for i := 1; i <= 3; i++ {
		conf.Stores = append(conf.Stores, Store{
			ID:        uint64(i),
			Status:    metapb.StoreState_Up,
			Capacity:  10 * gb,
			Available: 9 * gb,
		})
	}

	var id idAllocator
	id.setMaxID(3)
	for i := 0; i < 1000; i++ {
		peers := []*metapb.Peer{
			{Id: id.nextID(), StoreId: 1},
			{Id: id.nextID(), StoreId: 2},
			{Id: id.nextID(), StoreId: 3},
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
	conf.Checker = func(regions *core.RegionsInfo) bool {
		count1 := regions.GetStoreLeaderCount(1)
		count2 := regions.GetStoreLeaderCount(2)
		count3 := regions.GetStoreLeaderCount(3)
		simutil.Logger.Infof("leader counts: %v %v %v", count1, count2, count3)

		return count1 <= 350 &&
			count2 >= 300 &&
			count3 >= 300
	}
	return &conf
}
