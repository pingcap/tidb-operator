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

package tiflashapi

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
)

const (
	storeStatusPath = "tiflash/store-status"
)

type Status string

const (
	Idle       Status = "Idle"
	Ready      Status = "Ready"
	Running    Status = "Running"
	Stopping   Status = "Stopping"
	Terminated Status = "Terminated"
)

type TiFlashClient interface {
	GetStoreStatus() (Status, error)
}

type tiflashClient struct {
	url        string
	httpClient *http.Client
}

// NewTiFlashClient returns a new TiFlashClient
func NewTiFlashClient(url string, timeout time.Duration, tlsConfig *tls.Config, disableKeepalive bool) TiFlashClient {
	return &tiflashClient{
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

func (c *tiflashClient) GetStoreStatus() (Status, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, storeStatusPath)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return "", err
	}

	return Status(body), nil
}
