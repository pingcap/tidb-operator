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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	certutil "github.com/pingcap/tidb-operator/pkg/util/crypto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	glog "k8s.io/klog"
)

// SecretControlInterface manages certificates used by TiDB clusters
type SecretControlInterface interface {
	Create(tc *v1alpha1.TidbCluster, certOpts *TiDBClusterCertOptions, cert []byte, key []byte) error
	Load(ns string, secretName string) ([]byte, []byte, error)
	Check(ns string, secretName string) bool
}

type realSecretControl struct {
	kubeCli      kubernetes.Interface
	secretLister corelisters.SecretLister
}

// NewRealSecretControl creates a new SecretControlInterface
func NewRealSecretControl(
	kubeCli kubernetes.Interface,
	secretLister corelisters.SecretLister,
) SecretControlInterface {
	return &realSecretControl{
		kubeCli:      kubeCli,
		secretLister: secretLister,
	}
}

func (rsc *realSecretControl) Create(tc *v1alpha1.TidbCluster, certOpts *TiDBClusterCertOptions, cert []byte, key []byte) error {
	secretName := fmt.Sprintf("%s-%s", certOpts.Instance, certOpts.Suffix)

	secretLabel := label.New().Instance(certOpts.Instance).
		Component(certOpts.Component).Labels()

	secret := &corev1.Secret{
		ObjectMeta: types.ObjectMeta{
			Name:            secretName,
			Labels:          secretLabel,
			OwnerReferences: []metav1.OwnerReference{GetOwnerRef(tc)},
		},
		Data: map[string][]byte{
			"cert": cert,
			"key":  key,
		},
	}

	_, err := rsc.kubeCli.CoreV1().Secrets(certOpts.Namespace).Create(secret)
	if err == nil {
		glog.Infof("save cert to secret %s/%s", certOpts.Namespace, secretName)
	}
	return err
}

// Load loads cert and key from Secret matching the name
func (rsc *realSecretControl) Load(ns string, secretName string) ([]byte, []byte, error) {
	secret, err := rsc.secretLister.Secrets(ns).Get(secretName)
	if err != nil {
		return nil, nil, err
	}

	return secret.Data["cert"], secret.Data["key"], nil
}

// Check returns true if the secret already exist
func (rsc *realSecretControl) Check(ns string, secretName string) bool {
	certBytes, keyBytes, err := rsc.Load(ns, secretName)
	if err != nil {
		glog.Errorf("certificate validation failed for [%s/%s], error loading cert from secret, %v", ns, secretName, err)
		return false
	}

	// validate if the certificate is valid
	block, _ := pem.Decode(certBytes)
	if block == nil {
		glog.Errorf("certificate validation failed for [%s/%s], can not decode cert to PEM", ns, secretName)
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		glog.Errorf("certificate validation failed for [%s/%s], can not parse cert, %v", ns, secretName, err)
		return false
	}
	rootCAs, err := certutil.ReadCACerts()
	if err != nil {
		glog.Errorf("certificate validation failed for [%s/%s], error loading CAs, %v", ns, secretName, err)
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
		glog.Errorf("certificate validation failed for [%s/%s], %v", ns, secretName, err)
		return false
	}

	// validate if the certificate and private key matches
	_, err = tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		glog.Errorf("certificate validation failed for [%s/%s], error loading key pair, %v", ns, secretName, err)
		return false
	}

	return true
}

var _ SecretControlInterface = &realSecretControl{}

type FakeSecretControl struct {
	realSecretControl
}

func NewFakeSecretControl(
	kubeCli kubernetes.Interface,
	secretLister corelisters.SecretLister,
) SecretControlInterface {
	return &realSecretControl{
		kubeCli:      kubeCli,
		secretLister: secretLister,
	}
}
