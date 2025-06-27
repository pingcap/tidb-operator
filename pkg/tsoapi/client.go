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

package tsoapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	httputil "github.com/pingcap/tidb-operator/pkg/utils/http"
)

// Primary is the type for transfer primary
type Primary struct {
	NewPrimary string `json:"new_primary"`
}

// TSOClient provides tso api
type TSOClient interface {
	// GetHealth returns ping result
	// GetHealth() error

	// NOTE: this only transfer the leader of the default keyspace group
	TransferTSOLeader(ctx context.Context, transferee string) error
}

const (
	apiPrefix = "/tso/api/v1"
	// transfer tso leaders
	tsoLeaderTransferPrefix = apiPrefix + "/primary/transfer"
)

type client struct {
	url        string
	httpClient *http.Client
	tlsConfig  *tls.Config
}

// NewTSOClient returns a new TSOClient
func NewTSOClient(url string, timeout time.Duration, tlsConfig *tls.Config) TSOClient {
	return &client{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
		},
		tlsConfig: tlsConfig,
	}
}

func (c *client) TransferTSOLeader(ctx context.Context, name string) error {
	apiURL := c.url + tsoLeaderTransferPrefix
	data, err := json.Marshal(&Primary{
		NewPrimary: name,
	})
	if err != nil {
		return err
	}
	if _, err := httputil.PostBodyOK(ctx, c.httpClient, apiURL, bytes.NewBuffer(data)); err != nil {
		return fmt.Errorf("failed to transfer tso leader to %s, error: %w", name, err)
	}

	return nil
}
