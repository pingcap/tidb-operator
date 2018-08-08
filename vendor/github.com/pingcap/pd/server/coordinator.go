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
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/eraftpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/pkg/logutil"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/namespace"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
)

const (
	runSchedulerCheckInterval = 3 * time.Second
	collectFactor             = 0.8
	historyKeepTime           = 5 * time.Minute
	maxScheduleRetries        = 10

	regionheartbeatSendChanCap = 1024
	hotRegionScheduleName      = "balance-hot-region-scheduler"

	patrolScanRegionLimit = 128 // It takes about 14 minutes to iterate 1 million regions.
)

var (
	errSchedulerExisted  = errors.New("scheduler existed")
	errSchedulerNotFound = errors.New("scheduler not found")
)

type coordinator struct {
	sync.RWMutex

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	cluster          *clusterInfo
	limiter          *schedule.Limiter
	replicaChecker   *schedule.ReplicaChecker
	regionScatterer  *schedule.RegionScatterer
	namespaceChecker *schedule.NamespaceChecker
	mergeChecker     *schedule.MergeChecker
	operators        map[uint64]*schedule.Operator
	schedulers       map[string]*scheduleController
	classifier       namespace.Classifier
	histories        *list.List
	hbStreams        *heartbeatStreams
}

func newCoordinator(cluster *clusterInfo, hbStreams *heartbeatStreams, classifier namespace.Classifier) *coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &coordinator{
		ctx:              ctx,
		cancel:           cancel,
		cluster:          cluster,
		limiter:          schedule.NewLimiter(),
		replicaChecker:   schedule.NewReplicaChecker(cluster, classifier),
		regionScatterer:  schedule.NewRegionScatterer(cluster, classifier),
		namespaceChecker: schedule.NewNamespaceChecker(cluster, classifier),
		mergeChecker:     schedule.NewMergeChecker(cluster, classifier),
		operators:        make(map[uint64]*schedule.Operator),
		schedulers:       make(map[string]*scheduleController),
		classifier:       classifier,
		histories:        list.New(),
		hbStreams:        hbStreams,
	}
}

func (c *coordinator) dispatch(region *core.RegionInfo) {
	// Check existed operator.
	if op := c.getOperator(region.GetId()); op != nil {
		timeout := op.IsTimeout()
		if step := op.Check(region); step != nil && !timeout {
			operatorCounter.WithLabelValues(op.Desc(), "check").Inc()
			c.sendScheduleCommand(region, step)
			return
		}
		if op.IsFinish() {
			log.Infof("[region %v] operator finish: %s", region.GetId(), op)
			operatorCounter.WithLabelValues(op.Desc(), "finish").Inc()
			operatorDuration.WithLabelValues(op.Desc()).Observe(op.ElapsedTime().Seconds())
			c.pushHistory(op)
			c.removeOperator(op)
		} else if timeout {
			log.Infof("[region %v] operator timeout: %s", region.GetId(), op)
			operatorCounter.WithLabelValues(op.Desc(), "timeout").Inc()
			c.removeOperator(op)
		}
	}
}

func (c *coordinator) patrolRegions() {
	defer logutil.LogPanic()

	defer c.wg.Done()
	timer := time.NewTimer(c.cluster.GetPatrolRegionInterval())
	defer timer.Stop()

	log.Info("coordinator: start patrol regions")
	start := time.Now()
	var key []byte
	for {
		select {
		case <-timer.C:
			timer.Reset(c.cluster.GetPatrolRegionInterval())
		case <-c.ctx.Done():
			return
		}

		regions := c.cluster.ScanRegions(key, patrolScanRegionLimit)
		if len(regions) == 0 {
			// reset scan key.
			key = nil
			continue
		}

		for _, region := range regions {
			// Skip the region if there is already a pending operator.
			if c.getOperator(region.GetId()) != nil {
				continue
			}

			key = region.GetEndKey()

			if c.checkRegion(region) {
				break
			}
		}
		// update label level isolation statistics.
		c.cluster.updateRegionsLabelLevelStats(regions)
		if len(key) == 0 {
			patrolCheckRegionsHistogram.Observe(time.Since(start).Seconds())
			start = time.Now()
		}
	}
}

