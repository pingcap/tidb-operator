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

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
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
	PdAddr string `toml:"pd_addr"`
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
	if err := c.Validate(); err != nil {
		return err
	}

	if cluster.IsTLSClusterEnabled() {
		c.Security.CAPath = path.Join(v1alpha1.TiFlashClusterTLSMountPath, corev1.ServiceAccountRootCAKey)
		c.Security.CertPath = path.Join(v1alpha1.TiFlashClusterTLSMountPath, corev1.TLSCertKey)
		c.Security.KeyPath = path.Join(v1alpha1.TiFlashClusterTLSMountPath, corev1.TLSPrivateKeyKey)
	}

	for i := range tiflash.Spec.Volumes {
		vol := &tiflash.Spec.Volumes[i]
		for _, usage := range vol.For {
			if usage.Type != v1alpha1.VolumeUsageTypeTiFlashData {
				continue
			}
			dataDir := vol.Path
			if usage.SubPath != "" {
				dataDir = path.Join(vol.Path, usage.SubPath)
			}

			c.TmpPath = getTmpPath(dataDir)
			c.Storage.Main.Dir = []string{getMainStorageDir(dataDir)}
			c.Storage.Raft.Dir = []string{getRaftStorageDir(dataDir)}
			c.Flash.Proxy.DataDir = getProxyDataDir(dataDir)
			c.Logger.Log = GetServerLogPath(dataDir)
			c.Logger.Errorlog = GetErrorLogPath(dataDir)
		}
	}

	c.Flash.ServiceAddr = GetServiceAddr(tiflash)
	// /etc/tiflash/proxy.toml
	c.Flash.Proxy.Config = path.Join(v1alpha1.DirNameConfigTiFlash, v1alpha1.ConfigFileTiFlashProxyName)
	c.Flash.Proxy.Addr = getProxyAddr(tiflash)
	c.Flash.Proxy.AdvertiseAddr = getProxyAdvertiseAddr(tiflash)
	c.Flash.Proxy.AdvertiseStatusAddr = getProxyAdvertiseStatusAddr(tiflash)

	c.Raft.PdAddr = cluster.Status.PD

	c.Status.MetricsPort = int(tiflash.GetMetricsPort())

	return nil
}

//nolint:gocyclo // refactor if possible
func (c *Config) Validate() error {
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

	if c.Storage.Main.Dir != nil {
		fields = append(fields, "storage.main.dir")
	}
	if c.Storage.Raft.Dir != nil {
		fields = append(fields, "storage.raft.dir")
	}

	if c.Flash.ServiceAddr != "" {
		fields = append(fields, "flash.service_addr")
	}
	if c.Flash.Proxy.Config != "" {
		fields = append(fields, "flash.proxy.config")
	}
	if c.Flash.Proxy.DataDir != "" {
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

	if c.Raft.PdAddr != "" {
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

func GetServiceAddr(tiflash *v1alpha1.TiFlash) string {
	ns := tiflash.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", tiflash.PodName(), tiflash.Spec.Subdomain, ns, tiflash.GetFlashPort())
}

func getProxyAddr(tiflash *v1alpha1.TiFlash) string {
	return fmt.Sprintf("[::]:%d", tiflash.GetProxyPort())
}

func getProxyAdvertiseAddr(tiflash *v1alpha1.TiFlash) string {
	ns := tiflash.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", tiflash.PodName(), tiflash.Spec.Subdomain, ns, tiflash.GetProxyPort())
}

func getProxyAdvertiseStatusAddr(tiflash *v1alpha1.TiFlash) string {
	ns := tiflash.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", tiflash.PodName(), tiflash.Spec.Subdomain, ns, tiflash.GetProxyStatusPort())
}

func GetServerLogPath(dataDir string) string {
	return fmt.Sprintf("%s/logs/server.log", dataDir)
}

func GetErrorLogPath(dataDir string) string {
	return fmt.Sprintf("%s/logs/error.log", dataDir)
}

func getTmpPath(dataDir string) string {
	return fmt.Sprintf("%s/tmp", dataDir)
}

func getMainStorageDir(dataDir string) string {
	return fmt.Sprintf("%s/db", dataDir)
}

func getRaftStorageDir(dataDir string) string {
	return fmt.Sprintf("%s/kvstore", dataDir)
}

func getProxyDataDir(dataDir string) string {
	return fmt.Sprintf("%s/proxy", dataDir)
}
