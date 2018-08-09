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

package api

import (
	"fmt"
	"math/rand"

	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server"
	"github.com/pingcap/pd/server/core"
)

var _ = Suite(&testRegionSuite{})

type testRegionSuite struct {
	svr       *server.Server
	cleanup   cleanUpFunc
	urlPrefix string
}

func (s *testRegionSuite) SetUpSuite(c *C) {
	s.svr, s.cleanup = mustNewServer(c)
	mustWaitLeader(c, []*server.Server{s.svr})

	addr := s.svr.GetAddr()
	s.urlPrefix = fmt.Sprintf("%s%s/api/v1", addr, apiPrefix)

	mustBootstrapCluster(c, s.svr)
}

func (s *testRegionSuite) TearDownSuite(c *C) {
	s.cleanup()
}

func newTestRegionInfo(regionID, storeID uint64, start, end []byte) *core.RegionInfo {
	leader := &metapb.Peer{
		Id:      regionID,
		StoreId: storeID,
	}

	return &core.RegionInfo{
		Region: &metapb.Region{
			Id:       regionID,
			StartKey: start,
			EndKey:   end,
			Peers:    []*metapb.Peer{leader},
		},
		Leader:          leader,
		ApproximateSize: 10,
		ApproximateRows: 10,
	}
}

func (s *testRegionSuite) TestRegion(c *C) {
	r := newTestRegionInfo(2, 1, []byte("a"), []byte("b"))
	mustRegionHeartbeat(c, s.svr, r)
	url := fmt.Sprintf("%s/region/id/%d", s.urlPrefix, r.GetId())
	r1 := &regionInfo{}
	err := readJSONWithURL(url, r1)
	c.Assert(err, IsNil)
	c.Assert(r1, DeepEquals, newRegionInfo(r))

	url = fmt.Sprintf("%s/region/key/%s", s.urlPrefix, "a")
	r2 := &regionInfo{}
	err = readJSONWithURL(url, r2)
	c.Assert(err, IsNil)
	c.Assert(r2, DeepEquals, newRegionInfo(r))
}

func (s *testRegionSuite) TestTopFlow(c *C) {
	r1 := newTestRegionInfo(1, 1, []byte("a"), []byte("b"))
	r1.WrittenBytes, r1.ReadBytes = 1000, 1000
	mustRegionHeartbeat(c, s.svr, r1)
	r2 := newTestRegionInfo(2, 1, []byte("b"), []byte("c"))
	r2.WrittenBytes, r2.ReadBytes = 2000, 0
	mustRegionHeartbeat(c, s.svr, r2)
	r3 := newTestRegionInfo(3, 1, []byte("c"), []byte("d"))
	r3.WrittenBytes, r3.ReadBytes = 500, 800
	mustRegionHeartbeat(c, s.svr, r3)
	s.checkTopFlow(c, fmt.Sprintf("%s/regions/writeflow", s.urlPrefix), []uint64{2, 1, 3})
	s.checkTopFlow(c, fmt.Sprintf("%s/regions/readflow", s.urlPrefix), []uint64{1, 3, 2})
	s.checkTopFlow(c, fmt.Sprintf("%s/regions/writeflow?limit=2", s.urlPrefix), []uint64{2, 1})
}

func (s *testRegionSuite) checkTopFlow(c *C, url string, regionIDs []uint64) {
	regions := &regionsInfo{}
	err := readJSONWithURL(url, regions)
	c.Assert(err, IsNil)
	c.Assert(regions.Count, Equals, len(regionIDs))
	for i, r := range regions.Regions {
		c.Assert(r.ID, Equals, regionIDs[i])
	}
}

func (s *testRegionSuite) TestTopN(c *C) {
	writtenBytes := []uint64{10, 10, 9, 5, 3, 2, 2, 1, 0, 0}
	for n := 0; n <= len(writtenBytes)+1; n++ {
		regions := make([]*core.RegionInfo, 0, len(writtenBytes))
		for _, i := range rand.Perm(len(writtenBytes)) {
			id := uint64(i + 1)
			region := newTestRegionInfo(id, id, nil, nil)
			region.WrittenBytes = uint64(writtenBytes[i])
			regions = append(regions, region)
		}
		topN := topNRegions(regions, func(a, b *core.RegionInfo) bool { return a.WrittenBytes < b.WrittenBytes }, n)
		if n > len(writtenBytes) {
			c.Assert(len(topN), Equals, len(writtenBytes))
		} else {
			c.Assert(len(topN), Equals, n)
		}
		for i := range topN {
			c.Assert(topN[i].WrittenBytes, Equals, writtenBytes[i])
		}
	}
}
