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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

type Config struct {
	Server   Server   `toml:"server"`
	Storage  Storage  `toml:"storage"`
	PD       PD       `toml:"pd"`
	Security Security `toml:"security"`

	DFS      *DFS      `toml:"dfs,omitempty"`
	KVEngine *KVEngine `toml:"kvengine,omitempty"`
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

type Security struct {
	// CAPath is the path of file that contains list of trusted SSL CAs.
	CAPath string `toml:"ca-path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert-path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key-path"`
}

type DFS struct {
	RemoteCompactorAddr string `toml:"remote-compactor-addr"`
}

type KVEngine struct {
	RemoteCoprocessorAddr string `toml:"remote-coprocessor-addr"`
	RemoteWorkerAddr      string `toml:"remote-worker-addr"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, tikv *v1alpha1.TiKV) error {
	if err := c.Validate(); err != nil {
		return err
	}

	tls := coreutil.IsTLSClusterEnabled(cluster)

	if tls {
		c.Security.CAPath = path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.TLSPrivateKeyKey)
	}

	c.Server.Addr = coreutil.ListenAddress(coreutil.TiKVClientPort(tikv))
	c.Server.AdvertiseAddr = coreutil.InstanceAdvertiseAddress[scope.TiKV](cluster, tikv, coreutil.TiKVClientPort(tikv))
	c.Server.StatusAddr = coreutil.ListenAddress(coreutil.TiKVStatusPort(tikv))
	c.Server.AdvertiseStatusAddr = coreutil.InstanceAdvertiseAddress[scope.TiKV](cluster, tikv, coreutil.TiKVStatusPort(tikv))
	c.PD.Endpoints = []string{cluster.Status.PD}

	for i := range tikv.Spec.Volumes {
		vol := &tikv.Spec.Volumes[i]
		for _, mount := range vol.Mounts {
			if mount.Type == v1alpha1.VolumeMountTypeTiKVData {
				c.Storage.DataDir = mount.MountPath
			}
		}
	}

	if c.Storage.DataDir == "" {
		c.Storage.DataDir = v1alpha1.VolumeMountTiKVDataDefaultPath
	}

	if tikv.Spec.RemoteWorkers != nil {
		if c.KVEngine == nil {
			c.KVEngine = &KVEngine{}
		}
		if c.DFS == nil {
			c.DFS = &DFS{}
		}
		compactor := tikv.Spec.RemoteWorkers.Worker.Name
		copr := tikv.Spec.RemoteWorkers.Worker.Name
		if tikv.Spec.RemoteWorkers.Coprocessor != nil {
			copr = tikv.Spec.RemoteWorkers.Coprocessor.Name
		}

		c.KVEngine.RemoteCoprocessorAddr = coreutil.TiKVWorkerCoprocessorURLFromRef(cluster, copr)
		c.KVEngine.RemoteWorkerAddr = coreutil.TiKVWorkerCoprocessorURLFromRef(cluster, copr)
		c.DFS.RemoteCompactorAddr = coreutil.TiKVWorkerCompactorURLFromRef(cluster, compactor)
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
	if c.Server.AdvertiseStatusAddr != "" {
		fields = append(fields, "server.advertise-status-addr")
	}

	if c.Storage.DataDir != "" {
		fields = append(fields, "storage.data-dir")
	}
	if len(c.PD.Endpoints) != 0 {
		fields = append(fields, "pd.endpoints")
	}

	if c.Security.CAPath != "" {
		fields = append(fields, "security.ca-path")
	}
	if c.Security.CertPath != "" {
		fields = append(fields, "security.cert-path")
	}
	if c.Security.KeyPath != "" {
		fields = append(fields, "security.key-path")
	}

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}
