// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"bytes"
	"strconv"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrNotBootstrapped is error info for cluster not bootstrapped
	ErrNotBootstrapped = errors.New("TiKV cluster not bootstrapped, please start TiKV first")
	// ErrOperatorNotFound is error info for operator not found
	ErrOperatorNotFound = errors.New("operator not found")
	// ErrAddOperator is error info for already have an operator when adding operator
	ErrAddOperator = errors.New("failed to add operator, maybe already have one")
	// ErrRegionNotAdjacent is error info for region not adjacent
	ErrRegionNotAdjacent = errors.New("two regions are not adjacent")
	// ErrRegionNotFound is error info for region not found
	ErrRegionNotFound = func(regionID uint64) error {
		return errors.Errorf("region %v not found", regionID)
	}
	// ErrRegionAbnormalPeer is error info for region has abonormal peer
	ErrRegionAbnormalPeer = func(regionID uint64) error {
		return errors.Errorf("region %v has abnormal peer", regionID)
	}
	// ErrRegionIsStale is error info for region is stale
	ErrRegionIsStale = func(region *metapb.Region, origin *metapb.Region) error {
		return errors.Errorf("region is stale: region %v origin %v", region, origin)
	}
)

// Handler is a helper to export methods to handle API/RPC requests.
type Handler struct {
	s   *Server
	opt *scheduleOption
}

func newHandler(s *Server) *Handler {
	return &Handler{s: s, opt: s.scheduleOpt}
}

func (h *Handler) getCoordinator() (*coordinator, error) {
	cluster := h.s.GetRaftCluster()
	if cluster == nil {
		return nil, errors.Trace(ErrNotBootstrapped)
	}
	return cluster.coordinator, nil
}

// GetSchedulers returns all names of schedulers.
func (h *Handler) GetSchedulers() ([]string, error) {
	c, err := h.getCoordinator()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return c.getSchedulers(), nil
}

// GetStores returns all stores in the cluster.
func (h *Handler) GetStores() ([]*core.StoreInfo, error) {
	cluster := h.s.GetRaftCluster()
	if cluster == nil {
		return nil, errors.Trace(ErrNotBootstrapped)
	}
	storeMetas := cluster.GetStores()
	stores := make([]*core.StoreInfo, 0, len(storeMetas))
	for _, s := range storeMetas {
		store, err := cluster.GetStore(s.GetId())
		if err != nil {
			return nil, errors.Trace(err)
		}
		stores = append(stores, store)
	}
	return stores, nil
}

// GetHotWriteRegions gets all hot write regions stats.
func (h *Handler) GetHotWriteRegions() *core.StoreHotRegionInfos {
	c, err := h.getCoordinator()
	if err != nil {
		return nil
	}
	return c.getHotWriteRegions()
}

// GetHotReadRegions gets all hot read regions stats.
func (h *Handler) GetHotReadRegions() *core.StoreHotRegionInfos {
	c, err := h.getCoordinator()
	if err != nil {
		return nil
	}
	return c.getHotReadRegions()
}

// GetHotBytesWriteStores gets all hot write stores stats.
func (h *Handler) GetHotBytesWriteStores() map[uint64]uint64 {
	return h.s.cluster.cachedCluster.getStoresBytesWriteStat()
}

// GetHotBytesReadStores gets all hot write stores stats.
func (h *Handler) GetHotBytesReadStores() map[uint64]uint64 {
	return h.s.cluster.cachedCluster.getStoresBytesReadStat()
}

// GetHotKeysWriteStores gets all hot write stores stats.
func (h *Handler) GetHotKeysWriteStores() map[uint64]uint64 {
	return h.s.cluster.cachedCluster.getStoresKeysWriteStat()
}

// GetHotKeysReadStores gets all hot write stores stats.
func (h *Handler) GetHotKeysReadStores() map[uint64]uint64 {
	return h.s.cluster.cachedCluster.getStoresKeysReadStat()
}

// AddScheduler adds a scheduler.
func (h *Handler) AddScheduler(name string, args ...string) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}
	s, err := schedule.CreateScheduler(name, c.limiter, args...)
	if err != nil {
		return errors.Trace(err)
	}
	log.Infof("create scheduler %s", s.GetName())
	if err = c.addScheduler(s, args...); err != nil {
		log.Errorf("can not add scheduler %v: %v", s.GetName(), err)
	} else if err = h.opt.persist(c.cluster.kv); err != nil {
		log.Errorf("can not persist scheduler config: %v", err)
	}
	return errors.Trace(err)
}

