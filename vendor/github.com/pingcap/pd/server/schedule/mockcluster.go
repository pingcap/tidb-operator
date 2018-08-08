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

package schedule

import (
	"fmt"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/namespace"
	log "github.com/sirupsen/logrus"
)

// MockCluster is used to mock clusterInfo for test use
type MockCluster struct {
	*BasicCluster
	id *core.MockIDAllocator
	*MockSchedulerOptions
}

// NewMockCluster creates a new MockCluster
func NewMockCluster(opt *MockSchedulerOptions) *MockCluster {
	return &MockCluster{
		BasicCluster:         NewBasicCluster(),
		id:                   core.NewMockIDAllocator(),
		MockSchedulerOptions: opt,
	}
}

func (mc *MockCluster) allocID() (uint64, error) {
	return mc.id.Alloc()
}

// ScanRegions scan region with start key, until number greater than limit.
func (mc *MockCluster) ScanRegions(startKey []byte, limit int) []*core.RegionInfo {
	return mc.Regions.ScanRange(startKey, limit)
}

// LoadRegion put region info without leader
func (mc *MockCluster) LoadRegion(regionID uint64, followerIds ...uint64) {
	//  regions load from etcd will have no leader
	r := mc.newMockRegionInfo(regionID, 0, followerIds...)
	r.Leader = nil
	mc.PutRegion(r)
}

// IsRegionHot checks if the region is hot
func (mc *MockCluster) IsRegionHot(id uint64) bool {
	return mc.BasicCluster.IsRegionHot(id, mc.GetHotRegionLowThreshold())
}

// RandHotRegionFromStore random picks a hot region in specify store.
func (mc *MockCluster) RandHotRegionFromStore(store uint64, kind FlowKind) *core.RegionInfo {
	r := mc.HotCache.RandHotRegionFromStore(store, kind, mc.GetHotRegionLowThreshold())
	if r == nil {
		return nil
	}
	return mc.GetRegion(r.RegionID)
}

// AllocPeer allocs a new peer on a store.
func (mc *MockCluster) AllocPeer(storeID uint64) (*metapb.Peer, error) {
	peerID, err := mc.allocID()
	if err != nil {
		log.Errorf("failed to alloc peer: %v", err)
		return nil, errors.Trace(err)
	}
	peer := &metapb.Peer{
		Id:      peerID,
		StoreId: storeID,
	}
	return peer, nil
}

// SetStoreUp sets store state to be up.
func (mc *MockCluster) SetStoreUp(storeID uint64) {
	store := mc.GetStore(storeID)
	store.State = metapb.StoreState_Up
	store.LastHeartbeatTS = time.Now()
	mc.PutStore(store)
}

// SetStoreDisconnect changes a store's state to disconnected.
func (mc *MockCluster) SetStoreDisconnect(storeID uint64) {
	store := mc.GetStore(storeID)
	store.State = metapb.StoreState_Up
	store.LastHeartbeatTS = time.Now().Add(-time.Second * 30)
	mc.PutStore(store)
}

// SetStoreDown sets store down.
func (mc *MockCluster) SetStoreDown(storeID uint64) {
	store := mc.GetStore(storeID)
	store.State = metapb.StoreState_Up
	store.LastHeartbeatTS = time.Time{}
	mc.PutStore(store)
}

// SetStoreOffline sets store state to be offline.
func (mc *MockCluster) SetStoreOffline(storeID uint64) {
	store := mc.GetStore(storeID)
	store.State = metapb.StoreState_Offline
	mc.PutStore(store)
}

// SetStoreBusy sets store busy.
func (mc *MockCluster) SetStoreBusy(storeID uint64, busy bool) {
	store := mc.GetStore(storeID)
	store.Stats.IsBusy = busy
	store.LastHeartbeatTS = time.Now()
	mc.PutStore(store)
}

// AddLeaderStore adds store with specified count of leader.
func (mc *MockCluster) AddLeaderStore(storeID uint64, leaderCount int) {
	store := core.NewStoreInfo(&metapb.Store{Id: storeID})
	store.Stats = &pdpb.StoreStats{}
	store.LastHeartbeatTS = time.Now()
	store.LeaderCount = leaderCount
	store.LeaderSize = int64(leaderCount) * 10
	store.Stats.Capacity = 1000 * (1 << 20)
	store.Stats.Available = store.Stats.Capacity - uint64(store.LeaderSize)
	mc.PutStore(store)
}

