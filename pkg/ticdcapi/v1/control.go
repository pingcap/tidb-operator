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

package ticdcapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tlsutil "github.com/pingcap/tidb-operator/pkg/utils/tls"
)

var (
	defaultTiCDCClientTimeout = 5 * time.Second
	// TODO: customize the port
	defaultTiCDCStatusPort = 8300
)

// IiDBControlInterface is the interface that knows how to manage and get client for TiCDC.
type TiCDCControlInterface interface {
	// GetTiCDCPodClient provides TiCDCClient of a TiCDC pod.
	GetTiCDCPodClient(ctx context.Context, cli client.Client,
		namespace, tcName, podName, clusterDomain string, tlsEnabled bool) (TiCDCClient, error)
}

// defaultTiCDCControl is the default implementation of TiCDCControlInterface.
type defaultTiCDCControl struct{}

// NewDefaultTiCDCControl returns a defaultTiCDCControl instance.
func NewDefaultTiCDCControl() TiCDCControlInterface {
	return &defaultTiCDCControl{}
}

// GetTiCDCPodClient provides TiCDCClient of a TiCDC pod.
// NOTE: add cache for TiCDCClient if necessary.
func (*defaultTiCDCControl) GetTiCDCPodClient(ctx context.Context, cli client.Client,
	namespace, tcName, podName, clusterDomain string, tlsEnabled bool,
) (TiCDCClient, error) {
	var tlsConfig *tls.Config
	var err error
	scheme := "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, cli, namespace, coreutil.TLSClusterClientSecretName(tcName))
		if err != nil {
			return NewTiCDCClient(TiCDCPodClientURL(namespace, tcName, podName, clusterDomain, scheme),
					podName, defaultTiCDCClientTimeout, tlsConfig),
				fmt.Errorf("unable to get tls config for TiCDC cluster %q, ticdc client may not work: %w", tcName, err)
		}
		return NewTiCDCClient(TiCDCPodClientURL(namespace, tcName, podName, clusterDomain, scheme),
			podName, defaultTiCDCClientTimeout, tlsConfig), nil
	}

	return NewTiCDCClient(TiCDCPodClientURL(namespace, tcName, podName, clusterDomain, scheme),
		podName, defaultTiCDCClientTimeout, tlsConfig), nil
}

// TiCDCPodClientURL builds the URL of a TiCDC pod client.
func TiCDCPodClientURL(namespace, clusterName, podName, clusterDomain, scheme string) string {
	if clusterDomain != "" {
		return fmt.Sprintf("%s://%s.%s-ticdc-peer.%s.svc.%s:%d",
			scheme, podName, clusterName, namespace, clusterDomain, defaultTiCDCStatusPort)
	}
	return fmt.Sprintf("%s://%s.%s-ticdc-peer.%s:%d",
		scheme, podName, clusterName, namespace, defaultTiCDCStatusPort)
}
