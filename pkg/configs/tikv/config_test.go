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

package tikv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func TestValidate(t *testing.T) {
	cfgValid := &Config{}
	err := cfgValid.Validate()
	assert.NoError(t, err)

	cfgInvalid := &Config{
		Server: Server{
			Addr:                ":20160",
			AdvertiseAddr:       "advertise-addr",
			StatusAddr:          "status-addr",
			AdvertiseStatusAddr: "advertise-status-addr",
		},
		Storage: Storage{
			DataDir: "/var/lib/tikv",
		},
		PD: PD{
			Endpoints: []string{"pd-0", "pd-1", "pd-2"},
		},
		Security: Security{
			CAPath:   "/path/to/ca",
			CertPath: "/path/to/cert",
			KeyPath:  "/path/to/key",
		},
	}

	err = cfgInvalid.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server.addr")
	assert.Contains(t, err.Error(), "server.advertise-addr")
	assert.Contains(t, err.Error(), "server.status-addr")
	assert.Contains(t, err.Error(), "server.advertise-status-addr")
	assert.Contains(t, err.Error(), "storage.data-dir")
	assert.Contains(t, err.Error(), "pd.endpoints")
	assert.Contains(t, err.Error(), "security.ca-path")
	assert.Contains(t, err.Error(), "security.cert-path")
	assert.Contains(t, err.Error(), "security.key-path")
}

func TestOverlay(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "https://db-pd.ns1.svc:2379",
		},
	}
	tikv := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "basic-0",
		},
		Spec: v1alpha1.TiKVSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "basic-tikv-peer",
			TiKVTemplateSpec: v1alpha1.TiKVTemplateSpec{
				Volumes: []v1alpha1.Volume{
					{
						Name: "data",
						Path: "/var/lib/tikv",
						For: []v1alpha1.VolumeUsage{
							{
								Type: v1alpha1.VolumeUsageTypeTiKVData,
							},
						},
						Storage: resource.Quantity{
							Format: "1Gi",
						},
					},
				},
				Config: v1alpha1.ConfigFile(`[log]
level = "info"`),
			},
		},
	}

	cfg := &Config{}
	err := cfg.Overlay(cluster, tikv)
	assert.NoError(t, err)
	assert.Equal(t, "[::]:20160", cfg.Server.Addr)
	assert.Equal(t, "basic-tikv-0.basic-tikv-peer.ns1:20160", cfg.Server.AdvertiseAddr)
	assert.Equal(t, "[::]:20180", cfg.Server.StatusAddr)
	assert.Equal(t, "basic-tikv-0.basic-tikv-peer.ns1:20180", cfg.Server.AdvertiseStatusAddr)
	assert.Equal(t, "/var/lib/tikv", cfg.Storage.DataDir)
	assert.Equal(t, []string{"https://db-pd.ns1.svc:2379"}, cfg.PD.Endpoints)
	assert.Equal(t, "/var/lib/tikv-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/tikv-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/tikv-tls/tls.key", cfg.Security.KeyPath)
}
