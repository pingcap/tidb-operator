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

package replicationworker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/configs/common"
)

func TestValidate(t *testing.T) {
	cfgValid := &Config{}
	err := cfgValid.Validate()
	require.NoError(t, err)

	cfgInvalid := &Config{
		Server: ReplicationWorker{
			Enabled:       false,
			GRPCAddr:      "grpc-addr",
			AdvertiseAddr: "advertise-addr",
			MergedEngine: MergedEngine{
				MergedStoreID: 1024,
			},
		},
		DataDir: "/var/lib/tikv",
		Security: common.TiKVSecurity{
			CAPath:   "/path/to/ca",
			CertPath: "/path/to/cert",
			KeyPath:  "/path/to/key",
		},
	}

	err = cfgInvalid.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "replication-worker.grpc-addr")
	assert.Contains(t, err.Error(), "replication-worker.advertise-addr")
	assert.Contains(t, err.Error(), "data-dir")
	assert.Contains(t, err.Error(), "replication-worker.merged-engine.merged-store-id")
	assert.Contains(t, err.Error(), "security.ca-path")
	assert.Contains(t, err.Error(), "security.cert-path")
	assert.Contains(t, err.Error(), "security.key-path")
}

func TestOverlay(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
		},
	}
	rw := &v1alpha1.ReplicationWorker{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "basic-0",
		},
		Spec: v1alpha1.ReplicationWorkerSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "basic-rw-peer",
			ReplicationWorkerTemplateSpec: v1alpha1.ReplicationWorkerTemplateSpec{
				Volumes: []v1alpha1.Volume{
					{
						Name: "data",
						Mounts: []v1alpha1.VolumeMount{
							{
								Type:      v1alpha1.VolumeMountTypeReplicationWorkerData,
								MountPath: "/custom/path/to/data",
							},
						},
						Storage: resource.Quantity{
							Format: "1Gi",
						},
					},
				},
			},
		},
	}

	cfg := &Config{}
	err := cfg.Overlay(cluster, rw)
	require.NoError(t, err)
	assert.Equal(t, "[::]:19160", cfg.Server.GRPCAddr)
	assert.Equal(t, "basic-repl-worker-0.basic-rw-peer.ns1:19160", cfg.Server.AdvertiseAddr)
	assert.Equal(t, fixedMergedStoreID, cfg.Server.MergedEngine.MergedStoreID)
	assert.Equal(t, "/custom/path/to/data", cfg.DataDir)
	assert.Equal(t, "/var/lib/tikv-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/tikv-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/tikv-tls/tls.key", cfg.Security.KeyPath)

	// test default data dir when no TiKVData mount specified
	rw2 := rw.DeepCopy()
	rw2.Spec.Volumes = []v1alpha1.Volume{
		{
			Name: "other",
			Mounts: []v1alpha1.VolumeMount{
				{
					Type: v1alpha1.VolumeMountTypeTiDBSlowLog,
				},
			},
		},
	}
	cfg2 := &Config{}
	err = cfg2.Overlay(cluster, rw2)
	require.NoError(t, err)
	assert.Equal(t, v1alpha1.VolumeMountReplicationWorkerDataDefaultPath, cfg2.DataDir)

	// test non-TLS cluster
	cluster3 := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{},
	}
	cfg3 := &Config{}
	err = cfg3.Overlay(cluster3, rw)
	require.NoError(t, err)
	assert.Empty(t, cfg3.Security.CAPath)
	assert.Empty(t, cfg3.Security.CertPath)
	assert.Empty(t, cfg3.Security.KeyPath)
}
