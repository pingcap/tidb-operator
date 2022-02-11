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

package tngm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type NGMConprofConfig struct {
	Enable               bool `json:"enable,omitempty"`
	ProfileSeconds       int  `json:"profile_seconds,omitempty"`
	IntervalSeconds      int  `json:"interval_seconds,omitempty"`
	TimeoutSeconds       int  `json:"timeout_seconds,omitempty"`
	DataRetentionSeconds int  `json:"data_retention_seconds,omitempty"`
}

type NGMConfig struct {
	Addr          string `json:"addr"`
	AdvertiseAddr string `json:"advertise_addr"`

	Conprof *NGMConprofConfig `json:"continuous_profiling,omitempty"`
}

type ConfigureNGMReq struct {
	Conprof *NGMConprofConfig `json:"continuous_profiling,omitempty"`
}

func GetConfig(addr string) (*NGMConfig, error) {
	url := fmt.Sprintf("http://%s/config", addr)

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

	config := &NGMConfig{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal failed: %s", err)
	}

	return config, nil
}

func SetConfig(addr string, req *ConfigureNGMReq) error {
	url := fmt.Sprintf("http://%s/config", addr)

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal failed %s", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
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
