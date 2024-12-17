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
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

type Config struct {
	Server     Server     `toml:"server"`
	Storage    Storage    `toml:"storage"`
	PD         PD         `toml:"pd"`
	RaftEngine RaftEngine `toml:"raft-engine"`
	RocksDB    RocksDB    `toml:"rocksdb"`
	Security   Security   `toml:"security"`
}

type Server struct {
	Addr                string `toml:"addr"`
	AdvertiseAddr       string `toml:"advertise-addr"`
	StatusAddr          string `toml:"status-addr"`
	AdvertiseStatusAddr string `toml:"advertise-status-addr"`
}

type Storage struct {
	DataDir string `toml:"data-dir"`
}

type PD struct {
	Endpoints []string `toml:"endpoints"`
}

type RaftEngine struct {
	// Only for validation, it cannot be disabled
	Enable *bool  `toml:"enable"`
	Dir    string `toml:"dir"`
}

type RocksDB struct {
	WALDir string `toml:"wal-dir"`
}

type Security struct {
	// CAPath is the path of file that contains list of trusted SSL CAs.
	CAPath string `toml:"ca-path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert-path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key-path"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, tikv *v1alpha1.TiKV) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if cluster.IsTLSClusterEnabled() {
		c.Security.CAPath = path.Join(v1alpha1.TiKVClusterTLSMountPath, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.TiKVClusterTLSMountPath, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.TiKVClusterTLSMountPath, corev1.TLSPrivateKeyKey)
	}

	c.Server.Addr = getClientURLs(tikv)
	c.Server.AdvertiseAddr = GetAdvertiseClientURLs(tikv)
	c.Server.StatusAddr = getStatusURLs(tikv)
	c.Server.AdvertiseStatusAddr = getAdvertiseStatusURLs(tikv)
	c.PD.Endpoints = []string{cluster.Status.PD}

	for i := range tikv.Spec.Volumes {
		vol := &tikv.Spec.Volumes[i]
		for _, usage := range vol.For {
			switch usage.Type {
			case v1alpha1.VolumeUsageTypeTiKVData:
				p := string(usage.Type)
				if usage.SubPath != "" {
					p = usage.SubPath
				}
				c.Storage.DataDir = path.Join(vol.Path, p)
			case v1alpha1.VolumeUsageTypeTiKVRaftEngine:
				p := string(usage.Type)
				if usage.SubPath != "" {
					p = usage.SubPath
				}
				c.RaftEngine.Dir = path.Join(vol.Path, p)
			case v1alpha1.VolumeUsageTypeTiKVWAL:
				p := string(usage.Type)
				if usage.SubPath != "" {
					p = usage.SubPath
				}
				c.RocksDB.WALDir = path.Join(vol.Path, p)
			}
		}
	}

	return nil
}

func (c *Config) Validate() error {
	fields := []string{}

	if c.Server.Addr != "" {
		fields = append(fields, "server.addr")
	}
	if c.Server.AdvertiseAddr != "" {
		fields = append(fields, "server.advertise-addr")
	}
	if c.Server.StatusAddr != "" {
		fields = append(fields, "server.status-addr")
	}
	if c.Storage.DataDir != "" {
		fields = append(fields, "storage.data-dir")
	}
	if len(c.PD.Endpoints) != 0 {
		fields = append(fields, "pd.endpoints")
	}

	if c.RaftEngine.Enable != nil {
		fields = append(fields, "raft-engine.enable")
	}
	if c.RaftEngine.Dir != "" {
		fields = append(fields, "raft-engine.dir")
	}
	if c.RocksDB.WALDir != "" {
		fields = append(fields, "rocksdb.wal-dir")
	}

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}

func getClientURLs(tikv *v1alpha1.TiKV) string {
	return fmt.Sprintf("[::]:%d", tikv.GetClientPort())
}

func GetAdvertiseClientURLs(tikv *v1alpha1.TiKV) string {
	ns := tikv.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", tikv.Name, tikv.Spec.Subdomain, ns, tikv.GetClientPort())
}

func getStatusURLs(tikv *v1alpha1.TiKV) string {
	return fmt.Sprintf("[::]:%d", tikv.GetStatusPort())
}

func getAdvertiseStatusURLs(tikv *v1alpha1.TiKV) string {
	ns := tikv.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", tikv.Name, tikv.Spec.Subdomain, ns, tikv.GetStatusPort())
}