// AddRegionStore adds store with specified count of region.
func (mc *MockCluster) AddRegionStore(storeID uint64, regionCount int) {
	store := core.NewStoreInfo(&metapb.Store{Id: storeID})
	store.Stats = &pdpb.StoreStats{}
	store.LastHeartbeatTS = time.Now()
	store.RegionCount = regionCount
	store.RegionSize = int64(regionCount) * 10
	store.Stats.Capacity = 1000 * (1 << 20)
	store.Stats.Available = store.Stats.Capacity - uint64(store.RegionSize)
	mc.PutStore(store)
}

// AddLabelsStore adds store with specified count of region and labels.
func (mc *MockCluster) AddLabelsStore(storeID uint64, regionCount int, labels map[string]string) {
	mc.AddRegionStore(storeID, regionCount)
	store := mc.GetStore(storeID)
	for k, v := range labels {
		store.Labels = append(store.Labels, &metapb.StoreLabel{Key: k, Value: v})
	}
	mc.PutStore(store)
}

// AddLeaderRegion adds region with specified leader and followers.
func (mc *MockCluster) AddLeaderRegion(regionID uint64, leaderID uint64, followerIds ...uint64) {
	regionInfo := mc.newMockRegionInfo(regionID, leaderID, followerIds...)
	regionInfo.ApproximateSize = 10
	regionInfo.ApproximateRows = 10
	mc.PutRegion(regionInfo)
}

// AddLeaderRegionWithRange adds region with specified leader, followers and key range.
func (mc *MockCluster) AddLeaderRegionWithRange(regionID uint64, startKey string, endKey string, leaderID uint64, followerIds ...uint64) {
	r := mc.newMockRegionInfo(regionID, leaderID, followerIds...)
	r.StartKey = []byte(startKey)
	r.EndKey = []byte(endKey)
	mc.PutRegion(r)
}

// AddLeaderRegionWithReadInfo adds region with specified leader, followers and read info.
func (mc *MockCluster) AddLeaderRegionWithReadInfo(regionID uint64, leaderID uint64, readBytes uint64, followerIds ...uint64) {
	r := mc.newMockRegionInfo(regionID, leaderID, followerIds...)
	r.ReadBytes = readBytes
	isUpdate, item := mc.BasicCluster.CheckReadStatus(r)
	if isUpdate {
		mc.HotCache.Update(regionID, item, ReadFlow)
	}
	mc.PutRegion(r)
}

// AddLeaderRegionWithWriteInfo adds region with specified leader, followers and write info.
func (mc *MockCluster) AddLeaderRegionWithWriteInfo(regionID uint64, leaderID uint64, writtenBytes uint64, followerIds ...uint64) {
	r := mc.newMockRegionInfo(regionID, leaderID, followerIds...)
	r.WrittenBytes = writtenBytes
	isUpdate, item := mc.BasicCluster.CheckWriteStatus(r)
	if isUpdate {
		mc.HotCache.Update(regionID, item, WriteFlow)
	}
	mc.PutRegion(r)
}

// UpdateStoreLeaderWeight updates store leader weight.
func (mc *MockCluster) UpdateStoreLeaderWeight(storeID uint64, weight float64) {
	store := mc.GetStore(storeID)
	store.LeaderWeight = weight
	mc.PutStore(store)
}

// UpdateStoreRegionWeight updates store region weight.
func (mc *MockCluster) UpdateStoreRegionWeight(storeID uint64, weight float64) {
	store := mc.GetStore(storeID)
	store.RegionWeight = weight
	mc.PutStore(store)
}

// UpdateStoreLeaderSize updates store leader size.
func (mc *MockCluster) UpdateStoreLeaderSize(storeID uint64, size int64) {
	store := mc.GetStore(storeID)
	store.LeaderSize = size
	store.Stats.Available = store.Stats.Capacity - uint64(store.LeaderSize)
	mc.PutStore(store)
}

// UpdateStoreRegionSize updates store region size.
func (mc *MockCluster) UpdateStoreRegionSize(storeID uint64, size int64) {
	store := mc.GetStore(storeID)
	store.RegionSize = size
	store.Stats.Available = store.Stats.Capacity - uint64(store.RegionSize)
	mc.PutStore(store)
}

// UpdateLeaderCount updates store leader count.
func (mc *MockCluster) UpdateLeaderCount(storeID uint64, leaderCount int) {
	store := mc.GetStore(storeID)
	store.LeaderCount = leaderCount
	store.LeaderSize = int64(leaderCount) * 10
	mc.PutStore(store)
}

