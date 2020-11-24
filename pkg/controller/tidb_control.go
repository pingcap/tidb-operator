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

package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	"github.com/pingcap/tidb/config"
	"k8s.io/client-go/kubernetes"
)

const (
	// https://github.com/pingcap/tidb/blob/master/owner/manager.go#L183
	// NotDDLOwnerError is the error message which was returned when the tidb node is not a ddl owner
	NotDDLOwnerError = "This node is not a ddl owner, can't be resigned."
	timeout          = 5 * time.Second
)

type DBInfo struct {
	IsOwner bool `json:"is_owner"`
}

// TiDBControlInterface is the interface that knows how to manage tidb peers
type TiDBControlInterface interface {
	// GetHealth returns tidb's health info
	GetHealth(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error)
	// Get TIDB info return tidb's DBInfo
	GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*DBInfo, error)
	// GetSettings return the TiDB instance settings
	GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error)
}

// defaultTiDBControl is default implementation of TiDBControlInterface.
type defaultTiDBControl struct {
	httpClient
	// for unit test only
	testURL string
}

// NewDefaultTiDBControl returns a defaultTiDBControl instance
func NewDefaultTiDBControl(kubeCli kubernetes.Interface) *defaultTiDBControl {
	return &defaultTiDBControl{httpClient: httpClient{kubeCli: kubeCli}}
}

func (c *defaultTiDBControl) GetHealth(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return false, err
	}

	baseURL := c.getBaseURL(tc, ordinal)
	url := fmt.Sprintf("%s/status", baseURL)
	_, err = getBodyOK(httpClient, url)
	return err == nil, nil
}

func (c *defaultTiDBControl) GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*DBInfo, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return nil, err
	}

	baseURL := c.getBaseURL(tc, ordinal)
	url := fmt.Sprintf("%s/info", baseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %s:%v URL: %s", string(body), res.StatusCode, url))
		return nil, errMsg
	}
	info := DBInfo{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *defaultTiDBControl) GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return nil, err
	}

	baseURL := c.getBaseURL(tc, ordinal)
	url := fmt.Sprintf("%s/settings", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %s:%v URL: %s", string(body), res.StatusCode, url))
		return nil, errMsg
	}
	info := config.Config{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func getBodyOK(httpClient *http.Client, apiURL string) ([]byte, error) {
	res, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %s:%v URL %s", string(body), res.StatusCode, apiURL))
		return nil, errMsg
	}

	return body, err
}

func (c *defaultTiDBControl) getBaseURL(tc *v1alpha1.TidbCluster, ordinal int32) string {
	if c.testURL != "" {
		return c.testURL
	}

	tcName := tc.GetName()
	ns := tc.GetNamespace()
	scheme := tc.Scheme()
	hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), ordinal)

	return fmt.Sprintf("%s://%s.%s.%s:10080", scheme, hostName, TiDBPeerMemberName(tcName), ns)
}

// FakeTiDBControl is a fake implementation of TiDBControlInterface.
type FakeTiDBControl struct {
	healthInfo   map[string]bool
	tiDBInfo     *DBInfo
	getInfoError error
	tidbConfig   *config.Config
}

// NewFakeTiDBControl returns a FakeTiDBControl instance
func NewFakeTiDBControl() *FakeTiDBControl {
	return &FakeTiDBControl{}
}

// SetHealth set health info for FakeTiDBControl
func (c *FakeTiDBControl) SetHealth(healthInfo map[string]bool) {
	c.healthInfo = healthInfo
}

func (c *FakeTiDBControl) GetHealth(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	podName := fmt.Sprintf("%s-%d", TiDBMemberName(tc.GetName()), ordinal)
	if c.healthInfo == nil {
		return false, nil
	}
	if health, ok := c.healthInfo[podName]; ok {
		return health, nil
	}
	return false, nil
}

func (c *FakeTiDBControl) GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*DBInfo, error) {
	return c.tiDBInfo, c.getInfoError
}

func (c *FakeTiDBControl) GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error) {
	return c.tidbConfig, c.getInfoError
}