// RemoveScheduler removes a scheduler by name.
func (h *Handler) RemoveScheduler(name string) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}
	if err = c.removeScheduler(name); err != nil {
		log.Errorf("can not remove scheduler %v: %v", name, err)
	} else if err = h.opt.persist(c.cluster.kv); err != nil {
		log.Errorf("can not persist scheduler config: %v", err)
	}
	return errors.Trace(err)
}

// AddBalanceLeaderScheduler adds a balance-leader-scheduler.
func (h *Handler) AddBalanceLeaderScheduler() error {
	return h.AddScheduler("balance-leader")
}

// AddBalanceRegionScheduler adds a balance-region-scheduler.
func (h *Handler) AddBalanceRegionScheduler() error {
	return h.AddScheduler("balance-region")
}

// AddBalanceHotRegionScheduler adds a balance-hot-region-scheduler.
func (h *Handler) AddBalanceHotRegionScheduler() error {
	return h.AddScheduler("hot-region")
}

// AddLabelScheduler adds a label-scheduler.
func (h *Handler) AddLabelScheduler() error {
	return h.AddScheduler("label")
}

// AddScatterRangeScheduler adds a balance-range-leader-scheduler
func (h *Handler) AddScatterRangeScheduler(args ...string) error {
	return h.AddScheduler("scatter-range", args...)
}

// AddAdjacentRegionScheduler adds a balance-adjacent-region-scheduler.
func (h *Handler) AddAdjacentRegionScheduler(args ...string) error {
	return h.AddScheduler("adjacent-region", args...)
}

// AddGrantLeaderScheduler adds a grant-leader-scheduler.
func (h *Handler) AddGrantLeaderScheduler(storeID uint64) error {
	return h.AddScheduler("grant-leader", strconv.FormatUint(storeID, 10))
}

// AddEvictLeaderScheduler adds an evict-leader-scheduler.
func (h *Handler) AddEvictLeaderScheduler(storeID uint64) error {
	return h.AddScheduler("evict-leader", strconv.FormatUint(storeID, 10))
}

// AddShuffleLeaderScheduler adds a shuffle-leader-scheduler.
func (h *Handler) AddShuffleLeaderScheduler() error {
	return h.AddScheduler("shuffle-leader")
}

// AddShuffleRegionScheduler adds a shuffle-region-scheduler.
func (h *Handler) AddShuffleRegionScheduler() error {
	return h.AddScheduler("shuffle-region")
}

// AddRandomMergeScheduler adds a random-merge-scheduler.
func (h *Handler) AddRandomMergeScheduler() error {
	return h.AddScheduler("random-merge")
}

// GetOperator returns the region operator.
func (h *Handler) GetOperator(regionID uint64) (*schedule.Operator, error) {
	c, err := h.getCoordinator()
	if err != nil {
		return nil, errors.Trace(err)
	}

	op := c.getOperator(regionID)
	if op == nil {
		return nil, ErrOperatorNotFound
	}

	return op, nil
}

// RemoveOperator removes the region operator.
func (h *Handler) RemoveOperator(regionID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	op := c.getOperator(regionID)
	if op == nil {
		return ErrOperatorNotFound
	}

	c.removeOperator(op)
	return nil
}

// GetOperators returns the running operators.
func (h *Handler) GetOperators() ([]*schedule.Operator, error) {
	c, err := h.getCoordinator()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return c.getOperators(), nil
}

// GetAdminOperators returns the running admin operators.
func (h *Handler) GetAdminOperators() ([]*schedule.Operator, error) {
	return h.GetOperatorsOfKind(schedule.OpAdmin)
}

// GetLeaderOperators returns the running leader operators.
func (h *Handler) GetLeaderOperators() ([]*schedule.Operator, error) {
	return h.GetOperatorsOfKind(schedule.OpLeader)
}

// GetRegionOperators returns the running region operators.
func (h *Handler) GetRegionOperators() ([]*schedule.Operator, error) {
	return h.GetOperatorsOfKind(schedule.OpRegion)
}

// GetOperatorsOfKind returns the running operators of the kind.
func (h *Handler) GetOperatorsOfKind(mask schedule.OperatorKind) ([]*schedule.Operator, error) {
	ops, err := h.GetOperators()
	if err != nil {
		return nil, errors.Trace(err)
	}
	var results []*schedule.Operator
	for _, op := range ops {
		if op.Kind()&mask != 0 {
			results = append(results, op)
		}
	}
	return results, nil
}

// GetHistory returns finished operators' history since start.
func (h *Handler) GetHistory(start time.Time) ([]schedule.OperatorHistory, error) {
	c, err := h.getCoordinator()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return c.getHistory(start), nil
}

var errAddOperator = errors.New("failed to add operator, maybe already have one")

