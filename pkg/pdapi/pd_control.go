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

package pdapi

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// Namespace is a newtype of a string
type Namespace string

// PDControlInterface is an interface that knows how to manage and get tidb cluster's PD client
type PDControlInterface interface {
	// GetPDClient provides PDClient of the tidb cluster.
	GetPDClient(namespace Namespace, tcName string, tlsEnabled bool) PDClient
	// GetPDEtcdClient provides PD etcd Client of the tidb cluster.
	GetPDEtcdClient(namespace Namespace, tcName string, tlsEnabled bool) (PDEtcdClient, error)
}

// defaultPDControl is the default implementation of PDControlInterface.
type defaultPDControl struct {
	mutex         sync.Mutex
	etcdmutex     sync.Mutex
	kubeCli       kubernetes.Interface
	pdClients     map[string]PDClient
	pdEtcdClients map[string]PDEtcdClient
}

// NewDefaultPDControl returns a defaultPDControl instance
func NewDefaultPDControl(kubeCli kubernetes.Interface) PDControlInterface {
	return &defaultPDControl{kubeCli: kubeCli, pdClients: map[string]PDClient{}, pdEtcdClients: map[string]PDEtcdClient{}}
}

func (pdc *defaultPDControl) GetPDEtcdClient(namespace Namespace, tcName string, tlsEnabled bool) (PDEtcdClient, error) {
	pdc.etcdmutex.Lock()
	defer pdc.etcdmutex.Unlock()

	var tlsConfig *tls.Config
	var err error

	if tlsEnabled {
		tlsConfig, err = GetTLSConfig(pdc.kubeCli, namespace, tcName, util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			klog.Errorf("Unable to get tls config for tidb cluster %q, pd etcd client may not work: %v", tcName, err)
			return nil, err
		}
		return NewPdEtcdClient(PDEtcdClientURL(namespace, tcName), DefaultTimeout, tlsConfig)
	}
	key := pdEtcdClientKey(namespace, tcName)
	if _, ok := pdc.pdEtcdClients[key]; !ok {
		pdetcdClient, err := NewPdEtcdClient(PDEtcdClientURL(namespace, tcName), DefaultTimeout, nil)
		if err != nil {
			return nil, err
		}
		pdc.pdEtcdClients[key] = pdetcdClient
	}
	return pdc.pdEtcdClients[key], nil
}

// GetPDClient provides a PDClient of real pd cluster,if the PDClient not existing, it will create new one.
func (pdc *defaultPDControl) GetPDClient(namespace Namespace, tcName string, tlsEnabled bool) PDClient {
	pdc.mutex.Lock()
	defer pdc.mutex.Unlock()

	var tlsConfig *tls.Config
	var err error
	var scheme = "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = GetTLSConfig(pdc.kubeCli, namespace, tcName, util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			klog.Errorf("Unable to get tls config for tidb cluster %q, pd client may not work: %v", tcName, err)
			return &pdClient{url: PdClientURL(namespace, tcName, scheme), httpClient: &http.Client{Timeout: DefaultTimeout}}
		}

		return NewPDClient(PdClientURL(namespace, tcName, scheme), DefaultTimeout, tlsConfig)
	}

	key := pdClientKey(scheme, namespace, tcName)
	if _, ok := pdc.pdClients[key]; !ok {
		pdc.pdClients[key] = NewPDClient(PdClientURL(namespace, tcName, scheme), DefaultTimeout, nil)
	}
	return pdc.pdClients[key]
}

// pdClientKey returns the pd client key
func pdClientKey(scheme string, namespace Namespace, clusterName string) string {
	return fmt.Sprintf("%s.%s.%s", scheme, clusterName, string(namespace))
}

func pdEtcdClientKey(namespace Namespace, clusterName string) string {
	return fmt.Sprintf("%s.%s", clusterName, string(namespace))
}

// pdClientUrl builds the url of pd client
func PdClientURL(namespace Namespace, clusterName string, scheme string) string {
	return fmt.Sprintf("%s://%s-pd.%s:2379", scheme, clusterName, string(namespace))
}

func PDEtcdClientURL(namespace Namespace, clusterName string) string {
	return fmt.Sprintf("%s-pd.%s:2379", clusterName, string(namespace))
}

// FakePDControl implements a fake version of PDControlInterface.
type FakePDControl struct {
	defaultPDControl
}

func NewFakePDControl(kubeCli kubernetes.Interface) *FakePDControl {
	return &FakePDControl{
		defaultPDControl{kubeCli: kubeCli, pdClients: map[string]PDClient{}},
	}
}

func (fpc *FakePDControl) SetPDClient(namespace Namespace, tcName string, pdclient PDClient) {
	fpc.defaultPDControl.pdClients[pdClientKey("http", namespace, tcName)] = pdclient
}
