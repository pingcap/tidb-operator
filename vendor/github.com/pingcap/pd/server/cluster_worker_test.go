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
	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
)

var _ = Suite(&testClusterWorkerSuite{})

type testClusterWorkerSuite struct{}

func (s *testClusterWorkerSuite) TestReportSplit(c *C) {
	var cluster RaftCluster
	left := &metapb.Region{Id: 1, StartKey: []byte("a"), EndKey: []byte("b")}
	right := &metapb.Region{Id: 2, StartKey: []byte("b"), EndKey: []byte("c")}
	_, err := cluster.handleReportSplit(&pdpb.ReportSplitRequest{Left: left, Right: right})
	c.Assert(err, IsNil)
	_, err = cluster.handleReportSplit(&pdpb.ReportSplitRequest{Left: right, Right: left})
	c.Assert(err, NotNil)
}
