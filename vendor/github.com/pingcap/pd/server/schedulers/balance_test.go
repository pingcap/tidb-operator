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

package schedulers

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/pkg/testutil"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/namespace"
	"github.com/pingcap/pd/server/schedule"
)

func newTestReplication(mso *schedule.MockSchedulerOptions, maxReplicas int, locationLabels ...string) {
	mso.MaxReplicas = maxReplicas
	mso.LocationLabels = locationLabels
}

var _ = Suite(&testBalanceSpeedSuite{})

type testBalanceSpeedSuite struct{}

type testBalanceSpeedCase struct {
	sourceCount    uint64
	targetCount    uint64
	regionSize     int64
	expectedResult bool
}

func (s *testBalanceSpeedSuite) TestShouldBalance(c *C) {
	tests := []testBalanceSpeedCase{
		// all store capacity is 1024MB
		// size = count * 10

		// target size is zero
		{2, 0, 1, true},
		{2, 0, 10, false},
		// all in high space stage
		{10, 5, 1, true},
		{10, 5, 20, false},
		{10, 10, 1, false},
		{10, 10, 20, false},
		// all in transition stage
		{70, 50, 1, true},
		{70, 50, 50, false},
		{70, 70, 1, false},
		// all in low space stage
		{90, 80, 1, true},
		{90, 80, 50, false},
		{90, 90, 1, false},
		// one in high space stage, other in transition stage
		{65, 55, 5, true},
		{65, 50, 50, false},
		// one in transition space stage, other in low space stage
		{80, 70, 5, true},
		{80, 70, 50, false},
	}

	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)
	// create a region to control average region size.
	tc.AddLeaderRegion(1, 1, 2)

	for _, t := range tests {
		tc.AddLeaderStore(1, int(t.sourceCount))
		tc.AddLeaderStore(2, int(t.targetCount))
		source := tc.GetStore(1)
		target := tc.GetStore(2)
		region := tc.GetRegion(1)
		region.ApproximateSize = t.regionSize
		tc.PutRegion(region.Clone())
		c.Assert(shouldBalance(tc, source, target, region, core.LeaderKind, schedule.NewOpInfluence(nil, tc)), Equals, t.expectedResult)
	}

	for _, t := range tests {
		tc.AddRegionStore(1, int(t.sourceCount))
		tc.AddRegionStore(2, int(t.targetCount))
		source := tc.GetStore(1)
		target := tc.GetStore(2)
		region := tc.GetRegion(1)
		region.ApproximateSize = t.regionSize
		tc.PutRegion(region)
		c.Assert(shouldBalance(tc, source, target, region, core.RegionKind, schedule.NewOpInfluence(nil, tc)), Equals, t.expectedResult)
	}
}

func (s *testBalanceSpeedSuite) TestBalanceLimit(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)
	tc.AddLeaderStore(1, 10)
	tc.AddLeaderStore(2, 20)
	tc.AddLeaderStore(3, 30)

	// StandDeviation is sqrt((10^2+0+10^2)/3).
	c.Assert(adjustBalanceLimit(tc, core.LeaderKind), Equals, uint64(math.Sqrt(200.0/3.0)))

	tc.SetStoreOffline(1)
	// StandDeviation is sqrt((5^2+5^2)/2).
	c.Assert(adjustBalanceLimit(tc, core.LeaderKind), Equals, uint64(math.Sqrt(50.0/2.0)))
}

var _ = Suite(&testBalanceLeaderSchedulerSuite{})

type testBalanceLeaderSchedulerSuite struct {
	tc *schedule.MockCluster
	lb schedule.Scheduler
}

func (s *testBalanceLeaderSchedulerSuite) SetUpTest(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	s.tc = schedule.NewMockCluster(opt)
	lb, err := schedule.CreateScheduler("balance-leader", schedule.NewLimiter())
	c.Assert(err, IsNil)
	s.lb = lb
}

func (s *testBalanceLeaderSchedulerSuite) schedule(operators []*schedule.Operator) []*schedule.Operator {
	return s.lb.Schedule(s.tc, schedule.NewOpInfluence(operators, s.tc))
}

func (s *testBalanceLeaderSchedulerSuite) TestBalanceLimit(c *C) {
	// Stores:     1    2    3    4
	// Leaders:    1    0    0    0
	// Region1:    L    F    F    F
	s.tc.AddLeaderStore(1, 1)
	s.tc.AddLeaderStore(2, 0)
	s.tc.AddLeaderStore(3, 0)
	s.tc.AddLeaderStore(4, 0)
	s.tc.AddLeaderRegion(1, 1, 2, 3, 4)
	c.Check(s.schedule(nil), IsNil)

	// Stores:     1    2    3    4
	// Leaders:    16   0    0    0
	// Region1:    L    F    F    F
	s.tc.UpdateLeaderCount(1, 16)
	c.Check(s.schedule(nil), NotNil)

	// Stores:     1    2    3    4
	// Leaders:    7    8    9   10
	// Region1:    F    F    F    L
	s.tc.UpdateLeaderCount(1, 7)
	s.tc.UpdateLeaderCount(2, 8)
	s.tc.UpdateLeaderCount(3, 9)
	s.tc.UpdateLeaderCount(4, 10)
	s.tc.AddLeaderRegion(1, 4, 1, 2, 3)
	c.Check(s.schedule(nil), IsNil)

	// Stores:     1    2    3    4
	// Leaders:    7    8    9   16
	// Region1:    F    F    F    L
	s.tc.UpdateLeaderCount(4, 16)
	c.Check(s.schedule(nil), NotNil)
}

