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

package schedulers

import (
	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/pkg/testutil"
	"github.com/pingcap/pd/server/namespace"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

var _ = Suite(&testShuffleLeaderSuite{})

type testShuffleLeaderSuite struct{}

func (s *testShuffleLeaderSuite) TestShuffle(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	sl, err := schedule.CreateScheduler("shuffle-leader", schedule.NewLimiter())
	c.Assert(err, IsNil)
	c.Assert(sl.Schedule(tc, schedule.OpInfluence{}), IsNil)

	// Add stores 1,2,3,4
	tc.AddLeaderStore(1, 6)
	tc.AddLeaderStore(2, 7)
	tc.AddLeaderStore(3, 8)
	tc.AddLeaderStore(4, 9)
	// Add regions 1,2,3,4 with leaders in stores 1,2,3,4
	tc.AddLeaderRegion(1, 1, 2, 3, 4)
	tc.AddLeaderRegion(2, 2, 3, 4, 1)
	tc.AddLeaderRegion(3, 3, 4, 1, 2)
	tc.AddLeaderRegion(4, 4, 1, 2, 3)

	for i := 0; i < 4; i++ {
		op := sl.Schedule(tc, schedule.NewOpInfluence(nil, tc))
		c.Assert(op, NotNil)
		c.Assert(op[0].Kind(), Equals, schedule.OpLeader|schedule.OpAdmin)
	}
}

var _ = Suite(&testBalanceAdjacentRegionSuite{})

type testBalanceAdjacentRegionSuite struct{}

func (s *testBalanceAdjacentRegionSuite) TestBalance(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	sc, err := schedule.CreateScheduler("adjacent-region", schedule.NewLimiter())
	c.Assert(err, IsNil)
	c.Assert(sc.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)

	// Add stores 1,2,3,4
	tc.AddLeaderStore(1, 5)
	tc.AddLeaderStore(2, 0)
	tc.AddLeaderStore(3, 0)
	tc.AddLeaderStore(4, 0)
	// Add regions
	tc.AddLeaderRegionWithRange(1, "", "a", 1, 2, 3)
	tc.AddLeaderRegionWithRange(2, "a", "b", 1, 2, 3)
	tc.AddLeaderRegionWithRange(3, "b", "c", 1, 3, 4)
	tc.AddLeaderRegionWithRange(4, "c", "d", 1, 2, 3)
	tc.AddLeaderRegionWithRange(5, "e", "f", 1, 2, 3)
	tc.AddLeaderRegionWithRange(6, "f", "g", 1, 2, 3)
	tc.AddLeaderRegionWithRange(7, "z", "", 1, 2, 3)

	// check and do operator
	// transfer peer from store 1 to 4 for region 1 because the distribution of
	// the two regions is same, we will transfer the peer, which is leader now,
	// to a new store
	testutil.CheckTransferPeerWithLeaderTransfer(c, sc.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpAdjacent, 1, 4)
	// suppose we add peer in store 4, transfer leader to store 2, remove peer in store 1
	tc.AddLeaderRegionWithRange(1, "", "a", 2, 3, 4)

	// transfer leader from store 1 to store 2 for region 2 because we have a different peer location,
	// we can directly transfer leader to peer 2. we priority to transfer leader because less overhead
	testutil.CheckTransferLeader(c, sc.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpAdjacent, 1, 2)
	tc.AddLeaderRegionWithRange(2, "a", "b", 2, 1, 3)

	// transfer leader from store 1 to store 2 for region 3
	testutil.CheckTransferLeader(c, sc.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpAdjacent, 1, 4)
	tc.AddLeaderRegionWithRange(3, "b", "c", 4, 1, 3)

	// transfer peer from store 1 to store 4 for region 5
	// the region 5 just adjacent the region 6
	testutil.CheckTransferPeerWithLeaderTransfer(c, sc.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpAdjacent, 1, 4)
	tc.AddLeaderRegionWithRange(5, "e", "f", 2, 3, 4)

	c.Assert(sc.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)
	c.Assert(sc.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)
	testutil.CheckTransferLeader(c, sc.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpAdjacent, 2, 4)
	tc.AddLeaderRegionWithRange(1, "", "a", 4, 2, 3)
	for i := 0; i < 10; i++ {
		c.Assert(sc.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)
	}
}

type sequencer struct {
	maxID uint64
	curID uint64
}

func newSequencer(maxID uint64) *sequencer {
	return &sequencer{
		maxID: maxID,
		curID: 0,
	}
}

func (s *sequencer) next() uint64 {
	s.curID++
	if s.curID > s.maxID {
		s.curID = 1
	}
	return s.curID
}

var _ = Suite(&testScatterRegionSuite{})

type testScatterRegionSuite struct{}

func (s *testScatterRegionSuite) TestSixStores(c *C) {
	s.scatter(c, 6, 4)
}

func (s *testScatterRegionSuite) TestFiveStores(c *C) {
	s.scatter(c, 5, 5)
}

func (s *testScatterRegionSuite) scatter(c *C, numStores, numRegions uint64) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	// Add stores 1~6.
	for i := uint64(1); i <= numStores; i++ {
		tc.AddRegionStore(i, 0)
	}

	// Add regions 1~4.
	seq := newSequencer(numStores)
	for i := uint64(1); i <= numRegions; i++ {
		tc.AddLeaderRegion(i, seq.next(), seq.next(), seq.next())
	}

	scatterer := schedule.NewRegionScatterer(tc, namespace.DefaultClassifier)

	for i := uint64(1); i <= numRegions; i++ {
		region := tc.GetRegion(i)
		if op := scatterer.Scatter(region); op != nil {
			log.Info(op)
			tc.ApplyOperator(op)
		}
	}

	countPeers := make(map[uint64]uint64)
	for i := uint64(1); i <= numRegions; i++ {
		region := tc.GetRegion(i)
		for _, peer := range region.GetPeers() {
			countPeers[peer.GetStoreId()]++
		}
	}

	// Each store should have the same number of peers.
	for _, count := range countPeers {
		c.Assert(count, Equals, numRegions*3/numStores)
	}
}

