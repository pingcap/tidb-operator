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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/httputil"
	certutil "github.com/pingcap/tidb-operator/pkg/util/crypto"
	"github.com/pingcap/tidb/config"
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
	GetHealth(tc *v1alpha1.TidbCluster) map[string]bool
	// Get TIDB info return tidb's DBInfo
	GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*DBInfo, error)
	// GetSettings return the TiDB instance settings
	GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error)
}

// defaultTiDBControl is default implementation of TiDBControlInterface.
type defaultTiDBControl struct {
	httpClient *http.Client
}

// NewDefaultTiDBControl returns a defaultTiDBControl instance
func NewDefaultTiDBControl() TiDBControlInterface {
	return &defaultTiDBControl{httpClient: &http.Client{Timeout: timeout}}
}

func (tdc *defaultTiDBControl) useTLSHTTPClient(enableTLS bool) error {
	if enableTLS {
		rootCAs, err := certutil.ReadCACerts()
		if err != nil {
			return err
		}
		config := &tls.Config{
			RootCAs: rootCAs,
		}
		tdc.httpClient.Transport = &http.Transport{TLSClientConfig: config}
	}
	return nil
}

func (tdc *defaultTiDBControl) GetHealth(tc *v1alpha1.TidbCluster) map[string]bool {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	scheme := tc.Scheme()

	result := map[string]bool{}

	if err := tdc.useTLSHTTPClient(tc.Spec.EnableTLSCluster); err != nil {
		return result
	}

	for i := 0; i < int(tc.TiDBStsActualReplicas()); i++ {
		hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), i)
		url := fmt.Sprintf("%s://%s.%s.%s:10080/status", scheme, hostName, TiDBPeerMemberName(tcName), ns)
		_, err := tdc.getBodyOK(url)
		if err != nil {
			result[hostName] = false
		} else {
			result[hostName] = true
		}
	}
	return result
}

func (tdc *defaultTiDBControl) GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*DBInfo, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	scheme := tc.Scheme()
	if err := tdc.useTLSHTTPClient(tc.Spec.EnableTLSCluster); err != nil {
		return nil, err
	}

	hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), ordinal)
	url := fmt.Sprintf("%s://%s.%s.%s:10080/info", scheme, hostName, TiDBPeerMemberName(tcName), ns)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := tdc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %v URL: %s", res.StatusCode, url))
		return nil, errMsg
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	info := DBInfo{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (tdc *defaultTiDBControl) GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	scheme := tc.Scheme()
	if err := tdc.useTLSHTTPClient(tc.Spec.EnableTLSCluster); err != nil {
		return nil, err
	}

	hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), ordinal)
	url := fmt.Sprintf("%s://%s.%s.%s:10080/settings", scheme, hostName, TiDBPeerMemberName(tcName), ns)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := tdc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %v URL: %s", res.StatusCode, url))
		return nil, errMsg
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	info := config.Config{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (tdc *defaultTiDBControl) getBodyOK(apiURL string) ([]byte, error) {
	res, err := tdc.httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %v URL %s", res.StatusCode, apiURL))
		return nil, errMsg
	}

	defer httputil.DeferClose(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, err
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
func (ftd *FakeTiDBControl) SetHealth(healthInfo map[string]bool) {
	ftd.healthInfo = healthInfo
}

func (ftd *FakeTiDBControl) GetHealth(_ *v1alpha1.TidbCluster) map[string]bool {
	return ftd.healthInfo
}

func (ftd *FakeTiDBControl) GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*DBInfo, error) {
	return ftd.tiDBInfo, ftd.getInfoError
}

func (ftd *FakeTiDBControl) GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error) {
	return ftd.tidbConfig, ftd.getInfoError
}
