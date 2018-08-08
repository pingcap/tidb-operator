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

package schedule

import (
	"bytes"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	log "github.com/sirupsen/logrus"

	"github.com/pingcap/pd/server/core"
)

const (
	// LeaderOperatorWaitTime is the duration that when a leader operator lives
	// longer than it, the operator will be considered timeout.
	LeaderOperatorWaitTime = 10 * time.Second
	// RegionOperatorWaitTime is the duration that when a region operator lives
	// longer than it, the operator will be considered timeout.
	RegionOperatorWaitTime = 10 * time.Minute
)

// OperatorStep describes the basic scheduling steps that can not be subdivided.
type OperatorStep interface {
	fmt.Stringer
	IsFinish(region *core.RegionInfo) bool
	Influence(opInfluence OpInfluence, region *core.RegionInfo)
}

// TransferLeader is an OperatorStep that transfers a region's leader.
type TransferLeader struct {
	FromStore, ToStore uint64
}

func (tl TransferLeader) String() string {
	return fmt.Sprintf("transfer leader from store %v to store %v", tl.FromStore, tl.ToStore)
}

// IsFinish checks if current step is finished.
func (tl TransferLeader) IsFinish(region *core.RegionInfo) bool {
	return region.Leader.GetStoreId() == tl.ToStore
}

// Influence calculates the store difference that current step make
func (tl TransferLeader) Influence(opInfluence OpInfluence, region *core.RegionInfo) {
	from := opInfluence.GetStoreInfluence(tl.FromStore)
	to := opInfluence.GetStoreInfluence(tl.ToStore)

	from.LeaderSize -= region.ApproximateSize
	from.LeaderCount--
	to.LeaderSize += region.ApproximateSize
	to.LeaderCount++
}

// AddPeer is an OperatorStep that adds a region peer.
type AddPeer struct {
	ToStore, PeerID uint64
}

func (ap AddPeer) String() string {
	return fmt.Sprintf("add peer %v on store %v", ap.PeerID, ap.ToStore)
}

// IsFinish checks if current step is finished.
func (ap AddPeer) IsFinish(region *core.RegionInfo) bool {
	if p := region.GetStoreVoter(ap.ToStore); p != nil {
		if p.GetId() != ap.PeerID {
			log.Warnf("expect %v, but obtain voter %v", ap.String(), p.GetId())
			return false
		}
		return region.GetPendingVoter(p.GetId()) == nil
	}
	return false
}

// Influence calculates the store difference that current step make
func (ap AddPeer) Influence(opInfluence OpInfluence, region *core.RegionInfo) {
	to := opInfluence.GetStoreInfluence(ap.ToStore)

	to.RegionSize += region.ApproximateSize
	to.RegionCount++
}

// AddLearner is an OperatorStep that adds a region learner peer.
type AddLearner struct {
	ToStore, PeerID uint64
}

func (al AddLearner) String() string {
	return fmt.Sprintf("add learner peer %v on store %v", al.PeerID, al.ToStore)
}

// IsFinish checks if current step is finished.
func (al AddLearner) IsFinish(region *core.RegionInfo) bool {
	if p := region.GetStoreLearner(al.ToStore); p != nil {
		if p.GetId() != al.PeerID {
			log.Warnf("expect %v, but obtain learner %v", al.String(), p.GetId())
			return false
		}
		return region.GetPendingLearner(p.GetId()) == nil
	}
	return false
}

// Influence calculates the store difference that current step make
func (al AddLearner) Influence(opInfluence OpInfluence, region *core.RegionInfo) {
	to := opInfluence.GetStoreInfluence(al.ToStore)

	to.RegionSize += region.ApproximateSize
	to.RegionCount++
}

// PromoteLearner is an OperatorStep that promotes a region learner peer to normal voter.
type PromoteLearner struct {
	ToStore, PeerID uint64
}

func (pl PromoteLearner) String() string {
	return fmt.Sprintf("promote learner peer %v on store %v to voter", pl.PeerID, pl.ToStore)
}

// IsFinish checks if current step is finished.
func (pl PromoteLearner) IsFinish(region *core.RegionInfo) bool {
	if p := region.GetStoreVoter(pl.ToStore); p != nil {
		if p.GetId() != pl.PeerID {
			log.Warnf("expect %v, but obtain voter %v", pl.String(), p.GetId())
		}
		return p.GetId() == pl.PeerID
	}
	return false
}

// Influence calculates the store difference that current step make
func (pl PromoteLearner) Influence(opInfluence OpInfluence, region *core.RegionInfo) {}

// RemovePeer is an OperatorStep that removes a region peer.
type RemovePeer struct {
	FromStore uint64
}

