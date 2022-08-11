// Copyright 2018 PingCAP, Inc.
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

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

type CaptureStatus struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	IsOwner bool   `json:"is_owner"`
}

type captureInfo struct {
	ID            string `json:"id"`
	IsOwner       bool   `json:"is_owner"`
	AdvertiseAddr string `json:"address"`
}

// drainCaptureRequest is request for manual `DrainCapture`
type drainCaptureRequest struct {
	CaptureID string `json:"capture_id"`
}

// drainCaptureResp is response for manual `DrainCapture`
type drainCaptureResp struct {
	CurrentTableCount int `json:"current_table_count"`
}

// TiCDCControlInterface is the interface that knows how to manage ticdc captures
type TiCDCControlInterface interface {
	// GetStatus returns ticdc's status
	GetStatus(tc *v1alpha1.TidbCluster, ordinal int32) (*CaptureStatus, error)
	// DrainCapture remove capture ownership and moves its tables to other captures.
	// Returns the number of tables in the capture.
	// If there is only one capture, it always return 0.
	DrainCapture(tc *v1alpha1.TidbCluster, ordinal int32) (tableCount int, retry bool, err error)
	// ResignOwner tries to resign ownership from the current capture.
	// Returns true if the capture has already resigned ownership,
	// otherwise caller should retry resign owner.
	// If there is only one capture, it always return true.
	ResignOwner(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error)
	// IsHealthy gets the healthy status of TiCDC cluster.
	// Returns true if the TiCDC cluster is heathy.
	IsHealthy(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error)
}

// defaultTiCDCControl is default implementation of TiCDCControlInterface.
type defaultTiCDCControl struct {
	httpClient
	// for unit test only
	testURL string
}

// NewDefaultTiCDCControl returns a defaultTiCDCControl instance
func NewDefaultTiCDCControl(secretLister corelisterv1.SecretLister) *defaultTiCDCControl {
	return &defaultTiCDCControl{httpClient: httpClient{secretLister: secretLister}}
}

func (c *defaultTiCDCControl) GetStatus(tc *v1alpha1.TidbCluster, ordinal int32) (*CaptureStatus, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		return nil, err
	}

	baseURL := c.getBaseURL(tc, ordinal)
	url := fmt.Sprintf("%s/status", baseURL)
	body, err := getBodyOK(httpClient, url)
	if err != nil {
		return nil, err
	}

	status := CaptureStatus{}
	err = json.Unmarshal(body, &status)
	return &status, err
}

func (c *defaultTiCDCControl) DrainCapture(tc *v1alpha1.TidbCluster, ordinal int32) (int, bool, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		klog.Warningf("ticdc control: drain capture is failed, error: %v", err)
		return 0, false, err
	}

	baseURL := c.getBaseURL(tc, ordinal)

	captures, retry, err := getCaptures(httpClient, baseURL)
	if err != nil {
		klog.Warningf("ticdc control: drain capture is failed, error: %v", err)
		return 0, false, err
	}
	if retry {
		// Let caller retry drain capture.
		return 0, true, nil
	}
	if len(captures) == 1 {
		// No way to drain a single node TiCDC cluster, ignore.
		return 0, false, nil
	}

	this, owner := getOrdinalAndOwnerCaptureInfo(tc, ordinal, captures)
	if this == nil {
		addr := getCaptureAdvertiseAddressPrefix(tc, ordinal)
		return 0, false, fmt.Errorf("capture not found, address: %s, captures: %+v", addr, captures)
	}
	if owner == nil {
		return 0, false, fmt.Errorf("owner not found, captures: %+v", captures)
	}

	payload := drainCaptureRequest{
		CaptureID: this.ID,
	}
	payloadBody, err := json.Marshal(payload)
	if err != nil {
		return 0, false, fmt.Errorf("ticdc drain capture failed, marshal request error: %v", err)
	}
	req, err := http.NewRequest("PUT", baseURL+"/api/v1/captures/drain", bytes.NewReader(payloadBody))
	if err != nil {
		return 0, false, fmt.Errorf("ticdc drain capture failed, new request error: %v", err)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("ticdc drain capture failed, request error: %v", err)
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusNotFound {
		// It is likely the TiCDC does not support the API, ignore.
		klog.Infof("ticdc control: %s does not support drain capture, skip", this.AdvertiseAddr)
		return 0, false, nil
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		// TiCDC is not ready, retry.
		klog.Infof("ticdc control: %s service unavailable drain capture, retry", this.AdvertiseAddr)
		return 0, true, nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, false, fmt.Errorf("ticdc drain capture failed, read response error: %v", err)
	}

	var resp drainCaptureResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		// It is likely the TiCDC does not support the API, ignore.
		return 0, false, nil
	}
	return resp.CurrentTableCount, false, nil
}

