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

package dmworker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestOverlay(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "tc",
		},
	}
	dw := &v1alpha1.DMWorker{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "wg-0",
		},
		Spec: v1alpha1.DMWorkerSpec{
			Cluster: v1alpha1.ClusterReference{Name: "tc"},
			DMGroupRef: v1alpha1.DMGroupReference{
				Name: "dmg",
			},
			Subdomain: "workers",
			DMWorkerTemplateSpec: v1alpha1.DMWorkerTemplateSpec{
				RelayVolume: &v1alpha1.Volume{
					Name: "relay",
					Mounts: []v1alpha1.VolumeMount{
						{Type: v1alpha1.VolumeMountTypeDMWorkerRelay},
					},
				},
			},
		},
	}

	cfg := &Config{}
	require.NoError(t, cfg.Overlay(cluster, dw, "dmg.ns1.svc:8261"))
	assert.Equal(t, "wg-0", cfg.Name)
	assert.Equal(t, "[::]:8262", cfg.WorkerAddr)
	assert.Equal(t, "wg-dm-worker-0.workers.ns1:8262", cfg.AdvertiseAddr)
	assert.Equal(t, "dmg.ns1.svc:8261", cfg.Join)
	assert.Equal(t, v1alpha1.VolumeMountDMWorkerRelayDefaultPath, cfg.RelayDir)

	dw.Spec.RelayVolume.Mounts[0].MountPath = "/custom/relay"
	cfg = &Config{}
	require.NoError(t, cfg.Overlay(cluster, dw, "dmg.ns1.svc:8261"))
	assert.Equal(t, "/custom/relay", cfg.RelayDir)

	cluster.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
	cfg = &Config{}
	require.NoError(t, cfg.Overlay(cluster, dw, "dmg.ns1.svc:8261"))
	assert.Equal(t, "/var/lib/dm-worker-tls/ca.crt", cfg.SSLCA)
	assert.Equal(t, "/var/lib/dm-worker-tls/tls.crt", cfg.SSLCert)
	assert.Equal(t, "/var/lib/dm-worker-tls/tls.key", cfg.SSLKey)
}