// AddTransferLeaderOperator adds an operator to transfer leader to the store.
func (h *Handler) AddTransferLeaderOperator(regionID uint64, storeID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}
	newLeader := region.GetStoreVoter(storeID)
	if newLeader == nil {
		return errors.Errorf("region has no voter in store %v", storeID)
	}

	step := schedule.TransferLeader{FromStore: region.Leader.GetStoreId(), ToStore: newLeader.GetStoreId()}
	op := schedule.NewOperator("adminTransferLeader", regionID, region.GetRegionEpoch(), schedule.OpAdmin|schedule.OpLeader, step)
	if ok := c.addOperator(op); !ok {
		return errors.Trace(errAddOperator)
	}
	return nil
}

// AddTransferRegionOperator adds an operator to transfer region to the stores.
func (h *Handler) AddTransferRegionOperator(regionID uint64, storeIDs map[uint64]struct{}) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}

	var steps []schedule.OperatorStep

	// Add missing peers.
	for id := range storeIDs {
		if c.cluster.GetStore(id) == nil {
			return core.ErrStoreNotFound(id)
		}
		if region.GetStorePeer(id) != nil {
			continue
		}
		peer, err := c.cluster.AllocPeer(id)
		if err != nil {
			return errors.Trace(err)
		}
		if c.cluster.IsRaftLearnerEnabled() {
			steps = append(steps,
				schedule.AddLearner{ToStore: id, PeerID: peer.Id},
				schedule.PromoteLearner{ToStore: id, PeerID: peer.Id},
			)
		} else {
			steps = append(steps, schedule.AddPeer{ToStore: id, PeerID: peer.Id})
		}
	}

	// Remove redundant peers.
	for _, peer := range region.GetPeers() {
		if _, ok := storeIDs[peer.GetStoreId()]; ok {
			continue
		}
		steps = append(steps, schedule.RemovePeer{FromStore: peer.GetStoreId()})
	}

	op := schedule.NewOperator("adminMoveRegion", regionID, region.GetRegionEpoch(), schedule.OpAdmin|schedule.OpRegion, steps...)
	if ok := c.addOperator(op); !ok {
		return errors.Trace(errAddOperator)
	}
	return nil
}

// AddTransferPeerOperator adds an operator to transfer peer.
func (h *Handler) AddTransferPeerOperator(regionID uint64, fromStoreID, toStoreID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}

	oldPeer := region.GetStorePeer(fromStoreID)
	if oldPeer == nil {
		return errors.Errorf("region has no peer in store %v", fromStoreID)
	}

	if c.cluster.GetStore(toStoreID) == nil {
		return core.ErrStoreNotFound(toStoreID)
	}
	newPeer, err := c.cluster.AllocPeer(toStoreID)
	if err != nil {
		return errors.Trace(err)
	}

	op := schedule.CreateMovePeerOperator("adminMovePeer", c.cluster, region, schedule.OpAdmin, fromStoreID, toStoreID, newPeer.GetId())
	if ok := c.addOperator(op); !ok {
		return errors.Trace(errAddOperator)
	}
	return nil
}

// AddAddPeerOperator adds an operator to add peer.
func (h *Handler) AddAddPeerOperator(regionID uint64, toStoreID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}

	if region.GetStorePeer(toStoreID) != nil {
		return errors.Errorf("region already has peer in store %v", toStoreID)
	}

	if c.cluster.GetStore(toStoreID) == nil {
		return core.ErrStoreNotFound(toStoreID)
	}
	newPeer, err := c.cluster.AllocPeer(toStoreID)
	if err != nil {
		return errors.Trace(err)
	}

	var steps []schedule.OperatorStep
	if c.cluster.IsRaftLearnerEnabled() {
		steps = []schedule.OperatorStep{
			schedule.AddLearner{ToStore: toStoreID, PeerID: newPeer.GetId()},
			schedule.PromoteLearner{ToStore: toStoreID, PeerID: newPeer.GetId()},
		}
	} else {
		steps = []schedule.OperatorStep{
			schedule.AddPeer{ToStore: toStoreID, PeerID: newPeer.GetId()},
		}
	}
	op := schedule.NewOperator("adminAddPeer", regionID, region.GetRegionEpoch(), schedule.OpAdmin|schedule.OpRegion, steps...)
	if ok := c.addOperator(op); !ok {
		return errors.Trace(errAddOperator)
	}
	return nil
}