func (s *testBalanceLeaderSchedulerSuite) TestScheduleWithOpInfluence(c *C) {
	// Stores:     1    2    3    4
	// Leaders:    7    8    9   14
	// Region1:    F    F    F    L
	s.tc.AddLeaderStore(1, 7)
	s.tc.AddLeaderStore(2, 8)
	s.tc.AddLeaderStore(3, 9)
	s.tc.AddLeaderStore(4, 14)
	s.tc.AddLeaderRegion(1, 4, 1, 2, 3)
	op := s.schedule(nil)[0]
	c.Check(op, NotNil)
	// After considering the scheduled operator, leaders of store1 and store4 are 8
	// and 13 respectively. As the `TolerantSizeRatio` is 2.5, `shouldBalance`
	// returns false when leader differece is not greater than 5.
	c.Check(s.schedule([]*schedule.Operator{op}), IsNil)

	// Stores:     1    2    3    4
	// Leaders:    8    8    9   13
	// Region1:    F    F    F    L
	s.tc.UpdateLeaderCount(1, 8)
	s.tc.UpdateLeaderCount(2, 8)
	s.tc.UpdateLeaderCount(3, 9)
	s.tc.UpdateLeaderCount(4, 13)
	s.tc.AddLeaderRegion(1, 4, 1, 2, 3)
	c.Check(s.schedule(nil), IsNil)
}

func (s *testBalanceLeaderSchedulerSuite) TestBalanceFilter(c *C) {
	// Stores:     1    2    3    4
	// Leaders:    1    2    3   16
	// Region1:    F    F    F    L
	s.tc.AddLeaderStore(1, 1)
	s.tc.AddLeaderStore(2, 2)
	s.tc.AddLeaderStore(3, 3)
	s.tc.AddLeaderStore(4, 16)
	s.tc.AddLeaderRegion(1, 4, 1, 2, 3)

	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 4, 1)
	// Test stateFilter.
	// if store 4 is offline, we schould consider it
	// because it still provides services
	s.tc.SetStoreOffline(4)
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 4, 1)
	// If store 1 is down, it will be filtered,
	// store 2 becomes the store with least leaders.
	s.tc.SetStoreDown(1)
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 4, 2)

	// Test healthFilter.
	// If store 2 is busy, it will be filtered,
	// store 3 becomes the store with least leaders.
	s.tc.SetStoreBusy(2, true)
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 4, 3)

	// Test disconnectFilter.
	// If store 3 is disconnected, no operator can be created.
	s.tc.SetStoreDisconnect(3)
	c.Assert(s.schedule(nil), HasLen, 0)
}

func (s *testBalanceLeaderSchedulerSuite) TestLeaderWeight(c *C) {
	// Stores:	1	2	3	4
	// Leaders:    10      10      10      10
	// Weight:    0.5     0.9       1       2
	// Region1:     L       F       F       F

	s.tc.AddLeaderStore(1, 10)
	s.tc.AddLeaderStore(2, 10)
	s.tc.AddLeaderStore(3, 10)
	s.tc.AddLeaderStore(4, 10)
	s.tc.UpdateStoreLeaderWeight(1, 0.5)
	s.tc.UpdateStoreLeaderWeight(2, 0.9)
	s.tc.UpdateStoreLeaderWeight(3, 1)
	s.tc.UpdateStoreLeaderWeight(4, 2)
	s.tc.AddLeaderRegion(1, 1, 2, 3, 4)
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 1, 4)
	s.tc.UpdateLeaderCount(4, 30)
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 1, 3)
}

func (s *testBalanceLeaderSchedulerSuite) TestBalanceSelector(c *C) {
	// Stores:     1    2    3    4
	// Leaders:    1    2    3   16
	// Region1:    -    F    F    L
	// Region2:    F    F    L    -
	s.tc.AddLeaderStore(1, 1)
	s.tc.AddLeaderStore(2, 2)
	s.tc.AddLeaderStore(3, 3)
	s.tc.AddLeaderStore(4, 16)
	s.tc.AddLeaderRegion(1, 4, 2, 3)
	s.tc.AddLeaderRegion(2, 3, 1, 2)
	// store4 has max leader score, store1 has min leader score.
	// The scheduler try to move a leader out of 16 first.
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 4, 2)

	// Stores:     1    2    3    4
	// Leaders:    1    14   15   16
	// Region1:    -    F    F    L
	// Region2:    F    F    L    -
	s.tc.UpdateLeaderCount(2, 14)
	s.tc.UpdateLeaderCount(3, 15)
	// Cannot move leader out of store4, move a leader into store1.
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 3, 1)

	// Stores:     1    2    3    4
	// Leaders:    1    2    15   16
	// Region1:    -    F    L    F
	// Region2:    L    F    F    -
	s.tc.AddLeaderStore(2, 2)
	s.tc.AddLeaderRegion(1, 3, 2, 4)
	s.tc.AddLeaderRegion(2, 1, 2, 3)
	// No leader in store16, no follower in store1. No operator is created.
	c.Assert(s.schedule(nil), IsNil)
	// store4 and store1 are marked taint.
	// Now source and target are store3 and store2.
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 3, 2)

	// Stores:     1    2    3    4
	// Leaders:    9    10   10   11
	// Region1:    -    F    F    L
	// Region2:    L    F    F    -
	s.tc.AddLeaderStore(1, 10)
	s.tc.AddLeaderStore(2, 10)
	s.tc.AddLeaderStore(3, 10)
	s.tc.AddLeaderStore(4, 10)
	s.tc.AddLeaderRegion(1, 4, 2, 3)
	s.tc.AddLeaderRegion(2, 1, 2, 3)
	// The cluster is balanced.
	c.Assert(s.schedule(nil), IsNil) // store1, store4 are marked taint.
	c.Assert(s.schedule(nil), IsNil) // store2, store3 are marked taint.

	// store3's leader drops:
	// Stores:     1    2    3    4
	// Leaders:    11   13   0    16
	// Region1:    -    F    F    L
	// Region2:    L    F    F    -
	s.tc.AddLeaderStore(1, 11)
	s.tc.AddLeaderStore(2, 13)
	s.tc.AddLeaderStore(3, 0)
	s.tc.AddLeaderStore(4, 16)
	c.Assert(s.schedule(nil), IsNil)                                              // All stores are marked taint.
	testutil.CheckTransferLeader(c, s.schedule(nil)[0], schedule.OpBalance, 4, 3) // The taint store will be clear.
}

