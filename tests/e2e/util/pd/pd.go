// Copyright 2021 PingCAP, Inc.
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

package pd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Member struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peerURLs"`
	ClientURLs []string `json:"clientURLs"`
}

type GetMembersResponse struct {
	Members []Member `json:"members"`
}

func GetMembersV2(addr string) (*GetMembersResponse, error) {
	url := fmt.Sprintf("http://%s/v2/members", addr)

	httpResp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	data, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %s", err)
	}

	if httpResp.StatusCode < http.StatusContinue || httpResp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("code %s msg %s", httpResp.Status, string(data))
	}

	resp := &GetMembersResponse{}
	err = json.Unmarshal(data, resp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal failed: %s", err)
	}

	return resp, nil
}

func UpdateMemberPeerURLs(addr string, id string, peerURLs []string) error {
	url := fmt.Sprintf("http://%s/v2/members/%s", addr, id)

	member := Member{
		PeerURLs: peerURLs,
	}
	data, err := json.Marshal(member)
	if err != nil {
		return fmt.Errorf("marshal failed %s", err)
	}

	httpReq, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("create req failed %s", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < http.StatusContinue || httpResp.StatusCode >= http.StatusBadRequest {
		respData, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			return fmt.Errorf("read body failed: %s", err)
		}

		return fmt.Errorf("code %s msg %s", httpResp.Status, string(respData))
	}

	return nil
}
