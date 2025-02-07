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

package tikvapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tlsutil "github.com/pingcap/tidb-operator/pkg/utils/tls"
)

const (
	defaultTiKVClientTimeout = 5 * time.Second
	defaultTiKVStatusPort    = 20180
)

// TiKVControlInterface is an interface that knows how to manage and get client for TiKV.
type TiKVControlInterface interface {
	// GetTiKVPodClient provides TiKVClient of a TiKV pod.
	GetTiKVPodClient(ctx context.Context, cli client.Client,
		namespace, tcName, podName, clusterDomain string, tlsEnabled bool) (TiKVClient, error)
}

// defaultTiKVControl is the default implementation of TiKVControlInterface.
type defaultTiKVControl struct {
}

// NewDefaultTiKVControl returns a defaultTiKVControl instance.
func NewDefaultTiKVControl() TiKVControlInterface {
	return &defaultTiKVControl{}
}

// GetTiKVPodClient provides TiKVClient of a TiKV pod.
// NOTE: add cache for TiKVClient if necessary.
func (*defaultTiKVControl) GetTiKVPodClient(ctx context.Context, cli client.Client,
	namespace, tcName, podName, clusterDomain string, tlsEnabled bool,
) (TiKVClient, error) {
	var tlsConfig *tls.Config
	var err error
	scheme := "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, cli, namespace, v1alpha1.TLSClusterClientSecretName(tcName))
		if err != nil {
			return NewTiKVClient(TiKVPodClientURL(namespace, tcName, podName, scheme, clusterDomain), defaultTiKVClientTimeout, tlsConfig, true),
				fmt.Errorf("unable to get tls config for TiDB cluster %q, tikv client may not work: %w", tcName, err)
		}
		return NewTiKVClient(TiKVPodClientURL(namespace, tcName, podName, scheme, clusterDomain), defaultTiKVClientTimeout, tlsConfig, true), nil
	}

	return NewTiKVClient(TiKVPodClientURL(namespace, tcName, podName, scheme, clusterDomain), defaultTiKVClientTimeout, tlsConfig, true), nil
}

// TiKVPodClientURL builds the URL of a tikv pod client.
func TiKVPodClientURL(namespace, clusterName, podName, scheme, clusterDomain string) string {
	if clusterDomain != "" {
		return fmt.Sprintf("%s://%s.%s-tikv-peer.%s.svc.%s:%d",
			scheme, podName, clusterName, namespace, clusterDomain, defaultTiKVStatusPort)
	}
	return fmt.Sprintf("%s://%s.%s-tikv-peer.%s:%d",
		scheme, podName, clusterName, namespace, defaultTiKVStatusPort)
}
