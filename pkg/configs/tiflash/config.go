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
	"fmt"
	"path"
	"slices"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

// Config is a subset config of tiflash
// Only our managed fields are defined in here.
// ref: https://github.com/pingcap/tiflash/blob/master/etc/config-template.toml
// NOTE: the following config items are set in TiDB Operator v1, but not in TiDB Operator v2 as they are removed from v6.1
// - flash.tidb_status_addr
// - flash.flash_cluster
type Config struct {
	// NOTE: in TiFlash, some fields use "_" instead of "-"

	TmpPath string  `toml:"tmp_path"`
	Storage Storage `toml:"storage"`
	Flash   Flash   `toml:"flash"`
	Raft    Raft    `toml:"raft"`
	Status  Status  `toml:"status"`
	Logger  Logger  `toml:"logger"`

	Security Security `toml:"security"`
}

type Storage struct {
	Main StorageMain `toml:"main"`
	Raft StorageRaft `toml:"raft"`
}

type StorageMain struct {
	Dir []string `toml:"dir"`
}

type StorageRaft struct {
	Dir []string `toml:"dir"`
}

type Flash struct {
	ServiceAddr string `toml:"service_addr"`

	Proxy Proxy `toml:"proxy"`
}

type Proxy struct {
	Config              string `toml:"config"`
	DataDir             string `toml:"data-dir"`
	Addr                string `toml:"addr"`
	AdvertiseAddr       string `toml:"advertise-addr"`
	AdvertiseStatusAddr string `toml:"advertise-status-addr"`
}

type Raft struct {
	PDAddr string `toml:"pd_addr"`
}

type Status struct {
	MetricsPort int `toml:"metrics_port"`
}

type Logger struct {
	Log      string `toml:"log"`
	Errorlog string `toml:"errorlog"`
}

type Security struct {
	// CAPath is the path of file that contains list of trusted SSL CAs.
	CAPath string `toml:"ca_path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert_path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key_path"`
}

func (c *Config) Overlay(cluster *v1alpha1.Cluster, tiflash *v1alpha1.TiFlash) error {
	var dataDir string
	for i := range tiflash.Spec.Volumes {
		vol := &tiflash.Spec.Volumes[i]
		for _, mount := range vol.Mounts {
			if mount.Type != v1alpha1.VolumeMountTypeTiFlashData {
				continue
			}
			dataDir = mount.MountPath
		}
	}

	if err := c.Validate(dataDir); err != nil {
		return err
	}

	if coreutil.IsTLSClusterEnabled(cluster) {
		c.Security.CAPath = path.Join(v1alpha1.DirPathClusterTLSTiFlash, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.DirPathClusterTLSTiFlash, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.DirPathClusterTLSTiFlash, corev1.TLSPrivateKeyKey)
	}

	c.TmpPath = getTmpPath(dataDir)
	c.Storage.Main.Dir = []string{getMainStorageDir(dataDir)}
	c.Storage.Raft.Dir = []string{getRaftStorageDir(dataDir)}
	c.Flash.Proxy.DataDir = getProxyDataDir(dataDir)
	c.Logger.Log = GetServerLogPath(dataDir)
	c.Logger.Errorlog = GetErrorLogPath(dataDir)

	c.Flash.ServiceAddr = coreutil.InstanceAdvertiseAddress[scope.TiFlash](cluster, tiflash, coreutil.TiFlashFlashPort(tiflash))
	// /etc/tiflash/proxy.toml
	c.Flash.Proxy.Config = path.Join(v1alpha1.DirPathConfigTiFlash, v1alpha1.FileNameConfigTiFlashProxy)
	c.Flash.Proxy.Addr = coreutil.ListenAddress(coreutil.TiFlashProxyPort(tiflash))
	c.Flash.Proxy.AdvertiseAddr = coreutil.InstanceAdvertiseAddress[scope.TiFlash](cluster, tiflash, coreutil.TiFlashProxyPort(tiflash))
	c.Flash.Proxy.AdvertiseStatusAddr = coreutil.InstanceAdvertiseAddress[scope.TiFlash](cluster, tiflash, coreutil.TiFlashProxyStatusPort(tiflash))

	c.Raft.PDAddr = cluster.Status.PD

	c.Status.MetricsPort = int(coreutil.TiFlashMetricsPort(tiflash))

	return nil
}