func (rp RemovePeer) String() string {
	return fmt.Sprintf("remove peer on store %v", rp.FromStore)
}

// IsFinish checks if current step is finished.
func (rp RemovePeer) IsFinish(region *core.RegionInfo) bool {
	return region.GetStorePeer(rp.FromStore) == nil
}

// Influence calculates the store difference that current step make
func (rp RemovePeer) Influence(opInfluence OpInfluence, region *core.RegionInfo) {
	from := opInfluence.GetStoreInfluence(rp.FromStore)

	from.RegionSize -= region.ApproximateSize
	from.RegionCount--
}

// MergeRegion is an OperatorStep that merge two regions.
type MergeRegion struct {
	FromRegion *metapb.Region
	ToRegion   *metapb.Region
	// there are two regions involved in merge process,
	// so to keep them from other scheduler,
	// both of them should add MerRegion operatorStep.
	// But actually, tikv just need the region want to be merged to get the merge request,
	// thus use a IsPssive mark to indicate that
	// this region doesn't need to send merge request to tikv.
	IsPassive bool
}

func (mr MergeRegion) String() string {
	return fmt.Sprintf("merge region %v into region %v", mr.FromRegion.GetId(), mr.ToRegion.GetId())
}

// IsFinish checks if current step is finished
func (mr MergeRegion) IsFinish(region *core.RegionInfo) bool {
	if mr.IsPassive {
		return bytes.Compare(region.Region.StartKey, mr.ToRegion.StartKey) != 0 || bytes.Compare(region.Region.EndKey, mr.ToRegion.EndKey) != 0
	}
	return false
}

// Influence calculates the store difference that current step make
func (mr MergeRegion) Influence(opInfluence OpInfluence, region *core.RegionInfo) {
	if mr.IsPassive {
		for _, p := range region.GetPeers() {
			o := opInfluence.GetStoreInfluence(p.GetStoreId())
			o.RegionCount--
			if region.Leader.GetId() == p.GetId() {
				o.LeaderCount--
			}
		}
	}
}

// SplitRegion is an OperatorStep that splits a region.
type SplitRegion struct {
	StartKey, EndKey []byte
}

func (sr SplitRegion) String() string {
	return "split region"
}

// IsFinish checks if current step is finished.
func (sr SplitRegion) IsFinish(region *core.RegionInfo) bool {
	return !bytes.Equal(region.StartKey, sr.StartKey) || !bytes.Equal(region.EndKey, sr.EndKey)
}

// Influence calculates the store difference that current step make.
func (sr SplitRegion) Influence(opInfluence OpInfluence, region *core.RegionInfo) {
	for _, p := range region.Peers {
		inf := opInfluence.GetStoreInfluence(p.GetStoreId())
		inf.RegionCount++
		if region.Leader.GetId() == p.GetId() {
			inf.LeaderCount++
		}
	}
}

// Operator contains execution steps generated by scheduler.
type Operator struct {
	desc        string
	regionID    uint64
	regionEpoch *metapb.RegionEpoch
	kind        OperatorKind
	steps       []OperatorStep
	currentStep int32
	createTime  time.Time
	stepTime    int64
	level       core.PriorityLevel
}

// NewOperator creates a new operator.
func NewOperator(desc string, regionID uint64, regionEpoch *metapb.RegionEpoch, kind OperatorKind, steps ...OperatorStep) *Operator {
	return &Operator{
		desc:        desc,
		regionID:    regionID,
		regionEpoch: regionEpoch,
		kind:        kind,
		steps:       steps,
		createTime:  time.Now(),
		stepTime:    time.Now().UnixNano(),
		level:       core.NormalPriority,
	}
}

func (o *Operator) String() string {
	s := fmt.Sprintf("%s (kind:%s, region:%v(%v,%v), createAt:%s, currentStep:%v, steps:%+v) ", o.desc, o.kind, o.regionID, o.regionEpoch.GetVersion(), o.regionEpoch.GetConfVer(), o.createTime, atomic.LoadInt32(&o.currentStep), o.steps)
	if o.IsTimeout() {
		s = s + "timeout"
	}
	if o.IsFinish() {
		s = s + "finished"
	}
	return s
}

// MarshalJSON serialize custom types to JSON
func (o *Operator) MarshalJSON() ([]byte, error) {
	return []byte(`"` + o.String() + `"`), nil
}

// Desc returns the operator's short description.
func (o *Operator) Desc() string {
	return o.desc
}

// SetDesc sets the description for the operator.
func (o *Operator) SetDesc(desc string) {
	o.desc = desc
}

// AttachKind attaches an operator kind for the operator.
func (o *Operator) AttachKind(kind OperatorKind) {
	o.kind |= kind
}

