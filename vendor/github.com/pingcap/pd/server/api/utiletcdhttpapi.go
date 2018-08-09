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

package api

import (
	"fmt"
	"time"

	"github.com/juju/errors"
)

const (
	//etcd peer detail API
	//return struct https://github.com/coreos/etcd/blob/master/etcdserver/stats/server.go
	etcdPeerStatsAPI = "/v2/stats/self"
)

// PeerStats is the etcd peers' stats.
type PeerStats struct {
	Name       string    `json:"name"`
	ID         string    `json:"id"`
	State      string    `json:"state"`
	StartTime  time.Time `json:"startTime"`
	LeaderInfo struct {
		Leader    string    `json:"leader"`
		Uptime    string    `json:"uptime"`
		StartTime time.Time `json:"startTime"`
	} `json:"leaderInfo"`
	RecvAppendRequestCnt int `json:"recvAppendRequestCnt"`
	SendAppendRequestCnt int `json:"sendAppendRequestCnt"`
}

func getEtcdPeerStats(etcdClientURL string) (*PeerStats, error) {
	ps := &PeerStats{}
	resp, err := doGet(fmt.Sprintf("%s%s", etcdClientURL, etcdPeerStatsAPI))
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()
	if err := readJSON(resp.Body, ps); err != nil {
		return nil, errors.Trace(err)
	}
	return ps, nil
}
