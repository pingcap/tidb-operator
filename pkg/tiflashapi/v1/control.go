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

package tiflashapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tlsutil "github.com/pingcap/tidb-operator/pkg/utils/tls"
)

const (
	defaultTiFlashClientTimeout   = 5 * time.Second
	defaultTiFlashProxyStatusPort = 20292
)

// TiFlashControlInterface is an interface that knows how to manage and get client for TiFlash.
type TiFlashControlInterface interface {
	// GetTiFlashPodClient provides TiFlashClient of a TiFlash pod.
	GetTiFlashPodClient(ctx context.Context, cli client.Client,
		namespace, tcName, podName string, tlsEnabled bool) (TiFlashClient, error)
}

// defaultTiFlashControl is the default implementation of TiFlashControlInterface.
type defaultTiFlashControl struct {
}

// NewDefaultTiFlashControl returns a defaultTiFlashControl instance
func NewDefaultTiFlashControl() TiFlashControlInterface {
	return &defaultTiFlashControl{}
}

// GetTiFlashPodClient provides TiFlashClient of a TiFlash pod.
// NOTE: add cache for TiFlashClient if necessary.
func (*defaultTiFlashControl) GetTiFlashPodClient(ctx context.Context, cli client.Client,
	namespace, tcName, podName string, tlsEnabled bool,
) (TiFlashClient, error) {
	var tlsConfig *tls.Config
	var err error
	scheme := "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, cli, namespace, v1alpha1.TLSClusterClientSecretName(tcName))
		if err != nil {
			return NewTiFlashClient(TiFlashPodClientURL(namespace, tcName, podName, scheme), defaultTiFlashClientTimeout, tlsConfig, true),
				fmt.Errorf("unable to get tls config for TiDB cluster %q, tiflash client may not work: %w", tcName, err)
		}
		return NewTiFlashClient(TiFlashPodClientURL(namespace, tcName, podName, scheme), defaultTiFlashClientTimeout, tlsConfig, true), nil
	}

	return NewTiFlashClient(TiFlashPodClientURL(namespace, tcName, podName, scheme), defaultTiFlashClientTimeout, tlsConfig, true), nil
}

// TiFlashPodClientURL builds the URL of a TiFlash pod client.
func TiFlashPodClientURL(namespace, clusterName, podName, scheme string) string {
	return fmt.Sprintf("%s://%s.%s-tiflash-peer.%s:%d",
		scheme, podName, clusterName, namespace, defaultTiFlashProxyStatusPort)
}