func (c *coordinator) checkRegion(region *core.RegionInfo) bool {
	// If PD has restarted, it need to check learners added before and promote them.
	// Don't check isRaftLearnerEnabled cause it may be disable learner feature but still some learners to promote.
	for _, p := range region.GetLearners() {
		if region.GetPendingLearner(p.GetId()) != nil {
			continue
		}
		step := schedule.PromoteLearner{
			ToStore: p.GetStoreId(),
			PeerID:  p.GetId(),
		}
		op := schedule.NewOperator("promoteLearner", region.GetId(), region.GetRegionEpoch(), schedule.OpRegion, step)
		if c.addOperator(op) {
			return true
		}
	}

	if op := c.namespaceChecker.Check(region); op != nil {
		if c.addOperator(op) {
			return true
		}
	}
	if c.limiter.OperatorCount(schedule.OpReplica) < c.cluster.GetReplicaScheduleLimit() {
		if op := c.replicaChecker.Check(region); op != nil {
			if c.addOperator(op) {
				return true
			}
		}
	}
	if c.limiter.OperatorCount(schedule.OpMerge) < c.cluster.GetMergeScheduleLimit() {
		if op1, op2 := c.mergeChecker.Check(region); op1 != nil && op2 != nil {
			// make sure two operators can add successfully altogether
			if c.addOperator(op1, op2) {
				return true
			}
		}
	}
	return false
}

func (c *coordinator) run() {
	ticker := time.NewTicker(runSchedulerCheckInterval)
	defer ticker.Stop()
	log.Info("coordinator: Start collect cluster information")
	for {
		if c.shouldRun() {
			log.Info("coordinator: Cluster information is prepared")
			break
		}
		select {
		case <-ticker.C:
		case <-c.ctx.Done():
			log.Info("coordinator: Stopped coordinator")
			return
		}
	}
	log.Info("coordinator: Run scheduler")

	k := 0
	scheduleCfg := c.cluster.opt.load()
	for _, schedulerCfg := range scheduleCfg.Schedulers {
		if schedulerCfg.Disable {
			scheduleCfg.Schedulers[k] = schedulerCfg
			k++
			log.Info("skip create ", schedulerCfg.Type)
			continue
		}
		s, err := schedule.CreateScheduler(schedulerCfg.Type, c.limiter, schedulerCfg.Args...)
		if err != nil {
			log.Errorf("can not create scheduler %s: %v", schedulerCfg.Type, err)
		} else {
			log.Infof("create scheduler %s", s.GetName())
			if err = c.addScheduler(s, schedulerCfg.Args...); err != nil {
				log.Errorf("can not add scheduler %s: %v", s.GetName(), err)
			}
		}

		// only record valid scheduler config
		if err == nil {
			scheduleCfg.Schedulers[k] = schedulerCfg
			k++
		}
	}

	// remove invalid scheduler config and persist
	scheduleCfg.Schedulers = scheduleCfg.Schedulers[:k]
	if err := c.cluster.opt.persist(c.cluster.kv); err != nil {
		log.Errorf("can't persist schedule config: %v", err)
	}

	c.wg.Add(1)
	go c.patrolRegions()
}

func (c *coordinator) stop() {
	c.cancel()
	c.wg.Wait()
}

// Hack to retrive info from scheduler.
// TODO: remove it.
type hasHotStatus interface {
	GetHotReadStatus() *core.StoreHotRegionInfos
	GetHotWriteStatus() *core.StoreHotRegionInfos
}

func (c *coordinator) getHotWriteRegions() *core.StoreHotRegionInfos {
	c.RLock()
	defer c.RUnlock()
	s, ok := c.schedulers[hotRegionScheduleName]
	if !ok {
		return nil
	}
	if h, ok := s.Scheduler.(hasHotStatus); ok {
		return h.GetHotWriteStatus()
	}
	return nil
}

func (c *coordinator) getHotReadRegions() *core.StoreHotRegionInfos {
	c.RLock()
	defer c.RUnlock()
	s, ok := c.schedulers[hotRegionScheduleName]
	if !ok {
		return nil
	}
	if h, ok := s.Scheduler.(hasHotStatus); ok {
		return h.GetHotReadStatus()
	}
	return nil
}

func (c *coordinator) getSchedulers() []string {
	c.RLock()
	defer c.RUnlock()

	names := make([]string, 0, len(c.schedulers))
	for name := range c.schedulers {
		names = append(names, name)
	}
	return names
}

