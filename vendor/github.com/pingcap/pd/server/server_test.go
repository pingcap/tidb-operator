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

package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/coreos/etcd/clientv3"
	. "github.com/pingcap/check"
	"github.com/pingcap/pd/pkg/testutil"
)

func TestServer(t *testing.T) {
	EnableZap = true
	TestingT(t)
}

type cleanupFunc func()

func newTestServer(c *C) (*Server, cleanupFunc) {
	cfg := NewTestSingleConfig()

	svr, err := CreateServer(cfg, nil)
	c.Assert(err, IsNil)

	cleanup := func() {
		svr.Close()
		cleanServer(svr.cfg)
	}

	return svr, cleanup
}

func mustRunTestServer(c *C) (*Server, cleanupFunc) {
	server, cleanup := newTestServer(c)
	err := server.Run(context.TODO())
	c.Assert(err, IsNil)
	mustWaitLeader(c, []*Server{server})
	return server, cleanup
}

func mustWaitLeader(c *C, svrs []*Server) *Server {
	var leader *Server
	testutil.WaitUntil(c, func(c *C) bool {
		for _, s := range svrs {
			if s.IsLeader() {
				leader = s
				return true
			}
		}
		return false
	})
	return leader
}

var _ = Suite(&testLeaderServerSuite{})

type testLeaderServerSuite struct {
	svrs       map[string]*Server
	leaderPath string
}

func mustGetEtcdClient(c *C, svrs map[string]*Server) *clientv3.Client {
	for _, svr := range svrs {
		return svr.GetClient()
	}
	c.Fatal("etcd client none available")
	return nil
}

func (s *testLeaderServerSuite) SetUpSuite(c *C) {
	s.svrs = make(map[string]*Server)

	cfgs := NewTestMultiConfig(3)

	ch := make(chan *Server, 3)
	for i := 0; i < 3; i++ {
		cfg := cfgs[i]

		go func() {
			svr, err := CreateServer(cfg, nil)
			c.Assert(err, IsNil)
			err = svr.Run(context.TODO())
			c.Assert(err, IsNil)
			ch <- svr
		}()
	}

	for i := 0; i < 3; i++ {
		svr := <-ch
		s.svrs[svr.GetAddr()] = svr
		s.leaderPath = svr.getLeaderPath()
	}
}

func (s *testLeaderServerSuite) TearDownSuite(c *C) {
	for _, svr := range s.svrs {
		svr.Close()
		cleanServer(svr.cfg)
	}
}

var _ = Suite(&testServerSuite{})

type testServerSuite struct{}

func newTestServersWithCfgs(c *C, cfgs []*Config) ([]*Server, cleanupFunc) {
	svrs := make([]*Server, 0, len(cfgs))

	ch := make(chan *Server)
	for _, cfg := range cfgs {
		go func(cfg *Config) {
			svr, err := CreateServer(cfg, nil)
			c.Assert(err, IsNil)
			err = svr.Run(context.TODO())
			c.Assert(err, IsNil)
			ch <- svr
		}(cfg)
	}

	for i := 0; i < len(cfgs); i++ {
		svrs = append(svrs, <-ch)
	}
	mustWaitLeader(c, svrs)

	cleanup := func() {
		for _, svr := range svrs {
			svr.Close()
		}
		for _, cfg := range cfgs {
			cleanServer(cfg)
		}
	}

	return svrs, cleanup
}

func (s *testServerSuite) TestCheckClusterID(c *C) {
	cfgs := NewTestMultiConfig(2)
	for i, cfg := range cfgs {
		cfg.DataDir = fmt.Sprintf("/tmp/test_pd_check_clusterID_%d", i)
		// Clean up before testing.
		cleanServer(cfg)
	}
	originInitial := cfgs[0].InitialCluster
	for _, cfg := range cfgs {
		cfg.InitialCluster = fmt.Sprintf("%s=%s", cfg.Name, cfg.PeerUrls)
	}

	cfgA, cfgB := cfgs[0], cfgs[1]
	// Start a standalone cluster
	// TODO: clean up. For now tests failed because:
	//    etcdserver: failed to purge snap file ...
	svrsA, _ := newTestServersWithCfgs(c, []*Config{cfgA})
	// Close it.
	for _, svr := range svrsA {
		svr.Close()
	}

	// Start another cluster.
	_, cleanB := newTestServersWithCfgs(c, []*Config{cfgB})
	defer cleanB()

	// Start previous cluster, expect an error.
	cfgA.InitialCluster = originInitial
	svr, err := CreateServer(cfgA, nil)
	c.Assert(err, IsNil)
	err = svr.Run(context.TODO())
	c.Assert(err, NotNil)
}
