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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/client-go/kubernetes"
)

type CaptureStatus struct {
	ID string `json:"id"`
}

// TiCDCControlInterface is the interface that knows how to manage ticdc captures
type TiCDCControlInterface interface {
	// GetStatus returns ticdc's status
	GetStatus(tc *v1alpha1.TidbCluster, ordinal int32) (*CaptureStatus, error)
}

// defaultTiCDCControl is default implementation of TiCDCControlInterface.
type defaultTiCDCControl struct {
	kubeCli kubernetes.Interface
	// for unit test only
	testURL string
}

// NewDefaultTiCDCControl returns a defaultTiCDCControl instance
func NewDefaultTiCDCControl(kubeCli kubernetes.Interface) *defaultTiCDCControl {
	return &defaultTiCDCControl{kubeCli: kubeCli}
}

func (tcc *defaultTiCDCControl) GetStatus(tc *v1alpha1.TidbCluster, ordinal int32) (*CaptureStatus, error) {
	httpClient, err := tcc.getHTTPClient(tc)
	if err != nil {
		return nil, err
	}

	baseURL := tcc.getBaseURL(tc, ordinal)
	url := fmt.Sprintf("%s/status", baseURL)
	body, err := getBodyOK(httpClient, url)
	if err != nil {
		return nil, err
	}

	status := CaptureStatus{}
	err = json.Unmarshal(body, &status)
	return &status, err
}

func (tcc *defaultTiCDCControl) getHTTPClient(tc *v1alpha1.TidbCluster) (*http.Client, error) {
	return &http.Client{Timeout: timeout}, nil
}

func (tcc *defaultTiCDCControl) getBaseURL(tc *v1alpha1.TidbCluster, ordinal int32) string {
	if tcc.testURL != "" {
		return tcc.testURL
	}

	tcName := tc.GetName()
	ns := tc.GetNamespace()
	scheme := tc.Scheme()
	hostName := fmt.Sprintf("%s-%d", TiCDCMemberName(tcName), ordinal)

	return fmt.Sprintf("%s://%s.%s.%s:8301", scheme, hostName, TiCDCPeerMemberName(tcName), ns)
}

// FakeTiCDCControl is a fake implementation of TiCDCControlInterface.
type FakeTiCDCControl struct {
	status *CaptureStatus
}

// NewFakeTiCDCControl returns a FakeTiCDCControl instance
func NewFakeTiCDCControl() *FakeTiCDCControl {
	return &FakeTiCDCControl{}
}

// SetHealth set health info for FakeTiCDCControl
func (ftd *FakeTiCDCControl) SetStatus(status *CaptureStatus) {
	ftd.status = status
}