var _ = Suite(&testBalanceRegionSchedulerSuite{})

type testBalanceRegionSchedulerSuite struct{}

func (s *testBalanceRegionSchedulerSuite) TestBalance(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	sb, err := schedule.CreateScheduler("balance-region", schedule.NewLimiter())
	c.Assert(err, IsNil)
	cache := sb.(*balanceRegionScheduler).taintStores

	opt.SetMaxReplicas(1)

	// Add stores 1,2,3,4.
	tc.AddRegionStore(1, 6)
	tc.AddRegionStore(2, 8)
	tc.AddRegionStore(3, 8)
	tc.AddRegionStore(4, 16)
	// Add region 1 with leader in store 4.
	tc.AddLeaderRegion(1, 4)
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 4, 1)

	// Test stateFilter.
	tc.SetStoreOffline(1)
	tc.UpdateRegionCount(2, 6)
	cache.Remove(4)
	// When store 1 is offline, it will be filtered,
	// store 2 becomes the store with least regions.
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 4, 2)
	opt.SetMaxReplicas(3)
	c.Assert(sb.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)

	cache.Clear()
	opt.SetMaxReplicas(1)
	c.Assert(sb.Schedule(tc, schedule.NewOpInfluence(nil, tc)), NotNil)
}

func (s *testBalanceRegionSchedulerSuite) TestReplicas3(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	newTestReplication(opt, 3, "zone", "rack", "host")

	sb, err := schedule.CreateScheduler("balance-region", schedule.NewLimiter())
	c.Assert(err, IsNil)
	cache := sb.(*balanceRegionScheduler).taintStores

	// Store 1 has the largest region score, so the balancer try to replace peer in store 1.
	tc.AddLabelsStore(1, 16, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(2, 15, map[string]string{"zone": "z1", "rack": "r2", "host": "h1"})
	tc.AddLabelsStore(3, 14, map[string]string{"zone": "z1", "rack": "r2", "host": "h2"})

	tc.AddLeaderRegion(1, 1, 2, 3)
	// This schedule try to replace peer in store 1, but we have no other stores,
	// so store 1 will be set in the cache and skipped next schedule.
	c.Assert(sb.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)
	c.Assert(cache.Exists(1), IsTrue)

	// Store 4 has smaller region score than store 2.
	tc.AddLabelsStore(4, 2, map[string]string{"zone": "z1", "rack": "r2", "host": "h1"})
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 2, 4)

	// Store 5 has smaller region score than store 1.
	tc.AddLabelsStore(5, 2, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	cache.Remove(1) // Delete store 1 from cache, or it will be skipped.
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 1, 5)

	// Store 6 has smaller region score than store 5.
	tc.AddLabelsStore(6, 1, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 1, 6)

	// Store 7 has smaller region score with store 6.
	tc.AddLabelsStore(7, 0, map[string]string{"zone": "z1", "rack": "r1", "host": "h2"})
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 1, 7)

	// If store 7 is not available, will choose store 6.
	tc.SetStoreDown(7)
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 1, 6)

	// Store 8 has smaller region score than store 7, but the distinct score decrease.
	tc.AddLabelsStore(8, 1, map[string]string{"zone": "z1", "rack": "r2", "host": "h3"})
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 1, 6)

	// Take down 4,5,6,7
	tc.SetStoreDown(4)
	tc.SetStoreDown(5)
	tc.SetStoreDown(6)
	tc.SetStoreDown(7)
	c.Assert(sb.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)
	c.Assert(cache.Exists(1), IsTrue)
	cache.Remove(1)

	// Store 9 has different zone with other stores but larger region score than store 1.
	tc.AddLabelsStore(9, 20, map[string]string{"zone": "z2", "rack": "r1", "host": "h1"})
	c.Assert(sb.Schedule(tc, schedule.NewOpInfluence(nil, tc)), IsNil)
}

