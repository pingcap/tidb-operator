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
}

var (
	membersPrefix = "apis/v1alpha1/members"
)

type ListMemberRespHeader struct {
	Result bool   `json:"result,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

type MastersInfo struct {
	Name       string   `json:"name,omitempty"`
	MemberID   uint64   `json:"memberID,omitempty"`
	Alive      bool     `json:"alive,omitempty"`
	PeerURLs   []string `json:"peerURLs,omitempty"`
	ClientURLs []string `json:"clientURLs,omitempty"`
}

type MembersMaster struct {
	Msg     string         `json:"msg,omitempty"`
	Masters []*MastersInfo `json:"masters,omitempty"`
}

type ListMemberMaster struct {
	MembersMaster `json:"master,omitempty"`
}

type MastersResp struct {
	ListMemberRespHeader `json:",inline"`

	ListMemberResp []*ListMemberMaster `json:"members,omitempty"`
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
		return nil, err
	}
	if !listMemberResp.Result {
		return nil, fmt.Errorf("unable to list members info from dm-master, err: %s", listMemberResp.Msg)
	}
	if len(listMemberResp.ListMemberResp) != 1 {
		return nil, fmt.Errorf("invalid list members resp: %s", string(body))
	}

	return listMemberResp.ListMemberResp[0].Masters, nil
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
