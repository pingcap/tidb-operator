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
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

// Namespace is a newtype of a string
type Namespace string

// Option configures the PDClient
type Option func(c *clientConfig)

// ClusterRef sets the cluster domain of TC, it is used when generating the PD address from TC.
func ClusterRef(clusterDomain string) Option {
	return func(c *clientConfig) {
		c.clusterDomain = clusterDomain
	}
}

// TLSCertFromTC indicates that the clients use certs from specified TC's secret.
func TLSCertFromTC(ns Namespace, tcName string) Option {
	return func(c *clientConfig) {
		c.tlsSecretNamespace = ns
		c.tlsSecretName = util.ClusterClientTLSSecretName(tcName)
	}
}

// TLSCertFromTC indicates that clients use certs from specified secret.
func TLSCertFromSecret(ns Namespace, secret string) Option {
	return func(c *clientConfig) {
		c.tlsSecretNamespace = ns
		c.tlsSecretName = secret
	}
}

// SpecifyClient specify PD addr without generating
func SpecifyClient(clientURL, clientKey string) Option {
	return func(c *clientConfig) {
		c.clientURL = clientURL
		c.clientKey = clientKey
	}
}

// PDControlInterface is an interface that knows how to manage and get tidb cluster's PD client
type PDControlInterface interface {
	// GetPDClient provides PDClient of the tidb cluster.
	GetPDClient(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) PDClient
	// GetPDEtcdClient provides PD etcd Client of the tidb cluster.
	GetPDEtcdClient(namespace Namespace, tcName string, tlsEnabled bool) (PDEtcdClient, error)
	// GetEndpoints return the endpoints and client tls.Config to connection pd/etcd.
	GetEndpoints(namespace Namespace, tcName string, tlsEnabled bool) (endpoints []string, tlsConfig *tls.Config, err error)
}

type clientConfig struct {
	clusterDomain string

	// clientURL is PD addr. If it is empty, will generate from target TC
	clientURL string
	// clientKey is client name. If it is empty, will generate from target TC
	clientKey string

	tlsEnable          bool
	tlsSecretNamespace Namespace
	tlsSecretName      string
}

func (c *clientConfig) applyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// complete populate and correct config
func (c *clientConfig) complete(namespace Namespace, tcName string) {
	scheme := "http"
	if c.tlsEnable {
		scheme = "https"
		if c.tlsSecretName == "" {
			c.tlsSecretNamespace = namespace
			c.tlsSecretName = util.ClusterClientTLSSecretName(tcName)
		}
	}

	if c.clientURL == "" {
		c.clientURL = genClientUrl(namespace, tcName, scheme, c.clusterDomain)
	}
	if c.clientKey == "" {
		c.clientKey = genClientKey(scheme, namespace, tcName, c.clusterDomain)
	}
}

// defaultPDControl is the default implementation of PDControlInterface.
type defaultPDControl struct {
	secretLister corelisterv1.SecretLister

	mutex     sync.Mutex
	pdClients map[string]PDClient

	etcdmutex     sync.Mutex
	pdEtcdClients map[string]PDEtcdClient
}

type noOpClose struct {
	PDEtcdClient
}

func (c *noOpClose) Close() error {
	return nil
}

// NewDefaultPDControl returns a defaultPDControl instance
func NewDefaultPDControl(secretLister corelisterv1.SecretLister) PDControlInterface {
	return &defaultPDControl{secretLister: secretLister, pdClients: map[string]PDClient{}, pdEtcdClients: map[string]PDEtcdClient{}}
}

// NewDefaultPDControl returns a defaultPDControl instance
func NewDefaultPDControlByCli(kubeCli kubernetes.Interface) PDControlInterface {
	return &defaultPDControl{pdClients: map[string]PDClient{}, pdEtcdClients: map[string]PDEtcdClient{}}
}

func (c *defaultPDControl) GetEndpoints(namespace Namespace, tcName string, tlsEnabled bool) (endpoints []string, tlsConfig *tls.Config, err error) {
	if tlsEnabled {
		tlsConfig, err = GetTLSConfig(c.secretLister, namespace, util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			return nil, nil, err
		}
	}

	endpoints = []string{PDEtcdClientURL(namespace, tcName)}

	return
}

