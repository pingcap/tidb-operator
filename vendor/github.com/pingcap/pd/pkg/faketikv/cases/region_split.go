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
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/pkg/faketikv/simutil"
	"github.com/pingcap/pd/server/core"
)

func newRegionSplit() *Conf {
	var conf Conf
	// Initialize the cluster
	for i := 1; i <= 3; i++ {
		conf.Stores = append(conf.Stores, Store{
			ID:        uint64(i),
			Status:    metapb.StoreState_Up,
			Capacity:  10 * gb,
			Available: 9 * gb,
		})
	}
	peers := []*metapb.Peer{
		{Id: 4, StoreId: 1},
	}
	conf.Regions = append(conf.Regions, Region{
		ID:     5,
		Peers:  peers,
		Leader: peers[0],
		Size:   1 * mb,
		Rows:   10000,
	})
	conf.MaxID = 5
	conf.RegionSplitSize = 128 * mb
	conf.RegionSplitRows = 10000
	// Events description
	e := &WriteFlowOnSpotInner{}
	e.Step = func(tick int64) map[string]int64 {
		return map[string]int64{
			"foobar": 8 * mb,
		}
	}
	conf.Events = []EventInner{e}

	// Checker description
	conf.Checker = func(regions *core.RegionsInfo) bool {
		count1 := regions.GetStoreRegionCount(1)
		count2 := regions.GetStoreRegionCount(2)
		count3 := regions.GetStoreRegionCount(3)
		simutil.Logger.Infof("region counts: %v %v %v", count1, count2, count3)
		return count1 > 5 && count2 > 5 && count3 > 5
	}
	return &conf
}
