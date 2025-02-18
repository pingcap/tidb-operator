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

package ticdc

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
		Addr:          "[::]:8300",
		AdvertiseAddr: "basic-ticdc-0.basic-ticdc-peer.default.svc",
		Security: Security{
			CAPath:   "/path/to/ca",
			CertPath: "/path/to/cert",
			KeyPath:  "/path/to/key",
		},
	}

	err = cfgInvalid.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "addr")
	assert.Contains(t, err.Error(), "advertise-addr")
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
			PD: "https://basic-pd.ns1:2379",
		},
	}
	ticdc := &v1alpha1.TiCDC{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "basic-0",
		},
		Spec: v1alpha1.TiCDCSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "basic-ticdc-peer",
			TiCDCTemplateSpec: v1alpha1.TiCDCTemplateSpec{
				Volumes: []v1alpha1.Volume{
					{
						Name: "data",
						Mounts: []v1alpha1.VolumeMount{
							{
								Type: "data",
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
	err := cfg.Overlay(cluster, ticdc)
	require.NoError(t, err)
	assert.Equal(t, "[::]:8300", cfg.Addr)
	assert.Equal(t, "basic-ticdc-0.basic-ticdc-peer.ns1:8300", cfg.AdvertiseAddr)
	assert.Equal(t, "/var/lib/ticdc-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/ticdc-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/ticdc-tls/tls.key", cfg.Security.KeyPath)
}
