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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"

	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
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
			CheckRedirect: httputil.RejectRedirect,
			Timeout:       timeout,
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("fail to fetch metric families from %s, error: %w", apiURL, err)
	}
	// Use the configured client so its timeout and redirect policy also apply
	// to metrics requests.
	req.Header.Set("Accept", "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;"+
		"encoding=delimited;q=0.7,text/plain;version=0.0.4;q=0.3")
	//nolint:bodyclose,gosec // bodyclose: handled by DeferClose; gosec: URL is constructed from internal config
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fail to fetch metric families from %s, error: %w", apiURL, err)
	}
	defer httputil.DeferClose(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fail to fetch metric families from %s: status %d", apiURL, resp.StatusCode)
	}
	mfChan := make(chan *dto.MetricFamily, metricChanSize)
	errChan := make(chan error, 1)

	go func() {
		if err := prom2json.ParseResponse(resp, mfChan); err != nil {
			errChan <- fmt.Errorf("fail to fetch metric families from %s, error: %w", apiURL, err)
			return
		}
		errChan <- nil
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

	if fetchErr := <-errChan; fetchErr != nil {
		return 0, fetchErr
	}
	return 0, fmt.Errorf("metric %s{type=\"%s\"} not found for %s", metricNameRegionCount, metricLabelNameLeaderCount, apiURL)
}
