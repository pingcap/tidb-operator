// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
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
)

const (
	metricsPath                = "metrics"
	metricNameRegionCount      = "tikv_raftstore_region_count"
	metricLabelNameLeaderCount = "leader"

	metricChanSize = 1024
)

// TiKVClient provides TiKV server's APIs used by TiDB Operator.
type TiKVClient interface {
	// GetLeaderCount gets region leader count of this TiKV store.
	GetLeaderCount() (int, error)
}

// tikvClient is the default implementation of TiKVClient.
type tikvClient struct {
	url        string
	httpClient *http.Client
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

func (c *tikvClient) GetLeaderCount() (int, error) {
	// we need to get the region leader count via the metrics API.
	apiURL := fmt.Sprintf("%s/%s", c.url, metricsPath)
	transport := c.httpClient.Transport
	mfChan := make(chan *dto.MetricFamily, metricChanSize)

	var fetchErr error
	go func() {
		if err := prom2json.FetchMetricFamilies(apiURL, mfChan, transport); err != nil {
			fetchErr = fmt.Errorf("fail to fetch metric families from %s, error: %w", apiURL, err)
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
				if m, ok := m.(prom2json.Metric); ok && m.Labels["type"] == metricLabelNameLeaderCount {
					return strconv.Atoi(m.Value)
				}
			}
		}
	}

	if fetchErr != nil {
		return 0, fetchErr
	}
	return 0, fmt.Errorf("metric %s{type=\"%s\"} not found for %s", metricNameRegionCount, metricLabelNameLeaderCount, apiURL)
}
