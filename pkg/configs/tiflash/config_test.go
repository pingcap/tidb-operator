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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func TestValidate(t *testing.T) {
	cfgValid := &Config{}
	err := cfgValid.Validate()
	require.NoError(t, err)

	cfgInvalid := &Config{
		TmpPath: "/data0/tmp",
		Storage: Storage{
			Main: StorageMain{
				Dir: []string{"/data0/db"},
			},
			Raft: StorageRaft{
				Dir: []string{"/data0/kvstore"},
			},
		},
		Flash: Flash{
			ServiceAddr: "basic-tiflash-0.basic-tiflash-peer.ns1.svc:3930",
			Proxy: Proxy{
				Config:              "/path/to/proxy-config",
				DataDir:             "/path/to/proxy-data",
				Addr:                "[::]:20170",
				AdvertiseAddr:       "basic-tiflash-0.basic-tiflash-peer.ns1.svc:20170",
				AdvertiseStatusAddr: "basic-tiflash-0.basic-tiflash-peer.ns1.svc:20292",
			},
		},
		Raft: Raft{
			PdAddr: "http://basic-pd.ns1.svc:2379",
		},
		Status: Status{
			MetricsPort: 8234,
		},
		Logger: Logger{
			Log:      "/path/to/log",
			Errorlog: "/path/to/errorlog",
		},
		Security: Security{
			CAPath:   "/path/to/ca",
			CertPath: "/path/to/cert",
			KeyPath:  "/path/to/key",
		},
	}

	err = cfgInvalid.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "tmp_path")
	require.Contains(t, err.Error(), "storage.main.dir")
	require.Contains(t, err.Error(), "storage.raft.dir")
	require.Contains(t, err.Error(), "flash.service_addr")
	require.Contains(t, err.Error(), "flash.proxy.config")
	require.Contains(t, err.Error(), "flash.proxy.data-dir")
	require.Contains(t, err.Error(), "flash.proxy.addr")
	require.Contains(t, err.Error(), "flash.proxy.advertise-addr")
	require.Contains(t, err.Error(), "flash.proxy.advertise-status-addr")
	require.Contains(t, err.Error(), "raft.pd_addr")
	require.Contains(t, err.Error(), "status.metrics_port")
	require.Contains(t, err.Error(), "logger.log")
	require.Contains(t, err.Error(), "logger.errorlog")
	require.Contains(t, err.Error(), "security.ca_path")
	require.Contains(t, err.Error(), "security.cert_path")
	require.Contains(t, err.Error(), "security.key_path")
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
	tiflash := &v1alpha1.TiFlash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "basic-0",
		},
		Spec: v1alpha1.TiFlashSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "basic-tiflash-peer",
			TiFlashTemplateSpec: v1alpha1.TiFlashTemplateSpec{
				Volumes: []v1alpha1.Volume{
					{
						Name: "data",
						Path: "/data0",
					},
				},
			},
		},
	}

	cfg := &Config{}
	err := cfg.Overlay(cluster, tiflash)
	require.NoError(t, err)
	assert.Equal(t, "/data0/tmp", cfg.TmpPath)
	assert.Equal(t, []string{"/data0/db"}, cfg.Storage.Main.Dir)
	assert.Equal(t, []string{"/data0/kvstore"}, cfg.Storage.Raft.Dir)
	assert.Equal(t, "basic-tiflash-0.basic-tiflash-peer.ns1:3930", cfg.Flash.ServiceAddr)
	assert.Equal(t, "/etc/tiflash/proxy.toml", cfg.Flash.Proxy.Config)
	assert.Equal(t, "/data0/proxy", cfg.Flash.Proxy.DataDir)
	assert.Equal(t, "[::]:20170", cfg.Flash.Proxy.Addr)
	assert.Equal(t, "basic-tiflash-0.basic-tiflash-peer.ns1:20170", cfg.Flash.Proxy.AdvertiseAddr)
	assert.Equal(t, "basic-tiflash-0.basic-tiflash-peer.ns1:20292", cfg.Flash.Proxy.AdvertiseStatusAddr)
	assert.Equal(t, "https://basic-pd.ns1:2379", cfg.Raft.PdAddr)
	assert.Equal(t, "/data0/logs/server.log", cfg.Logger.Log)
	assert.Equal(t, "/data0/logs/error.log", cfg.Logger.Errorlog)
	assert.Equal(t, 8234, cfg.Status.MetricsPort)
	assert.Equal(t, "/var/lib/tiflash-tls/ca.crt", cfg.Security.CAPath)
	assert.Equal(t, "/var/lib/tiflash-tls/tls.crt", cfg.Security.CertPath)
	assert.Equal(t, "/var/lib/tiflash-tls/tls.key", cfg.Security.KeyPath)
}
