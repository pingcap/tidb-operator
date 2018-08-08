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

package integration

import (
	"testing"

	. "github.com/pingcap/check"
	"github.com/pingcap/pd/pkg/tempurl"
	"github.com/pingcap/pd/pkg/testutil"
	"github.com/pingcap/pd/server"
)

func TestAll(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&integrationTestSuite{})

type integrationTestSuite struct{}

func (s *integrationTestSuite) SetUpSuite(c *C) {
	server.EnableZap = true
}

func (s *integrationTestSuite) TestUpdateAdvertiseUrls(c *C) {
	c.Parallel()

	cluster, err := newTestCluster(2)
	c.Assert(err, IsNil)
	defer cluster.Destory()

	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)

	// AdvertisePeerUrls should equals to PeerUrls.
	for _, conf := range cluster.config.InitialServers {
		serverConf := cluster.GetServer(conf.Name).GetConfig()
		c.Assert(serverConf.AdvertisePeerUrls, Equals, conf.PeerURLs)
		c.Assert(serverConf.AdvertiseClientUrls, Equals, conf.ClientURLs)
	}

	err = cluster.StopAll()
	c.Assert(err, IsNil)

	// Change config will not affect peer urls.
	// Recreate servers with new peer URLs.
	for _, conf := range cluster.config.InitialServers {
		conf.AdvertisePeerURLs = conf.PeerURLs + "," + tempurl.Alloc()
	}
	for _, conf := range cluster.config.InitialServers {
		serverConf, e := conf.Generate()
		c.Assert(e, IsNil)
		s, e := newTestServer(serverConf)
		c.Assert(e, IsNil)
		cluster.servers[conf.Name] = s
	}
	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)
	for _, conf := range cluster.config.InitialServers {
		serverConf := cluster.GetServer(conf.Name).GetConfig()
		c.Assert(serverConf.AdvertisePeerUrls, Equals, conf.PeerURLs)
	}
}

func (s *integrationTestSuite) TestClusterID(c *C) {
	c.Parallel()

	cluster, err := newTestCluster(3)
	c.Assert(err, IsNil)
	defer cluster.Destory()

	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)

	clusterID := cluster.GetServer("pd1").GetClusterID()
	for _, s := range cluster.servers {
		c.Assert(s.GetClusterID(), Equals, clusterID)
	}

	// Restart all PDs.
	err = cluster.StopAll()
	c.Assert(err, IsNil)
	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)

	// All PDs should have the same cluster ID as before.
	for _, s := range cluster.servers {
		c.Assert(s.GetClusterID(), Equals, clusterID)
	}
}

func (s *integrationTestSuite) TestLeader(c *C) {
	c.Parallel()

	cluster, err := newTestCluster(3)
	c.Assert(err, IsNil)
	defer cluster.Destory()

	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)

	leader1 := cluster.WaitLeader()
	c.Assert(leader1, Not(Equals), "")

	err = cluster.GetServer(leader1).Stop()
	c.Assert(err, IsNil)
	testutil.WaitUntil(c, func(c *C) bool {
		leader := cluster.GetLeader()
		return leader != leader1
	})
}
