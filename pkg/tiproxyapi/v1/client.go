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

package tiproxyapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"

	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
)

const (
	healthPath          = "api/debug/health"
	healthUnhealthyPath = "api/debug/health/unhealthy"
	configPath          = "api/admin/config"
)

// TiProxyClient is the interface that knows how to control tiproxy clusters.
type TiProxyClient interface {
	// IsHealthy checks if the TiProxy is healthy.
	IsHealthy(ctx context.Context) (bool, error)
	// MarkUnhealthy makes the TiProxy health endpoint report unhealthy.
	MarkUnhealthy(ctx context.Context) error
	// GetGracefulWaitBeforeShutdown returns the current graceful-wait-before-shutdown config.
	GetGracefulWaitBeforeShutdown(ctx context.Context) (int, error)
	// SetLabels sets the labels for TiProxy.
	SetLabels(ctx context.Context, labels map[string]string) error
}

// tiproxyClient is the default implementation of TiProxyClient.
type tiproxyClient struct {
	url        string
	httpClient *http.Client
}

// NewTiProxyClient returns a new TiProxyClient.
func NewTiProxyClient(addr string, timeout time.Duration, tlsConfig *tls.Config) TiProxyClient {
	var disableKeepalive bool
	if tlsConfig != nil {
		disableKeepalive = true
	}
	scheme := "http://"
	if tlsConfig != nil {
		scheme = "https://"
	}
	return &tiproxyClient{
		url: scheme + addr,
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

func (c *tiproxyClient) IsHealthy(ctx context.Context) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, healthPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return false, err
	}

	//nolint:bodyclose,gosec // bodyclose: has been handled; gosec: URL is constructed from trusted internal config
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer httputil.DeferClose(resp.Body)

	// NOTE: we don't check the response body here.
	return resp.StatusCode < http.StatusBadRequest, nil
}

func (c *tiproxyClient) MarkUnhealthy(ctx context.Context) error {
	apiURL := fmt.Sprintf("%s/%s", c.url, healthUnhealthyPath)
	_, err := httputil.PostBodyOK(ctx, c.httpClient, apiURL, nil)
	return err
}

func (c *tiproxyClient) GetGracefulWaitBeforeShutdown(ctx context.Context) (int, error) {
	type proxyConfig struct {
		Proxy struct {
			GracefulWaitBeforeShutdown int `json:"graceful-wait-before-shutdown"`
		} `json:"proxy"`
	}

	apiURL := fmt.Sprintf("%s/%s", c.url, configPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")

	//nolint:bodyclose,gosec // bodyclose: has been handled; gosec: URL is constructed from trusted internal config
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer httputil.DeferClose(resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		bodyErr := httputil.ReadErrorBody(resp.Body)
		if bodyErr == nil {
			return 0, httputil.Errorf(resp.StatusCode, "error response to %s", apiURL)
		}
		return 0, httputil.Errorf(resp.StatusCode, "error response to %s: %s", apiURL, bodyErr.Error())
	}

	var cfg proxyConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return 0, err
	}

	return cfg.Proxy.GracefulWaitBeforeShutdown, nil
}

func (c *tiproxyClient) SetLabels(ctx context.Context, labels map[string]string) error {
	type labelConfig struct {
		Labels map[string]string `toml:"labels"`
	}
	cfg := labelConfig{Labels: labels}
	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(cfg); err != nil {
		return fmt.Errorf("encode labels to toml failed, error: %w", err)
	}

	apiURL := fmt.Sprintf("%s/%s", c.url, configPath)
	_, err := httputil.PutBodyOK(ctx, c.httpClient, apiURL, &buffer)
	return err
}
