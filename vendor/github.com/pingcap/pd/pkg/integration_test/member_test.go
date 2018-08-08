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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/juju/errors"
	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/pkg/testutil"
)

func (s *integrationTestSuite) TestMemberDelete(c *C) {
	c.Parallel()

	cluster, err := newTestCluster(3)
	c.Assert(err, IsNil)
	defer cluster.Destory()

	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)
	cluster.WaitLeader()
	clientURL := cluster.GetServer("pd1").GetConfig().ClientUrls
	pd3ID := strconv.FormatUint(cluster.GetServer("pd3").GetServerID(), 10)
	httpClient := &http.Client{Timeout: 15 * time.Second}

	var table = []struct {
		path    string
		status  int
		members []*serverConfig
	}{
		{path: "name/foobar", status: http.StatusNotFound},
		{path: "name/pd2", members: []*serverConfig{cluster.config.InitialServers[0], cluster.config.InitialServers[1]}},
		{path: "name/pd2", status: http.StatusNotFound},
		{path: "id/" + pd3ID, members: []*serverConfig{cluster.config.InitialServers[0]}},
	}
	for _, t := range table {
		c.Log(time.Now(), "try to delete:", t.path)
		testutil.WaitUntil(c, func(c *C) bool {
			addr := clientURL + "/pd/api/v1/members/" + t.path
			req, err := http.NewRequest("DELETE", addr, nil)
			c.Assert(err, IsNil)
			res, err := httpClient.Do(req)
			c.Assert(err, IsNil)
			defer res.Body.Close()
			// Check by status.
			if t.status != 0 {
				if res.StatusCode != t.status {
					time.Sleep(time.Second)
					return false
				}
				return true
			}
			// Check by member list.
			cluster.WaitLeader()
			if err = s.checkMemberList(c, clientURL, t.members); err != nil {
				c.Logf("check member fail: %v", err)
				time.Sleep(time.Second)
				return false
			}
			return true
		})
	}
}

func (s *integrationTestSuite) checkMemberList(c *C, clientURL string, configs []*serverConfig) error {
	httpClient := &http.Client{Timeout: 15 * time.Second}
	addr := clientURL + "/pd/api/v1/members"
	res, err := httpClient.Get(addr)
	c.Assert(err, IsNil)
	defer res.Body.Close()
	buf, err := ioutil.ReadAll(res.Body)
	c.Assert(err, IsNil)
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("load members failed, status: %v, data: %q", res.StatusCode, buf)
	}
	data := make(map[string][]*pdpb.Member)
	json.Unmarshal(buf, &data)
	if len(data["members"]) != len(configs) {
		return errors.Errorf("member length not match, %v vs %v", len(data["members"]), len(configs))
	}
	for _, member := range data["members"] {
		for _, cfg := range configs {
			if member.GetName() == cfg.Name {
				c.Assert(member.ClientUrls, DeepEquals, []string{cfg.ClientURLs})
				c.Assert(member.PeerUrls, DeepEquals, []string{cfg.PeerURLs})
			}
		}
	}
	return nil
}

func (s *integrationTestSuite) TestLeaderPriority(c *C) {
	c.Parallel()

	cluster, err := newTestCluster(3)
	c.Assert(err, IsNil)
	defer cluster.Destory()

	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)

	cluster.WaitLeader()

	leader1, err := cluster.GetServer("pd1").GetEtcdLeader()
	c.Assert(err, IsNil)
	server1 := cluster.GetServer(leader1)
	addr := server1.GetConfig().ClientUrls
	// PD leader should sync with etcd leader.
	testutil.WaitUntil(c, func(c *C) bool {
		return cluster.GetLeader() == leader1
	})
	// Bind a lower priority to current leader.
	s.post(c, addr+"/pd/api/v1/members/name/"+leader1, `{"leader-priority": -1}`)
	// Wait etcd leader change.
	leader2 := s.waitEtcdLeaderChange(c, server1, leader1)
	// PD leader should sync with etcd leader again.
	testutil.WaitUntil(c, func(c *C) bool {
		return cluster.GetLeader() == leader2
	})
}

func (s *integrationTestSuite) post(c *C, url string, body string) {
	testutil.WaitUntil(c, func(c *C) bool {
		res, err := http.Post(url, "", bytes.NewBufferString(body))
		c.Assert(err, IsNil)
		b, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		c.Assert(err, IsNil)
		c.Logf("post %s, status: %v res: %s", url, res.StatusCode, string(b))
		return res.StatusCode == http.StatusOK
	})
}

func (s *integrationTestSuite) waitEtcdLeaderChange(c *C, server *testServer, old string) string {
	var leader string
	testutil.WaitUntil(c, func(c *C) bool {
		var err error
		leader, err = server.GetEtcdLeader()
		if err != nil {
			return false
		}
		if leader == old {
			// Priority check could be slow. So we sleep longer here.
			time.Sleep(5 * time.Second)
		}
		return leader != old
	})
	return leader
}

func (s *integrationTestSuite) TestLeaderResign(c *C) {
	c.Parallel()

	cluster, err := newTestCluster(3)
	c.Assert(err, IsNil)
	defer cluster.Destory()

	err = cluster.RunInitialServers()
	c.Assert(err, IsNil)

	leader1 := cluster.WaitLeader()
	addr1 := cluster.GetServer(leader1).GetConfig().ClientUrls

	s.post(c, addr1+"/pd/api/v1/leader/resign", "")
	leader2 := s.waitLeaderChange(c, cluster, leader1)
	c.Log("leader2:", leader2)
	addr2 := cluster.GetServer(leader2).GetConfig().ClientUrls
	s.post(c, addr2+"/pd/api/v1/leader/transfer/"+leader1, "")
	leader3 := s.waitLeaderChange(c, cluster, leader2)
	c.Assert(leader3, Equals, leader1)
}

func (s *integrationTestSuite) waitLeaderChange(c *C, cluster *testCluster, old string) string {
	var leader string
	testutil.WaitUntil(c, func(c *C) bool {
		leader = cluster.GetLeader()
		if leader == old || leader == "" {
			time.Sleep(time.Second)
			return false
		}
		return true
	})
	return leader
}