func (s *testBalanceRegionSchedulerSuite) TestReplicas5(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	newTestReplication(opt, 5, "zone", "rack", "host")

	sb, err := schedule.CreateScheduler("balance-region", schedule.NewLimiter())
	c.Assert(err, IsNil)

	tc.AddLabelsStore(1, 4, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(2, 5, map[string]string{"zone": "z2", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(3, 6, map[string]string{"zone": "z3", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(4, 7, map[string]string{"zone": "z4", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(5, 28, map[string]string{"zone": "z5", "rack": "r1", "host": "h1"})

	tc.AddLeaderRegion(1, 1, 2, 3, 4, 5)

	// Store 6 has smaller region score.
	tc.AddLabelsStore(6, 1, map[string]string{"zone": "z5", "rack": "r2", "host": "h1"})
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 5, 6)

	// Store 7 has larger region score and same distinct score with store 6.
	tc.AddLabelsStore(7, 5, map[string]string{"zone": "z6", "rack": "r1", "host": "h1"})
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 5, 6)

	// Store 1 has smaller region score and higher distinct score.
	tc.AddLeaderRegion(1, 2, 3, 4, 5, 6)
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 5, 1)

	// Store 6 has smaller region score and higher distinct score.
	tc.AddLabelsStore(11, 29, map[string]string{"zone": "z1", "rack": "r2", "host": "h1"})
	tc.AddLabelsStore(12, 8, map[string]string{"zone": "z2", "rack": "r2", "host": "h1"})
	tc.AddLabelsStore(13, 7, map[string]string{"zone": "z3", "rack": "r2", "host": "h1"})
	tc.AddLeaderRegion(1, 2, 3, 11, 12, 13)
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 11, 6)
}

func (s *testBalanceRegionSchedulerSuite) TestStoreWeight(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	sb, err := schedule.CreateScheduler("balance-region", schedule.NewLimiter())
	c.Assert(err, IsNil)
	opt.SetMaxReplicas(1)

	tc.AddRegionStore(1, 10)
	tc.AddRegionStore(2, 10)
	tc.AddRegionStore(3, 10)
	tc.AddRegionStore(4, 10)
	tc.UpdateStoreRegionWeight(1, 0.5)
	tc.UpdateStoreRegionWeight(2, 0.9)
	tc.UpdateStoreRegionWeight(3, 1.0)
	tc.UpdateStoreRegionWeight(4, 2.0)

	tc.AddLeaderRegion(1, 1)
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 1, 4)

	tc.UpdateRegionCount(4, 30)
	testutil.CheckTransferPeer(c, sb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpBalance, 1, 3)
}

var _ = Suite(&testReplicaCheckerSuite{})

type testReplicaCheckerSuite struct{}

func (s *testReplicaCheckerSuite) TestBasic(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	rc := schedule.NewReplicaChecker(tc, namespace.DefaultClassifier)

	opt.MaxSnapshotCount = 2

	// Add stores 1,2,3,4.
	tc.AddRegionStore(1, 4)
	tc.AddRegionStore(2, 3)
	tc.AddRegionStore(3, 2)
	tc.AddRegionStore(4, 1)
	// Add region 1 with leader in store 1 and follower in store 2.
	tc.AddLeaderRegion(1, 1, 2)

	// Region has 2 peers, we need to add a new peer.
	region := tc.GetRegion(1)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 4)

	// Test healthFilter.
	// If store 4 is down, we add to store 3.
	tc.SetStoreDown(4)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 3)
	tc.SetStoreUp(4)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 4)

	// Test snapshotCountFilter.
	// If snapshotCount > MaxSnapshotCount, we add to store 3.
	tc.UpdateSnapshotCount(4, 3)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 3)
	// If snapshotCount < MaxSnapshotCount, we can add peer again.
	tc.UpdateSnapshotCount(4, 1)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 4)

	// Add peer in store 4, and we have enough replicas.
	peer4, _ := tc.AllocPeer(4)
	region.AddPeer(peer4)
	c.Assert(rc.Check(region), IsNil)

	// Add peer in store 3, and we have redundant replicas.
	peer3, _ := tc.AllocPeer(3)
	region.AddPeer(peer3)
	testutil.CheckRemovePeer(c, rc.Check(region), 1)
	region.RemoveStorePeer(1)

	// Peer in store 2 is down, remove it.
	tc.SetStoreDown(2)
	downPeer := &pdpb.PeerStats{
		Peer:        region.GetStorePeer(2),
		DownSeconds: 24 * 60 * 60,
	}
	region.DownPeers = append(region.DownPeers, downPeer)
	testutil.CheckRemovePeer(c, rc.Check(region), 2)
	region.DownPeers = nil
	c.Assert(rc.Check(region), IsNil)

	// Peer in store 3 is offline, transfer peer to store 1.
	tc.SetStoreOffline(3)
	testutil.CheckTransferPeer(c, rc.Check(region), schedule.OpReplica, 3, 1)
}

func (s *testReplicaCheckerSuite) TestLostStore(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	tc.AddRegionStore(1, 1)
	tc.AddRegionStore(2, 1)

	rc := schedule.NewReplicaChecker(tc, namespace.DefaultClassifier)

	// now region peer in store 1,2,3.but we just have store 1,2
	// This happens only in recovering the PD tc
	// should not panic
	tc.AddLeaderRegion(1, 1, 2, 3)
	region := tc.GetRegion(1)
	op := rc.Check(region)
	c.Assert(op, IsNil)
}

