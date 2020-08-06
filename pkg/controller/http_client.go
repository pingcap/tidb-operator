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

package controller

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type httpClient struct {
	kubeCli kubernetes.Interface
}

func (hc *httpClient) getHTTPClient(tc *v1alpha1.TidbCluster) (*http.Client, error) {
	httpClient := &http.Client{Timeout: timeout}
	if !tc.IsTLSClusterEnabled() {
		return httpClient, nil
	}

	tcName := tc.Name
	ns := tc.Namespace
	secretName := util.ClusterClientTLSSecretName(tcName)
	secret, err := hc.kubeCli.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	clientCert, certExists := secret.Data[v1.TLSCertKey]
	clientKey, keyExists := secret.Data[v1.TLSPrivateKeyKey]
	if !certExists || !keyExists {
		return nil, fmt.Errorf("cert or key does not exist in secret %s/%s", ns, secretName)
	}

	tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", ns, secretName, err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(secret.Data[v1.ServiceAccountRootCAKey])
	config := &tls.Config{
		RootCAs:      rootCAs,
		Certificates: []tls.Certificate{tlsCert},
	}
	httpClient.Transport = &http.Transport{TLSClientConfig: config, DisableKeepAlives: true}

	return httpClient, nil
}