func (c *defaultPDControl) GetPDEtcdClient(namespace Namespace, tcName string, tlsEnabled bool) (PDEtcdClient, error) {
	c.etcdmutex.Lock()
	defer c.etcdmutex.Unlock()

	var tlsConfig *tls.Config
	var err error

	if tlsEnabled {
		tlsConfig, err = GetTLSConfig(c.secretLister, namespace, util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			klog.Errorf("Unable to get tls config for tidb cluster %q, pd etcd client may not work: %v", tcName, err)
			return nil, err
		}
	}

	key := pdEtcdClientKey(namespace, tcName, tlsEnabled)
	if _, ok := c.pdEtcdClients[key]; !ok {
		pdetcdClient, err := NewPdEtcdClient(PDEtcdClientURL(namespace, tcName), DefaultTimeout, tlsConfig)
		if err != nil {
			return nil, err
		}

		// For the current behavior to create a new client each time,
		// it's to cover the case that the secret containing the certs may be rotated to avoid expiration,
		// in which case, the client may fail and we have to restart the tidb-controller-manager container
		if tlsEnabled {
			return pdetcdClient, nil
		}

		c.pdEtcdClients[key] = &noOpClose{PDEtcdClient: pdetcdClient}
	}

	return c.pdEtcdClients[key], nil
}

// GetPDClient provides a PDClient of real pd cluster, if the PDClient not existing, it will create new one.
func (pdc *defaultPDControl) GetPDClient(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) PDClient {
	config := &clientConfig{}

	config.tlsEnable = tlsEnabled
	config.applyOptions(opts...)

	config.complete(namespace, tcName)

	pdc.mutex.Lock()
	defer pdc.mutex.Unlock()

	if config.tlsEnable {
		tlsConfig, err := GetTLSConfig(pdc.secretLister, config.tlsSecretNamespace, config.tlsSecretName)
		if err != nil {
			klog.Errorf("Unable to get tls config for tidb cluster %q in %s, pd client may not work: %v", tcName, namespace, err)
			return &pdClient{url: config.clientURL, httpClient: &http.Client{Timeout: DefaultTimeout}}
		}

		return NewPDClient(config.clientURL, DefaultTimeout, tlsConfig)
	}
	if _, ok := pdc.pdClients[config.clientKey]; !ok {
		pdc.pdClients[config.clientKey] = NewPDClient(config.clientURL, DefaultTimeout, nil)
	}
	return pdc.pdClients[config.clientKey]
}

func genClientKey(scheme string, namespace Namespace, clusterName string, clusterDomain string) string {
	if len(clusterDomain) == 0 {
		return fmt.Sprintf("%s.%s.%s", scheme, clusterName, string(namespace))
	}
	return fmt.Sprintf("%s.%s.%s.%s", scheme, clusterName, string(namespace), clusterDomain)
}

func pdEtcdClientKey(namespace Namespace, clusterName string, tlsEnabled bool) string {
	return fmt.Sprintf("%s.%s.%v", clusterName, string(namespace), tlsEnabled)
}

// genClientUrl builds the url of cluster pd client
func genClientUrl(namespace Namespace, clusterName string, scheme string, clusterDomain string) string {
	if len(namespace) == 0 {
		return fmt.Sprintf("%s://%s-pd:2379", scheme, clusterName)
	}
	if len(clusterDomain) == 0 {
		return fmt.Sprintf("%s://%s-pd.%s:2379", scheme, clusterName, string(namespace))
	}
	return fmt.Sprintf("%s://%s-pd-peer.%s.svc.%s:2379", scheme, clusterName, string(namespace), clusterDomain)
}

func PDEtcdClientURL(namespace Namespace, clusterName string) string {
	return fmt.Sprintf("%s-pd.%s:2379", clusterName, string(namespace))
}

// FakePDControl implements a fake version of PDControlInterface.
type FakePDControl struct {
	defaultPDControl
}

func NewFakePDControl(secretLister corelisterv1.SecretLister) *FakePDControl {
	return &FakePDControl{
		defaultPDControl{secretLister: secretLister, pdClients: map[string]PDClient{}},
	}
}

func (fpc *FakePDControl) SetPDClient(namespace Namespace, tcName string, pdclient PDClient) {
	fpc.defaultPDControl.pdClients[genClientKey("http", namespace, tcName, "")] = pdclient
}

func (fpc *FakePDControl) SetPDClientWithClusterDomain(namespace Namespace, tcName string, tcClusterDomain string, pdclient PDClient) {
	fpc.defaultPDControl.pdClients[genClientKey("http", namespace, tcName, tcClusterDomain)] = pdclient
}

func (fpc *FakePDControl) SetPDClientWithAddress(peerURL string, pdclient PDClient) {
	fpc.defaultPDControl.pdClients[peerURL] = pdclient
}