func (s *testReplicaCheckerSuite) TestOffline(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	newTestReplication(opt, 3, "zone", "rack", "host")

	rc := schedule.NewReplicaChecker(tc, namespace.DefaultClassifier)

	tc.AddLabelsStore(1, 1, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(2, 2, map[string]string{"zone": "z2", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(3, 3, map[string]string{"zone": "z3", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(4, 4, map[string]string{"zone": "z3", "rack": "r2", "host": "h1"})

	tc.AddLeaderRegion(1, 1)
	region := tc.GetRegion(1)

	// Store 2 has different zone and smallest region score.
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 2)
	peer2, _ := tc.AllocPeer(2)
	region.AddPeer(peer2)

	// Store 3 has different zone and smallest region score.
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 3)
	peer3, _ := tc.AllocPeer(3)
	region.AddPeer(peer3)

	// Store 4 has the same zone with store 3 and larger region score.
	peer4, _ := tc.AllocPeer(4)
	region.AddPeer(peer4)
	testutil.CheckRemovePeer(c, rc.Check(region), 4)

	// Test healthFilter.
	tc.SetStoreBusy(4, true)
	c.Assert(rc.Check(region), IsNil)
	tc.SetStoreBusy(4, false)
	testutil.CheckRemovePeer(c, rc.Check(region), 4)

	// Test offline
	// the number of region peers more than the maxReplicas
	// remove the peer
	tc.SetStoreOffline(3)
	testutil.CheckRemovePeer(c, rc.Check(region), 3)
	region.RemoveStorePeer(4)
	// the number of region peers equals the maxReplicas
	// Transfer peer to store 4.
	testutil.CheckTransferPeer(c, rc.Check(region), schedule.OpReplica, 3, 4)

	// Store 5 has a same label score with store 4,but the region score smaller than store 4, we will choose store 5.
	tc.AddLabelsStore(5, 3, map[string]string{"zone": "z4", "rack": "r1", "host": "h1"})
	testutil.CheckTransferPeer(c, rc.Check(region), schedule.OpReplica, 3, 5)
	// Store 5 has too many snapshots, choose store 4
	tc.UpdateSnapshotCount(5, 10)
	testutil.CheckTransferPeer(c, rc.Check(region), schedule.OpReplica, 3, 4)
	tc.UpdatePendingPeerCount(4, 30)
	c.Assert(rc.Check(region), IsNil)
}

func (s *testReplicaCheckerSuite) TestDistinctScore(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	newTestReplication(opt, 3, "zone", "rack", "host")

	rc := schedule.NewReplicaChecker(tc, namespace.DefaultClassifier)

	tc.AddLabelsStore(1, 9, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	tc.AddLabelsStore(2, 8, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})

	// We need 3 replicas.
	tc.AddLeaderRegion(1, 1)
	region := tc.GetRegion(1)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 2)
	peer2, _ := tc.AllocPeer(2)
	region.AddPeer(peer2)

	// Store 1,2,3 have the same zone, rack, and host.
	tc.AddLabelsStore(3, 5, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 3)

	// Store 4 has smaller region score.
	tc.AddLabelsStore(4, 4, map[string]string{"zone": "z1", "rack": "r1", "host": "h1"})
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 4)

	// Store 5 has a different host.
	tc.AddLabelsStore(5, 5, map[string]string{"zone": "z1", "rack": "r1", "host": "h2"})
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 5)

	// Store 6 has a different rack.
	tc.AddLabelsStore(6, 6, map[string]string{"zone": "z1", "rack": "r2", "host": "h1"})
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 6)

	// Store 7 has a different zone.
	tc.AddLabelsStore(7, 7, map[string]string{"zone": "z2", "rack": "r1", "host": "h1"})
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 7)

	// Test stateFilter.
	tc.SetStoreOffline(7)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 6)
	tc.SetStoreUp(7)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 7)

	// Add peer to store 7.
	peer7, _ := tc.AllocPeer(7)
	region.AddPeer(peer7)

	// Replace peer in store 1 with store 6 because it has a different rack.
	testutil.CheckTransferPeer(c, rc.Check(region), schedule.OpReplica, 1, 6)
	peer6, _ := tc.AllocPeer(6)
	region.AddPeer(peer6)
	testutil.CheckRemovePeer(c, rc.Check(region), 1)
	region.RemoveStorePeer(1)
	c.Assert(rc.Check(region), IsNil)

	// Store 8 has the same zone and different rack with store 7.
	// Store 1 has the same zone and different rack with store 6.
	// So store 8 and store 1 are equivalent.
	tc.AddLabelsStore(8, 1, map[string]string{"zone": "z2", "rack": "r2", "host": "h1"})
	c.Assert(rc.Check(region), IsNil)

	// Store 10 has a different zone.
	// Store 2 and 6 have the same distinct score, but store 2 has larger region score.
	// So replace peer in store 2 with store 10.
	tc.AddLabelsStore(10, 1, map[string]string{"zone": "z3", "rack": "r1", "host": "h1"})
	testutil.CheckTransferPeer(c, rc.Check(region), schedule.OpReplica, 2, 10)
	peer10, _ := tc.AllocPeer(10)
	region.AddPeer(peer10)
	testutil.CheckRemovePeer(c, rc.Check(region), 2)
	region.RemoveStorePeer(2)
	c.Assert(rc.Check(region), IsNil)
}

func (s *testReplicaCheckerSuite) TestDistinctScore2(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)

	newTestReplication(opt, 5, "zone", "host")

	rc := schedule.NewReplicaChecker(tc, namespace.DefaultClassifier)

	tc.AddLabelsStore(1, 1, map[string]string{"zone": "z1", "host": "h1"})
	tc.AddLabelsStore(2, 1, map[string]string{"zone": "z1", "host": "h2"})
	tc.AddLabelsStore(3, 1, map[string]string{"zone": "z1", "host": "h3"})
	tc.AddLabelsStore(4, 1, map[string]string{"zone": "z2", "host": "h1"})
	tc.AddLabelsStore(5, 1, map[string]string{"zone": "z2", "host": "h2"})
	tc.AddLabelsStore(6, 1, map[string]string{"zone": "z3", "host": "h1"})

	tc.AddLeaderRegion(1, 1, 2, 4)
	region := tc.GetRegion(1)

	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 6)
	peer6, _ := tc.AllocPeer(6)
	region.AddPeer(peer6)

	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 5)
	peer5, _ := tc.AllocPeer(5)
	region.AddPeer(peer5)

	c.Assert(rc.Check(region), IsNil)
}

func (s *testReplicaCheckerSuite) TestStorageThreshold(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	opt.LocationLabels = []string{"zone"}
	tc := schedule.NewMockCluster(opt)
	rc := schedule.NewReplicaChecker(tc, namespace.DefaultClassifier)

	tc.AddLabelsStore(1, 1, map[string]string{"zone": "z1"})
	tc.UpdateStorageRatio(1, 0.5, 0.5)
	tc.AddLabelsStore(2, 1, map[string]string{"zone": "z1"})
	tc.UpdateStorageRatio(1, 0.1, 0.9)
	tc.AddLabelsStore(3, 1, map[string]string{"zone": "z2"})
	tc.AddLabelsStore(4, 0, map[string]string{"zone": "z3"})

	tc.AddLeaderRegion(1, 1, 2, 3)
	region := tc.GetRegion(1)

	// Move peer to better location.
	tc.UpdateStorageRatio(4, 0, 1)
	testutil.CheckTransferPeer(c, rc.Check(region), schedule.OpReplica, 1, 4)
	// If store4 is almost full, do not add peer on it.
	tc.UpdateStorageRatio(4, 0.9, 0.1)
	c.Assert(rc.Check(region), IsNil)

	tc.AddLeaderRegion(2, 1, 3)
	region = tc.GetRegion(2)
	// Add peer on store4.
	tc.UpdateStorageRatio(4, 0, 1)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 4)
	// If store4 is almost full, do not add peer on it.
	tc.UpdateStorageRatio(4, 0.8, 0)
	testutil.CheckAddPeer(c, rc.Check(region), schedule.OpReplica, 2)
}

var _ = Suite(&testMergeCheckerSuite{})

type testMergeCheckerSuite struct {
	cluster *schedule.MockCluster
	mc      *schedule.MergeChecker
	regions []*core.RegionInfo
}

