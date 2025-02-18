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

package ticdcapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	httputil "github.com/pingcap/tidb-operator/pkg/utils/http"
)

const (
	statusPath       = "status"
	healthPath       = "api/v1/health"
	getCapturesPath  = "api/v1/captures"
	drainCapturePath = "api/v1/captures/drain"
	resignOwnerPath  = "api/v1/owner/resign"
)

// TiCDCClient provides TiDB server's APIs used by TiDB Operator.
// NOTE: Some of these APIs are only available in v6.3.0 or later.
type TiCDCClient interface {
	// GetStatus returns ticdc's status
	GetStatus(ctx context.Context) (*CaptureStatus, error)
	// DrainCapture remove capture ownership and moves its tables to other captures.
	// Returns the number of tables in the capture.
	// If there is only one capture, it always return 0.
	DrainCapture(ctx context.Context) (tableCount int, err error)
	// ResignOwner tries to resign ownership from the current capture.
	// Returns true if the capture has already resigned ownership,
	// otherwise caller should retry resign owner.
	// If there is only one capture, it always return true.
	ResignOwner(ctx context.Context) (ok bool, err error)
	// IsHealthy gets the healthy status of TiCDC cluster.
	// Returns true if the TiCDC cluster is healthy.
	IsHealthy(ctx context.Context) (ok bool, err error)
}

// ticdcClient is the default implementation of TiCDCClient.
type ticdcClient struct {
	url          string
	instanceName string
	httpClient   *http.Client
}

// NewTiCDCClient returns a new TiCDCClient.
func NewTiCDCClient(url, instanceName string, timeout time.Duration, tlsConfig *tls.Config) TiCDCClient {
	var disableKeepalive bool
	if tlsConfig != nil {
		disableKeepalive = true
	}
	return &ticdcClient{
		url:          url,
		instanceName: instanceName,
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

func (c *ticdcClient) GetStatus(ctx context.Context) (*CaptureStatus, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, statusPath)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	var status CaptureStatus
	err = json.Unmarshal(body, &status)
	return &status, err
}

func (c *ticdcClient) DrainCapture(ctx context.Context) (int, error) {
	captures, err := c.getCaptures(ctx)
	if err != nil {
		return 0, err
	}
	if len(captures) < 2 {
		// No way to drain a single node TiCDC cluster, ignore
		return 0, nil
	}

	this, owner := getThisAndOwnerCaptureInfo(c.instanceName, captures)
	if this == nil {
		return 0, fmt.Errorf("capture not found, instance: %s, captures: %+v", c.instanceName, captures)
	}
	if owner == nil {
		return 0, fmt.Errorf("owner not found, captures: %+v", captures)
	}

	payload := drainCaptureRequest{
		CaptureID: this.ID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("ticdc drain capture failed, marshal request error: %w", err)
	}
	apiURL := fmt.Sprintf("%s/%s", c.url, drainCapturePath)
	// Put instead of Post here
	body, err := httputil.PutBodyOK(ctx, c.httpClient, apiURL, bytes.NewBuffer(data))
	if err != nil {
		return 0, fmt.Errorf("ticdc drain capture failed, request error: %w", err)
	}

	var resp drainCaptureResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		// It is likely the TiCDC does not support the API, ignore
		return 0, nil
	}
	return resp.CurrentTableCount, nil
}

func (c *ticdcClient) ResignOwner(ctx context.Context) (bool, error) {
	captures, err := c.getCaptures(ctx)
	if err != nil {
		return false, err
	}
	if len(captures) < 2 {
		// No way to resign owner in a single node TiCDC cluster, ignore
		return true, nil
	}

	this, owner := getThisAndOwnerCaptureInfo(c.instanceName, captures)
	if owner != nil && this != nil {
		if owner.ID != this.ID {
			// Ownership has been transferred another capture
			return true, nil
		}
	} else {
		// Owner or this capture not found, resign ownership from the capture is
		// meaning less, ignore
		return true, nil
	}

	apiURL := fmt.Sprintf("%s/%s", c.url, resignOwnerPath)
	_, err = httputil.PostBodyOK(ctx, c.httpClient, apiURL, nil)
	// always return false, let the caller to retry and check the above condition
	return false, err
}

func (c *ticdcClient) IsHealthy(ctx context.Context) (bool, error) {
	captures, err := c.getCaptures(ctx)
	if err != nil {
		return false, err
	}
	if len(captures) == 0 {
		// No way to get capture info, ignore
		return true, nil
	}

	_, owner := getThisAndOwnerCaptureInfo(c.instanceName, captures)
	if owner == nil {
		// Unhealthy, owner is not found
		return false, nil
	}

	apiURL := fmt.Sprintf("%s/%s", c.url, healthPath)
	_, err = httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *ticdcClient) getCaptures(ctx context.Context) ([]captureInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, getCapturesPath)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}

	var resp []captureInfo
	err = json.Unmarshal(body, &resp)
	return resp, err
}

func getThisAndOwnerCaptureInfo(thisInstance string, captures []captureInfo) (this, owner *captureInfo) {
	for i := range captures {
		cp := &captures[i]
		sl := strings.Split(cp.AdvertiseAddr, ".")
		if len(sl) > 0 && sl[0] == thisInstance {
			this = cp
		}
		if cp.IsOwner {
			owner = cp
		}
	}
	return
}
