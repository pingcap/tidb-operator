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
	"sync"
	"time"

	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

const (
	DefaultTimeout = 5 * time.Second
)

// TiFlashControlInterface is an interface that knows how to manage and get client for TiFlash
type TiFlashControlInterface interface {
	// GetTiFlashPodClient provides TiFlashClient of the TiFlash cluster.
	GetTiFlashPodClient(namespace string, tcName string, podName string, tlsEnabled bool) TiFlashClient
}

// defaultTiFlashControl is the default implementation of TiFlashControlInterface.
type defaultTiFlashControl struct {
	mutex        sync.Mutex
	secretLister corelisterv1.SecretLister
}

// NewDefaultTiFlashControl returns a defaultTiFlashControl instance
func NewDefaultTiFlashControl(secretLister corelisterv1.SecretLister) TiFlashControlInterface {
	return &defaultTiFlashControl{secretLister: secretLister}
}

func (tc *defaultTiFlashControl) GetTiFlashPodClient(namespace string, tcName string, podName string, tlsEnabled bool) TiFlashClient {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	var tlsConfig *tls.Config
	var err error
	var scheme = "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = pdapi.GetTLSConfig(tc.secretLister, pdapi.Namespace(namespace), util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			klog.Errorf("Unable to get tls config for TiFlash cluster %q, tiflash client may not work: %v", tcName, err)
			return NewTiFlashClient(TiFlashPodClientURL(namespace, tcName, podName, scheme), DefaultTimeout, tlsConfig, true)
		}

		return NewTiFlashClient(TiFlashPodClientURL(namespace, tcName, podName, scheme), DefaultTimeout, tlsConfig, true)
	}

	return NewTiFlashClient(TiFlashPodClientURL(namespace, tcName, podName, scheme), DefaultTimeout, tlsConfig, true)
}

func tiflashPodClientKey(schema, namespace, clusterName, podName string) string {
	return fmt.Sprintf("%s.%s.%s.%s", schema, clusterName, namespace, podName)
}

// TiFlashPodClientURL builds the url of tiflash pod client
func TiFlashPodClientURL(namespace, clusterName, podName, scheme string) string {
	return fmt.Sprintf("%s://%s.%s-tiflash-peer.%s:20292", scheme, podName, clusterName, namespace)
}

// FakeTiFlashControl implements a fake version of TiFlashControlInterface.
type FakeTiFlashControl struct {
	defaultTiFlashControl
	tiflashPodClients map[string]TiFlashClient
}

func NewFakeTiFlashControl(secretLister corelisterv1.SecretLister) *FakeTiFlashControl {
	return &FakeTiFlashControl{
		defaultTiFlashControl: defaultTiFlashControl{secretLister: secretLister},
		tiflashPodClients:     map[string]TiFlashClient{},
	}
}

func (ftc *FakeTiFlashControl) SetTiFlashPodClient(namespace, tcName, podName string, tiflashPodClient TiFlashClient) {
	ftc.tiflashPodClients[tiflashPodClientKey("http", namespace, tcName, podName)] = tiflashPodClient
}

func (ftc *FakeTiFlashControl) GetTiFlashPodClient(namespace, tcName, podName string, tlsEnabled bool) TiFlashClient {
	return ftc.tiflashPodClients[tiflashPodClientKey("http", namespace, tcName, podName)]
}
