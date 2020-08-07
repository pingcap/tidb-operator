// Copyright 2020 PingCAP, Inc.
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

package dmapi

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/pingcap/tidb-operator/pkg/pdapi"

	"github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/klog"

	"k8s.io/client-go/kubernetes"
)

// Namespace is a newtype of a string
type Namespace pdapi.Namespace

// MasterControlInterface is an interface that knows how to manage and get dm cluster's master client
type MasterControlInterface interface {
	// GetMasterClient provides MasterClient of the dm cluster.
	GetMasterClient(namespace Namespace, tcName string, tlsEnabled bool) MasterClient
}

// defaultMasterControl is the default implementation of MasterControlInterface.
type defaultMasterControl struct {
	mutex         sync.Mutex
	kubeCli       kubernetes.Interface
	masterClients map[string]MasterClient
}

// NewDefaultMasterControl returns a defaultMasterControl instance
func NewDefaultMasterControl(kubeCli kubernetes.Interface) MasterControlInterface {
	return &defaultMasterControl{kubeCli: kubeCli, masterClients: map[string]MasterClient{}}
}

// GetPDClient provides a PDClient of real pd cluster,if the PDClient not existing, it will create new one.
func (mc *defaultMasterControl) GetMasterClient(namespace Namespace, dcName string, tlsEnabled bool) MasterClient {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	var tlsConfig *tls.Config
	var err error
	var scheme = "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = pdapi.GetTLSConfig(mc.kubeCli, pdapi.Namespace(namespace), dcName, util.ClusterClientTLSSecretName(dcName))
		if err != nil {
			klog.Errorf("Unable to get tls config for dm cluster %q, master client may not work: %v", dcName, err)
			return &masterClient{url: MasterClientURL(namespace, dcName, scheme), httpClient: &http.Client{Timeout: DefaultTimeout}}
		}

		return NewMasterClient(MasterClientURL(namespace, dcName, scheme), DefaultTimeout, tlsConfig)
	}

	key := masterClientKey(scheme, namespace, dcName)
	if _, ok := mc.masterClients[key]; !ok {
		mc.masterClients[key] = NewMasterClient(MasterClientURL(namespace, dcName, scheme), DefaultTimeout, nil)
	}
	return mc.masterClients[key]
}

// masterClientKey returns the master client key
func masterClientKey(scheme string, namespace Namespace, clusterName string) string {
	return fmt.Sprintf("%s.%s.%s", scheme, clusterName, string(namespace))
}

// MasterClientURL builds the url of master client
func MasterClientURL(namespace Namespace, clusterName string, scheme string) string {
	return fmt.Sprintf("%s://%s-master.%s:2379", scheme, clusterName, string(namespace))
}

// FakeMasterControl implements a fake version of MasterControlInterface.
type FakeMasterControl struct {
	defaultMasterControl
}

func NewFakeMasterControl(kubeCli kubernetes.Interface) *FakeMasterControl {
	return &FakeMasterControl{
		defaultMasterControl{kubeCli: kubeCli, masterClients: map[string]MasterClient{}},
	}
}

func (fmc *FakeMasterControl) SetMasterClient(namespace Namespace, tcName string, masterClient MasterClient) {
	fmc.defaultMasterControl.masterClients[masterClientKey("http", namespace, tcName)] = masterClient
}
