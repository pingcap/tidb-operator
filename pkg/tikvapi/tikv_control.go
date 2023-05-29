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

package tikvapi

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

// TiKVControlInterface is an interface that knows how to manage and get client for TiKV
type TiKVControlInterface interface {
	// GetTiKVPodClient provides TiKVClient of the TiKV cluster.
	GetTiKVPodClient(namespace string, tcName string, podName string, tlsEnabled bool) TiKVClient
}

// defaultTiKVControl is the default implementation of TiKVControlInterface.
type defaultTiKVControl struct {
	mutex        sync.Mutex
	secretLister corelisterv1.SecretLister
	tikvClients  map[string]TiKVClient
}

// NewDefaultTiKVControl returns a defaultTiKVControl instance
func NewDefaultTiKVControl(secretLister corelisterv1.SecretLister) TiKVControlInterface {
	return &defaultTiKVControl{secretLister: secretLister, tikvClients: map[string]TiKVClient{}}
}

func (tc *defaultTiKVControl) GetTiKVPodClient(namespace string, tcName string, podName string, tlsEnabled bool) TiKVClient {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	var tlsConfig *tls.Config
	var err error
	var scheme = "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = pdapi.GetTLSConfig(tc.secretLister, pdapi.Namespace(namespace), util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			klog.Errorf("Unable to get tls config for TiKV cluster %q, tikv client may not work: %v", tcName, err)
			return NewTiKVClient(TiKVPodClientURL(namespace, tcName, podName, scheme), DefaultTimeout, tlsConfig, true)
		}

		return NewTiKVClient(TiKVPodClientURL(namespace, tcName, podName, scheme), DefaultTimeout, tlsConfig, true)
	}

	return NewTiKVClient(TiKVPodClientURL(namespace, tcName, podName, scheme), DefaultTimeout, tlsConfig, true)
}

func tikvPodClientKey(schema, namespace, clusterName, podName string) string {
	return fmt.Sprintf("%s.%s.%s.%s", schema, clusterName, namespace, podName)
}

// TiKVPodClientURL builds the url of tikv pod client
func TiKVPodClientURL(namespace, clusterName, podName, scheme string) string {
	return fmt.Sprintf("%s://%s.%s-tikv-peer.%s:%d", scheme, podName, clusterName, namespace, v1alpha1.DefaultTiKVStatusPort)
}

// FakeTiKVControl implements a fake version of TiKVControlInterface.
type FakeTiKVControl struct {
	defaultTiKVControl
	tikvPodClients map[string]TiKVClient
}

func NewFakeTiKVControl(secretLister corelisterv1.SecretLister) *FakeTiKVControl {
	return &FakeTiKVControl{
		defaultTiKVControl: defaultTiKVControl{secretLister: secretLister, tikvClients: map[string]TiKVClient{}},
		tikvPodClients:     map[string]TiKVClient{},
	}
}

func (ftc *FakeTiKVControl) SetTiKVPodClient(namespace, tcName, podName string, tikvPodClient TiKVClient) {
	ftc.tikvPodClients[tikvPodClientKey("http", namespace, tcName, podName)] = tikvPodClient
}

func (ftc *FakeTiKVControl) GetTiKVPodClient(namespace, tcName, podName string, tlsEnabled bool) TiKVClient {
	return ftc.tikvPodClients[tikvPodClientKey("http", namespace, tcName, podName)]
}
