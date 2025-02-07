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

package tiflash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestProxyValidate(t *testing.T) {
	cfgValid := &ProxyConfig{}
	err := cfgValid.Validate()
	require.NoError(t, err)

	cfgInvalid := &ProxyConfig{
		LogLevel: "info",
		Server: ProxyServer{
			StatusAddr: "[::]:20292",
		},
		Security: ProxySecurity{
			CAPath:   "/path/to/ca",
			CertPath: "/path/to/cert",
			KeyPath:  "/path/to/key",
		},
	}

	err = cfgInvalid.Validate()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "log-level") // can be set by the user
	assert.Contains(t, err.Error(), "server.status-addr")
	assert.Contains(t, err.Error(), "security.ca-path")
	assert.Contains(t, err.Error(), "security.cert-path")
	assert.Contains(t, err.Error(), "security.key-path")
}

func TestProxyOverlay(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "https://basic-pd.ns1:2379",
		},
	}
	tiflash := &v1alpha1.TiFlash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "basic-0",
		},
	}

	cfg := &ProxyConfig{
		LogLevel: "debug",
	}
	err := cfg.Overlay(cluster, tiflash)
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "[::]:20292", cfg.Server.StatusAddr)
	assert.Equal(t, "/var/lib/tiflash-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/tiflash-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/tiflash-tls/tls.key", cfg.Security.KeyPath)
}