// AddRemovePeerOperator adds an operator to remove peer.
func (h *Handler) AddRemovePeerOperator(regionID uint64, fromStoreID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}

	if region.GetStorePeer(fromStoreID) == nil {
		return errors.Errorf("region has no peer in store %v", fromStoreID)
	}

	op := schedule.CreateRemovePeerOperator("adminRemovePeer", c.cluster, schedule.OpAdmin, region, fromStoreID)
	if ok := c.addOperator(op); !ok {
		return errors.Trace(errAddOperator)
	}
	return nil
}

// AddMergeRegionOperator adds an operator to merge region.
func (h *Handler) AddMergeRegionOperator(regionID uint64, targetID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}

	target := c.cluster.GetRegion(targetID)
	if target == nil {
		return ErrRegionNotFound(targetID)
	}

	if len(region.DownPeers) > 0 || len(region.PendingPeers) > 0 || len(region.Learners) > 0 ||
		len(region.Region.GetPeers()) != c.cluster.GetMaxReplicas() {
		return ErrRegionAbnormalPeer(regionID)
	}

	if len(target.DownPeers) > 0 || len(target.PendingPeers) > 0 || len(target.Learners) > 0 ||
		len(target.Region.GetPeers()) != c.cluster.GetMaxReplicas() {
		return ErrRegionAbnormalPeer(targetID)
	}

	// for the case first region (start key is nil) with the last region (end key is nil) but not adjacent
	if (bytes.Compare(region.StartKey, target.EndKey) != 0 || len(region.StartKey) == 0) &&
		(bytes.Compare(region.EndKey, target.StartKey) != 0 || len(region.EndKey) == 0) {
		return ErrRegionNotAdjacent
	}

	op1, op2, err := schedule.CreateMergeRegionOperator("adminMergeRegion", c.cluster, region, target, schedule.OpAdmin)
	if err != nil {
		return errors.Trace(err)
	}
	if ok := c.addOperator(op1, op2); !ok {
		return errors.Trace(ErrAddOperator)
	}
	return nil
}

// AddSplitRegionOperator adds an operator to split a region.
func (h *Handler) AddSplitRegionOperator(regionID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}

	step := schedule.SplitRegion{StartKey: region.StartKey, EndKey: region.EndKey}
	op := schedule.NewOperator("adminSplitRegion", regionID, region.GetRegionEpoch(), schedule.OpAdmin, step)
	if ok := c.addOperator(op); !ok {
		return errors.Trace(errAddOperator)
	}
	return nil
}

// AddScatterRegionOperator adds an operator to scatter a region.
func (h *Handler) AddScatterRegionOperator(regionID uint64) error {
	c, err := h.getCoordinator()
	if err != nil {
		return errors.Trace(err)
	}

	region := c.cluster.GetRegion(regionID)
	if region == nil {
		return ErrRegionNotFound(regionID)
	}

	op := c.regionScatterer.Scatter(region)
	if op == nil {
		return nil
	}
	if ok := c.addOperator(op); !ok {
		return errors.Trace(errAddOperator)
	}
	return nil
}

// GetDownPeerRegions gets the region with down peer.
func (h *Handler) GetDownPeerRegions() ([]*core.RegionInfo, error) {
	c := h.s.GetRaftCluster()
	if c == nil {
		return nil, ErrNotBootstrapped
	}
	return c.cachedCluster.GetRegionStatsByType(downPeer), nil
}

// GetExtraPeerRegions gets the region exceeds the specified number of peers.
func (h *Handler) GetExtraPeerRegions() ([]*core.RegionInfo, error) {
	c := h.s.GetRaftCluster()
	if c == nil {
		return nil, ErrNotBootstrapped
	}
	return c.cachedCluster.GetRegionStatsByType(extraPeer), nil
}

// GetMissPeerRegions gets the region less than the specified number of peers.
func (h *Handler) GetMissPeerRegions() ([]*core.RegionInfo, error) {
	c := h.s.GetRaftCluster()
	if c == nil {
		return nil, ErrNotBootstrapped
	}
	return c.cachedCluster.GetRegionStatsByType(missPeer), nil
}

// GetPendingPeerRegions gets the region with pending peer.
func (h *Handler) GetPendingPeerRegions() ([]*core.RegionInfo, error) {
	c := h.s.GetRaftCluster()
	if c == nil {
		return nil, ErrNotBootstrapped
	}
	return c.cachedCluster.GetRegionStatsByType(pendingPeer), nil
}

// GetIncorrectNamespaceRegions gets the region with incorrect namespace peer.
func (h *Handler) GetIncorrectNamespaceRegions() ([]*core.RegionInfo, error) {
	c := h.s.GetRaftCluster()
	if c == nil {
		return nil, ErrNotBootstrapped
	}
	return c.cachedCluster.GetRegionStatsByType(incorrectNamespace), nil
}
