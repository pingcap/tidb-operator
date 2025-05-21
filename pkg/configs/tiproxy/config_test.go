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

package tidb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantErr   bool
		errFields []string
	}{
		{
			name:    "valid empty config",
			config:  &Config{},
			wantErr: false,
		},
		{
			name: "invalid config with managed fields",
			config: &Config{
				Proxy: Proxy{
					Address:          "0.0.0.0:4000",
					AdvertiseAddress: "tiproxy-0.tiproxy-peer.default.svc",
					PDAddress:        "pd:2379",
				},
				API: API{
					Address: "0.0.0.0:3080",
				},
				Security: Security{
					ServerSQLTLS: TLSConfig{
						Cert: "/path/to/cert",
						Key:  "/path/to/key",
						CA:   "/path/to/ca",
					},
				},
			},
			wantErr: true,
			errFields: []string{
				"proxy.address",
				"proxy.advertise-address",
				"proxy.pd-address",
				"api.address",
				"security.server-sql-tls.cert",
				"security.server-sql-tls.key",
				"security.server-sql-tls.ca",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				for _, field := range tt.errFields {
					assert.Contains(t, err.Error(), field)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOverlay(t *testing.T) {
	tests := []struct {
		name    string
		cluster *v1alpha1.Cluster
		tiproxy *v1alpha1.TiProxy
		want    *Config
		wantErr bool
	}{
		{
			name: "basic config without TLS",
			cluster: &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					PD: "http://db-pd.ns1:2379",
				},
			},
			tiproxy: &v1alpha1.TiProxy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "db-foo",
				},
				Spec: v1alpha1.TiProxySpec{
					Subdomain: "db-tiproxy-peer",
				},
			},
			want: &Config{
				Proxy: Proxy{
					Address:          "0.0.0.0:6000",
					AdvertiseAddress: "db-tiproxy-foo.db-tiproxy-peer.ns1.svc",
					PDAddress:        "db-pd.ns1:2379",
				},
				API: API{
					Address: "0.0.0.0:3080",
				},
			},
			wantErr: false,
		},
		{
			name: "config with cluster TLS enabled",
			cluster: &v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{
						Enabled: true,
					},
				},
				Status: v1alpha1.ClusterStatus{
					PD: "https://db-pd.ns1:2379",
				},
			},
			tiproxy: &v1alpha1.TiProxy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "db-foo",
				},
				Spec: v1alpha1.TiProxySpec{
					Subdomain: "tiproxy-peer",
				},
			},
			want: &Config{
				Proxy: Proxy{
					Address:          "0.0.0.0:6000",
					AdvertiseAddress: "db-tiproxy-foo.tiproxy-peer.ns1.svc",
					PDAddress:        "db-pd.ns1:2379",
				},
				API: API{
					Address: "0.0.0.0:3080",
				},
				Security: Security{
					ClusterTLS: TLSConfig{
						CA:   "/var/lib/tiproxy-tls/ca.crt",
						Cert: "/var/lib/tiproxy-tls/tls.crt",
						Key:  "/var/lib/tiproxy-tls/tls.key",
					},
					ServerHTTPTLS: TLSConfig{
						CA:     "/var/lib/tiproxy-tls/ca.crt",
						Cert:   "/var/lib/tiproxy-tls/tls.crt",
						Key:    "/var/lib/tiproxy-tls/tls.key",
						SkipCA: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "config with MySQL TLS enabled",
			cluster: &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					PD: "http://db-pd.ns1:2379",
				},
			},
			tiproxy: &v1alpha1.TiProxy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "db-foo",
				},
				Spec: v1alpha1.TiProxySpec{
					Subdomain: "tiproxy-peer",
					TiProxyTemplateSpec: v1alpha1.TiProxyTemplateSpec{
						Security: &v1alpha1.TiProxySecurity{
							TLS: &v1alpha1.TiProxyTLS{
								MySQL: &v1alpha1.TLS{
									Enabled: true,
								},
							},
						},
					},
				},
			},
			want: &Config{
				Proxy: Proxy{
					Address:          "0.0.0.0:6000",
					AdvertiseAddress: "db-tiproxy-foo.tiproxy-peer.ns1.svc",
					PDAddress:        "db-pd.ns1:2379",
				},
				API: API{
					Address: "0.0.0.0:3080",
				},
				Security: Security{
					ServerSQLTLS: TLSConfig{
						CA:   "/var/lib/tiproxy-sql-tls/ca.crt",
						Cert: "/var/lib/tiproxy-sql-tls/tls.crt",
						Key:  "/var/lib/tiproxy-sql-tls/tls.key",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "config with backend tls enabled",
			cluster: &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					PD: "http://db-pd.ns1:2379",
				},
			},
			tiproxy: &v1alpha1.TiProxy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "db-foo",
				},
				Spec: v1alpha1.TiProxySpec{
					Subdomain: "tiproxy-peer",
					TiProxyTemplateSpec: v1alpha1.TiProxyTemplateSpec{
						Security: &v1alpha1.TiProxySecurity{
							TLS: &v1alpha1.TiProxyTLS{
								Backend: &v1alpha1.TLS{
									Enabled: true,
									SkipCA:  true,
								},
							},
						},
					},
				},
			},
			want: &Config{
				Proxy: Proxy{
					Address:          "0.0.0.0:6000",
					AdvertiseAddress: "db-tiproxy-foo.tiproxy-peer.ns1.svc",
					PDAddress:        "db-pd.ns1:2379",
				},
				API: API{
					Address: "0.0.0.0:3080",
				},
				Security: Security{
					SQLTLS: TLSConfig{
						CA:     "/var/lib/tiproxy-tidb-tls/ca.crt",
						Cert:   "/var/lib/tiproxy-tidb-tls/tls.crt",
						Key:    "/var/lib/tiproxy-tidb-tls/tls.key",
						SkipCA: true,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &Config{}
			err := got.Overlay(tt.cluster, tt.tiproxy)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
