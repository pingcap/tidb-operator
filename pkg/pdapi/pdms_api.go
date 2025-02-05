// Copyright 2023 PingCAP, Inc.
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

package pdapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	"k8s.io/klog/v2"
)

// PDMSClient provides pd MS server's api
type PDMSClient interface {
	// GetHealth returns ping result
	GetHealth() error
	// TransferPrimary transfers the primary to the newPrimary
	TransferPrimary(newPrimary string) error
}

var (
	pdMSHealthPrefix          = "api/v1/health"
	pdMSPrimaryTransferPrefix = "api/v1/primary/transfer"
)

// pdMSClient is default implementation of PDClient
type pdMSClient struct {
	serviceName string
	url         string
	httpClient  *http.Client
}

// NewPDMSClient returns a new PDClient
func NewPDMSClient(serviceName, url string, timeout time.Duration, tlsConfig *tls.Config) *pdMSClient {
	var disableKeepalive bool
	if tlsConfig != nil {
		disableKeepalive = true
	}
	return &pdMSClient{
		serviceName: serviceName,
		url:         url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: disableKeepalive},
		},
	}
}

func (c *pdMSClient) GetHealth() error {
	// only support TSO service
	if c.serviceName != TSOServiceName {
		klog.Errorf("only support TSO service, but got %s", c.serviceName)
		return nil
	}
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, c.serviceName, pdMSHealthPrefix)
	_, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return err
	}
	return nil
}

func (c *pdMSClient) TransferPrimary(newPrimary string) error {
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, c.serviceName, pdMSPrimaryTransferPrefix)
	data, err := json.Marshal(struct {
		NewPrimary string `json:"new_primary"`
	}{
		NewPrimary: newPrimary,
	})
	if err != nil {
		return err
	}
	_, err = httputil.PostBodyOK(c.httpClient, apiURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	return nil
}