func (c *defaultTiCDCControl) ResignOwner(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		klog.Warningf("ticdc control: resign owner failed, error: %v", err)
		return false, err
	}

	baseURL := c.getBaseURL(tc, ordinal)
	captures, retry, err := getCaptures(httpClient, baseURL)
	if err != nil {
		klog.Warningf("ticdc control: resign owner failed, error: %v", err)
		return false, err
	}
	if retry {
		// Let caller retry resign owner.
		return false, nil
	}
	if len(captures) == 1 {
		// No way to resign owner in a single node TiCDC cluster, ignore.
		return true, nil
	}

	this, owner := getOrdinalAndOwnerCaptureInfo(tc, ordinal, captures)
	if owner != nil && this != nil {
		if owner.ID != this.ID {
			// Ownership has been transferred another capture.
			return true, nil
		}
	} else {
		// Owner or this capture not found, resign ownership from the capture is
		// meaning less, ignore.
		return true, nil
	}

	res, err := httpClient.Post(baseURL+"/api/v1/owner/resign", "", nil)
	if err != nil {
		return false, fmt.Errorf("ticdc resign owner failed, request error: %v", err)
	}
	httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusNotFound {
		// It is likely the TiCDC does not support the API, ignore.
		klog.Infof("ticdc control: %s does not support resign owner, skip", this.AdvertiseAddr)
		return true, nil
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		// Let caller retry resign owner.
		klog.Infof("ticdc control: %s service unavailable resign owner, retry", this.AdvertiseAddr)
		return false, nil
	}
	return false, nil
}

func (c *defaultTiCDCControl) IsHealthy(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	httpClient, err := c.getHTTPClient(tc)
	if err != nil {
		klog.Warningf("ticdc control: get http client failed, error: %v", err)
		return false, err
	}

	captures, retry, err := getCaptures(httpClient, c.getBaseURL(tc, ordinal))
	if err != nil {
		klog.Warningf("ticdc control: get capture failed, error: %v", err)
		return false, err
	}
	if retry {
		// Let caller retry.
		return false, nil
	}

	_, owner := getOrdinalAndOwnerCaptureInfo(tc, ordinal, captures)
	if owner == nil {
		// Unhealthy, owner is not found.
		return false, nil
	}

	healthURL := fmt.Sprintf("%s://%s/api/v1/health", tc.Scheme(), owner.AdvertiseAddr)
	res, err := httpClient.Get(healthURL)
	if err != nil {
		return false, fmt.Errorf("ticdc get health failed, request error: %v", err)
	}
	httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusNotFound {
		// It is likely the TiCDC does not support the API, ignore.
		klog.Infof("ticdc control: %s does not support health API, skip", owner.AdvertiseAddr)
		return true, nil
	}
	if res.StatusCode == http.StatusInternalServerError {
		// Let caller retry get health.
		klog.Infof("ticdc control: %s report unhealthy, retry", owner.AdvertiseAddr)
		return false, nil
	}
	return true, nil
}

