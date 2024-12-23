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

package pd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func TestValidate(t *testing.T) {
	cfgValid := &Config{}
	err := cfgValid.Validate()
	require.NoError(t, err)

	cfgInvalid := &Config{
		Name:                "pd-0",
		DataDir:             "/var/lib/pd",
		ClientUrls:          "https://[::]:2379",
		PeerUrls:            "https://[::]:2380",
		AdvertiseClientUrls: "https://pd-0.default.svc:2379",
		AdvertisePeerUrls:   "https://pd-0.default.svc:2380",
		InitialCluster:      "pd-0=https://pd-0.default.svc:2380,pd-1=https://pd-1.default.svc:2380",
		InitialClusterState: InitialClusterStateNew,
		InitialClusterToken: "pd-cluster",
		Join:                "pd-2=https://pd-2.default.svc:2380",
		Security: Security{
			CAPath:   "/path/to/ca",
			CertPath: "/path/to/cert",
			KeyPath:  "/path/to/key",
		},
	}

	err = cfgInvalid.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "data-dir")
	assert.Contains(t, err.Error(), "client-urls")
	assert.Contains(t, err.Error(), "peer-urls")
	assert.Contains(t, err.Error(), "advertise-client-urls")
	assert.Contains(t, err.Error(), "advertise-peer-urls")
	assert.Contains(t, err.Error(), "initial-cluster")
	assert.Contains(t, err.Error(), "initial-cluster-state")
	assert.Contains(t, err.Error(), "initial-cluster-token")
	assert.Contains(t, err.Error(), "join")
	assert.Contains(t, err.Error(), "security.cacert-path")
	assert.Contains(t, err.Error(), "security.cert-path")
	assert.Contains(t, err.Error(), "security.key-path")
}

func TestOverlay(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "https://db-pd.default.svc:2379",
		},
	}
	pd := &v1alpha1.PD{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "basic-0",
		},
		Spec: v1alpha1.PDSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "basic",
			PDTemplateSpec: v1alpha1.PDTemplateSpec{
				Volumes: []v1alpha1.Volume{
					{
						Name: "data",
						Path: "/var/lib/pd",
						For: []v1alpha1.VolumeUsage{
							{
								Type: v1alpha1.VolumeUsageTypePDData,
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
	peers := []*v1alpha1.PD{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "basic-0",
			},
			Spec: v1alpha1.PDSpec{
				Subdomain: "default",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "basic-1",
			},
			Spec: v1alpha1.PDSpec{
				Subdomain: "default",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "basic-2",
			},
			Spec: v1alpha1.PDSpec{
				Subdomain: "default",
			},
		},
	}

	// join to an existing cluster
	cfg := &Config{}
	err := cfg.Overlay(cluster, pd, peers)
	require.NoError(t, err)
	assert.Equal(t, "basic-0", cfg.Name)
	assert.Equal(t, "/var/lib/pd", cfg.DataDir)
	assert.Equal(t, "https://[::]:2379", cfg.ClientUrls)
	assert.Equal(t, "https://[::]:2380", cfg.PeerUrls)
	assert.Equal(t, "https://basic-pd-0.basic.default:2379", cfg.AdvertiseClientUrls)
	assert.Equal(t, "https://basic-pd-0.basic.default:2380", cfg.AdvertisePeerUrls)
	assert.Equal(t, "", cfg.InitialCluster)
	assert.Equal(t, "", cfg.InitialClusterState)
	assert.Equal(t, "cluster-1", cfg.InitialClusterToken)
	assert.Equal(t, "/var/lib/pd-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/pd-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/pd-tls/tls.key", cfg.Security.KeyPath)

	// init a new cluster
	cluster2 := cluster.DeepCopy()
	cluster2.Status.PD = ""
	pd2 := pd.DeepCopy()
	pd2.Annotations = map[string]string{
		v1alpha1.AnnoKeyInitialClusterNum: "3",
	}
	cfg2 := &Config{}
	err = cfg2.Overlay(cluster2, pd2, peers)
	require.NoError(t, err)
	assert.Equal(t, "basic-0", cfg2.Name)
	assert.Equal(t, "/var/lib/pd", cfg2.DataDir)
	assert.Equal(t, "https://[::]:2379", cfg2.ClientUrls)
	assert.Equal(t, "https://[::]:2380", cfg2.PeerUrls)
	assert.Equal(t, "https://basic-pd-0.basic.default:2379", cfg2.AdvertiseClientUrls)
	assert.Equal(t, "https://basic-pd-0.basic.default:2380", cfg2.AdvertisePeerUrls)
	assert.Equal(t, "basic-0=https://basic-pd-0.default.default:2380,basic-1=https://basic-pd-1.default.default:2380,basic-2=https://basic-pd-2.default.default:2380", cfg2.InitialCluster)
	assert.Equal(t, InitialClusterStateNew, cfg2.InitialClusterState)
	assert.Equal(t, "cluster-1", cfg2.InitialClusterToken)
	assert.Equal(t, "/var/lib/pd-tls/ca.crt", cfg2.Security.CAPath)
	assert.Equal(t, "/var/lib/pd-tls/tls.crt", cfg2.Security.CertPath)
	assert.Equal(t, "/var/lib/pd-tls/tls.key", cfg2.Security.KeyPath)
}
