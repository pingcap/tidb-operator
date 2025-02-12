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

package tidbapi

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
	defaultTiDBClientTimeout = 5 * time.Second
	defaultTiDBStatusPort    = 10080
)

// IiDBControlInterface is the interface that knows how to manage and get client for TiDB.
type TiDBControlInterface interface {
	// GetTiDBPodClient provides TiDBClient of a TiDB pod.
	GetTiDBPodClient(ctx context.Context, cli client.Client,
		namespace, tcName, podName, clusterDomain string, tlsEnabled bool) (TiDBClient, error)
}

// defaultTiDBControl is the default implementation of TiDBControlInterface.
type defaultTiDBControl struct{}

// NewDefaultTiDBControl returns a defaultTiDBControl instance.
func NewDefaultTiDBControl() TiDBControlInterface {
	return &defaultTiDBControl{}
}

// GetTiDBPodClient provides TiDBClient of a TiDB pod.
// NOTE: add cache for TiDBClient if necessary.
func (*defaultTiDBControl) GetTiDBPodClient(ctx context.Context, cli client.Client,
	namespace, tcName, podName, clusterDomain string, tlsEnabled bool,
) (TiDBClient, error) {
	var tlsConfig *tls.Config
	var err error
	scheme := "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, cli, namespace, coreutil.TLSClusterClientSecretName(tcName))
		if err != nil {
			return NewTiDBClient(TiDBPodClientURL(namespace, tcName, podName, clusterDomain, scheme), defaultTiDBClientTimeout, tlsConfig),
				fmt.Errorf("unable to get tls config for TiDB cluster %q, tidb client may not work: %w", tcName, err)
		}
		return NewTiDBClient(TiDBPodClientURL(namespace, tcName, podName, clusterDomain, scheme), defaultTiDBClientTimeout, tlsConfig), nil
	}

	return NewTiDBClient(TiDBPodClientURL(namespace, tcName, podName, clusterDomain, scheme), defaultTiDBClientTimeout, tlsConfig), nil
}

// TiDBPodClientURL builds the URL of a tidb pod client.
func TiDBPodClientURL(namespace, clusterName, podName, clusterDomain, scheme string) string {
	if clusterDomain != "" {
		return fmt.Sprintf("%s://%s.%s-tidb-peer.%s.svc.%s:%d",
			scheme, podName, clusterName, namespace, clusterDomain, defaultTiDBStatusPort)
	}
	return fmt.Sprintf("%s://%s.%s-tidb-peer.%s:%d",
		scheme, podName, clusterName, namespace, defaultTiDBStatusPort)
}
