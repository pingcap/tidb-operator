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
package schedule

import (
	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server/core"
)

var _ = Suite(&testFiltersSuite{})

type testFiltersSuite struct{}

func (s *testReplicationSuite) TestPendingPeerFilter(c *C) {
	filter := NewPendingPeerCountFilter()
	opt := NewMockSchedulerOptions()
	tc := NewMockCluster(opt)
	store := core.NewStoreInfo(&metapb.Store{Id: 1})
	c.Assert(filter.FilterSource(tc, store), IsFalse)
	store.PendingPeerCount = 30
	c.Assert(filter.FilterSource(tc, store), IsTrue)
	c.Assert(filter.FilterTarget(tc, store), IsTrue)
	// set to 0 means no limit
	opt.MaxPendingPeerCount = 0
	c.Assert(filter.FilterSource(tc, store), IsFalse)
	c.Assert(filter.FilterTarget(tc, store), IsFalse)
}
