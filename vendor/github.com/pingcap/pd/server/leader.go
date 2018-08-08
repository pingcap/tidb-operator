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
	"context"
	"math/rand"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/juju/errors"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/pkg/etcdutil"
	"github.com/pingcap/pd/pkg/logutil"
	log "github.com/sirupsen/logrus"
)

var (
	errNoLeader = errors.New("no leader")
)

// IsLeader returns whether server is leader or not.
func (s *Server) IsLeader() bool {
	return atomic.LoadInt64(&s.isLeader) == 1
}

func (s *Server) enableLeader(b bool) {
	value := int64(0)
	if b {
		value = 1
	}

	atomic.StoreInt64(&s.isLeader, value)
}

func (s *Server) getLeaderPath() string {
	return path.Join(s.rootPath, "leader")
}

func (s *Server) startLeaderLoop() {
	s.leaderLoopCtx, s.leaderLoopCancel = context.WithCancel(context.Background())
	s.leaderLoopWg.Add(2)
	go s.leaderLoop()
	go s.etcdLeaderLoop()
}

func (s *Server) stopLeaderLoop() {
	s.leaderLoopCancel()
	s.leaderLoopWg.Wait()
}

func (s *Server) leaderLoop() {
	defer logutil.LogPanic()
	defer s.leaderLoopWg.Done()

	for {
		if s.isClosed() {
			log.Infof("server is closed, return leader loop")
			return
		}

		leader, err := getLeader(s.client, s.getLeaderPath())
		if err != nil {
			log.Errorf("get leader err %v", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if leader != nil {
			if s.isSameLeader(leader) {
				// oh, we are already leader, we may meet something wrong
				// in previous campaignLeader. we can delete and campaign again.
				log.Warnf("leader is still %s, delete and campaign again", leader)
				if err = s.deleteLeaderKey(); err != nil {
					log.Errorf("delete leader key err %s", err)
					time.Sleep(200 * time.Millisecond)
					continue
				}
			} else {
				log.Infof("leader is %s, watch it", leader)
				s.watchLeader()
				log.Info("leader changed, try to campaign leader")
			}
		}

		etcdLeader := s.etcd.Server.Lead()
		if etcdLeader != s.ID() {
			log.Infof("%v is not etcd leader, skip campaign leader and check later", s.Name())
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if err = s.campaignLeader(); err != nil {
			log.Errorf("campaign leader err %s", errors.ErrorStack(err))
		}
	}
}

func (s *Server) etcdLeaderLoop() {
	defer logutil.LogPanic()
	defer s.leaderLoopWg.Done()

	ctx, cancel := context.WithCancel(s.leaderLoopCtx)
	defer cancel()
	for {
		select {
		case <-time.After(s.cfg.leaderPriorityCheckInterval.Duration):
			etcdLeader := s.etcd.Server.Lead()
			if etcdLeader == s.ID() {
				break
			}
			myPriority, err := s.GetMemberLeaderPriority(s.ID())
			if err != nil {
				log.Errorf("failed to load leader priority: %v", err)
				break
			}
			leaderPriority, err := s.GetMemberLeaderPriority(etcdLeader)
			if err != nil {
				log.Errorf("failed to load leader priority: %v", err)
				break
			}
			if myPriority > leaderPriority {
				err := s.etcd.Server.MoveLeader(ctx, etcdLeader, s.ID())
				if err != nil {
					log.Errorf("failed to transfer etcd leader: %v", err)
				} else {
					log.Infof("etcd leader moved from %v to %v", etcdLeader, s.ID())
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func getLeaderAddr(leader *pdpb.Member) string {
	return strings.Join(leader.GetClientUrls(), ",")
}

// getLeader gets server leader from etcd.
func getLeader(c *clientv3.Client, leaderPath string) (*pdpb.Member, error) {
	leader := &pdpb.Member{}
	ok, err := getProtoMsg(c, leaderPath, leader)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !ok {
		return nil, nil
	}

	return leader, nil
}

// GetLeader gets pd cluster leader.
func (s *Server) GetLeader() (*pdpb.Member, error) {
	if s.isClosed() {
		return nil, errors.New("server is closed")
	}
	leader, err := getLeader(s.client, s.getLeaderPath())
	if err != nil {
		return nil, errors.Trace(err)
	}
	if leader == nil {
		return nil, errors.Trace(errNoLeader)
	}
	return leader, nil
}

func (s *Server) isSameLeader(leader *pdpb.Member) bool {
	return leader.GetMemberId() == s.ID()
}

func (s *Server) marshalLeader() string {
	leader := &pdpb.Member{
		Name:       s.Name(),
		MemberId:   s.ID(),
		ClientUrls: strings.Split(s.cfg.AdvertiseClientUrls, ","),
		PeerUrls:   strings.Split(s.cfg.AdvertisePeerUrls, ","),
	}

	data, err := leader.Marshal()
	if err != nil {
		// can't fail, so panic here.
		log.Fatalf("marshal leader %s err %v", leader, err)
	}

	return string(data)
}

func (s *Server) campaignLeader() error {
	log.Debugf("begin to campaign leader %s", s.Name())

	lessor := clientv3.NewLease(s.client)
	defer lessor.Close()

	start := time.Now()
	ctx, cancel := context.WithTimeout(s.client.Ctx(), requestTimeout)
	leaseResp, err := lessor.Grant(ctx, s.cfg.LeaderLease)
	cancel()

	if cost := time.Since(start); cost > slowRequestTime {
		log.Warnf("lessor grants too slow, cost %s", cost)
	}

	if err != nil {
		return errors.Trace(err)
	}

	leaderKey := s.getLeaderPath()
	// The leader key must not exist, so the CreateRevision is 0.
	resp, err := s.txn().
		If(clientv3.Compare(clientv3.CreateRevision(leaderKey), "=", 0)).
		Then(clientv3.OpPut(leaderKey, s.leaderValue, clientv3.WithLease(clientv3.LeaseID(leaseResp.ID)))).
		Commit()
	if err != nil {
		return errors.Trace(err)
	}
	if !resp.Succeeded {
		return errors.New("campaign leader failed, other server may campaign ok")
	}

	// Make the leader keepalived.
	ctx, cancel = context.WithCancel(s.leaderLoopCtx)
	defer cancel()

	ch, err := lessor.KeepAlive(ctx, clientv3.LeaseID(leaseResp.ID))
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("campaign leader ok %s", s.Name())

	err = s.scheduleOpt.reload(s.kv)
	if err != nil {
		return errors.Trace(err)
	}
	// Try to create raft cluster.
	err = s.createRaftCluster()
	if err != nil {
		return errors.Trace(err)
	}
	defer s.stopRaftCluster()

	log.Debug("sync timestamp for tso")
	if err = s.syncTimestamp(); err != nil {
		return errors.Trace(err)
	}
	defer s.ts.Store(&atomicObject{
		physical: zeroTime,
	})

	s.enableLeader(true)
	defer s.enableLeader(false)

	log.Infof("PD cluster leader %s is ready to serve", s.Name())

	tsTicker := time.NewTicker(updateTimestampStep)
	defer tsTicker.Stop()

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				log.Info("keep alive channel is closed")
				return nil
			}
		case <-tsTicker.C:
			if err = s.updateTimestamp(); err != nil {
				return errors.Trace(err)
			}
			etcdLeader := s.etcd.Server.Lead()
			if etcdLeader != s.ID() {
				log.Infof("etcd leader changed, %s resigns leadership", s.Name())
				return nil
			}
		case <-ctx.Done():
			return errors.New("server closed")
		}
	}
}

func (s *Server) watchLeader() {
	watcher := clientv3.NewWatcher(s.client)
	defer watcher.Close()

	ctx, cancel := context.WithCancel(s.leaderLoopCtx)
	defer cancel()

	for {
		rch := watcher.Watch(ctx, s.getLeaderPath())
		for wresp := range rch {
			if wresp.Canceled {
				return
			}

			for _, ev := range wresp.Events {
				if ev.Type == mvccpb.DELETE {
					log.Info("leader is deleted")
					return
				}
			}
		}

		select {
		case <-ctx.Done():
			// server closed, return
			return
		default:
		}
	}
}

// ResignLeader resigns current PD's leadership. If nextLeader is empty, all
// other pd-servers can campaign.
func (s *Server) ResignLeader(nextLeader string) error {
	log.Infof("%s tries to resign leader with next leader directive: %v", s.Name(), nextLeader)
	// Determine next leaders.
	var leaderIDs []uint64
	res, err := etcdutil.ListEtcdMembers(s.client)
	if err != nil {
		return errors.Trace(err)
	}
	for _, member := range res.Members {
		if (nextLeader == "" && member.ID != s.id) || (nextLeader != "" && member.Name == nextLeader) {
			leaderIDs = append(leaderIDs, member.GetID())
		}
	}
	if len(leaderIDs) == 0 {
		return errors.New("no valid pd to transfer leader")
	}
	nextLeaderID := leaderIDs[rand.Intn(len(leaderIDs))]
	log.Infof("%s ready to resign leader, next leader: %v", s.Name(), nextLeaderID)
	err = s.etcd.Server.MoveLeader(s.leaderLoopCtx, s.ID(), nextLeaderID)
	return errors.Trace(err)
}

func (s *Server) deleteLeaderKey() error {
	// delete leader itself and let others start a new election again.
	leaderKey := s.getLeaderPath()
	resp, err := s.leaderTxn().Then(clientv3.OpDelete(leaderKey)).Commit()
	if err != nil {
		return errors.Trace(err)
	}
	if !resp.Succeeded {
		return errors.New("resign leader failed, we are not leader already")
	}

	return nil
}

func (s *Server) leaderCmp() clientv3.Cmp {
	return clientv3.Compare(clientv3.Value(s.getLeaderPath()), "=", s.leaderValue)
}