// UpdateRegionCount updates store region count.
func (mc *MockCluster) UpdateRegionCount(storeID uint64, regionCount int) {
	store := mc.GetStore(storeID)
	store.RegionCount = regionCount
	store.RegionSize = int64(regionCount) * 10
	mc.PutStore(store)
}

// UpdateSnapshotCount updates store snapshot count.
func (mc *MockCluster) UpdateSnapshotCount(storeID uint64, snapshotCount int) {
	store := mc.GetStore(storeID)
	store.Stats.ApplyingSnapCount = uint32(snapshotCount)
	mc.PutStore(store)
}

// UpdatePendingPeerCount updates store pending peer count.
func (mc *MockCluster) UpdatePendingPeerCount(storeID uint64, pendingPeerCount int) {
	store := mc.GetStore(storeID)
	store.PendingPeerCount = pendingPeerCount
	mc.PutStore(store)
}

// UpdateStorageRatio updates store storage ratio count.
func (mc *MockCluster) UpdateStorageRatio(storeID uint64, usedRatio, availableRatio float64) {
	store := mc.GetStore(storeID)
	store.Stats.Capacity = 1000 * (1 << 20)
	store.Stats.UsedSize = uint64(float64(store.Stats.Capacity) * usedRatio)
	store.Stats.Available = uint64(float64(store.Stats.Capacity) * availableRatio)
	mc.PutStore(store)
}

// UpdateStorageWrittenBytes updates store written bytes.
func (mc *MockCluster) UpdateStorageWrittenBytes(storeID uint64, BytesWritten uint64) {
	store := mc.GetStore(storeID)
	store.Stats.BytesWritten = BytesWritten
	now := time.Now().Second()
	interval := &pdpb.TimeInterval{StartTimestamp: uint64(now - storeHeartBeatReportInterval), EndTimestamp: uint64(now)}
	store.Stats.Interval = interval
	mc.PutStore(store)
}

// UpdateStorageReadBytes updates store read bytes.
func (mc *MockCluster) UpdateStorageReadBytes(storeID uint64, BytesRead uint64) {
	store := mc.GetStore(storeID)
	now := time.Now().Second()
	interval := &pdpb.TimeInterval{StartTimestamp: uint64(now - storeHeartBeatReportInterval), EndTimestamp: uint64(now)}
	store.Stats.Interval = interval
	store.Stats.BytesRead = BytesRead
	mc.PutStore(store)
}

// UpdateStoreStatus updates store status.
func (mc *MockCluster) UpdateStoreStatus(id uint64) {
	mc.Stores.SetLeaderCount(id, mc.Regions.GetStoreLeaderCount(id))
	mc.Stores.SetRegionCount(id, mc.Regions.GetStoreRegionCount(id))
	mc.Stores.SetPendingPeerCount(id, mc.Regions.GetStorePendingPeerCount(id))
	mc.Stores.SetLeaderSize(id, mc.Regions.GetStoreLeaderRegionSize(id))
	mc.Stores.SetRegionSize(id, mc.Regions.GetStoreRegionSize(id))
	store := mc.Stores.GetStore(id)
	store.Stats = &pdpb.StoreStats{}
	store.Stats.Capacity = 1000 * (1 << 20)
	store.Stats.Available = store.Stats.Capacity - uint64(store.RegionSize)
	store.Stats.UsedSize = uint64(store.RegionSize)
	mc.PutStore(store)
}

func (mc *MockCluster) newMockRegionInfo(regionID uint64, leaderID uint64, followerIds ...uint64) *core.RegionInfo {
	region := &metapb.Region{
		Id:       regionID,
		StartKey: []byte(fmt.Sprintf("%20d", regionID)),
		EndKey:   []byte(fmt.Sprintf("%20d", regionID+1)),
	}
	leader, _ := mc.AllocPeer(leaderID)
	region.Peers = []*metapb.Peer{leader}
	for _, id := range followerIds {
		peer, _ := mc.AllocPeer(id)
		region.Peers = append(region.Peers, peer)
	}

	return core.NewRegionInfo(region, leader)
}

