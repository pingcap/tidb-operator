// Copyright 2019 PingCAP, Inc.
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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"

	certutil "github.com/pingcap/tidb-operator/pkg/util/crypto"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

// SecretControlInterface manages certificates used by TiDB clusters
type SecretControlInterface interface {
	Load(ns string, secretName string) ([]byte, []byte, error)
	Check(ns string, secretName string) bool
	Create(ns string, secret *v1.Secret) error
}

type realSecretControl struct {
	kubeCli      kubernetes.Interface
	secretLister corelisterv1.SecretLister
	recorder     record.EventRecorder
}

// NewRealSecretControl creates a new SecretControlInterface
func NewRealSecretControl(
	kubeCli kubernetes.Interface, secretLister corelisterv1.SecretLister, recorder record.EventRecorder,
) SecretControlInterface {
	return &realSecretControl{
		kubeCli:      kubeCli,
		secretLister: secretLister,
		recorder:     recorder,
	}
}

// Load loads cert and key from Secret matching the name
func (c *realSecretControl) Load(ns string, secretName string) ([]byte, []byte, error) {
	secret, err := c.secretLister.Secrets(ns).Get(secretName)
	if err != nil {
		return nil, nil, err
	}

	return secret.Data[v1.TLSCertKey], secret.Data[v1.TLSPrivateKeyKey], nil
}

// Check returns true if the secret already exist
func (c *realSecretControl) Check(ns string, secretName string) bool {
	certBytes, keyBytes, err := c.Load(ns, secretName)
	if err != nil {
		klog.Errorf("certificate validation failed for [%s/%s], error loading cert from secret, %v", ns, secretName, err)
		return false
	}

	// validate if the certificate is valid
	block, _ := pem.Decode(certBytes)
	if block == nil {
		klog.Errorf("certificate validation failed for [%s/%s], can not decode cert to PEM", ns, secretName)
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		klog.Errorf("certificate validation failed for [%s/%s], can not parse cert, %v", ns, secretName, err)
		return false
	}
	rootCAs, err := certutil.ReadCACerts()
	if err != nil {
		klog.Errorf("certificate validation failed for [%s/%s], error loading CAs, %v", ns, secretName, err)
		return false
	}

	verifyOpts := x509.VerifyOptions{
		Roots: rootCAs,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
	}
	_, err = cert.Verify(verifyOpts)
	if err != nil {
		klog.Errorf("certificate validation failed for [%s/%s], %v", ns, secretName, err)
		return false
	}

	// validate if the certificate and private key matches
	_, err = tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		klog.Errorf("certificate validation failed for [%s/%s], error loading key pair, %v", ns, secretName, err)
		return false
	}

	return true
}

// Create create secret
func (c *realSecretControl) Create(ns string, secret *v1.Secret) error {
	_, err := c.kubeCli.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

var _ SecretControlInterface = &realSecretControl{}

// FakeSecretControl is a fake SecretControlInterface
type FakeSecretControl struct {
	SecretLister        corelisterv1.SecretLister
	SecretIndexer       cache.Indexer
	createSecretTracker RequestTracker
}

// NewFakeSecretControl returns a FakeSecretControl
func NewFakeSecretControl(secretInformer coreinformers.SecretInformer) *FakeSecretControl {
	return &FakeSecretControl{
		secretInformer.Lister(),
		secretInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

func (c *FakeSecretControl) Create(ns string, secret *v1.Secret) error {
	defer c.createSecretTracker.Inc()
	if c.createSecretTracker.ErrorReady() {
		defer c.createSecretTracker.Reset()
		return c.createSecretTracker.GetError()
	}

	err := c.SecretIndexer.Add(secret)
	if err != nil {
		return err
	}
	return nil
}

func (c *FakeSecretControl) Load(ns string, secretName string) ([]byte, []byte, error) {

	return nil, nil, nil
}

func (c *FakeSecretControl) Check(ns string, secretName string) bool {
	return true
}
