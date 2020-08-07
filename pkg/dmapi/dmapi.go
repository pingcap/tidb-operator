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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	dmpb "github.com/pingcap/dm/dm/pb"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
)

const (
	DefaultTimeout = 5 * time.Second
)

// MasterClient provides master server's api
type MasterClient interface {
	// GetMasters returns all master members from cluster
	GetMembers(string) (*MembersInfo, error)
	// GetMasters returns all master members from cluster
	GetMasters() (*MembersInfo, error)
}

var (
	membersPrefix = "apis/v1alpha1/members"
)

// MembersInfo is master members info returned from dm-master RESTful interface
//type Members map[string][]*pdpb.Member
type MembersInfo struct {
	Masters []*dmpb.MasterInfo
	Workers []*dmpb.WorkerInfo
	Leader  *dmpb.ListLeaderMember
}

// masterClient is default implementation of MasterClient
type masterClient struct {
	url        string
	httpClient *http.Client
}

func (mc *masterClient) GetMembers(apiURL string) (*MembersInfo, error) {
	body, err := httputil.GetBodyOK(mc.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	listMemberResp := &dmpb.ListMemberResponse{}
	err = jsonpb.Unmarshal(strings.NewReader(string(body)), listMemberResp)
	if err != nil {
		return nil, err
	}
	if !listMemberResp.Result {
		return nil, fmt.Errorf("unable to list members info from dm-master, err: %s", listMemberResp.Msg)
	}
	membersInfo := &MembersInfo{
		Masters: make([]*dmpb.MasterInfo, 0),
		Workers: make([]*dmpb.WorkerInfo, 0),
	}
	for _, member := range listMemberResp.GetMembers() {
		if leader := member.GetLeader(); leader != nil {
			membersInfo.Leader = leader
		} else if masters := member.GetMaster(); masters != nil {
			membersInfo.Masters = append(membersInfo.Masters, masters.Masters...)
		} else if workers := member.GetWorker(); workers != nil {
			membersInfo.Workers = append(membersInfo.Workers, workers.Workers...)
		}
	}
	return membersInfo, nil
}

func (mc *masterClient) GetMasters() (*MembersInfo, error) {
	query := "?leader=true&master=true"
	apiURL := fmt.Sprintf("%s/%s%s", mc.url, membersPrefix, query)
	return mc.GetMembers(apiURL)
}

// NewMasterClient returns a new MasterClient
func NewMasterClient(url string, timeout time.Duration, tlsConfig *tls.Config) MasterClient {
	var disableKeepalive bool
	if tlsConfig != nil {
		disableKeepalive = true
	}
	return &masterClient{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: disableKeepalive},
		},
	}
}