func (c *defaultTiCDCControl) getBaseURL(tc *v1alpha1.TidbCluster, ordinal int32) string {
	if c.testURL != "" {
		return c.testURL
	}

	scheme := tc.Scheme()
	addr := getCaptureAdvertiseAddressPrefix(tc, ordinal)
	return fmt.Sprintf("%s://%s:8301", scheme, addr)
}

// getCaptureAdvertiseAddressPrefix is the prefix of TiCDC advertiseAddress
// which is composed by ${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.${ClusterDomain}:8301
// this function return a string "${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}"
func getCaptureAdvertiseAddressPrefix(tc *v1alpha1.TidbCluster, ordinal int32) string {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	hostName := fmt.Sprintf("%s-%d", TiCDCMemberName(tcName), ordinal)

	return fmt.Sprintf("%s.%s.%s", hostName, TiCDCPeerMemberName(tcName), ns)
}

func getCaptures(httpClient *http.Client, baseURL string) ([]captureInfo, bool, error) {
	res, err := httpClient.Get(baseURL + "/api/v1/captures")
	if err != nil {
		return nil, false, fmt.Errorf("ticdc get captures failed, request error: %v", err)
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusNotFound {
		// It is likely the TiCDC does not support the API, ignore.
		return nil, false, nil
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		// TiCDC is not ready, retry.
		return nil, true, nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, false, fmt.Errorf("ticdc get captures failed, read response error: %v", err)
	}
	var resp []captureInfo
	err = json.Unmarshal(body, &resp)
	if err != nil {
		// It is likely the TiCDC does not support the API, ignore.
		return nil, false, nil
	}
	return resp, false, nil
}

func getOrdinalAndOwnerCaptureInfo(
	tc *v1alpha1.TidbCluster, ordinal int32, captures []captureInfo,
) (this, owner *captureInfo) {
	addrPrefix := getCaptureAdvertiseAddressPrefix(tc, ordinal)
	for i := range captures {
		cp := &captures[i]
		if strings.Contains(cp.AdvertiseAddr, addrPrefix) {
			this = cp
		}
		if cp.IsOwner {
			owner = cp
		}
	}
	return
}

// FakeTiCDCControl is a fake implementation of TiCDCControlInterface.
type FakeTiCDCControl struct {
	GetStatusFn    func(tc *v1alpha1.TidbCluster, ordinal int32) (*CaptureStatus, error)
	DrainCaptureFn func(tc *v1alpha1.TidbCluster, ordinal int32) (tableCount int, retry bool, err error)
	ResignOwnerFn  func(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error)
	IsHealthyFn    func(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error)
}

// NewFakeTiCDCControl returns a FakeTiCDCControl instance
func NewFakeTiCDCControl() *FakeTiCDCControl {
	return &FakeTiCDCControl{}
}

func (c *FakeTiCDCControl) GetStatus(tc *v1alpha1.TidbCluster, ordinal int32) (*CaptureStatus, error) {
	if c.GetStatusFn == nil {
		return nil, fmt.Errorf("undefined GetStatus")
	}
	return c.GetStatusFn(tc, ordinal)
}

func (c *FakeTiCDCControl) DrainCapture(tc *v1alpha1.TidbCluster, ordinal int32) (tableCount int, retry bool, err error) {
	if c.DrainCaptureFn == nil {
		return 0, false, fmt.Errorf("undefined DrainCapture")
	}
	return c.DrainCaptureFn(tc, ordinal)
}

func (c *FakeTiCDCControl) ResignOwner(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error) {
	if c.ResignOwnerFn == nil {
		return true, fmt.Errorf("undefined ResignOwner")
	}
	return c.ResignOwnerFn(tc, ordinal)
}

func (c *FakeTiCDCControl) IsHealthy(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	if c.IsHealthyFn == nil {
		return true, fmt.Errorf("undefined IsHealthy")
	}
	return c.IsHealthyFn(tc, ordinal)
}
