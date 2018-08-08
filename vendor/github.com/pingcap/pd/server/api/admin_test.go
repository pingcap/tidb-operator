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

package api

import (
	"fmt"
	"net/http"

	. "github.com/pingcap/check"
	"github.com/pingcap/pd/server"
)

var _ = Suite(&testAdminSuite{})

type testAdminSuite struct {
	svr       *server.Server
	cleanup   cleanUpFunc
	urlPrefix string
}

func (s *testAdminSuite) SetUpSuite(c *C) {
	s.svr, s.cleanup = mustNewServer(c)
	mustWaitLeader(c, []*server.Server{s.svr})

	addr := s.svr.GetAddr()
	s.urlPrefix = fmt.Sprintf("%s%s/api/v1", addr, apiPrefix)

	mustBootstrapCluster(c, s.svr)
}

func (s *testAdminSuite) TearDownSuite(c *C) {
	s.cleanup()
}

func (s *testAdminSuite) TestDropRegion(c *C) {
	cluster := s.svr.GetRaftCluster()

	// Update region's epoch to (100, 100).
	region := cluster.GetRegionInfoByKey([]byte("foo"))
	region.RegionEpoch.ConfVer, region.RegionEpoch.Version = 100, 100
	err := cluster.HandleRegionHeartbeat(region)
	c.Assert(err, IsNil)

	// Region epoch cannot decrease.
	region.RegionEpoch.ConfVer, region.RegionEpoch.Version = 50, 50
	err = cluster.HandleRegionHeartbeat(region)
	c.Assert(err, NotNil)

	// After drop region from cache, lower version is accepted.
	url := fmt.Sprintf("%s/admin/cache/region/%d", s.urlPrefix, region.Id)
	req, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	res, err := http.DefaultClient.Do(req)
	c.Assert(err, IsNil)
	c.Assert(res.StatusCode, Equals, http.StatusOK)
	res.Body.Close()
	err = cluster.HandleRegionHeartbeat(region)
	c.Assert(err, IsNil)

	region = cluster.GetRegionInfoByKey([]byte("foo"))
	c.Assert(region.GetRegionEpoch().ConfVer, Equals, uint64(50))
	c.Assert(region.GetRegionEpoch().Version, Equals, uint64(50))
}