// RegionID returns the region that operator is targeted.
func (o *Operator) RegionID() uint64 {
	return o.regionID
}

// RegionEpoch returns the region's epoch that is attched to the operator.
func (o *Operator) RegionEpoch() *metapb.RegionEpoch {
	return o.regionEpoch
}

// Kind returns operator's kind.
func (o *Operator) Kind() OperatorKind {
	return o.kind
}

// ElapsedTime returns duration since it was created.
func (o *Operator) ElapsedTime() time.Duration {
	return time.Since(o.createTime)
}

// Len returns the operator's steps count.
func (o *Operator) Len() int {
	return len(o.steps)
}

// Step returns the i-th step.
func (o *Operator) Step(i int) OperatorStep {
	if i >= 0 && i < len(o.steps) {
		return o.steps[i]
	}
	return nil
}

// Check checks if current step is finished, returns next step to take action.
// It's safe to be called by multiple goroutine concurrently.
func (o *Operator) Check(region *core.RegionInfo) OperatorStep {
	for step := atomic.LoadInt32(&o.currentStep); int(step) < len(o.steps); step++ {
		if o.steps[int(step)].IsFinish(region) {
			operatorStepDuration.WithLabelValues(reflect.TypeOf(o.steps[int(step)]).Name()).
				Observe(time.Since(time.Unix(0, atomic.LoadInt64(&o.stepTime))).Seconds())
			atomic.StoreInt32(&o.currentStep, step+1)
			atomic.StoreInt64(&o.stepTime, time.Now().UnixNano())
		} else {
			return o.steps[int(step)]
		}
	}
	return nil
}

// SetPriorityLevel set the priority level for operator
func (o *Operator) SetPriorityLevel(level core.PriorityLevel) {
	o.level = level
}

// GetPriorityLevel get the priority level
func (o *Operator) GetPriorityLevel() core.PriorityLevel {
	return o.level
}

// IsFinish checks if all steps are finished.
func (o *Operator) IsFinish() bool {
	return atomic.LoadInt32(&o.currentStep) >= int32(len(o.steps))
}

// IsTimeout checks the operator's create time and determines if it is timeout.
func (o *Operator) IsTimeout() bool {
	if o.IsFinish() {
		return false
	}
	if o.kind&OpRegion != 0 {
		return time.Since(o.createTime) > RegionOperatorWaitTime
	}
	return time.Since(o.createTime) > LeaderOperatorWaitTime
}

// Influence calculates the store difference which unfinished operator steps make
func (o *Operator) Influence(opInfluence OpInfluence, region *core.RegionInfo) {
	for step := atomic.LoadInt32(&o.currentStep); int(step) < len(o.steps); step++ {
		if !o.steps[int(step)].IsFinish(region) {
			o.steps[int(step)].Influence(opInfluence, region)
		}
	}
}

// OperatorHistory is used to log and visualize completed operators.
type OperatorHistory struct {
	FinishTime time.Time
	From, To   uint64
	Kind       core.ResourceKind
}

// History transfers the operator's steps to operator histories.
func (o *Operator) History() []OperatorHistory {
	now := time.Now()
	var histories []OperatorHistory
	var addPeerStores, removePeerStores []uint64
	for _, step := range o.steps {
		switch s := step.(type) {
		case TransferLeader:
			histories = append(histories, OperatorHistory{
				FinishTime: now,
				From:       s.FromStore,
				To:         s.ToStore,
				Kind:       core.LeaderKind,
			})
		case AddPeer:
			addPeerStores = append(addPeerStores, s.ToStore)
		case AddLearner:
			addPeerStores = append(addPeerStores, s.ToStore)
		case RemovePeer:
			removePeerStores = append(removePeerStores, s.FromStore)
		}
	}
	for i := range addPeerStores {
		if i < len(removePeerStores) {
			histories = append(histories, OperatorHistory{
				FinishTime: now,
				From:       removePeerStores[i],
				To:         addPeerStores[i],
				Kind:       core.RegionKind,
			})
		}
	}
	return histories
}

// CreateRemovePeerOperator creates an Operator that removes a peer from region.
func CreateRemovePeerOperator(desc string, cluster Cluster, kind OperatorKind, region *core.RegionInfo, storeID uint64) *Operator {
	removeKind, steps := removePeerSteps(cluster, region, storeID)
	return NewOperator(desc, region.GetId(), region.GetRegionEpoch(), removeKind|kind, steps...)
}

