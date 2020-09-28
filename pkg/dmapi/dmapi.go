// Copyright 2020 PingCAP, Inc.
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

package dmapi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
)

const (
	DefaultTimeout = 5 * time.Second
)

// MasterClient provides master server's api
type MasterClient interface {
	// GetMasters returns all master members from cluster
	GetMasters() ([]*MastersInfo, error)
	GetWorkers() ([]*WorkersInfo, error)
	GetLeader() (MembersLeader, error)
	EvictLeader() error
	DeleteMaster(name string) error
	DeleteWorker(name string) error
}

var (
	membersPrefix = "apis/v1alpha1/members"
	leaderPrefix  = "apis/v1alpha1/leader"
)

type RespHeader struct {
	Result bool   `json:"result,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

type MastersInfo struct {
	Name       string   `json:"name,omitempty"`
	MemberID   string   `json:"memberID,omitempty"`
	Alive      bool     `json:"alive,omitempty"`
	PeerURLs   []string `json:"peerURLs,omitempty"`
	ClientURLs []string `json:"clientURLs,omitempty"`
}

type WorkersInfo struct {
	Name   string `json:"name,omitempty"`
	Addr   string `json:"addr,omitempty"`
	Stage  string `json:"stage,omitempty"`
	Source string `json:"source,omitempty"`
}

type MembersMaster struct {
	Msg     string         `json:"msg,omitempty"`
	Masters []*MastersInfo `json:"masters,omitempty"`
}

type MembersWorker struct {
	Msg     string         `json:"msg,omitempty"`
	Workers []*WorkersInfo `json:"workers,omitempty"`
}

type MembersLeader struct {
	Msg  string `json:"msg,omitempty"`
	Name string `json:"name,omitempty"`
	Addr string `json:"addr,omitempty"`
}

type ListMemberMaster struct {
	MembersMaster `json:"master,omitempty"`
}

type ListMemberWorker struct {
	MembersWorker `json:"worker,omitempty"`
}

type ListMemberLeader struct {
	MembersLeader `json:"leader,omitempty"`
}

type MastersResp struct {
	RespHeader     `json:",inline"`
	ListMemberResp []*ListMemberMaster `json:"members,omitempty"`
}

type WorkerResp struct {
	RespHeader     `json:",inline"`
	ListMemberResp []*ListMemberWorker `json:"members,omitempty"`
}

type LeaderResp struct {
	RespHeader     `json:",inline"`
	ListMemberResp []*ListMemberLeader `json:"members,omitempty"`
}

// masterClient is default implementation of MasterClient
type masterClient struct {
	url        string
	httpClient *http.Client
}

func (mc *masterClient) GetMasters() ([]*MastersInfo, error) {
	query := "?master=true"
	apiURL := fmt.Sprintf("%s/%s%s", mc.url, membersPrefix, query)
	body, err := httputil.GetBodyOK(mc.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	listMemberResp := &MastersResp{}
	err = json.Unmarshal(body, listMemberResp)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal list masters resp: %s, err: %s", body, err)
	}
	if !listMemberResp.Result {
		return nil, fmt.Errorf("unable to list masters info, err: %s", listMemberResp.Msg)
	}
	if len(listMemberResp.ListMemberResp) != 1 {
		return nil, fmt.Errorf("invalid list masters resp: %s", body)
	}

	return listMemberResp.ListMemberResp[0].Masters, nil
}

func (mc *masterClient) GetWorkers() ([]*WorkersInfo, error) {
	query := "?worker=true"
	apiURL := fmt.Sprintf("%s/%s%s", mc.url, membersPrefix, query)
	body, err := httputil.GetBodyOK(mc.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	listMemberResp := &WorkerResp{}
	err = json.Unmarshal(body, listMemberResp)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal list workers resp: %s, err: %s", body, err)
	}
	if !listMemberResp.Result {
		return nil, fmt.Errorf("unable to list workers info, err: %s", listMemberResp.Msg)
	}
	if len(listMemberResp.ListMemberResp) != 1 {
		return nil, fmt.Errorf("invalid list workers resp: %s", body)
	}

	return listMemberResp.ListMemberResp[0].Workers, nil
}

func (mc *masterClient) GetLeader() (MembersLeader, error) {
	query := "?leader=true"
	apiURL := fmt.Sprintf("%s/%s%s", mc.url, membersPrefix, query)
	body, err := httputil.GetBodyOK(mc.httpClient, apiURL)
	if err != nil {
		return MembersLeader{}, err
	}
	listMemberResp := &LeaderResp{}
	err = json.Unmarshal(body, listMemberResp)
	if err != nil {
		return MembersLeader{}, fmt.Errorf("unable to unmarshal list leader resp: %s, err: %s", body, err)
	}
	if !listMemberResp.Result {
		return MembersLeader{}, fmt.Errorf("unable to get leader info, err: %s", listMemberResp.Msg)
	}
	if len(listMemberResp.ListMemberResp) != 1 {
		return MembersLeader{}, fmt.Errorf("invalid list leader resp: %s", body)
	}

	return listMemberResp.ListMemberResp[0].MembersLeader, nil
}

func (mc *masterClient) EvictLeader() error {
	query := "/1"
	apiURL := fmt.Sprintf("%s/%s%s", mc.url, leaderPrefix, query)
	body, err := httputil.PutBodyOK(mc.httpClient, apiURL)
	if err != nil {
		return err
	}
	evictLeaderResp := &RespHeader{}
	err = json.Unmarshal(body, evictLeaderResp)
	if err != nil {
		return fmt.Errorf("unable to unmarshal evict leader resp: %s, err: %s", body, err)
	}
	if !evictLeaderResp.Result {
		return fmt.Errorf("unable to evict leader, err: %s", evictLeaderResp.Msg)
	}

	return nil
}

func (mc *masterClient) deleteMember(query string) error {
	apiURL := fmt.Sprintf("%s/%s%s", mc.url, membersPrefix, query)
	body, err := httputil.DeleteBodyOK(mc.httpClient, apiURL)
	if err != nil {
		return err
	}
	deleteMemberResp := &RespHeader{}
	err = json.Unmarshal(body, deleteMemberResp)
	if err != nil {
		return fmt.Errorf("unable to unmarshal delete member resp: %s, query: %s, err: %s", body, query, err)
	}
	if !deleteMemberResp.Result {
		return fmt.Errorf("unable to delete member, query: %s, err: %s", query, deleteMemberResp.Msg)
	}

	return nil
}

func (mc *masterClient) DeleteMaster(name string) error {
	query := "/master/" + name
	return mc.deleteMember(query)
}

func (mc *masterClient) DeleteWorker(name string) error {
	query := "/worker/" + name
	return mc.deleteMember(query)
}

// NewMasterClient returns a new MasterClient
func NewMasterClient(url string, timeout time.Duration, tlsConfig *tls.Config, disableKeepalive bool) MasterClient {
	return &masterClient{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: disableKeepalive},
		},
	}
}
