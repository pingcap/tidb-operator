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

package apicall

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func GetClientTLSConfig[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](ctx context.Context, c client.Client, obj F) (*tls.Config, error) {
	caName := coreutil.ClientCASecretName[S](obj)
	certKeyPairName := coreutil.ClientCertKeyPairSecretName[S](obj)
	skipVerify := scope.From[S](obj).ClientInsecureSkipTLSVerify()

	rootCAs := x509.NewCertPool()
	if !skipVerify {
		ca, err := getCA(ctx, c, obj.GetNamespace(), caName)
		if err != nil {
			return nil, err
		}
		if !rootCAs.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to append ca certs")
		}
	}
	certKeyPair, err := getCertKeyPair(ctx, c, obj.GetNamespace(), certKeyPairName)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:   rootCAs,
		ClientCAs: rootCAs,
		// nolint: gosec // user specified
		InsecureSkipVerify: skipVerify,
		Certificates:       []tls.Certificate{*certKeyPair},
	}, nil
}

func getCA(ctx context.Context, c client.Client, ns, name string) ([]byte, error) {
	var secret corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", ns, name, err)
	}

	ca, ok := secret.Data[corev1.ServiceAccountRootCAKey]
	if !ok {
		return nil, fmt.Errorf("cannot find %s in secret %s/%s", corev1.ServiceAccountRootCAKey, ns, name)
	}
	return ca, nil
}

func getCertKeyPair(ctx context.Context, c client.Client, ns, name string) (*tls.Certificate, error) {
	var secret corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", ns, name, err)
	}

	cert, certExists := secret.Data[corev1.TLSCertKey]
	key, keyExists := secret.Data[corev1.TLSPrivateKeyKey]
	if !certExists || !keyExists {
		return nil, fmt.Errorf("cert or key does not exist in the secret %s/%s", secret.Namespace, secret.Name)
	}
	certKeyPair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from the secret %s/%s: %w", secret.Namespace, secret.Name, err)
	}
	return &certKeyPair, nil
}