// ApplyOperator mocks apply oeprator.
func (mc *MockCluster) ApplyOperator(op *Operator) {
	region := mc.GetRegion(op.RegionID())
	for !op.IsFinish() {
		if step := op.Check(region); step != nil {
			switch s := step.(type) {
			case TransferLeader:
				region.Leader = region.GetStorePeer(s.ToStore)
			case AddPeer:
				if region.GetStorePeer(s.ToStore) != nil {
					panic("Add peer that exists")
				}
				peer := &metapb.Peer{
					Id:      s.PeerID,
					StoreId: s.ToStore,
				}
				region.AddPeer(peer)
			case RemovePeer:
				if region.GetStorePeer(s.FromStore) == nil {
					panic("Remove peer that doesn't exist")
				}
				region.RemoveStorePeer(s.FromStore)
			case AddLearner:
				if region.GetStorePeer(s.ToStore) != nil {
					panic("Add learner that exists")
				}
				peer := &metapb.Peer{
					Id:        s.PeerID,
					StoreId:   s.ToStore,
					IsLearner: true,
				}
				region.AddPeer(peer)
			case PromoteLearner:
				if region.GetStoreLearner(s.ToStore) == nil {
					panic("promote peer that doesn't exist")
				}
				region.RemoveStorePeer(s.ToStore)
				peer := &metapb.Peer{
					Id:      s.PeerID,
					StoreId: s.ToStore,
				}
				region.AddPeer(peer)
			default:
				panic("Unknown operator step")
			}
		}
	}
	mc.PutRegion(region)
	for id := range region.GetStoreIds() {
		mc.UpdateStoreStatus(id)
	}
}

// GetOpt mocks method.
func (mc *MockCluster) GetOpt() NamespaceOptions {
	return mc.MockSchedulerOptions
}

// GetLeaderScheduleLimit mocks method.
func (mc *MockCluster) GetLeaderScheduleLimit() uint64 {
	return mc.MockSchedulerOptions.GetLeaderScheduleLimit(namespace.DefaultNamespace)
}

// GetRegionScheduleLimit mocks method.
func (mc *MockCluster) GetRegionScheduleLimit() uint64 {
	return mc.MockSchedulerOptions.GetRegionScheduleLimit(namespace.DefaultNamespace)
}

// GetReplicaScheduleLimit mocks method.
func (mc *MockCluster) GetReplicaScheduleLimit() uint64 {
	return mc.MockSchedulerOptions.GetReplicaScheduleLimit(namespace.DefaultNamespace)
}

// GetMergeScheduleLimit mocks method.
func (mc *MockCluster) GetMergeScheduleLimit() uint64 {
	return mc.MockSchedulerOptions.GetMergeScheduleLimit(namespace.DefaultNamespace)
}

// GetMaxReplicas mocks method.
func (mc *MockCluster) GetMaxReplicas() int {
	return mc.MockSchedulerOptions.GetMaxReplicas(namespace.DefaultNamespace)
}

// CheckLabelProperty checks label property.
func (mc *MockCluster) CheckLabelProperty(typ string, labels []*metapb.StoreLabel) bool {
	for _, cfg := range mc.LabelProperties[typ] {
		for _, l := range labels {
			if l.Key == cfg.Key && l.Value == cfg.Value {
				return true
			}
		}
	}
	return false
}

const (
	defaultMaxReplicas          = 3
	defaultMaxSnapshotCount     = 3
	defaultMaxPendingPeerCount  = 16
	defaultMaxMergeRegionSize   = 0
	defaultMaxMergeRegionRows   = 0
	defaultSplitMergeInterval   = 0
	defaultMaxStoreDownTime     = 30 * time.Minute
	defaultLeaderScheduleLimit  = 4
	defaultRegionScheduleLimit  = 4
	defaultReplicaScheduleLimit = 8
	defaultMergeScheduleLimit   = 8
	defaultTolerantSizeRatio    = 2.5
	defaultLowSpaceRatio        = 0.8
	defaultHighSpaceRatio       = 0.6
)

// MockSchedulerOptions is a mock of SchedulerOptions
// which implements Options interface
type MockSchedulerOptions struct {
	RegionScheduleLimit   uint64
	LeaderScheduleLimit   uint64
	ReplicaScheduleLimit  uint64
	MergeScheduleLimit    uint64
	MaxSnapshotCount      uint64
	MaxPendingPeerCount   uint64
	MaxMergeRegionSize    uint64
	MaxMergeRegionRows    uint64
	SplitMergeInterval    time.Duration
	MaxStoreDownTime      time.Duration
	MaxReplicas           int
	LocationLabels        []string
	HotRegionLowThreshold int
	TolerantSizeRatio     float64
	LowSpaceRatio         float64
	HighSpaceRatio        float64
	DisableLearner        bool
	LabelProperties       map[string][]*metapb.StoreLabel
}

