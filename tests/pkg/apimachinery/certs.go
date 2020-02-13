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

package apimachinery

import (
	"crypto/x509"
	"io/ioutil"
	"os"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

type CertContext struct {
	Cert        []byte
	Key         []byte
	SigningCert []byte
}

// Setup the server cert. For example, user apiservers and admission webhooks
// can use the cert to prove their identify to the kube-apiserver
func SetupServerCert(namespaceName, serviceName string) (*CertContext, error) {
	certDir, err := ioutil.TempDir("", "test-e2e-server-cert")
	if err != nil {
		klog.Errorf("Failed to create a temp dir for cert generation %v", err)
		return nil, err
	}
	defer os.RemoveAll(certDir)
	signingKey, err := pkiutil.NewPrivateKey()
	if err != nil {
		klog.Errorf("Failed to create CA private key %v", err)
		return nil, err
	}
	signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "e2e-server-cert-ca"}, signingKey)
	if err != nil {
		klog.Errorf("Failed to create CA cert for apiserver %v", err)
		return nil, err
	}
	caCertFile, err := ioutil.TempFile(certDir, "ca.crt")
	if err != nil {
		klog.Errorf("Failed to create a temp file for ca cert generation %v", err)
		return nil, err
	}
	if err := ioutil.WriteFile(caCertFile.Name(), pkiutil.EncodeCertPEM(signingCert), 0644); err != nil {
		klog.Errorf("Failed to write CA cert %v", err)
		return nil, err
	}
	key, err := pkiutil.NewPrivateKey()
	if err != nil {
		klog.Errorf("Failed to create private key for %v", err)
		return nil, err
	}
	signedCert, err := pkiutil.NewSignedCert(
		&cert.Config{
			CommonName: serviceName + "." + namespaceName + ".svc",
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		klog.Errorf("Failed to create cert%v", err)
		return nil, err
	}
	certFile, err := ioutil.TempFile(certDir, "server.crt")
	if err != nil {
		klog.Errorf("Failed to create a temp file for cert generation %v", err)
		return nil, err
	}
	keyFile, err := ioutil.TempFile(certDir, "server.key")
	if err != nil {
		klog.Errorf("Failed to create a temp file for key generation %v", err)
		return nil, err
	}
	if err = ioutil.WriteFile(certFile.Name(), pkiutil.EncodeCertPEM(signedCert), 0600); err != nil {
		klog.Errorf("Failed to write cert file %v", err)
		return nil, err
	}
	keyPEM, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return nil, err
	}
	if err = ioutil.WriteFile(keyFile.Name(), keyPEM, 0644); err != nil {
		klog.Errorf("Failed to write key file %v", err)
		return nil, err
	}
	return &CertContext{
		Cert:        pkiutil.EncodeCertPEM(signedCert),
		Key:         keyPEM,
		SigningCert: pkiutil.EncodeCertPEM(signingCert),
	}, nil
}