func (c *coordinator) collectSchedulerMetrics() {
	c.RLock()
	defer c.RUnlock()
	for _, s := range c.schedulers {
		var allowScheduler float64
		if s.AllowSchedule() {
			allowScheduler = 1
		}
		schedulerStatusGauge.WithLabelValues(s.GetName(), "allow").Set(allowScheduler)
	}
}

func (c *coordinator) collectHotSpotMetrics() {
	c.RLock()
	defer c.RUnlock()
	// collect hot write region metrics
	s, ok := c.schedulers[hotRegionScheduleName]
	if !ok {
		return
	}
	stores := c.cluster.GetStores()
	status := s.Scheduler.(hasHotStatus).GetHotWriteStatus()
	for _, s := range stores {
		store := fmt.Sprintf("store_%d", s.GetId())
		stat, ok := status.AsPeer[s.GetId()]
		if ok {
			totalWriteBytes := float64(stat.TotalFlowBytes)
			hotWriteRegionCount := float64(stat.RegionsCount)

			hotSpotStatusGauge.WithLabelValues(store, "total_written_bytes_as_peer").Set(totalWriteBytes)
			hotSpotStatusGauge.WithLabelValues(store, "hot_write_region_as_peer").Set(hotWriteRegionCount)
		} else {
			hotSpotStatusGauge.WithLabelValues(store, "total_written_bytes_as_peer").Set(0)
			hotSpotStatusGauge.WithLabelValues(store, "hot_write_region_as_peer").Set(0)
		}

		stat, ok = status.AsLeader[s.GetId()]
		if ok {
			totalWriteBytes := float64(stat.TotalFlowBytes)
			hotWriteRegionCount := float64(stat.RegionsCount)

			hotSpotStatusGauge.WithLabelValues(store, "total_written_bytes_as_leader").Set(totalWriteBytes)
			hotSpotStatusGauge.WithLabelValues(store, "hot_write_region_as_leader").Set(hotWriteRegionCount)
		} else {
			hotSpotStatusGauge.WithLabelValues(store, "total_written_bytes_as_leader").Set(0)
			hotSpotStatusGauge.WithLabelValues(store, "hot_write_region_as_leader").Set(0)
		}
	}

	// collect hot read region metrics
	status = s.Scheduler.(hasHotStatus).GetHotReadStatus()
	for _, s := range stores {
		store := fmt.Sprintf("store_%d", s.GetId())
		stat, ok := status.AsLeader[s.GetId()]
		if ok {
			totalReadBytes := float64(stat.TotalFlowBytes)
			hotReadRegionCount := float64(stat.RegionsCount)

			hotSpotStatusGauge.WithLabelValues(store, "total_read_bytes_as_leader").Set(totalReadBytes)
			hotSpotStatusGauge.WithLabelValues(store, "hot_read_region_as_leader").Set(hotReadRegionCount)
		} else {
			hotSpotStatusGauge.WithLabelValues(store, "total_read_bytes_as_leader").Set(0)
			hotSpotStatusGauge.WithLabelValues(store, "hot_read_region_as_leader").Set(0)
		}
	}

}

func (c *coordinator) shouldRun() bool {
	return c.cluster.isPrepared()
}

func (c *coordinator) addScheduler(scheduler schedule.Scheduler, args ...string) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.schedulers[scheduler.GetName()]; ok {
		return errSchedulerExisted
	}

	s := newScheduleController(c, scheduler)
	if err := s.Prepare(c.cluster); err != nil {
		return errors.Trace(err)
	}

	c.wg.Add(1)
	go c.runScheduler(s)
	c.schedulers[s.GetName()] = s
	c.cluster.opt.AddSchedulerCfg(s.GetType(), args)

	return nil
}

func (c *coordinator) removeScheduler(name string) error {
	c.Lock()
	defer c.Unlock()

	s, ok := c.schedulers[name]
	if !ok {
		return errSchedulerNotFound
	}

	s.Stop()
	delete(c.schedulers, name)

	if err := c.cluster.opt.RemoveSchedulerCfg(name); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *coordinator) runScheduler(s *scheduleController) {
	defer logutil.LogPanic()
	defer c.wg.Done()
	defer s.Cleanup(c.cluster)

	timer := time.NewTimer(s.GetInterval())
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			timer.Reset(s.GetInterval())
			if !s.AllowSchedule() {
				continue
			}
			opInfluence := schedule.NewOpInfluence(c.getOperators(), c.cluster)
			if op := s.Schedule(c.cluster, opInfluence); op != nil {
				c.addOperator(op...)
			}

		case <-s.Ctx().Done():
			log.Infof("%v stopped: %v", s.GetName(), s.Ctx().Err())
			return
		}
	}
}