func (s *testMergeCheckerSuite) SetUpTest(c *C) {
	cfg := schedule.NewMockSchedulerOptions()
	cfg.MaxMergeRegionSize = 2
	cfg.MaxMergeRegionRows = 2
	s.cluster = schedule.NewMockCluster(cfg)
	s.regions = []*core.RegionInfo{
		{
			Region: &metapb.Region{
				Id:       1,
				StartKey: []byte(""),
				EndKey:   []byte("a"),
				Peers: []*metapb.Peer{
					{Id: 101, StoreId: 1},
					{Id: 102, StoreId: 2},
				},
			},
			Leader:          &metapb.Peer{Id: 101, StoreId: 1},
			ApproximateSize: 1,
			ApproximateRows: 1,
		},
		{
			Region: &metapb.Region{
				Id:       2,
				StartKey: []byte("a"),
				EndKey:   []byte("t"),
				Peers: []*metapb.Peer{
					{Id: 103, StoreId: 1},
					{Id: 104, StoreId: 4},
					{Id: 105, StoreId: 5},
				},
			},
			Leader:          &metapb.Peer{Id: 104, StoreId: 4},
			ApproximateSize: 200,
			ApproximateRows: 200,
		},
		{
			Region: &metapb.Region{
				Id:       3,
				StartKey: []byte("t"),
				EndKey:   []byte("x"),
				Peers: []*metapb.Peer{
					{Id: 106, StoreId: 1},
					{Id: 107, StoreId: 5},
					{Id: 108, StoreId: 6},
				},
			},
			Leader:          &metapb.Peer{Id: 108, StoreId: 6},
			ApproximateSize: 1,
			ApproximateRows: 1,
		},
		{
			Region: &metapb.Region{
				Id:       4,
				StartKey: []byte("x"),
				EndKey:   []byte(""),
				Peers: []*metapb.Peer{
					{Id: 109, StoreId: 4},
				},
			},
			Leader:          &metapb.Peer{Id: 109, StoreId: 4},
			ApproximateSize: 10,
			ApproximateRows: 10,
		},
	}

	for _, region := range s.regions {
		c.Assert(s.cluster.PutRegion(region), IsNil)
	}

	s.mc = schedule.NewMergeChecker(s.cluster, namespace.DefaultClassifier)
}

func (s *testMergeCheckerSuite) TestBasic(c *C) {
	s.cluster.MockSchedulerOptions.SplitMergeInterval = time.Hour

	// should with same peer count
	op1, op2 := s.mc.Check(s.regions[0])
	c.Assert(op1, IsNil)
	c.Assert(op2, IsNil)
	// size should be small enough
	op1, op2 = s.mc.Check(s.regions[1])
	c.Assert(op1, IsNil)
	c.Assert(op2, IsNil)
	op1, op2 = s.mc.Check(s.regions[2])
	c.Assert(op1, NotNil)
	c.Assert(op2, NotNil)
	// Skip recently split regions.
	s.mc.RecordRegionSplit(s.regions[2].GetId())
	op1, op2 = s.mc.Check(s.regions[2])
	c.Assert(op1, IsNil)
	c.Assert(op2, IsNil)
	op1, op2 = s.mc.Check(s.regions[3])
	c.Assert(op1, IsNil)
	c.Assert(op2, IsNil)
}

func (s *testMergeCheckerSuite) checkSteps(c *C, op *schedule.Operator, steps []schedule.OperatorStep) {
	c.Assert(steps, NotNil)
	c.Assert(op.Len(), Equals, len(steps))
	for i := range steps {
		c.Assert(op.Step(i), DeepEquals, steps[i])
	}
}

func (s *testMergeCheckerSuite) TestMatchPeers(c *C) {
	// partial store overlap not including leader
	op1, op2 := s.mc.Check(s.regions[2])
	s.checkSteps(c, op1, []schedule.OperatorStep{
		schedule.AddLearner{ToStore: 4, PeerID: 1},
		schedule.PromoteLearner{ToStore: 4, PeerID: 1},
		schedule.TransferLeader{FromStore: 6, ToStore: 4},
		schedule.RemovePeer{FromStore: 6},
		schedule.MergeRegion{
			FromRegion: s.regions[2].Region,
			ToRegion:   s.regions[1].Region,
			IsPassive:  false,
		},
	})
	s.checkSteps(c, op2, []schedule.OperatorStep{
		schedule.MergeRegion{
			FromRegion: s.regions[2].Region,
			ToRegion:   s.regions[1].Region,
			IsPassive:  true,
		},
	})

	// partial store overlap including leader
	s.regions[2].Leader = &metapb.Peer{Id: 106, StoreId: 1}
	s.cluster.PutRegion(s.regions[2])
	op1, op2 = s.mc.Check(s.regions[2])
	s.checkSteps(c, op1, []schedule.OperatorStep{
		schedule.AddLearner{ToStore: 4, PeerID: 2},
		schedule.PromoteLearner{ToStore: 4, PeerID: 2},
		schedule.RemovePeer{FromStore: 6},
		schedule.MergeRegion{
			FromRegion: s.regions[2].Region,
			ToRegion:   s.regions[1].Region,
			IsPassive:  false,
		},
	})
	s.checkSteps(c, op2, []schedule.OperatorStep{
		schedule.MergeRegion{
			FromRegion: s.regions[2].Region,
			ToRegion:   s.regions[1].Region,
			IsPassive:  true,
		},
	})

	// all store overlap
	s.regions[2].Peers = []*metapb.Peer{
		{Id: 106, StoreId: 1},
		{Id: 107, StoreId: 5},
		{Id: 108, StoreId: 4},
	}
	s.cluster.PutRegion(s.regions[2])
	op1, op2 = s.mc.Check(s.regions[2])
	s.checkSteps(c, op1, []schedule.OperatorStep{
		schedule.MergeRegion{
			FromRegion: s.regions[2].Region,
			ToRegion:   s.regions[1].Region,
			IsPassive:  false,
		},
	})
	s.checkSteps(c, op2, []schedule.OperatorStep{
		schedule.MergeRegion{
			FromRegion: s.regions[2].Region,
			ToRegion:   s.regions[1].Region,
			IsPassive:  true,
		},
	})
}

