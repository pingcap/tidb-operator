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

package tiproxyapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tlsutil "github.com/pingcap/tidb-operator/pkg/utils/tls"
)

var (
	defaultTiProxyClientTimeout = 5 * time.Second
)

// TiProxyControlInterface is the interface that knows how to manage and get client for TiProxy.
type TiProxyControlInterface interface {
	// GetTiProxyPodClient provides TiProxyClient of a TiProxy pod.
	GetTiProxyPodClient(ctx context.Context, cli client.Client,
		namespace, tcName, podName, clusterDomain string, tlsEnabled bool) (TiProxyClient, error)
}

// defaultTiProxyControl is the default implementation of TiProxyControlInterface.
type defaultTiProxyControl struct{}

// NewDefaultTiProxyControl returns a defaultTiProxyControl instance.
func NewDefaultTiProxyControl() TiProxyControlInterface {
	return &defaultTiProxyControl{}
}

// GetTiProxyPodClient provides TiProxyClient of a TiProxy pod.
// NOTE: add cache for TiProxyClient if necessary.
func (*defaultTiProxyControl) GetTiProxyPodClient(ctx context.Context, cli client.Client,
	namespace, tcName, podName, clusterDomain string, tlsEnabled bool,
) (TiProxyClient, error) {
	var tlsConfig *tls.Config
	var err error
	scheme := "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, cli, namespace, coreutil.TLSClusterClientSecretName(tcName))
		if err != nil {
			return NewTiProxyClient(
					TiProxyPodClientURL(namespace, tcName, podName, clusterDomain, scheme),
					defaultTiProxyClientTimeout, tlsConfig),
				fmt.Errorf("unable to get tls config for TiProxy cluster %q, tiproxy client may not work: %w", tcName, err)
		}
		return NewTiProxyClient(
			TiProxyPodClientURL(namespace, tcName, podName, clusterDomain, scheme),
			defaultTiProxyClientTimeout, tlsConfig), nil
	}

	return NewTiProxyClient(
		TiProxyPodClientURL(namespace, tcName, podName, clusterDomain, scheme),
		defaultTiProxyClientTimeout, tlsConfig), nil
}

// TiProxyPodClientURL builds the URL of a tiproxy pod client.
func TiProxyPodClientURL(namespace, clusterName, podName, clusterDomain, scheme string) string {
	if clusterDomain != "" {
		return fmt.Sprintf("%s://%s.%s-tiproxy-peer.%s.svc.%s:%d",
			scheme, podName, clusterName, namespace, clusterDomain, v1alpha1.DefaultTiProxyPortAPI)
	}
	return fmt.Sprintf("%s://%s.%s-tiproxy-peer.%s:%d",
		scheme, podName, clusterName, namespace, v1alpha1.DefaultTiProxyPortAPI)
}
