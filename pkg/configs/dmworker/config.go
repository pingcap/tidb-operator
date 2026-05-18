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
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

// Config is a subset config of dm-worker.
// Only operator-managed fields are defined here.
type Config struct {
	Name          string `toml:"name"`
	WorkerAddr    string `toml:"worker-addr"`
	AdvertiseAddr string `toml:"advertise-addr"`
	// Join is the dm-master client address list (e.g., "host1:8261,host2:8261")
	Join     string `toml:"join"`
	RelayDir string `toml:"relay-dir"`

	// SSL fields are top-level in DM worker config (not nested under [security])
	SSLCA   string `toml:"ssl-ca"`
	SSLCert string `toml:"ssl-cert"`
	SSLKey  string `toml:"ssl-key"`
}

// Overlay fills in operator-managed config fields.
// dmMasterAddr is the dm-master client service address (e.g., "host:8261") used as the join target.
func (c *Config) Overlay(cluster *v1alpha1.Cluster, dw *v1alpha1.DMWorker, dmMasterAddr string) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.SSLCA = path.Join(v1alpha1.DirPathClusterTLSDMWorker, corev1.ServiceAccountRootCAKey)
		c.SSLCert = path.Join(v1alpha1.DirPathClusterTLSDMWorker, corev1.TLSCertKey)
		c.SSLKey = path.Join(v1alpha1.DirPathClusterTLSDMWorker, corev1.TLSPrivateKeyKey)
	}

	c.Name = dw.Name
	c.WorkerAddr = coreutil.ListenAddress(coreutil.DMWorkerPort(dw))
	c.AdvertiseAddr = coreutil.InstanceAdvertiseAddress[scope.DMWorker](cluster, dw, coreutil.DMWorkerPort(dw))
	c.Join = dmMasterAddr

	// relay dir from RelayVolume mounts
	if dw.Spec.RelayVolume != nil {
		for k := range dw.Spec.RelayVolume.Mounts {
			mount := &dw.Spec.RelayVolume.Mounts[k]
			if mount.Type == v1alpha1.VolumeMountTypeDMWorkerRelay {
				c.RelayDir = mount.MountPath
			}
		}
	}
	if c.RelayDir == "" {
		c.RelayDir = v1alpha1.VolumeMountDMWorkerRelayDefaultPath
	}

	return nil
}

func (c *Config) Validate() error {
	var fields []string

	if c.Name != "" {
		fields = append(fields, "name")
	}
	if c.WorkerAddr != "" {
		fields = append(fields, "worker-addr")
	}
	if c.AdvertiseAddr != "" {
		fields = append(fields, "advertise-addr")
	}
	if c.Join != "" {
		fields = append(fields, "join")
	}
	if c.RelayDir != "" {
		fields = append(fields, "relay-dir")
	}
	if c.SSLCA != "" {
		fields = append(fields, "ssl-ca")
	}
	if c.SSLCert != "" {
		fields = append(fields, "ssl-cert")
	}
	if c.SSLKey != "" {
		fields = append(fields, "ssl-key")
	}

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}
