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
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/configs/common"
)

const fixedMergedStoreID = 1024

type Config struct {
	Server   ReplicationWorker   `toml:"replication-worker"`
	DataDir  string              `toml:"data-dir"`
	Security common.TiKVSecurity `toml:"security"`
}

type ReplicationWorker struct {
	Enabled       bool         `toml:"enabled"`
	GRPCAddr      string       `toml:"grpc-addr"`
	AdvertiseAddr string       `toml:"advertise-addr"`
	MergedEngine  MergedEngine `toml:"merged-engine"`
}

type MergedEngine struct {
	// MergedStoreID is the ID of the merged store.
	// Different replication workers should have different IDs.
	MergedStoreID int `toml:"merged-store-id"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, rw *v1alpha1.ReplicationWorker) error {
	if err := c.Validate(); err != nil {
		return err
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.Security.CAPath = path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.DirPathClusterTLSTiKV, corev1.TLSPrivateKeyKey)
	}

	c.Server.Enabled = true
	c.Server.GRPCAddr = fmt.Sprintf("[::]:%d", coreutil.ReplicationWorkerGRPCPort(rw))
	c.Server.AdvertiseAddr = coreutil.ReplicationWorkerAdvertiseURL(rw)
	// TODO: fix this when we support multiple replication workers in one cluster
	c.Server.MergedEngine.MergedStoreID = fixedMergedStoreID

	for i := range rw.Spec.Volumes {
		vol := &rw.Spec.Volumes[i]
		for _, mount := range vol.Mounts {
			if mount.Type == v1alpha1.VolumeMountTypeReplicationWorkerData {
				c.DataDir = mount.MountPath
			}
		}
	}

	if c.DataDir == "" {
		c.DataDir = v1alpha1.VolumeMountReplicationWorkerDataDefaultPath
	}

	return nil
}

func (c *Config) Validate() error {
	var fields []string

	if c.Server.GRPCAddr != "" {
		fields = append(fields, "replication-worker.grpc-addr")
	}
	if c.Server.AdvertiseAddr != "" {
		fields = append(fields, "replication-worker.advertise-addr")
	}
	if c.DataDir != "" {
		fields = append(fields, "data-dir")
	}
	if c.Server.MergedEngine.MergedStoreID != 0 {
		fields = append(fields, "replication-worker.merged-engine.merged-store-id")
	}

	fields = append(fields, common.ValidateTiKVSecurity(c.Security)...)

	if len(fields) == 0 {
		return nil
	}

	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}
