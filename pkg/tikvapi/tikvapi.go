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

package tikvapi

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	"k8s.io/klog"
)

const (
	DefaultTimeout        = 5 * time.Second
	metricNameRegionCount = "tikv_raftstore_region_count"
	labelNameLeaderCount  = "leader"
	metricsPrefix         = "metrics"
)

// TiKVClient provides tikv server's api
type TiKVClient interface {
	GetLeaderCount() (int, error)
}

// tikvClient is default implementation of TiKVClient
type tikvClient struct {
	url        string
	httpClient *http.Client
}

// GetLeaderCount gets region leader count from the URL
func (c *tikvClient) GetLeaderCount() (int, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, metricsPrefix)
	transport := c.httpClient.Transport
	mfChan := make(chan *dto.MetricFamily, 1024)

	go func() {
		if err := prom2json.FetchMetricFamilies(apiURL, mfChan, transport); err != nil {
			klog.Errorf("Fail to get region leader count from %s, error: %v", apiURL, err)
		}
	}()

	fms := []*prom2json.Family{}
	for mfc := range mfChan {
		fm := prom2json.NewFamily(mfc)
		fms = append(fms, fm)
	}
	for _, fm := range fms {
		if fm.Name == metricNameRegionCount {
			for _, m := range fm.Metrics {
				if m, ok := m.(prom2json.Metric); ok && m.Labels["type"] == labelNameLeaderCount {
					return strconv.Atoi(m.Value)
				}
			}
		}
	}

	return 0, fmt.Errorf("metric %s{type=\"%s\"} not found for %s", metricNameRegionCount, labelNameLeaderCount, apiURL)
}

// NewTiKVClient returns a new TiKVClient
func NewTiKVClient(url string, timeout time.Duration, tlsConfig *tls.Config, disableKeepalive bool) TiKVClient {
	return &tikvClient{
		url: url,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig:       tlsConfig,
				DisableKeepAlives:     disableKeepalive,
				ResponseHeaderTimeout: 10 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).DialContext,
			},
		},
	}
}
