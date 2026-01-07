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

package tidbapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
)

const (
	statusPath           = "status"
	infoPath             = "info"
	labelsPath           = "labels"
	tidbPoolActivatePath = "tidb-pool/activate"
	tidbPoolStatusPath   = "tidb-pool/status"
)

// TiDBClient provides TiDB server's APIs used by TiDB Operator.
type TiDBClient interface {
	// GetHealth gets the health status of this TiDB server.
	GetHealth(ctx context.Context) (bool, error)
	// GetInfo gets the information of this TiDB server.
	GetInfo(ctx context.Context) (*ServerInfo, error)
	// SetServerLabels sets the labels of this TiDB server.
	SetServerLabels(ctx context.Context, labels map[string]string) error

	// GetPoolStatus get the tidb pool status(standby or activated) of the tidb
	GetPoolStatus(ctx context.Context) (*PoolStatus, error)
	// Activate sets the keyspace of a standby TiDB instance.
	Activate(ctx context.Context, keyspace string) error
}

// tidbClient is the default implementation of TiDBClient.
type tidbClient struct {
	url        string
	httpClient *http.Client
}

// NewTiDBClient returns a new TiDBClient.
func NewTiDBClient(url string, timeout time.Duration, tlsConfig *tls.Config) TiDBClient {
	var disableKeepalive bool
	if tlsConfig != nil {
		disableKeepalive = true
	}
	return &tidbClient{
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

func (c *tidbClient) GetHealth(ctx context.Context) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, statusPath)
	_, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	// NOTE: we don't check the response body here.
	return err == nil, err
}

func (c *tidbClient) GetInfo(ctx context.Context) (*ServerInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, infoPath)
	// NOTE: in TiDB Operator v1, we use "POST" to get the info.
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}

	info := ServerInfo{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// SetServerLabels will set labels of tidb server.
// TODO(liubo02): now this function call is not safe because labels maybe changed by others after get info is called.
func (c *tidbClient) SetServerLabels(ctx context.Context, labels map[string]string) error {
	buffer := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buffer).Encode(labels); err != nil {
		return fmt.Errorf("encode labels to json failed, error: %w", err)
	}

	apiURL := fmt.Sprintf("%s/%s", c.url, labelsPath)
	if _, err := httputil.PostBodyOK(ctx, c.httpClient, apiURL, buffer); err != nil {
		return err
	}
	return nil
}

func (c *tidbClient) Activate(ctx context.Context, keyspace string) error {
	buffer := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buffer).Encode(&ActivateRequest{
		KeyspaceName:   keyspace,
		RunAutoAnalyze: true,
		TiDBEnableDDL:  true,
	}); err != nil {
		return fmt.Errorf("encode request to json failed, error: %w", err)
	}

	apiURL := fmt.Sprintf("%s/%s", c.url, tidbPoolActivatePath)
	_, err := httputil.PostBodyOK(ctx, c.httpClient, apiURL, buffer)
	return err
}

func (c *tidbClient) GetPoolStatus(ctx context.Context) (*PoolStatus, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, tidbPoolStatusPath)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		if httputil.IsNotFound(err) {
			// If not found, state is activated and keyspace is unknown
			// This api is only available for the tidb started in the standby mode
			return &PoolStatus{
				State: PoolStateActivated,
			}, nil
		}
		return nil, err
	}

	status := PoolStatus{}
	err = json.Unmarshal(body, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}
