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
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/toml"
)

func TestValidate(t *testing.T) {
	cfgValid := &Config{}
	err := cfgValid.Validate()
	require.NoError(t, err)

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
	require.Error(t, err)
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
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "cluster-1",
		},
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "https://basic-pd.ns1:2379",
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
						Mounts: []v1alpha1.VolumeMount{
							{
								Type: v1alpha1.VolumeMountTypeTiKVData,
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
	err := cfg.Overlay(cluster, tikv, features.NewFromFeatures(nil))
	require.NoError(t, err)
	assert.Equal(t, "[::]:20160", cfg.Server.Addr)
	assert.Equal(t, "basic-tikv-0.basic-tikv-peer.ns1:20160", cfg.Server.AdvertiseAddr)
	assert.Equal(t, "[::]:20180", cfg.Server.StatusAddr)
	assert.Equal(t, "basic-tikv-0.basic-tikv-peer.ns1:20180", cfg.Server.AdvertiseStatusAddr)
	assert.Equal(t, "/var/lib/tikv", cfg.Storage.DataDir)
	assert.Equal(t, []string{"https://basic-pd.ns1:2379"}, cfg.PD.Endpoints)
	assert.Equal(t, "/var/lib/tikv-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/tikv-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/tikv-tls/tls.key", cfg.Security.KeyPath)
}

func TestOverlayRemoteWorkers(t *testing.T) {
	const (
		workerCoprocessorURL = "http://worker-a-tikv-worker.ns1:19000/coprocessor"
		workerCompactorURL   = "http://worker-a-tikv-worker.ns1:19000/compact"
		coprocessorURL       = "http://copr-a-tikv-worker.ns1:19000/coprocessor"
		compactorURL         = "http://compactor-a-tikv-worker.ns1:19000/compact"
	)

	cases := []struct {
		name                      string
		independentKVEngineWorker bool
		disableRemoteCoprocessor  bool
		omitKVEngineSection       bool
		omitExistingCoprocessor   bool
		coprocessor               string
		compactor                 string
		wantRemoteCoprocessorAddr string
		wantRemoteWorkerAddr      string
		wantRemoteCompactorAddr   string
	}{
		{
			name:                      "legacy default worker fallback",
			wantRemoteCoprocessorAddr: workerCoprocessorURL,
			wantRemoteWorkerAddr:      workerCoprocessorURL,
			wantRemoteCompactorAddr:   workerCompactorURL,
		},
		{
			name:                      "coprocessor ref with independent worker disabled",
			coprocessor:               "copr-a",
			compactor:                 "compactor-a",
			wantRemoteCoprocessorAddr: coprocessorURL,
			wantRemoteWorkerAddr:      coprocessorURL,
			wantRemoteCompactorAddr:   compactorURL,
		},
		{
			name:                     "disable coprocessor with independent worker disabled",
			disableRemoteCoprocessor: true,
			compactor:                "compactor-a",
			wantRemoteWorkerAddr:     workerCoprocessorURL,
			wantRemoteCompactorAddr:  compactorURL,
		},
		{
			name:                      "coprocessor ref with independent worker enabled",
			independentKVEngineWorker: true,
			coprocessor:               "copr-a",
			compactor:                 "compactor-a",
			wantRemoteCoprocessorAddr: coprocessorURL,
			wantRemoteWorkerAddr:      workerCoprocessorURL,
			wantRemoteCompactorAddr:   compactorURL,
		},
		{
			name:                      "disable coprocessor with independent worker enabled",
			independentKVEngineWorker: true,
			disableRemoteCoprocessor:  true,
			compactor:                 "compactor-a",
			wantRemoteWorkerAddr:      workerCoprocessorURL,
			wantRemoteCompactorAddr:   compactorURL,
		},
		{
			name:                     "disable coprocessor when original address is absent",
			disableRemoteCoprocessor: true,
			omitExistingCoprocessor:  true,
			compactor:                "compactor-a",
			wantRemoteWorkerAddr:     workerCoprocessorURL,
			wantRemoteCompactorAddr:  compactorURL,
		},
		{
			name:                     "disable coprocessor when original kvengine section is absent",
			disableRemoteCoprocessor: true,
			omitKVEngineSection:      true,
			compactor:                "compactor-a",
			wantRemoteWorkerAddr:     workerCoprocessorURL,
			wantRemoteCompactorAddr:  compactorURL,
		},
	}

	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			remoteWorkers := &v1alpha1.TiKVRemoteWorkers{
				Worker:                   v1alpha1.WorkerReference{Name: "worker-a"},
				DisableRemoteCoprocessor: tc.disableRemoteCoprocessor,
			}
			if tc.coprocessor != "" {
				remoteWorkers.Coprocessor = &v1alpha1.CoprocessorReference{Name: tc.coprocessor}
			}
			if tc.compactor != "" {
				remoteWorkers.Compactor = &v1alpha1.CompactorReference{Name: tc.compactor}
			}

			tikv := &v1alpha1.TiKV{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "basic-0",
				},
				Spec: v1alpha1.TiKVSpec{
					Subdomain: "basic-tikv-peer",
					TiKVTemplateSpec: v1alpha1.TiKVTemplateSpec{
						RemoteWorkers: remoteWorkers,
					},
				},
			}

			decoder, encoder := toml.Codec[Config]()
			cfg := &Config{}
			input := ""
			if !tc.omitKVEngineSection {
				input = "[kvengine]\n"
				if !tc.omitExistingCoprocessor {
					input += "remote-coprocessor-addr = \"configured-coprocessor\"\n"
				}
				input += "remote-worker-addr = \"configured-worker\"\n\n"
			}
			input += "[dfs]\nremote-compactor-addr = \"configured-compactor\"\n"
			require.NoError(t, decoder.Decode([]byte(input), cfg))
			gateFeatures := []metav1alpha1.Feature(nil)
			if tc.independentKVEngineWorker {
				gateFeatures = []metav1alpha1.Feature{metav1alpha1.IndependentKVEngineWorker}
			}

			require.NoError(t, cfg.Overlay(cluster, tikv, features.NewFromFeatures(gateFeatures)))
			require.NotNil(t, cfg.KVEngine.RemoteCoprocessorAddr)
			assert.Equal(t, tc.wantRemoteCoprocessorAddr, *cfg.KVEngine.RemoteCoprocessorAddr)
			assert.Equal(t, tc.wantRemoteWorkerAddr, cfg.KVEngine.RemoteWorkerAddr)
			assert.Equal(t, tc.wantRemoteCompactorAddr, cfg.DFS.RemoteCompactorAddr)

			if tc.disableRemoteCoprocessor {
				data, err := encoder.Encode(cfg)
				require.NoError(t, err)
				assert.Contains(t, string(data), "remote-coprocessor-addr = ''")
				assert.Contains(t, string(data), "remote-worker-addr = '"+tc.wantRemoteWorkerAddr+"'")
				assert.Contains(t, string(data), "remote-compactor-addr = '"+tc.wantRemoteCompactorAddr+"'")
			}
		})
	}
}

func TestOverlayWithoutRemoteWorkersPreservesAbsentCoprocessorAddr(t *testing.T) {
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"}}
	tikv := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns1", Name: "basic-0"},
		Spec: v1alpha1.TiKVSpec{
			Subdomain: "basic-tikv-peer",
		},
	}

	decoder, encoder := toml.Codec[Config]()
	cfg := &Config{}
	require.NoError(t, decoder.Decode([]byte(`[dfs]
remote-compactor-addr = "configured-compactor"
`), cfg))
	require.NoError(t, cfg.Overlay(cluster, tikv, features.NewFromFeatures(nil)))

	data, err := encoder.Encode(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "remote-coprocessor-addr")
}