var _ = Suite(&testRejectLeaderSuite{})

type testRejectLeaderSuite struct{}

func (s *testRejectLeaderSuite) TestRejectLeader(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	opt.LabelProperties = map[string][]*metapb.StoreLabel{
		schedule.RejectLeader: {{Key: "noleader", Value: "true"}},
	}
	tc := schedule.NewMockCluster(opt)

	// Add 3 stores 1,2,3.
	tc.AddLabelsStore(1, 1, map[string]string{"noleader": "true"})
	tc.UpdateLeaderCount(1, 1)
	tc.AddLeaderStore(2, 10)
	tc.AddLeaderStore(3, 0)
	// Add 2 regions with leader on 1 and 2.
	tc.AddLeaderRegion(1, 1, 2, 3)
	tc.AddLeaderRegion(2, 2, 1, 3)

	// The label scheduler transfers leader out of store1.
	sl, err := schedule.CreateScheduler("label", schedule.NewLimiter())
	c.Assert(err, IsNil)
	op := sl.Schedule(tc, schedule.NewOpInfluence(nil, tc))
	testutil.CheckTransferLeader(c, op[0], schedule.OpLeader, 1, 3)

	// If store3 is disconnected, transfer leader to store 2 instead.
	tc.SetStoreDisconnect(3)
	op = sl.Schedule(tc, schedule.NewOpInfluence(nil, tc))
	testutil.CheckTransferLeader(c, op[0], schedule.OpLeader, 1, 2)

	// As store3 is disconnected, store1 rejects leader. Balancer will not create
	// any operators.
	bs, err := schedule.CreateScheduler("balance-leader", schedule.NewLimiter())
	c.Assert(err, IsNil)
	op = bs.Schedule(tc, schedule.NewOpInfluence(nil, tc))
	c.Assert(op, IsNil)

	// Can't evict leader from store2, neither.
	el, err := schedule.CreateScheduler("evict-leader", schedule.NewLimiter(), "2")
	c.Assert(err, IsNil)
	op = el.Schedule(tc, schedule.NewOpInfluence(nil, tc))
	c.Assert(op, IsNil)

	// If the peer on store3 is pending, not trasnfer to store3 neither.
	tc.SetStoreUp(3)
	region := tc.Regions.GetRegion(1)
	for _, p := range region.Peers {
		if p.GetStoreId() == 3 {
			region.PendingPeers = append(region.PendingPeers, p)
			break
		}
	}
	tc.Regions.AddRegion(region)
	op = sl.Schedule(tc, schedule.NewOpInfluence(nil, tc))
	testutil.CheckTransferLeader(c, op[0], schedule.OpLeader, 1, 2)
}