var _ = Suite(&testBalanceHotWriteRegionSchedulerSuite{})

type testBalanceHotWriteRegionSchedulerSuite struct{}

func (s *testBalanceHotWriteRegionSchedulerSuite) TestBalance(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	newTestReplication(opt, 3, "zone", "host")
	tc := schedule.NewMockCluster(opt)
	hb, err := schedule.CreateScheduler("hot-write-region", schedule.NewLimiter())
	c.Assert(err, IsNil)

	// Add stores 1, 2, 3, 4, 5, 6  with region counts 3, 2, 2, 2, 0, 0.

	tc.AddLabelsStore(1, 3, map[string]string{"zone": "z1", "host": "h1"})
	tc.AddLabelsStore(2, 2, map[string]string{"zone": "z2", "host": "h2"})
	tc.AddLabelsStore(3, 2, map[string]string{"zone": "z3", "host": "h3"})
	tc.AddLabelsStore(4, 2, map[string]string{"zone": "z4", "host": "h4"})
	tc.AddLabelsStore(5, 0, map[string]string{"zone": "z2", "host": "h5"})
	tc.AddLabelsStore(6, 0, map[string]string{"zone": "z5", "host": "h6"})
	tc.AddLabelsStore(7, 0, map[string]string{"zone": "z5", "host": "h7"})
	tc.SetStoreDown(7)

	// Report store written bytes.
	tc.UpdateStorageWrittenBytes(1, 75*1024*1024)
	tc.UpdateStorageWrittenBytes(2, 45*1024*1024)
	tc.UpdateStorageWrittenBytes(3, 45*1024*1024)
	tc.UpdateStorageWrittenBytes(4, 60*1024*1024)
	tc.UpdateStorageWrittenBytes(5, 0)
	tc.UpdateStorageWrittenBytes(6, 0)

	// Region 1, 2 and 3 are hot regions.
	//| region_id | leader_sotre | follower_store | follower_store | written_bytes |
	//|-----------|--------------|----------------|----------------|---------------|
	//|     1     |       1      |        2       |       3        |      512KB    |
	//|     2     |       1      |        3       |       4        |      512KB    |
	//|     3     |       1      |        2       |       4        |      512KB    |
	tc.AddLeaderRegionWithWriteInfo(1, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 3)
	tc.AddLeaderRegionWithWriteInfo(2, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 3, 4)
	tc.AddLeaderRegionWithWriteInfo(3, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 4)
	opt.HotRegionLowThreshold = 0

	// Will transfer a hot region from store 1, because the total count of peers
	// which is hot for store 1 is more larger than other stores.
	op := hb.Schedule(tc, schedule.NewOpInfluence(nil, tc))
	c.Assert(op, NotNil)
	switch op[0].Len() {
	case 1:
		// balance by leader selected
		testutil.CheckTransferLeaderFrom(c, op[0], schedule.OpHotRegion, 1)
	case 3:
		// balance by peer selected
		if op[0].RegionID() == 2 {
			// peer in store 1 of the region 2 can transfer to store 5 or store 6 because of the label
			testutil.CheckTransferPeerWithLeaderTransferFrom(c, op[0], schedule.OpHotRegion, 1)
		} else {
			// peer in store 1 of the region 1,2 can only transfer to store 6
			testutil.CheckTransferPeerWithLeaderTransfer(c, op[0], schedule.OpHotRegion, 1, 6)
		}
	}
	// After transfer a hot region from store 1 to store 5
	//| region_id | leader_sotre | follower_store | follower_store | written_bytes |
	//|-----------|--------------|----------------|----------------|---------------|
	//|     1     |       1      |        2       |       3        |      512KB    |
	//|     2     |       1      |        3       |       4        |      512KB    |
	//|     3     |       6      |        2       |       4        |      512KB    |
	//|     4     |       5      |        6       |       1        |      512KB    |
	//|     5     |       3      |        4       |       5        |      512KB    |
	tc.UpdateStorageWrittenBytes(1, 60*1024*1024)
	tc.UpdateStorageWrittenBytes(2, 30*1024*1024)
	tc.UpdateStorageWrittenBytes(3, 60*1024*1024)
	tc.UpdateStorageWrittenBytes(4, 30*1024*1024)
	tc.UpdateStorageWrittenBytes(5, 0*1024*1024)
	tc.UpdateStorageWrittenBytes(6, 30*1024*1024)
	tc.AddLeaderRegionWithWriteInfo(1, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 3)
	tc.AddLeaderRegionWithWriteInfo(2, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 3)
	tc.AddLeaderRegionWithWriteInfo(3, 6, 512*1024*schedule.RegionHeartBeatReportInterval, 1, 4)
	tc.AddLeaderRegionWithWriteInfo(4, 5, 512*1024*schedule.RegionHeartBeatReportInterval, 6, 4)
	tc.AddLeaderRegionWithWriteInfo(5, 3, 512*1024*schedule.RegionHeartBeatReportInterval, 4, 5)
	// We can find that the leader of all hot regions are on store 1,
	// so one of the leader will transfer to another store.
	testutil.CheckTransferLeaderFrom(c, hb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpHotRegion, 1)

	// Should not panic if region not found.
	for i := uint64(1); i <= 3; i++ {
		tc.Regions.RemoveRegion(tc.GetRegion(i))
	}
	hb.Schedule(tc, schedule.NewOpInfluence(nil, tc))
}

type testBalanceHotReadRegionSchedulerSuite struct{}

