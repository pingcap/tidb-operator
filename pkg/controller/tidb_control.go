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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

const (
	// https://github.com/pingcap/tidb/blob/master/owner/manager.go#L183
	// NotDDLOwnerError is the error message which was returned when the tidb node is not a ddl owner
	NotDDLOwnerError               = "This node is not a ddl owner, can't be resigned."
	timeout                        = 5 * time.Second
	startUpgradeTimeout            = 30 * time.Second
	tidbUpgradeSuccessBody         = "success!"
	tidbUpgradeDuplicateStartBody  = "It's a duplicated operation and the cluster is already in upgrading state."
	tidbUpgradeDuplicateFinishBody = "It's a duplicated operation and the cluster is already in normal state."
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
	// SetServerLabels update TiDB's labels config
	SetServerLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error
	// StartUpgrade switches TiDB into upgrade state.
	StartUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error
	// FinishUpgrade switches TiDB back to normal state.
	FinishUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error
}

// defaultTiDBControl is default implementation of TiDBControlInterface.
type defaultTiDBControl struct {
	httpClient
	// for unit test only
	testURL string
}

// NewDefaultTiDBControl returns a defaultTiDBControl instance
func NewDefaultTiDBControl(secretLister corelisterv1.SecretLister) *defaultTiDBControl {
	return &defaultTiDBControl{httpClient: httpClient{secretLister: secretLister}}
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf("Error response %s:%v URL: %s", string(body), res.StatusCode, url)
		return nil, errMsg
	}
	info := DBInfo{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// SetServerLabels update TiDB's labels config
func (c *defaultTiDBControl) SetServerLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buffer).Encode(labels); err != nil {
		return fmt.Errorf("encode labels to json failed, error: %v", err)
	}

	url := fmt.Sprintf("%s/labels", c.getBaseURL(tc, ordinal))
	_, err = httputil.PostBodyOK(httpClient, url, buffer)
	return err
}

func (c *defaultTiDBControl) StartUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error {
	return c.upgrade(tc, ordinal, "start", c.upgradeClientTimeout("start"), tidbUpgradeDuplicateStartBody)
}

func (c *defaultTiDBControl) FinishUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error {
	return c.upgrade(tc, ordinal, "finish", c.upgradeClientTimeout("finish"), tidbUpgradeDuplicateFinishBody)
}

func (c *defaultTiDBControl) upgradeClientTimeout(op string) time.Duration {
	if op == "start" {
		return startUpgradeTimeout
	}
	return timeout
}

func (c *defaultTiDBControl) upgrade(tc *v1alpha1.TidbCluster, ordinal int32, op string, requestTimeout time.Duration, duplicateSuccessBody string) error {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return err
	}
	httpClient.Timeout = requestTimeout

	url := fmt.Sprintf("%s/upgrade/%s", c.getBaseURL(tc, ordinal), op)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	bodyText := string(body)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("tidb upgrade %s error response %s:%v URL: %s", op, bodyText, res.StatusCode, url)
	}
	successText := normalizeTiDBUpgradeResponse(body)
	if successText == tidbUpgradeSuccessBody || successText == duplicateSuccessBody {
		return nil
	}
	return fmt.Errorf("tidb upgrade %s unexpected response %q:%v URL: %s", op, bodyText, res.StatusCode, url)
}

func normalizeTiDBUpgradeResponse(body []byte) string {
	var text string
	if err := json.Unmarshal(body, &text); err == nil {
		return text
	}
	return string(body)
}

func getBodyOK(httpClient *http.Client, apiURL string) ([]byte, error) {
	res, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		errMsg := fmt.Errorf("Error response %s:%v URL %s", string(body), res.StatusCode, apiURL)
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

	baseURL := fmt.Sprintf("%s://%s.%s.%s:%d", scheme, hostName, TiDBPeerMemberName(tcName), ns, v1alpha1.DefaultTiDBStatusPort)
	if tc.Spec.ClusterDomain != "" {
		baseURL = fmt.Sprintf("%s://%s.%s.%s.svc.%s:%d", scheme, hostName, TiDBPeerMemberName(tcName), ns, tc.Spec.ClusterDomain, v1alpha1.DefaultTiDBStatusPort)
	}
	return baseURL
}

// FakeTiDBControl is a fake implementation of TiDBControlInterface.
type FakeTiDBControl struct {
	healthInfo            map[string]bool
	tiDBInfo              *DBInfo
	getInfoError          error
	setLabelsError        error
	startUpgradeError     error
	finishUpgradeError    error
	StartUpgradeOrdinals  []int32
	FinishUpgradeOrdinals []int32
}

// NewFakeTiDBControl returns a FakeTiDBControl instance
func NewFakeTiDBControl(secretLister corelisterv1.SecretLister) *FakeTiDBControl {
	return &FakeTiDBControl{}
}

// SetHealth set health info for FakeTiDBControl
func (c *FakeTiDBControl) SetHealth(healthInfo map[string]bool) {
	c.healthInfo = healthInfo
}

func (c *FakeTiDBControl) SetLabelsErr(err error) {
	c.setLabelsError = err
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

func (c *FakeTiDBControl) SetServerLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error {
	return c.setLabelsError
}

func (c *FakeTiDBControl) SetStartUpgradeError(err error) {
	c.startUpgradeError = err
}

func (c *FakeTiDBControl) SetFinishUpgradeError(err error) {
	c.finishUpgradeError = err
}

func (c *FakeTiDBControl) StartUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error {
	c.StartUpgradeOrdinals = append(c.StartUpgradeOrdinals, ordinal)
	return c.startUpgradeError
}

func (c *FakeTiDBControl) FinishUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error {
	c.FinishUpgradeOrdinals = append(c.FinishUpgradeOrdinals, ordinal)
	return c.finishUpgradeError
}
