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
	"sync"

	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// MasterControlInterface is an interface that knows how to manage and get dm cluster's master client
type MasterControlInterface interface {
	// GetMasterClient provides MasterClient of the dm cluster.
	GetMasterClient(namespace string, dcName string, tlsEnabled bool) MasterClient
	GetMasterPeerClient(namespace string, dcName, podName string, tlsEnabled bool) MasterClient
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

// GetMasterClient provides a MasterClient of real dm-master cluster, if the MasterClient not existing, it will create new one.
func (mc *defaultMasterControl) GetMasterClient(namespace string, dcName string, tlsEnabled bool) MasterClient {
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
			return NewMasterClient(MasterClientURL(namespace, dcName, scheme), DefaultTimeout, tlsConfig, true)
		}

		return NewMasterClient(MasterClientURL(namespace, dcName, scheme), DefaultTimeout, tlsConfig, true)
	}

	key := masterClientKey(scheme, namespace, dcName)
	if _, ok := mc.masterClients[key]; !ok {
		mc.masterClients[key] = NewMasterClient(MasterClientURL(namespace, dcName, scheme), DefaultTimeout, nil, false)
	}
	return mc.masterClients[key]
}

func (mc *defaultMasterControl) GetMasterPeerClient(namespace string, dcName string, podName string, tlsEnabled bool) MasterClient {
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
			return NewMasterClient(MasterPeerClientURL(namespace, dcName, podName, scheme), DefaultTimeout, tlsConfig, true)
		}

		return NewMasterClient(MasterPeerClientURL(namespace, dcName, podName, scheme), DefaultTimeout, tlsConfig, true)
	}

	return NewMasterClient(MasterPeerClientURL(namespace, dcName, podName, scheme), DefaultTimeout, tlsConfig, true)
}

// masterClientKey returns the master client key
func masterClientKey(scheme, namespace, clusterName string) string {
	return fmt.Sprintf("%s.%s.%s", scheme, clusterName, namespace)
}

func masterPeerClientKey(schema, namespace, clusterName, podName string) string {
	return fmt.Sprintf("%s.%s.%s.%s", schema, clusterName, namespace, podName)
}

// MasterClientURL builds the url of master client
func MasterClientURL(namespace, clusterName, scheme string) string {
	return fmt.Sprintf("%s://%s-dm-master.%s:8261", scheme, clusterName, namespace)
}

// MasterPeerClientURL builds the url of master peer client. It's used to evict leader because dm can't forward evict leader command now
func MasterPeerClientURL(namespace, clusterName, podName, scheme string) string {
	return fmt.Sprintf("%s://%s.%s-dm-master-peer.%s:8261", scheme, podName, clusterName, namespace)
}

// FakeMasterControl implements a fake version of MasterControlInterface.
type FakeMasterControl struct {
	defaultMasterControl
	masterPeerClients map[string]MasterClient
}

func NewFakeMasterControl(kubeCli kubernetes.Interface) *FakeMasterControl {
	return &FakeMasterControl{
		defaultMasterControl: defaultMasterControl{kubeCli: kubeCli, masterClients: map[string]MasterClient{}},
		masterPeerClients:    map[string]MasterClient{},
	}
}

func (fmc *FakeMasterControl) SetMasterClient(namespace, dcName string, masterClient MasterClient) {
	fmc.defaultMasterControl.masterClients[masterClientKey("http", namespace, dcName)] = masterClient
}

func (fmc *FakeMasterControl) SetMasterPeerClient(namespace, dcName, podName string, masterPeerClient MasterClient) {
	fmc.masterPeerClients[masterPeerClientKey("http", namespace, dcName, podName)] = masterPeerClient
}

func (fmc *FakeMasterControl) GetMasterClient(namespace string, dcName string, tlsEnabled bool) MasterClient {
	return fmc.defaultMasterControl.GetMasterClient(namespace, dcName, tlsEnabled)
}

func (fmc *FakeMasterControl) GetMasterPeerClient(namespace, dcName, podName string, tlsEnabled bool) MasterClient {
	return fmc.masterPeerClients[masterPeerClientKey("http", namespace, dcName, podName)]
}