//nolint:gocyclo // refactor if possible
func (c *Config) Validate(dataDir string) error {
	fields := []string{}

	if c.Security.CAPath != "" {
		fields = append(fields, "security.ca_path")
	}
	if c.Security.CertPath != "" {
		fields = append(fields, "security.cert_path")
	}
	if c.Security.KeyPath != "" {
		fields = append(fields, "security.key_path")
	}

	if c.TmpPath != "" {
		fields = append(fields, "tmp_path")
	}

	if c.Storage.Main.Dir != nil && !slices.Equal(c.Storage.Main.Dir, []string{getMainStorageDir(dataDir)}) {
		fields = append(fields, "storage.main.dir")
	}
	if c.Storage.Raft.Dir != nil && !slices.Equal(c.Storage.Raft.Dir, []string{getRaftStorageDir(dataDir)}) {
		fields = append(fields, "storage.raft.dir")
	}

	if c.Flash.ServiceAddr != "" {
		fields = append(fields, "flash.service_addr")
	}
	if c.Flash.Proxy.Config != "" {
		fields = append(fields, "flash.proxy.config")
	}
	if c.Flash.Proxy.DataDir != "" && c.Flash.Proxy.DataDir != getProxyDataDir(dataDir) {
		fields = append(fields, "flash.proxy.data-dir")
	}
	if c.Flash.Proxy.Addr != "" {
		fields = append(fields, "flash.proxy.addr")
	}
	if c.Flash.Proxy.AdvertiseAddr != "" {
		fields = append(fields, "flash.proxy.advertise-addr")
	}
	if c.Flash.Proxy.AdvertiseStatusAddr != "" {
		fields = append(fields, "flash.proxy.advertise-status-addr")
	}

	if c.Raft.PDAddr != "" {
		fields = append(fields, "raft.pd_addr")
	}

	if c.Logger.Log != "" {
		fields = append(fields, "logger.log")
	}
	if c.Logger.Errorlog != "" {
		fields = append(fields, "logger.errorlog")
	}

	if c.Status.MetricsPort != 0 {
		fields = append(fields, "status.metrics_port")
	}

	if len(fields) == 0 {
		return nil
	}
	return fmt.Errorf("%v: %w", fields, v1alpha1.ErrFieldIsManagedByOperator)
}

func GetDataDir(dataDir string) string {
	if dataDir == "" {
		return v1alpha1.VolumeMountTiFlashDataDefaultPath
	}

	return dataDir
}

func GetServerLogPath(dataDir string) string {
	dataDir = GetDataDir(dataDir)
	return fmt.Sprintf("%s/logs/server.log", dataDir)
}

func GetErrorLogPath(dataDir string) string {
	dataDir = GetDataDir(dataDir)
	return fmt.Sprintf("%s/logs/error.log", dataDir)
}

func getTmpPath(dataDir string) string {
	dataDir = GetDataDir(dataDir)
	return fmt.Sprintf("%s/tmp", dataDir)
}

func getMainStorageDir(dataDir string) string {
	dataDir = GetDataDir(dataDir)
	return fmt.Sprintf("%s/db", dataDir)
}

func getRaftStorageDir(dataDir string) string {
	dataDir = GetDataDir(dataDir)
	return fmt.Sprintf("%s/kvstore", dataDir)
}

func getProxyDataDir(dataDir string) string {
	dataDir = GetDataDir(dataDir)
	return fmt.Sprintf("%s/proxy", dataDir)
}
