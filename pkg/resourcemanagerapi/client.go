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

package resourcemanagerapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
)

// Primary is the request payload for transfer primary.
type Primary struct {
	NewPrimary string `json:"new_primary"`
}

// Client provides Resource Manager microservice APIs used by TiDB Operator.
type Client interface {
	TransferPrimary(ctx context.Context, transferee string) error
}

const (
	apiPrefix = "/resource-manager/api/v1"
	// transfer primary
	primaryTransferPrefix = apiPrefix + "/primary/transfer"
)

type client struct {
	url        string
	httpClient *http.Client
	tlsConfig  *tls.Config
}

// NewClient returns a new Resource Manager API client.
func NewClient(url string, timeout time.Duration, tlsConfig *tls.Config) Client {
	return &client{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
		},
		tlsConfig: tlsConfig,
	}
}

func (c *client) TransferPrimary(ctx context.Context, transferee string) error {
	apiURL := c.url + primaryTransferPrefix
	data, err := json.Marshal(&Primary{NewPrimary: transferee})
	if err != nil {
		return err
	}
	if _, err := httputil.PostBodyOK(ctx, c.httpClient, apiURL, bytes.NewBuffer(data)); err != nil {
		return fmt.Errorf("failed to transfer resource manager primary to %s, error: %w", transferee, err)
	}
	return nil
}
