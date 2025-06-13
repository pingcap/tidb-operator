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
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"

	httputil "github.com/pingcap/tidb-operator/pkg/utils/http"
)

const (
	healthPath = "api/debug/health"
	configPath = "api/admin/config/"
)

// TiProxyClient is the interface that knows how to control tiproxy clusters.
type TiProxyClient interface {
	// IsHealthy checks if the TiProxy is healthy.
	IsHealthy(ctx context.Context) (bool, error)
	// SetLabels sets the labels for TiProxy.
	SetLabels(ctx context.Context, labels map[string]string) error
}

// tiproxyClient is the default implementation of TiProxyClient.
type tiproxyClient struct {
	url        string
	httpClient *http.Client
}

// NewTiProxyClient returns a new TiProxyClient.
func NewTiProxyClient(url string, timeout time.Duration, tlsConfig *tls.Config) TiProxyClient {
	var disableKeepalive bool
	if tlsConfig != nil {
		disableKeepalive = true
	}
	return &tiproxyClient{
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

func (c *tiproxyClient) IsHealthy(ctx context.Context) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, healthPath)
	_, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	// NOTE: we don't check the response body here.
	return err == nil, err
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