func (c *coordinator) addOperatorLocked(op *schedule.Operator) bool {
	regionID := op.RegionID()

	log.Infof("[region %v] add operator: %s", regionID, op)

	// If there is an old operator, replace it. The priority should be checked
	// already.
	if old, ok := c.operators[regionID]; ok {
		log.Infof("[region %v] replace old operator: %s", regionID, old)
		operatorCounter.WithLabelValues(old.Desc(), "replaced").Inc()
		c.removeOperatorLocked(old)
	}

	c.operators[regionID] = op
	c.limiter.UpdateCounts(c.operators)

	if region := c.cluster.GetRegion(op.RegionID()); region != nil {
		if step := op.Check(region); step != nil {
			c.sendScheduleCommand(region, step)
		}
	}

	operatorCounter.WithLabelValues(op.Desc(), "create").Inc()
	return true
}

func (c *coordinator) addOperator(ops ...*schedule.Operator) bool {
	c.Lock()
	defer c.Unlock()

	for _, op := range ops {
		if !c.checkAddOperator(op) {
			operatorCounter.WithLabelValues(op.Desc(), "canceled").Inc()
			return false
		}
	}
	for _, op := range ops {
		c.addOperatorLocked(op)
	}

	return true
}

func (c *coordinator) checkAddOperator(op *schedule.Operator) bool {
	region := c.cluster.GetRegion(op.RegionID())
	if region == nil {
		log.Debugf("[region %v] region not found, cancel add operator", op.RegionID())
		return false
	}
	if region.GetRegionEpoch().GetVersion() != op.RegionEpoch().GetVersion() || region.GetRegionEpoch().GetConfVer() != op.RegionEpoch().GetConfVer() {
		log.Debugf("[region %v] region epoch not match, %v vs %v, cancel add operator", op.RegionID(), region.GetRegionEpoch(), op.RegionEpoch())
		return false
	}
	if old := c.operators[op.RegionID()]; old != nil && !isHigherPriorityOperator(op, old) {
		log.Debugf("[region %v] already have operator %s, cancel add operator", op.RegionID(), old)
		return false
	}
	return true
}

func isHigherPriorityOperator(new, old *schedule.Operator) bool {
	return new.GetPriorityLevel() < old.GetPriorityLevel()
}

func (c *coordinator) pushHistory(op *schedule.Operator) {
	c.Lock()
	defer c.Unlock()
	for _, h := range op.History() {
		c.histories.PushFront(h)
	}
}

func (c *coordinator) pruneHistory() {
	c.Lock()
	defer c.Unlock()
	p := c.histories.Back()
	for p != nil && time.Since(p.Value.(schedule.OperatorHistory).FinishTime) > historyKeepTime {
		prev := p.Prev()
		c.histories.Remove(p)
		p = prev
	}
}

func (c *coordinator) removeOperator(op *schedule.Operator) {
	c.Lock()
	defer c.Unlock()
	c.removeOperatorLocked(op)
}

func (c *coordinator) removeOperatorLocked(op *schedule.Operator) {
	regionID := op.RegionID()
	delete(c.operators, regionID)
	c.limiter.UpdateCounts(c.operators)
	operatorCounter.WithLabelValues(op.Desc(), "remove").Inc()
}

func (c *coordinator) getOperator(regionID uint64) *schedule.Operator {
	c.RLock()
	defer c.RUnlock()
	return c.operators[regionID]
}

func (c *coordinator) getOperators() []*schedule.Operator {
	c.RLock()
	defer c.RUnlock()

	operators := make([]*schedule.Operator, 0, len(c.operators))
	for _, op := range c.operators {
		operators = append(operators, op)
	}

	return operators
}

func (c *coordinator) getHistory(start time.Time) []schedule.OperatorHistory {
	c.RLock()
	defer c.RUnlock()
	histories := make([]schedule.OperatorHistory, 0, c.histories.Len())
	for p := c.histories.Front(); p != nil; p = p.Next() {
		history := p.Value.(schedule.OperatorHistory)
		if history.FinishTime.Before(start) {
			break
		}
		histories = append(histories, history)
	}
	return histories
}

