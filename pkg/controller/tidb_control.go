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

	"github.com/pingcap/tidb-operator/pkg/apis/httputil"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb/config"
)

const (
	// https://github.com/pingcap/tidb/blob/master/owner/manager.go#L183
	// NotDDLOwnerError is the error message which was returned when the tidb node is not a ddl owner
	NotDDLOwnerError = "This node is not a ddl owner, can't be resigned."
	timeout          = 5 * time.Second
)

type dbInfo struct {
	IsOwner bool `json:"is_owner"`
}

// TiDBControlInterface is the interface that knows how to manage tidb peers
type TiDBControlInterface interface {
	// GetHealth returns tidb's health info
	GetHealth(tc *v1alpha1.TidbCluster) map[string]bool
	// ResignDDLOwner resigns the ddl owner of tidb, if the tidb node is not a ddl owner returns (true,nil),else returns (false,err)
	ResignDDLOwner(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error)
	// Get TIDB info return tidb's dbInfo
	GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*dbInfo, error)
	// GetSettings return the TiDB instance settings
	GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error)
}

// defaultTiDBControl is default implementation of TiDBControlInterface.
type defaultTiDBControl struct {
	httpClient *http.Client
}

// NewDefaultTiDBControl returns a defaultTiDBControl instance
func NewDefaultTiDBControl() TiDBControlInterface {
	httpClient := &http.Client{Timeout: timeout}
	return &defaultTiDBControl{httpClient: httpClient}
}

func (tdc *defaultTiDBControl) GetHealth(tc *v1alpha1.TidbCluster) map[string]bool {
	tcName := tc.GetName()
	ns := tc.GetNamespace()

	result := map[string]bool{}
	for i := 0; i < int(tc.TiDBRealReplicas()); i++ {
		hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), i)
		url := fmt.Sprintf("http://%s.%s.%s:10080/status", hostName, TiDBPeerMemberName(tcName), ns)
		_, err := tdc.getBodyOK(url)
		if err != nil {
			result[hostName] = false
		} else {
			result[hostName] = true
		}
	}
	return result
}

func (tdc *defaultTiDBControl) ResignDDLOwner(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()

	hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), ordinal)
	url := fmt.Sprintf("http://%s.%s.%s:10080/ddl/owner/resign", hostName, TiDBPeerMemberName(tcName), ns)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return false, err
	}
	res, err := tdc.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer httputil.DeferClose(res.Body, &err)
	if res.StatusCode == http.StatusOK {
		return false, nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	if err2.Error() == NotDDLOwnerError {
		return true, nil
	}
	return false, err2
}

func (tdc *defaultTiDBControl) GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*dbInfo, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()

	hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), ordinal)
	url := fmt.Sprintf("http://%s.%s.%s:10080/info", hostName, TiDBPeerMemberName(tcName), ns)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := tdc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body, &err)
	if res.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %v URL: %s", res.StatusCode, url))
		return nil, errMsg
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	info := dbInfo{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (tdc *defaultTiDBControl) GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()

	hostName := fmt.Sprintf("%s-%d", TiDBMemberName(tcName), ordinal)
	url := fmt.Sprintf("http://%s.%s.%s:10080/settings", hostName, TiDBPeerMemberName(tcName), ns)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := tdc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body, &err)
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

	defer httputil.DeferClose(res.Body, &err)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

// FakeTiDBControl is a fake implementation of TiDBControlInterface.
type FakeTiDBControl struct {
	healthInfo          map[string]bool
	resignDDLOwnerError error
	notDDLOwner         bool
	tidbInfo            *dbInfo
	getInfoError        error
	tidbConfig          *config.Config
}

// NewFakeTiDBControl returns a FakeTiDBControl instance
func NewFakeTiDBControl() *FakeTiDBControl {
	return &FakeTiDBControl{}
}

// SetHealth set health info for FakeTiDBControl
func (ftd *FakeTiDBControl) SetHealth(healthInfo map[string]bool) {
	ftd.healthInfo = healthInfo
}

//  NotDDLOwner sets whether the tidb is the ddl owner
func (ftd *FakeTiDBControl) NotDDLOwner(notDDLOwner bool) {
	ftd.notDDLOwner = notDDLOwner
}

//  SetResignDDLOwner sets error of resign ddl owner for FakeTiDBControl
func (ftd *FakeTiDBControl) SetResignDDLOwnerError(err error) {
	ftd.resignDDLOwnerError = err
}

func (ftd *FakeTiDBControl) GetHealth(_ *v1alpha1.TidbCluster) map[string]bool {
	return ftd.healthInfo
}

func (ftd *FakeTiDBControl) ResignDDLOwner(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	return ftd.notDDLOwner, ftd.resignDDLOwnerError
}

func (ftd *FakeTiDBControl) GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*dbInfo, error) {
	return ftd.tidbInfo, ftd.getInfoError
}

func (ftd *FakeTiDBControl) GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error) {
	return ftd.tidbConfig, ftd.getInfoError
}