// NewMockSchedulerOptions creates a mock schedule option.
func NewMockSchedulerOptions() *MockSchedulerOptions {
	mso := &MockSchedulerOptions{}
	mso.RegionScheduleLimit = defaultRegionScheduleLimit
	mso.LeaderScheduleLimit = defaultLeaderScheduleLimit
	mso.ReplicaScheduleLimit = defaultReplicaScheduleLimit
	mso.MergeScheduleLimit = defaultMergeScheduleLimit
	mso.MaxSnapshotCount = defaultMaxSnapshotCount
	mso.MaxMergeRegionSize = defaultMaxMergeRegionSize
	mso.MaxMergeRegionRows = defaultMaxMergeRegionRows
	mso.SplitMergeInterval = defaultSplitMergeInterval
	mso.MaxStoreDownTime = defaultMaxStoreDownTime
	mso.MaxReplicas = defaultMaxReplicas
	mso.HotRegionLowThreshold = HotRegionLowThreshold
	mso.MaxPendingPeerCount = defaultMaxPendingPeerCount
	mso.TolerantSizeRatio = defaultTolerantSizeRatio
	mso.LowSpaceRatio = defaultLowSpaceRatio
	mso.HighSpaceRatio = defaultHighSpaceRatio
	return mso
}

// GetLeaderScheduleLimit mock method
func (mso *MockSchedulerOptions) GetLeaderScheduleLimit(name string) uint64 {
	return mso.LeaderScheduleLimit
}

// GetRegionScheduleLimit mock method
func (mso *MockSchedulerOptions) GetRegionScheduleLimit(name string) uint64 {
	return mso.RegionScheduleLimit
}

// GetReplicaScheduleLimit mock method
func (mso *MockSchedulerOptions) GetReplicaScheduleLimit(name string) uint64 {
	return mso.ReplicaScheduleLimit
}

// GetMergeScheduleLimit mock method
func (mso *MockSchedulerOptions) GetMergeScheduleLimit(name string) uint64 {
	return mso.MergeScheduleLimit
}

// GetMaxSnapshotCount mock method
func (mso *MockSchedulerOptions) GetMaxSnapshotCount() uint64 {
	return mso.MaxSnapshotCount
}

// GetMaxPendingPeerCount mock method
func (mso *MockSchedulerOptions) GetMaxPendingPeerCount() uint64 {
	return mso.MaxPendingPeerCount
}

// GetMaxMergeRegionSize mock method
func (mso *MockSchedulerOptions) GetMaxMergeRegionSize() uint64 {
	return mso.MaxMergeRegionSize
}

// GetMaxMergeRegionRows mock method
func (mso *MockSchedulerOptions) GetMaxMergeRegionRows() uint64 {
	return mso.MaxMergeRegionRows
}

// GetSplitMergeInterval mock method
func (mso *MockSchedulerOptions) GetSplitMergeInterval() time.Duration {
	return mso.SplitMergeInterval
}

// GetMaxStoreDownTime mock method
func (mso *MockSchedulerOptions) GetMaxStoreDownTime() time.Duration {
	return mso.MaxStoreDownTime
}

// GetMaxReplicas mock method
func (mso *MockSchedulerOptions) GetMaxReplicas(name string) int {
	return mso.MaxReplicas
}

// GetLocationLabels mock method
func (mso *MockSchedulerOptions) GetLocationLabels() []string {
	return mso.LocationLabels
}

// GetHotRegionLowThreshold mock method
func (mso *MockSchedulerOptions) GetHotRegionLowThreshold() int {
	return mso.HotRegionLowThreshold
}

// GetTolerantSizeRatio mock method
func (mso *MockSchedulerOptions) GetTolerantSizeRatio() float64 {
	return mso.TolerantSizeRatio
}

// GetLowSpaceRatio mock method
func (mso *MockSchedulerOptions) GetLowSpaceRatio() float64 {
	return mso.LowSpaceRatio
}

// GetHighSpaceRatio mock method
func (mso *MockSchedulerOptions) GetHighSpaceRatio() float64 {
	return mso.HighSpaceRatio
}

// SetMaxReplicas mock method
func (mso *MockSchedulerOptions) SetMaxReplicas(replicas int) {
	mso.MaxReplicas = replicas
}

// IsRaftLearnerEnabled mock method
func (mso *MockSchedulerOptions) IsRaftLearnerEnabled() bool {
	return !mso.DisableLearner
}
