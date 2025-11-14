// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tlsutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

// GetTLSConfigFromSecret returns *tls.Config for the given secret.
func GetTLSConfigFromSecret(ctx context.Context, cli client.Client, namespace, secretName string) (*tls.Config, error) {
	var secret corev1.Secret
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}
	return LoadTLSConfigFromSecret(&secret)
}

// LoadTLSConfigFromSecret loads *tls.Config from the given secret.
// The secret should often contain the following keys:
// - "ca.crt": CA certificate
// - "tls.crt": TLS certificate
// - "tls.key": TLS private key
func LoadTLSConfigFromSecret(secret *corev1.Secret) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()

	if !rootCAs.AppendCertsFromPEM(secret.Data[corev1.ServiceAccountRootCAKey]) {
		return nil, fmt.Errorf("failed to append ca certs")
	}

	clientCert, certExists := secret.Data[corev1.TLSCertKey]
	clientKey, keyExists := secret.Data[corev1.TLSPrivateKeyKey]
	if !certExists || !keyExists {
		return nil, fmt.Errorf("cert or key does not exist in the secret %s/%s", secret.Namespace, secret.Name)
	}
	tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from the secret %s/%s: %w", secret.Namespace, secret.Name, err)
	}

	//nolint:gosec // we didn't force to use a specific TLS version yet
	return &tls.Config{
		RootCAs:      rootCAs,
		ClientCAs:    rootCAs,
		Certificates: []tls.Certificate{tlsCert},
	}, nil
}