func (s *testBalanceHotReadRegionSchedulerSuite) TestBalance(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)
	hb, err := schedule.CreateScheduler("hot-read-region", schedule.NewLimiter())
	c.Assert(err, IsNil)

	// Add stores 1, 2, 3, 4, 5 with region counts 3, 2, 2, 2, 0.
	tc.AddRegionStore(1, 3)
	tc.AddRegionStore(2, 2)
	tc.AddRegionStore(3, 2)
	tc.AddRegionStore(4, 2)
	tc.AddRegionStore(5, 0)

	// Report store read bytes.
	tc.UpdateStorageReadBytes(1, 75*1024*1024)
	tc.UpdateStorageReadBytes(2, 45*1024*1024)
	tc.UpdateStorageReadBytes(3, 45*1024*1024)
	tc.UpdateStorageReadBytes(4, 60*1024*1024)
	tc.UpdateStorageReadBytes(5, 0)

	// Region 1, 2 and 3 are hot regions.
	//| region_id | leader_sotre | follower_store | follower_store |   read_bytes  |
	//|-----------|--------------|----------------|----------------|---------------|
	//|     1     |       1      |        2       |       3        |      512KB    |
	//|     2     |       2      |        1       |       3        |      512KB    |
	//|     3     |       1      |        2       |       3        |      512KB    |
	tc.AddLeaderRegionWithReadInfo(1, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 3)
	tc.AddLeaderRegionWithReadInfo(2, 2, 512*1024*schedule.RegionHeartBeatReportInterval, 1, 3)
	tc.AddLeaderRegionWithReadInfo(3, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 3)
	// lower than hot read flow rate, but higher than write flow rate
	tc.AddLeaderRegionWithReadInfo(11, 1, 24*1024*schedule.RegionHeartBeatReportInterval, 2, 3)
	opt.HotRegionLowThreshold = 0
	c.Assert(tc.IsRegionHot(1), IsTrue)
	c.Assert(tc.IsRegionHot(11), IsFalse)
	// check randomly pick hot region
	r := tc.RandHotRegionFromStore(2, schedule.ReadFlow)
	c.Assert(r, NotNil)
	c.Assert(r.GetId(), Equals, uint64(2))
	// check hot items
	stats := tc.HotCache.RegionStats(schedule.ReadFlow)
	c.Assert(len(stats), Equals, 3)
	for _, s := range stats {
		c.Assert(s.FlowBytes, Equals, uint64(512*1024))
	}
	// Will transfer a hot region leader from store 1 to store 3, because the total count of peers
	// which is hot for store 1 is more larger than other stores.
	testutil.CheckTransferLeader(c, hb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpHotRegion, 1, 3)
	// assume handle the operator
	tc.AddLeaderRegionWithReadInfo(3, 3, 512*1024*schedule.RegionHeartBeatReportInterval, 1, 2)

	// After transfer a hot region leader from store 1 to store 3
	// the tree region leader will be evenly distributed in three stores
	tc.UpdateStorageReadBytes(1, 60*1024*1024)
	tc.UpdateStorageReadBytes(2, 30*1024*1024)
	tc.UpdateStorageReadBytes(3, 60*1024*1024)
	tc.UpdateStorageReadBytes(4, 30*1024*1024)
	tc.UpdateStorageReadBytes(5, 30*1024*1024)
	tc.AddLeaderRegionWithReadInfo(4, 1, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 3)
	tc.AddLeaderRegionWithReadInfo(5, 4, 512*1024*schedule.RegionHeartBeatReportInterval, 2, 5)

	// Now appear two read hot region in store 1 and 4
	// We will Transfer peer from 1 to 5
	testutil.CheckTransferPeerWithLeaderTransfer(c, hb.Schedule(tc, schedule.NewOpInfluence(nil, tc))[0], schedule.OpHotRegion, 1, 5)

	// Should not panic if region not found.
	for i := uint64(1); i <= 3; i++ {
		tc.Regions.RemoveRegion(tc.GetRegion(i))
	}
	hb.Schedule(tc, schedule.NewOpInfluence(nil, tc))
}

var _ = Suite(&testScatterRangeLeaderSuite{})

type testScatterRangeLeaderSuite struct{}

func (s *testScatterRangeLeaderSuite) TestBalance(c *C) {
	opt := schedule.NewMockSchedulerOptions()
	tc := schedule.NewMockCluster(opt)
	// Add stores 1,2,3,4,5.
	tc.AddRegionStore(1, 0)
	tc.AddRegionStore(2, 0)
	tc.AddRegionStore(3, 0)
	tc.AddRegionStore(4, 0)
	tc.AddRegionStore(5, 0)
	var (
		id      uint64
		regions []*metapb.Region
	)
	for i := 0; i < 50; i++ {
		peers := []*metapb.Peer{
			{Id: id + 1, StoreId: 1},
			{Id: id + 2, StoreId: 2},
			{Id: id + 3, StoreId: 3},
		}
		regions = append(regions, &metapb.Region{
			Id:       id + 4,
			Peers:    peers,
			StartKey: []byte(fmt.Sprintf("s_%02d", i)),
			EndKey:   []byte(fmt.Sprintf("s_%02d", i+1)),
		})
		id += 4
	}
	// empty case
	regions[49].EndKey = []byte("")
	for _, meta := range regions {
		leader := rand.Intn(4) % 3
		regionInfo := core.NewRegionInfo(meta, meta.Peers[leader])
		regionInfo.ApproximateSize = 96
		regionInfo.ApproximateRows = 96
		tc.Regions.SetRegion(regionInfo)
	}
	for i := 0; i < 100; i++ {
		tc.AllocPeer(1)
	}
	for i := 1; i <= 5; i++ {
		tc.UpdateStoreStatus(uint64(i))
	}

	hb, err := schedule.CreateScheduler("scatter-range", schedule.NewLimiter(), "s_00", "s_50", "t")
	c.Assert(err, IsNil)
	limit := 0
	for {
		if limit > 100 {
			break
		}

		ops := hb.Schedule(tc, schedule.NewOpInfluence(nil, tc))
		if ops == nil {
			limit++
			continue
		}
		tc.ApplyOperator(ops[0])
	}
	for i := 1; i <= 5; i++ {
		leaderCount := tc.Regions.GetStoreLeaderCount(uint64(i))
		c.Assert(leaderCount, LessEqual, 12)
		regionCount := tc.Regions.GetStoreRegionCount(uint64(i))
		c.Assert(regionCount, LessEqual, 32)
	}
}