func (c *coordinator) sendScheduleCommand(region *core.RegionInfo, step schedule.OperatorStep) {
	log.Infof("[region %v] send schedule command: %s", region.GetId(), step)
	switch s := step.(type) {
	case schedule.TransferLeader:
		cmd := &pdpb.RegionHeartbeatResponse{
			TransferLeader: &pdpb.TransferLeader{
				Peer: region.GetStorePeer(s.ToStore),
			},
		}
		c.hbStreams.sendMsg(region, cmd)
	case schedule.AddPeer:
		if region.GetStorePeer(s.ToStore) != nil {
			// The newly added peer is pending.
			return
		}
		cmd := &pdpb.RegionHeartbeatResponse{
			ChangePeer: &pdpb.ChangePeer{
				ChangeType: eraftpb.ConfChangeType_AddNode,
				Peer: &metapb.Peer{
					Id:      s.PeerID,
					StoreId: s.ToStore,
				},
			},
		}
		c.hbStreams.sendMsg(region, cmd)
	case schedule.AddLearner:
		if region.GetStorePeer(s.ToStore) != nil {
			// The newly added peer is pending.
			return
		}
		cmd := &pdpb.RegionHeartbeatResponse{
			ChangePeer: &pdpb.ChangePeer{
				ChangeType: eraftpb.ConfChangeType_AddLearnerNode,
				Peer: &metapb.Peer{
					Id:        s.PeerID,
					StoreId:   s.ToStore,
					IsLearner: true,
				},
			},
		}
		c.hbStreams.sendMsg(region, cmd)
	case schedule.PromoteLearner:
		cmd := &pdpb.RegionHeartbeatResponse{
			ChangePeer: &pdpb.ChangePeer{
				// reuse AddNode type
				ChangeType: eraftpb.ConfChangeType_AddNode,
				Peer: &metapb.Peer{
					Id:      s.PeerID,
					StoreId: s.ToStore,
				},
			},
		}
		c.hbStreams.sendMsg(region, cmd)
	case schedule.RemovePeer:
		cmd := &pdpb.RegionHeartbeatResponse{
			ChangePeer: &pdpb.ChangePeer{
				ChangeType: eraftpb.ConfChangeType_RemoveNode,
				Peer:       region.GetStorePeer(s.FromStore),
			},
		}
		c.hbStreams.sendMsg(region, cmd)
	case schedule.MergeRegion:
		if s.IsPassive {
			return
		}
		cmd := &pdpb.RegionHeartbeatResponse{
			Merge: &pdpb.Merge{
				Target: s.ToRegion,
			},
		}
		c.hbStreams.sendMsg(region, cmd)
	case schedule.SplitRegion:
		cmd := &pdpb.RegionHeartbeatResponse{
			SplitRegion: &pdpb.SplitRegion{},
		}
		c.hbStreams.sendMsg(region, cmd)
	default:
		log.Errorf("unknown operatorStep: %v", step)
	}
}

type scheduleController struct {
	schedule.Scheduler
	cluster      *clusterInfo
	limiter      *schedule.Limiter
	classifier   namespace.Classifier
	nextInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
}

func newScheduleController(c *coordinator, s schedule.Scheduler) *scheduleController {
	ctx, cancel := context.WithCancel(c.ctx)
	return &scheduleController{
		Scheduler:    s,
		cluster:      c.cluster,
		limiter:      c.limiter,
		nextInterval: s.GetMinInterval(),
		classifier:   c.classifier,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (s *scheduleController) Ctx() context.Context {
	return s.ctx
}

func (s *scheduleController) Stop() {
	s.cancel()
}

func (s *scheduleController) Schedule(cluster schedule.Cluster, opInfluence schedule.OpInfluence) []*schedule.Operator {
	for i := 0; i < maxScheduleRetries; i++ {
		// If we have schedule, reset interval to the minimal interval.
		if op := scheduleByNamespace(cluster, s.classifier, s.Scheduler, opInfluence); op != nil {
			s.nextInterval = s.Scheduler.GetMinInterval()
			return op
		}
	}
	s.nextInterval = s.Scheduler.GetNextInterval(s.nextInterval)
	return nil
}

func (s *scheduleController) GetInterval() time.Duration {
	return s.nextInterval
}

func (s *scheduleController) AllowSchedule() bool {
	return s.Scheduler.IsScheduleAllowed(s.cluster)
}
