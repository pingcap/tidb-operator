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
	"context"
	"os"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/server"
	"github.com/pingcap/pd/server/api"
)

// testServer states.
const (
	Initial int32 = iota
	Running
	Stop
	Destroy
)

type testServer struct {
	sync.RWMutex
	server *server.Server
	state  int32
}

var initHTTPClientOnce sync.Once

func newTestServer(cfg *server.Config) (*testServer, error) {
	err := server.PrepareJoinCluster(cfg)
	if err != nil {
		return nil, errors.Trace(err)
	}
	svr, err := server.CreateServer(cfg, api.NewHandler)
	if err != nil {
		return nil, errors.Trace(err)
	}
	initHTTPClientOnce.Do(func() {
		err = server.InitHTTPClient(svr)
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &testServer{
		server: svr,
		state:  Initial,
	}, nil
}

func (s *testServer) Run(ctx context.Context) error {
	s.Lock()
	defer s.Unlock()
	if s.state != Initial && s.state != Stop {
		return errors.Errorf("server(state%d) cannot run", s.state)
	}
	if err := s.server.Run(ctx); err != nil {
		return errors.Trace(err)
	}
	s.state = Running
	return nil
}

func (s *testServer) Stop() error {
	s.Lock()
	defer s.Unlock()
	if s.state != Running {
		return errors.Errorf("server(state%d) cannot stop", s.state)
	}
	s.server.Close()
	s.state = Stop
	return nil
}

func (s *testServer) Destory() error {
	s.Lock()
	defer s.Unlock()
	if s.state == Running {
		s.server.Close()
	}
	os.RemoveAll(s.server.GetConfig().DataDir)
	s.state = Destroy
	return nil
}

func (s *testServer) State() int32 {
	s.RLock()
	defer s.RUnlock()
	return s.state
}

func (s *testServer) GetConfig() *server.Config {
	s.RLock()
	defer s.RUnlock()
	return s.server.GetConfig()
}

func (s *testServer) GetClusterID() uint64 {
	s.RLock()
	defer s.RUnlock()
	return s.server.ClusterID()
}

func (s *testServer) GetServerID() uint64 {
	s.RLock()
	defer s.RUnlock()
	return s.server.ID()
}

func (s *testServer) IsLeader() bool {
	s.RLock()
	defer s.RUnlock()
	return s.server.IsLeader()
}

func (s *testServer) GetEtcdLeader() (string, error) {
	s.RLock()
	defer s.RUnlock()
	req := &pdpb.GetMembersRequest{Header: &pdpb.RequestHeader{ClusterId: s.server.ClusterID()}}
	members, err := s.server.GetMembers(context.TODO(), req)
	if err != nil {
		return "", errors.Trace(err)
	}
	return members.GetEtcdLeader().GetName(), nil
}

func (s *testServer) GetEtcdClient() *clientv3.Client {
	s.RLock()
	defer s.RUnlock()
	return s.server.GetClient()
}

type testCluster struct {
	config  *clusterConfig
	servers map[string]*testServer
}

func newTestCluster(initialServerCount int) (*testCluster, error) {
	config := newClusterConfig(initialServerCount)
	servers := make(map[string]*testServer)
	for _, conf := range config.InitialServers {
		serverConf, err := conf.Generate()
		if err != nil {
			return nil, errors.Trace(err)
		}
		s, err := newTestServer(serverConf)
		if err != nil {
			return nil, errors.Trace(err)
		}
		servers[conf.Name] = s
	}
	return &testCluster{
		config:  config,
		servers: servers,
	}, nil
}

func (c *testCluster) RunServer(ctx context.Context, server *testServer) <-chan error {
	resC := make(chan error)
	go func() { resC <- server.Run(ctx) }()
	return resC
}

func (c *testCluster) RunServers(ctx context.Context, servers []*testServer) error {
	res := make([]<-chan error, len(servers))
	for i, s := range servers {
		res[i] = c.RunServer(ctx, s)
	}
	for _, c := range res {
		if err := <-c; err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (c *testCluster) RunInitialServers() error {
	var servers []*testServer
	for _, conf := range c.config.InitialServers {
		servers = append(servers, c.GetServer(conf.Name))
	}
	return c.RunServers(context.Background(), servers)
}

func (c *testCluster) StopAll() error {
	for _, s := range c.servers {
		if err := s.Stop(); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (c *testCluster) GetServer(name string) *testServer {
	return c.servers[name]
}

func (c *testCluster) GetLeader() string {
	for name, s := range c.servers {
		if s.IsLeader() {
			return name
		}
	}
	return ""
}

func (c *testCluster) WaitLeader() string {
	for i := 0; i < 100; i++ {
		if leader := c.GetLeader(); leader != "" {
			return leader
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ""
}

func (c *testCluster) Join() (*testServer, error) {
	conf, err := c.config.Join().Generate()
	if err != nil {
		return nil, errors.Trace(err)
	}
	s, err := newTestServer(conf)
	if err != nil {
		return nil, errors.Trace(err)
	}
	c.servers[conf.Name] = s
	return s, nil
}

func (c *testCluster) Destory() error {
	for _, s := range c.servers {
		err := s.Destory()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
