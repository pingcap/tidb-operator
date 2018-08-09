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

package testutil

import (
	check "github.com/pingcap/check"
	"github.com/pingcap/pd/server/schedule"
)

// CheckAddPeer checks if the operator is to add peer on specified store.
func CheckAddPeer(c *check.C, op *schedule.Operator, kind schedule.OperatorKind, storeID uint64) {
	c.Assert(op, check.NotNil)
	c.Assert(op.Len(), check.Equals, 2)
	c.Assert(op.Step(0).(schedule.AddLearner).ToStore, check.Equals, storeID)
	c.Assert(op.Step(1), check.FitsTypeOf, schedule.PromoteLearner{})
	kind |= schedule.OpRegion
	c.Assert(op.Kind()&kind, check.Equals, kind)
}

// CheckRemovePeer checks if the operator is to remove peer on specified store.
func CheckRemovePeer(c *check.C, op *schedule.Operator, storeID uint64) {
	if op.Len() == 1 {
		c.Assert(op.Step(0).(schedule.RemovePeer).FromStore, check.Equals, storeID)
	} else {
		c.Assert(op.Len(), check.Equals, 2)
		c.Assert(op.Step(0).(schedule.TransferLeader).FromStore, check.Equals, storeID)
		c.Assert(op.Step(1).(schedule.RemovePeer).FromStore, check.Equals, storeID)
	}
}

// CheckTransferLeader checks if the operator is to transfer leader between the specified source and target stores.
func CheckTransferLeader(c *check.C, op *schedule.Operator, kind schedule.OperatorKind, sourceID, targetID uint64) {
	c.Assert(op, check.NotNil)
	c.Assert(op.Len(), check.Equals, 1)
	c.Assert(op.Step(0), check.Equals, schedule.TransferLeader{FromStore: sourceID, ToStore: targetID})
	kind |= schedule.OpLeader
	c.Assert(op.Kind()&kind, check.Equals, kind)
}

// CheckTransferLeaderFrom checks if the operator is to transfer leader out of the specified store.
func CheckTransferLeaderFrom(c *check.C, op *schedule.Operator, kind schedule.OperatorKind, sourceID uint64) {
	c.Assert(op, check.NotNil)
	c.Assert(op.Len(), check.Equals, 1)
	c.Assert(op.Step(0).(schedule.TransferLeader).FromStore, check.Equals, sourceID)
	kind |= schedule.OpLeader
	c.Assert(op.Kind()&kind, check.Equals, kind)
}

// CheckTransferPeer checks if the operator is to transfer peer between the specified source and target stores.
func CheckTransferPeer(c *check.C, op *schedule.Operator, kind schedule.OperatorKind, sourceID, targetID uint64) {
	c.Assert(op, check.NotNil)
	if op.Len() == 3 {
		c.Assert(op.Step(0).(schedule.AddLearner).ToStore, check.Equals, targetID)
		c.Assert(op.Step(1), check.FitsTypeOf, schedule.PromoteLearner{})
		c.Assert(op.Step(2).(schedule.RemovePeer).FromStore, check.Equals, sourceID)
	} else {
		c.Assert(op.Len(), check.Equals, 4)
		c.Assert(op.Step(0).(schedule.AddLearner).ToStore, check.Equals, targetID)
		c.Assert(op.Step(1), check.FitsTypeOf, schedule.PromoteLearner{})
		c.Assert(op.Step(2).(schedule.TransferLeader).FromStore, check.Equals, sourceID)
		c.Assert(op.Step(3).(schedule.RemovePeer).FromStore, check.Equals, sourceID)
		kind |= schedule.OpLeader
	}
	kind |= schedule.OpRegion
	c.Assert(op.Kind()&kind, check.Equals, kind)
}

// CheckTransferPeerWithLeaderTransfer checks if the operator is to transfer
// peer between the specified source and target stores and it meanwhile
// trasnfers the leader out of source store.
func CheckTransferPeerWithLeaderTransfer(c *check.C, op *schedule.Operator, kind schedule.OperatorKind, sourceID, targetID uint64) {
	c.Assert(op, check.NotNil)
	c.Assert(op.Len(), check.Equals, 4)
	CheckTransferPeer(c, op, kind, sourceID, targetID)
}

// CheckTransferPeerWithLeaderTransferFrom checks if the operator is to transfer
// peer out of the specified store and it meanwhile transfers the leader out of
// the store.
func CheckTransferPeerWithLeaderTransferFrom(c *check.C, op *schedule.Operator, kind schedule.OperatorKind, sourceID uint64) {
	c.Assert(op, check.NotNil)
	c.Assert(op.Len(), check.Equals, 4)
	c.Assert(op.Step(0), check.FitsTypeOf, schedule.AddLearner{})
	c.Assert(op.Step(1), check.FitsTypeOf, schedule.PromoteLearner{})
	c.Assert(op.Step(2), check.FitsTypeOf, schedule.TransferLeader{})
	c.Assert(op.Step(3).(schedule.RemovePeer).FromStore, check.Equals, sourceID)
	kind |= (schedule.OpRegion | schedule.OpLeader)
	c.Assert(op.Kind()&kind, check.Equals, kind)
}
