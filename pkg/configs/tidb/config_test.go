// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tidb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func TestValidate(t *testing.T) {
	cfgValid := &Config{}
	err := cfgValid.Validate(true)
	require.NoError(t, err)

	cfgInvalid := &Config{
		Store:            "tikv",
		AdvertiseAddress: "basic-tidb-0.basic-tidb-peer.default.svc",
		Host:             "::",
		Path:             "basic-pd.default.svc",
		Security: Security{
			SSLCA:          "/path/to/ca",
			SSLCert:        "/path/to/cert",
			SSLKey:         "/path/to/key",
			ClusterSSLCA:   "/path/to/cluster-ca",
			ClusterSSLCert: "/path/to/cluster-cert",
			ClusterSSLKey:  "/path/to/cluster-key",
			AuthTokenJwks:  "/path/to/auth-token-jwks",
		},
		Log: Log{
			SlowQueryFile: "/path/to/slow-query-file",
		},
		InitializeSQLFile:          "/path/to/initialize-sql-file",
		GracefulWaitBeforeShutdown: 10,
	}

	err = cfgInvalid.Validate(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store")
	assert.Contains(t, err.Error(), "advertise-address")
	assert.Contains(t, err.Error(), "host")
	assert.Contains(t, err.Error(), "path")
	assert.Contains(t, err.Error(), "security.ssl-ca")
	assert.Contains(t, err.Error(), "security.ssl-cert")
	assert.Contains(t, err.Error(), "security.ssl-key")
	assert.Contains(t, err.Error(), "security.cluster-ssl-ca")
	assert.Contains(t, err.Error(), "security.cluster-ssl-cert")
	assert.Contains(t, err.Error(), "security.cluster-ssl-key")
	assert.Contains(t, err.Error(), "security.auth-token-jwks")
	assert.Contains(t, err.Error(), "log.slow-query-file")
	assert.Contains(t, err.Error(), "initialize-sql-file")
	assert.NotContains(t, err.Error(), "graceful-wait-before-shutdown") // can be set by the user
}

func TestOverlay(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "https://basic-pd.ns1:2379",
		},
	}
	tidb := &v1alpha1.TiDB{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "basic-0",
		},
		Spec: v1alpha1.TiDBSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "basic-tidb-peer",
			TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
				Security: &v1alpha1.TiDBSecurity{
					AuthToken: &v1alpha1.TiDBAuthToken{
						JWKs: corev1.LocalObjectReference{
							Name: "auth-token-jwks",
						},
					},
					BootstrapSQL: &corev1.LocalObjectReference{
						Name: "bootstrap-sql",
					},
					TLS: &v1alpha1.TiDBTLS{
						MySQL: &v1alpha1.TLS{
							Enabled: true,
						},
					},
				},
			},
		},
	}

	cfg := &Config{
		GracefulWaitBeforeShutdown: 100,
	}
	err := cfg.Overlay(cluster, tidb)
	require.NoError(t, err)
	assert.Equal(t, "tikv", cfg.Store)
	assert.Equal(t, "basic-tidb-0.basic-tidb-peer.ns1.svc", cfg.AdvertiseAddress)
	assert.Equal(t, "::", cfg.Host)
	assert.Equal(t, "basic-pd.ns1:2379", cfg.Path)
	assert.Equal(t, "/var/lib/tidb-sql-tls/ca.crt", cfg.Security.SSLCA)
	assert.Equal(t, "/var/lib/tidb-sql-tls/tls.crt", cfg.Security.SSLCert)
	assert.Equal(t, "/var/lib/tidb-sql-tls/tls.key", cfg.Security.SSLKey)
	assert.Equal(t, "/var/lib/tidb-tls/ca.crt", cfg.Security.ClusterSSLCA)
	assert.Equal(t, "/var/lib/tidb-tls/tls.crt", cfg.Security.ClusterSSLCert)
	assert.Equal(t, "/var/lib/tidb-tls/tls.key", cfg.Security.ClusterSSLKey)
	assert.Equal(t, "/var/lib/tidb-auth-token/tidb_auth_token_jwks.json", cfg.Security.AuthTokenJwks)
	assert.Equal(t, "/var/log/tidb/slowlog", cfg.Log.SlowQueryFile)
	assert.Equal(t, "/etc/tidb-bootstrap/bootstrap.sql", cfg.InitializeSQLFile)
	assert.Equal(t, 100, cfg.GracefulWaitBeforeShutdown)

	// store slowlog in PVC
	tidb2 := tidb.DeepCopy()
	tidb2.Spec.Volumes = []v1alpha1.Volume{
		{
			Name: "slowlog",
			Mounts: []v1alpha1.VolumeMount{
				{
					Type: v1alpha1.VolumeMountTypeTiDBSlowLog,
				},
			},
		},
	}
	cfg2 := &Config{}
	err = cfg2.Overlay(cluster, tidb2)
	require.NoError(t, err)
	assert.Equal(t, "/var/log/tidb/slowlog", cfg2.Log.SlowQueryFile)
}
