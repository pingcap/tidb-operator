// Copyright 2022 PingCAP, Inc.
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
	"io/ioutil"
	"net/http"

	"github.com/pingcap/TiProxy/lib/config"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

// TiProxyControlInterface is the interface that knows how to control tiproxy clusters
type TiProxyControlInterface interface {
	// IsPodHealthy gets the healthy status of TiProxy pod.
	IsPodHealthy(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error)
	// SetConfigProxy set the proxy part in config
	SetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32, cfg *config.ProxyServerOnline) error
	// GetConfigProxy get the proxy part in config
	GetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32) (*config.ProxyServerOnline, error)
}

// defaultTiProxyControl is default implementation of TiProxyControlInterface.
type defaultTiProxyControl struct {
	httpClient
}

// NewDefaultTiProxyControl returns a defaultTiProxyControl instance
func NewDefaultTiProxyControl(secretLister corelisterv1.SecretLister) *defaultTiProxyControl {
	return &defaultTiProxyControl{httpClient: httpClient{secretLister: secretLister}}
}

func (c *defaultTiProxyControl) getBaseURL(tc *v1alpha1.TidbCluster, ordinal int32) string {
	tcName := tc.GetName()
	return fmt.Sprintf("%s://%s.%s.%s:3080/api",
		tc.Scheme(),
		fmt.Sprintf("%s-%d", TiProxyMemberName(tcName), ordinal),
		TiProxyPeerMemberName(tcName),
		tc.GetNamespace(),
	)
}

func (c *defaultTiProxyControl) IsPodHealthy(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return false, err
	}
	res, err := httpClient.Get(fmt.Sprintf("%s/metrics", c.getBaseURL(tc, ordinal)))
	if err != nil {
		return false, err
	}
	defer httputil.DeferClose(res.Body)
	return res.StatusCode == http.StatusOK, nil
}

func (c *defaultTiProxyControl) GetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32) (*config.ProxyServerOnline, error) {
	// TODO: use TiProxy/lib/cli
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Get(fmt.Sprintf("%s/admin/config/proxy", c.getBaseURL(tc, ordinal)))
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	be, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", be)
	}
	ret := &config.ProxyServerOnline{}
	if err := json.Unmarshal(be, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *defaultTiProxyControl) SetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32, cfg *config.ProxyServerOnline) error {
	// TODO: use TiProxy/lib/cli
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return err
	}
	be, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	res, err := httpClient.Post(fmt.Sprintf("%s/admin/config/proxy", c.getBaseURL(tc, ordinal)), "application/json", bytes.NewReader(be))
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode != http.StatusOK {
		be, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", be)
	}
	return nil
}