// CreateMovePeerOperator creates an Operator that replaces an old peer with a new peer.
func CreateMovePeerOperator(desc string, cluster Cluster, region *core.RegionInfo, kind OperatorKind, oldStore, newStore uint64, peerID uint64) *Operator {
	removeKind, steps := removePeerSteps(cluster, region, oldStore)
	var st []OperatorStep
	if cluster.IsRaftLearnerEnabled() {
		st = []OperatorStep{
			AddLearner{ToStore: newStore, PeerID: peerID},
			PromoteLearner{ToStore: newStore, PeerID: peerID},
		}
	} else {
		st = []OperatorStep{
			AddPeer{ToStore: newStore, PeerID: peerID},
		}
	}
	steps = append(st, steps...)
	return NewOperator(desc, region.GetId(), region.GetRegionEpoch(), removeKind|kind|OpRegion, steps...)
}

// removePeerSteps returns the steps to safely remove a peer. It prevents removing leader by transfer its leadership first.
func removePeerSteps(cluster Cluster, region *core.RegionInfo, storeID uint64) (kind OperatorKind, steps []OperatorStep) {
	if region.Leader != nil && region.Leader.GetStoreId() == storeID {
		for id := range region.GetFollowers() {
			follower := cluster.GetStore(id)
			if follower != nil && !cluster.CheckLabelProperty(RejectLeader, follower.Labels) {
				steps = append(steps, TransferLeader{FromStore: storeID, ToStore: id})
				kind = OpLeader
				break
			}
		}
	}
	steps = append(steps, RemovePeer{FromStore: storeID})
	kind |= OpRegion
	return
}

// CreateMergeRegionOperator creates an Operator that merge two region into one
func CreateMergeRegionOperator(desc string, cluster Cluster, source *core.RegionInfo, target *core.RegionInfo, kind OperatorKind) (*Operator, *Operator, error) {
	steps, kinds, err := matchPeerSteps(cluster, source, target)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	steps = append(steps, MergeRegion{
		FromRegion: source.Region,
		ToRegion:   target.Region,
		IsPassive:  false,
	})

	op1 := NewOperator(desc, source.GetId(), source.GetRegionEpoch(), kinds|kind, steps...)
	op2 := NewOperator(desc, target.GetId(), target.GetRegionEpoch(), kind, MergeRegion{
		FromRegion: source.Region,
		ToRegion:   target.Region,
		IsPassive:  true,
	})

	return op1, op2, nil
}

// matchPeerSteps returns the steps to match the location of peer stores of source region with target's.
func matchPeerSteps(cluster Cluster, source *core.RegionInfo, target *core.RegionInfo) ([]OperatorStep, OperatorKind, error) {
	storeIDs := make(map[uint64]struct{})
	var steps []OperatorStep
	var kind OperatorKind

	sourcePeers := source.Region.GetPeers()
	targetPeers := target.Region.GetPeers()

	for _, peer := range targetPeers {
		storeIDs[peer.GetStoreId()] = struct{}{}
	}

	// Add missing peers.
	for id := range storeIDs {
		if source.GetStorePeer(id) != nil {
			continue
		}
		peer, err := cluster.AllocPeer(id)
		if err != nil {
			log.Debugf("peer alloc failed: %v", err)
			return nil, kind, errors.Trace(err)
		}
		if cluster.IsRaftLearnerEnabled() {
			steps = append(steps,
				AddLearner{ToStore: id, PeerID: peer.Id},
				PromoteLearner{ToStore: id, PeerID: peer.Id},
			)
		} else {
			steps = append(steps, AddPeer{ToStore: id, PeerID: peer.Id})
		}
		kind |= OpRegion
	}

	// Check whether to transfer leader or not
	intersection := getIntersectionStores(sourcePeers, targetPeers)
	leaderID := source.Leader.GetStoreId()
	isFound := false
	for _, storeID := range intersection {
		if storeID == leaderID {
			isFound = true
			break
		}
	}
	if !isFound {
		steps = append(steps, TransferLeader{FromStore: source.Leader.GetStoreId(), ToStore: target.Leader.GetStoreId()})
		kind |= OpLeader
	}

	// Remove redundant peers.
	for _, peer := range sourcePeers {
		if _, ok := storeIDs[peer.GetStoreId()]; ok {
			continue
		}
		steps = append(steps, RemovePeer{FromStore: peer.GetStoreId()})
		kind |= OpRegion
	}

	return steps, kind, nil
}

// getIntersectionStores returns the stores included in two region's peers.
func getIntersectionStores(a []*metapb.Peer, b []*metapb.Peer) []uint64 {
	set := make([]uint64, 0)
	hash := make(map[uint64]struct{})

	for _, peer := range a {
		hash[peer.GetStoreId()] = struct{}{}
	}

	for _, peer := range b {
		if _, found := hash[peer.GetStoreId()]; found {
			set = append(set, peer.GetStoreId())
		}
	}

	return set
}
