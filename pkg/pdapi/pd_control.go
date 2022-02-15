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

// ClusterRef sets the cluster domain of TC, it is used when generating the client address from TC.
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

// SpecifyClient specify client addr without generating
func SpecifyClient(clientURL, clientKey string) Option {
	return func(c *clientConfig) {
		c.clientURL = clientURL
		c.clientKey = clientKey
	}
}

// UseHeadlessService indicates that the clients use headless service to connect to PD.
func UseHeadlessService(headless bool) Option {
	return func(c *clientConfig) {
		c.headlessSvc = headless
	}
}

// PDControlInterface is an interface that knows how to manage and get tidb cluster's PD client
type PDControlInterface interface {
	// GetPDClient provides PDClient of the tidb cluster.
	GetPDClient(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) PDClient
	// GetPDEtcdClient provides PD etcd Client of the tidb cluster.
	GetPDEtcdClient(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) (PDEtcdClient, error)
	// GetEndpoints return the endpoints and client tls.Config to connection pd/etcd.
	GetEndpoints(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) (endpoints []string, tlsConfig *tls.Config, err error)
}

type clientConfig struct {
	clusterDomain string
	headlessSvc   bool // use headless service to connect, default to use service

	// clientURL is PD/Etcd addr. If it is empty, will generate from target TC
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

// completeForPDClient populate and correct config for pd client
func (c *clientConfig) completeForPDClient(namespace Namespace, tcName string) {
	scheme := "http"
	if c.tlsEnable {
		scheme = "https"
		if c.tlsSecretName == "" {
			c.tlsSecretNamespace = namespace
			c.tlsSecretName = util.ClusterClientTLSSecretName(tcName)
		}
	}

	if c.clientURL == "" {
		c.clientURL = genClientUrl(namespace, tcName, scheme, c.clusterDomain, c.headlessSvc)
	}
	if c.clientKey == "" {
		c.clientKey = genClientKey(scheme, namespace, tcName, c.clusterDomain)
	}
}

// completeForEtcdClient populate and correct config for pd etcd client
func (c *clientConfig) completeForEtcdClient(namespace Namespace, tcName string) {
	if c.tlsEnable {
		if c.tlsSecretName == "" {
			c.tlsSecretNamespace = namespace
			c.tlsSecretName = util.ClusterClientTLSSecretName(tcName)
		}
	}

	if c.clientURL == "" {
		c.clientURL = genEtcdClientUrl(namespace, tcName, c.clusterDomain, c.headlessSvc)
	}
	if c.clientKey == "" {
		c.clientKey = genEtcdClientKey(namespace, tcName, c.clusterDomain, c.tlsEnable)
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

func (pdc *defaultPDControl) GetEndpoints(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) (endpoints []string, tlsConfig *tls.Config, err error) {
	config := &clientConfig{}

	config.tlsEnable = tlsEnabled
	config.applyOptions(opts...)

	config.completeForEtcdClient(namespace, tcName)

	if config.tlsEnable {
		tlsConfig, err = GetTLSConfig(pdc.secretLister, config.tlsSecretNamespace, config.tlsSecretName)
		if err != nil {
			return nil, nil, err
		}
	}

	endpoints = []string{config.clientURL}

	return
}

func (pdc *defaultPDControl) GetPDEtcdClient(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) (PDEtcdClient, error) {
	config := &clientConfig{}

	config.tlsEnable = tlsEnabled
	config.applyOptions(opts...)

	config.completeForEtcdClient(namespace, tcName)

	pdc.etcdmutex.Lock()
	defer pdc.etcdmutex.Unlock()

	if config.tlsEnable {
		tlsConfig, err := GetTLSConfig(pdc.secretLister, config.tlsSecretNamespace, config.tlsSecretName)
		if err != nil {
			klog.Errorf("Unable to get tls config for tidb cluster %q in %s, pd client may not work: %v", tcName, namespace, err)
			return nil, err
		}
		return NewPdEtcdClient(config.clientURL, DefaultTimeout, tlsConfig)
	}

	if _, ok := pdc.pdEtcdClients[config.clientKey]; !ok {
		etcdCli, err := NewPdEtcdClient(config.clientURL, DefaultTimeout, nil)
		if err != nil {
			return nil, err
		}
		pdc.pdEtcdClients[config.clientKey] = &noOpClose{PDEtcdClient: etcdCli}
	}
	return pdc.pdEtcdClients[config.clientKey], nil
}

// GetPDClient provides a PDClient of real pd cluster, if the PDClient not existing, it will create new one.
func (pdc *defaultPDControl) GetPDClient(namespace Namespace, tcName string, tlsEnabled bool, opts ...Option) PDClient {
	config := &clientConfig{}

	config.tlsEnable = tlsEnabled
	config.applyOptions(opts...)

	config.completeForPDClient(namespace, tcName)

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

func genEtcdClientKey(namespace Namespace, clusterName string, clusterDomain string, tlsEnabled bool) string {
	if len(clusterDomain) == 0 {
		return fmt.Sprintf("%s.%s.%v", clusterName, string(namespace), tlsEnabled)
	}
	return fmt.Sprintf("%s.%s.%s.%v", clusterName, string(namespace), clusterDomain, tlsEnabled)
}

// genClientUrl builds the url of cluster pd client
func genClientUrl(namespace Namespace, clusterName string, scheme string, clusterDomain string, headlessSvc bool) string {
	svc := "pd"
	if headlessSvc {
		svc = "pd-peer"
	}
	if len(namespace) == 0 {
		return fmt.Sprintf("%s://%s-%s:2379", scheme, clusterName, svc)
	}
	if len(clusterDomain) == 0 {
		return fmt.Sprintf("%s://%s-%s.%s:2379", scheme, clusterName, svc, string(namespace))
	}
	return fmt.Sprintf("%s://%s-%s.%s.svc.%s:2379", scheme, clusterName, svc, string(namespace), clusterDomain)
}

// genEtcdClientUrl builds the url of cluster pd etcd client
func genEtcdClientUrl(namespace Namespace, clusterName, clusterDomain string, headlessSvc bool) string {
	svc := "pd"
	if headlessSvc {
		svc = "pd-peer"
	}
	if clusterDomain == "" {
		return fmt.Sprintf("%s-%s.%s:2379", clusterName, svc, string(namespace))
	}
	return fmt.Sprintf("%s-%s.%s.svc.%s:2379", clusterName, svc, string(namespace), clusterDomain)
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
