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

package server

import (
	"context"
	"time"

	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/pkg/testutil"
	"github.com/pingcap/pd/pkg/typeutil"
)

var _ = Suite(&testHeartbeatStreamSuite{})

type testHeartbeatStreamSuite struct {
	testClusterBaseSuite
	region *metapb.Region
}

func (s *testHeartbeatStreamSuite) SetUpSuite(c *C) {
	s.svr, s.cleanup = newTestServer(c)
	s.svr.cfg.heartbeatStreamBindInterval = typeutil.NewDuration(time.Second)
	err := s.svr.Run(context.TODO())
	c.Assert(err, IsNil)
	mustWaitLeader(c, []*Server{s.svr})
	s.grpcPDClient = mustNewGrpcClient(c, s.svr.GetAddr())

	bootstrapReq := s.newBootstrapRequest(c, s.svr.clusterID, "127.0.0.1:0")
	_, err = s.svr.bootstrapCluster(bootstrapReq)
	c.Assert(err, IsNil)
	s.region = bootstrapReq.Region
}

func (s *testHeartbeatStreamSuite) TearDownSuite(c *C) {
	s.cleanup()
}

func (s *testHeartbeatStreamSuite) TestActivity(c *C) {
	// Add a new store and an addPeer operator.
	storeID, err := s.svr.idAlloc.Alloc()
	c.Assert(err, IsNil)
	putStore(c, s.grpcPDClient, s.svr.clusterID, &metapb.Store{Id: storeID, Address: "127.0.0.1:1"})
	newHandler(s.svr).AddAddPeerOperator(s.region.GetId(), storeID)

	stream1, stream2 := newRegionheartbeatClient(c, s.grpcPDClient), newRegionheartbeatClient(c, s.grpcPDClient)
	checkActiveStream := func() int {
		select {
		case <-stream1.respCh:
			return 1
		case <-stream2.respCh:
			return 2
		case <-time.After(time.Second):
			return 0
		}
	}

	req := &pdpb.RegionHeartbeatRequest{
		Header: newRequestHeader(s.svr.clusterID),
		Leader: s.region.Peers[0],
		Region: s.region,
	}
	// Active stream is stream1.
	stream1.stream.Send(req)
	c.Assert(checkActiveStream(), Equals, 1)
	// Rebind to stream2.
	stream2.stream.Send(req)
	c.Assert(checkActiveStream(), Equals, 2)
	// Rebind to stream1 if no more heartbeats sent through stream2.
	testutil.WaitUntil(c, func(c *C) bool {
		stream1.stream.Send(req)
		return checkActiveStream() == 1
	})
}

type regionHeartbeatClient struct {
	stream pdpb.PD_RegionHeartbeatClient
	respCh chan *pdpb.RegionHeartbeatResponse
}

func newRegionheartbeatClient(c *C, grpcClient pdpb.PDClient) *regionHeartbeatClient {
	stream, err := grpcClient.RegionHeartbeat(context.Background())
	c.Assert(err, IsNil)
	ch := make(chan *pdpb.RegionHeartbeatResponse)
	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				return
			}
			ch <- res
		}
	}()
	return &regionHeartbeatClient{
		stream: stream,
		respCh: ch,
	}
}

func (c *regionHeartbeatClient) close() {
	c.stream.CloseSend()
}

func (c *regionHeartbeatClient) SendRecv(msg *pdpb.RegionHeartbeatRequest, timeout time.Duration) *pdpb.RegionHeartbeatResponse {
	c.stream.Send(msg)
	select {
	case <-time.After(timeout):
		return nil
	case res := <-c.respCh:
		return res
	}
}
