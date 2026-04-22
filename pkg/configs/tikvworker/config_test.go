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

package tikvworker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestValidate(t *testing.T) {
	cfgValid := &Config{}
	err := cfgValid.Validate()
	require.NoError(t, err)

	cfgInvalid := &Config{
		Addr: "0.0.0.0:19000",
		Storage: Storage{
			DataDir: "/var/lib/tikv-worker",
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
	assert.Contains(t, err.Error(), "addr")
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
	worker := &v1alpha1.TiKVWorker{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "basic-0",
		},
		Spec: v1alpha1.TiKVWorkerSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "basic-tikv-worker-peer",
			TiKVWorkerTemplateSpec: v1alpha1.TiKVWorkerTemplateSpec{
				Volumes: []v1alpha1.Volume{
					{
						Name: "data",
						Mounts: []v1alpha1.VolumeMount{
							{
								Type: v1alpha1.VolumeMountTypeTiKVWorkerData,
							},
						},
						Storage: resource.MustParse("1Gi"),
					},
				},
				Config: v1alpha1.ConfigFile(`[log]
level = "info"`),
			},
		},
	}

	cfg := &Config{}
	err := cfg.Overlay(cluster, worker)
	require.NoError(t, err)
	assert.Equal(t, "[::]:19000", cfg.Addr)
	assert.Equal(t, "/var/lib/tikv-worker", cfg.Storage.DataDir)
	assert.Equal(t, []string{"https://basic-pd.ns1:2379"}, cfg.PD.Endpoints)
	assert.Equal(t, "/var/lib/tikv-worker-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/tikv-worker-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/tikv-worker-tls/tls.key", cfg.Security.KeyPath)
}

func TestOverlayDataDirValidation(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "https://basic-pd.ns1:2379",
		},
	}

	t.Run("rejects managed data dir when data volume exists", func(t *testing.T) {
		cfg := &Config{
			Storage: Storage{
				DataDir: "/custom-data-dir",
			},
		}
		worker := &v1alpha1.TiKVWorker{
			Spec: v1alpha1.TiKVWorkerSpec{
				TiKVWorkerTemplateSpec: v1alpha1.TiKVWorkerTemplateSpec{
					Volumes: []v1alpha1.Volume{
						{
							Name: "data",
							Mounts: []v1alpha1.VolumeMount{
								{Type: v1alpha1.VolumeMountTypeTiKVWorkerData},
							},
						},
					},
				},
			},
		}

		err := cfg.Overlay(cluster, worker)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage.data-dir")
	})

	t.Run("keeps user data dir when no data volume exists", func(t *testing.T) {
		cfg := &Config{
			Storage: Storage{
				DataDir: "/custom-data-dir",
			},
		}
		worker := &v1alpha1.TiKVWorker{}

		err := cfg.Overlay(cluster, worker)
		require.NoError(t, err)
		assert.Equal(t, "/custom-data-dir", cfg.Storage.DataDir)
	})

	t.Run("does not treat empty mount type as data volume", func(t *testing.T) {
		cfg := &Config{
			Storage: Storage{
				DataDir: "/custom-data-dir",
			},
		}
		worker := &v1alpha1.TiKVWorker{
			Spec: v1alpha1.TiKVWorkerSpec{
				TiKVWorkerTemplateSpec: v1alpha1.TiKVWorkerTemplateSpec{
					Volumes: []v1alpha1.Volume{
						{
							Name: "custom",
							Mounts: []v1alpha1.VolumeMount{
								{MountPath: "/mnt/custom"},
							},
						},
					},
				},
			},
		}

		err := cfg.Overlay(cluster, worker)
		require.NoError(t, err)
		assert.Equal(t, "/custom-data-dir", cfg.Storage.DataDir)
	})
}
