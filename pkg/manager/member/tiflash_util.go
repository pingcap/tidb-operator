// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package member

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	defaultClusterLog = "/data0/logs/flash_cluster_manager.log"
	defaultProxyLog   = "/data0/logs/proxy.log"
	defaultErrorLog   = "/data0/logs/error.log"
	defaultServerLog  = "/data0/logs/server.log"
)

func buildTiFlashSidecarContainers(tc *v1alpha1.TidbCluster) []corev1.Container {
	spec := tc.Spec.TiFlash
	config := spec.Config.DeepCopy()
	image := tc.HelperImage()
	pullPolicy := tc.HelperImagePullPolicy()
	var containers []corev1.Container
	var resource corev1.ResourceRequirements
	if spec.LogTailer != nil {
		resource = controller.ContainerResource(spec.LogTailer.ResourceRequirements)
	}
	if config == nil {
		config = &v1alpha1.TiFlashConfig{}
	}
	setTiFlashLogConfigDefault(config)
	containers = append(containers, buildSidecarContainer("serverlog", *config.CommonConfig.FlashLogger.ServerLog, image, pullPolicy, resource))
	containers = append(containers, buildSidecarContainer("errorlog", *config.CommonConfig.FlashLogger.ErrorLog, image, pullPolicy, resource))
	containers = append(containers, buildSidecarContainer("proxylog", *config.CommonConfig.Flash.FlashProxy.LogFile, image, pullPolicy, resource))
	containers = append(containers, buildSidecarContainer("clusterlog", *config.CommonConfig.Flash.FlashCluster.ClusterLog, image, pullPolicy, resource))
	return containers
}

func buildSidecarContainer(name, path, image string,
	pullPolicy corev1.PullPolicy,
	resource corev1.ResourceRequirements) corev1.Container {
	splitPath := strings.Split(path, string(os.PathSeparator))
	// The log path should be at least /dir/base.log
	if len(splitPath) >= 3 {
		serverLogVolumeName := splitPath[1]
		serverLogMountDir := "/" + serverLogVolumeName
		return corev1.Container{
			Name:            name,
			Image:           image,
			ImagePullPolicy: pullPolicy,
			Resources:       resource,
			VolumeMounts: []corev1.VolumeMount{
				{Name: serverLogVolumeName, MountPath: serverLogMountDir},
			},
			Command: []string{
				"sh",
				"-c",
				fmt.Sprintf("touch %s; tail -n0 -F %s;", path, path),
			},
		}
	}
	return corev1.Container{
		Name:            name,
		Image:           image,
		ImagePullPolicy: pullPolicy,
		Resources:       resource,
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf("touch %s; tail -n0 -F %s;", path, path),
		},
	}
}

func getTiFlashConfig(tc *v1alpha1.TidbCluster) *v1alpha1.TiFlashConfig {
	config := tc.Spec.TiFlash.Config.DeepCopy()
	if config == nil {
		config = &v1alpha1.TiFlashConfig{}
	}

	if config.CommonConfig == nil {
		config.CommonConfig = &v1alpha1.CommonConfig{}
	}
	if config.CommonConfig.FlashDataPath == nil {
		var paths []string
		for k := range tc.Spec.TiFlash.StorageClaims {
			paths = append(paths, fmt.Sprintf("/data%d/db", k))
		}
		if len(paths) > 0 {
			dataPath := strings.Join(paths, ",")
			config.CommonConfig.FlashDataPath = pointer.StringPtr(dataPath)
		}
	}

	if tc.IsHeterogeneous() {
		setTiFlashConfigDefault(config, tc.Spec.Cluster.Name, tc.Name, tc.Namespace, tc.Spec.ClusterDomain)
	} else {
		setTiFlashConfigDefault(config, "", tc.Name, tc.Namespace, tc.Spec.ClusterDomain)
	}

	if tc.IsTLSClusterEnabled() {
		if config.CommonConfig.Security == nil {
			config.CommonConfig.Security = &v1alpha1.FlashSecurity{}
		}
		if config.ProxyConfig.Security == nil {
			config.ProxyConfig.Security = &v1alpha1.TiKVSecurityConfig{}
		}
		config.ProxyConfig.Security.CAPath = pointer.StringPtr(path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey))
		config.ProxyConfig.Security.CertPath = pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSCertKey))
		config.ProxyConfig.Security.KeyPath = pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey))
		config.CommonConfig.Security.CAPath = pointer.StringPtr(path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey))
		config.CommonConfig.Security.CertPath = pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSCertKey))
		config.CommonConfig.Security.KeyPath = pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey))
		if len(config.CommonConfig.Security.CertAllowedCN) > 0 && len(config.ProxyConfig.Security.CertAllowedCN) == 0 {
			config.ProxyConfig.Security.CertAllowedCN = config.CommonConfig.Security.CertAllowedCN
		}
		// unset the http ports
		config.CommonConfig.HTTPPort = nil
		config.CommonConfig.TCPPort = nil
	} else {
		// unset the https ports
		config.CommonConfig.HTTPSPort = nil
		config.CommonConfig.TCPPortSecure = nil
	}
	return config
}

func setTiFlashLogConfigDefault(config *v1alpha1.TiFlashConfig) {
	if config.CommonConfig == nil {
		config.CommonConfig = &v1alpha1.CommonConfig{}
	}
	if config.CommonConfig.Flash == nil {
		config.CommonConfig.Flash = &v1alpha1.Flash{}
	}
	if config.CommonConfig.Flash.FlashCluster == nil {
		config.CommonConfig.Flash.FlashCluster = &v1alpha1.FlashCluster{}
	}
	if config.CommonConfig.Flash.FlashCluster.ClusterLog == nil {
		config.CommonConfig.Flash.FlashCluster.ClusterLog = pointer.StringPtr(defaultClusterLog)
	}
	if config.CommonConfig.Flash.FlashProxy == nil {
		config.CommonConfig.Flash.FlashProxy = &v1alpha1.FlashProxy{}
	}
	if config.CommonConfig.Flash.FlashProxy.LogFile == nil {
		config.CommonConfig.Flash.FlashProxy.LogFile = pointer.StringPtr(defaultProxyLog)
	}

	if config.CommonConfig.FlashLogger == nil {
		config.CommonConfig.FlashLogger = &v1alpha1.FlashLogger{}
	}
	if config.CommonConfig.FlashLogger.ErrorLog == nil {
		config.CommonConfig.FlashLogger.ErrorLog = pointer.StringPtr(defaultErrorLog)
	}
	if config.CommonConfig.FlashLogger.ServerLog == nil {
		config.CommonConfig.FlashLogger.ServerLog = pointer.StringPtr(defaultServerLog)
	}
}

// setTiFlashConfigDefault sets default configs for TiFlash
func setTiFlashConfigDefault(config *v1alpha1.TiFlashConfig, externalClusterName, clusterName, ns, clusterDomain string) {
	if config.CommonConfig == nil {
		config.CommonConfig = &v1alpha1.CommonConfig{}
	}
	setTiFlashCommonConfigDefault(config.CommonConfig, externalClusterName, clusterName, ns, clusterDomain)
	if config.ProxyConfig == nil {
		config.ProxyConfig = &v1alpha1.ProxyConfig{}
	}
	setTiFlashProxyConfigDefault(config.ProxyConfig, clusterName, ns, clusterDomain)
}

func setTiFlashProxyConfigDefault(config *v1alpha1.ProxyConfig, clusterName, ns, clusterDomain string) {
	if config.LogLevel == nil {
		config.LogLevel = pointer.StringPtr("info")
	}
	if config.Server == nil {
		config.Server = &v1alpha1.FlashServerConfig{}
	}
	if config.Server.EngineAddr == nil {
		config.Server.EngineAddr = pointer.StringPtr(fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:3930", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns, controller.FormatClusterDomain(clusterDomain)))
	}
	if config.Server.StatusAddr == nil {
		config.Server.StatusAddr = pointer.StringPtr("0.0.0.0:20292")
	}
	if config.Server.AdvertiseStatusAddr == nil {
		config.Server.AdvertiseStatusAddr = pointer.StringPtr(fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:20292", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns, controller.FormatClusterDomain(clusterDomain)))
	}
}

func setTiFlashCommonConfigDefault(config *v1alpha1.CommonConfig, externalClusterName string, clusterName, ns, clusterDomain string) {
	if config.TmpPath == nil {
		config.TmpPath = pointer.StringPtr("/data0/tmp")
	}
	if config.DisplayName == nil {
		config.DisplayName = pointer.StringPtr("TiFlash")
	}
	if config.DefaultProfile == nil {
		config.DefaultProfile = pointer.StringPtr("default")
	}
	if config.FlashDataPath == nil {
		config.FlashDataPath = pointer.StringPtr("/data0/db")
	}
	if config.PathRealtimeMode == nil {
		b := false
		config.PathRealtimeMode = &b
	}
	if config.MarkCacheSize == nil {
		var m int64 = 5368709120
		config.MarkCacheSize = &m
	}
	if config.MinmaxIndexCacheSize == nil {
		var m int64 = 5368709120
		config.MinmaxIndexCacheSize = &m
	}
	if config.ListenHost == nil {
		config.ListenHost = pointer.StringPtr("0.0.0.0")
	}
	if config.TCPPort == nil {
		var p int32 = 9000
		config.TCPPort = &p
	}
	if config.TCPPortSecure == nil {
		var p int32 = 9000
		config.TCPPortSecure = &p
	}
	if config.HTTPSPort == nil {
		var p int32 = 8123
		config.HTTPSPort = &p
	}
	if config.HTTPPort == nil {
		var p int32 = 8123
		config.HTTPPort = &p
	}
	if config.InternalServerHTTPPort == nil {
		var p int32 = 9009
		config.InternalServerHTTPPort = &p
	}
	if config.Flash == nil {
		config.Flash = &v1alpha1.Flash{}
	}
	setTiFlashFlashConfigDefault(config.Flash, clusterName, ns, clusterDomain)
	if config.FlashLogger == nil {
		config.FlashLogger = &v1alpha1.FlashLogger{}
	}
	setTiFlashLoggerConfigDefault(config.FlashLogger)
	if config.FlashApplication == nil {
		config.FlashApplication = &v1alpha1.FlashApplication{}
	}
	setTiFlashApplicationConfigDefault(config.FlashApplication)
	if config.FlashRaft == nil {
		config.FlashRaft = &v1alpha1.FlashRaft{}
	}

	if len(externalClusterName) > 0 {
		setTiFlashRaftConfigDefault(config.FlashRaft, externalClusterName, ns)
	} else {
		setTiFlashRaftConfigDefault(config.FlashRaft, clusterName, ns)
	}

	if config.FlashStatus == nil {
		config.FlashStatus = &v1alpha1.FlashStatus{}
	}
	setTiFlashStatusConfigDefault(config.FlashStatus)
	if config.FlashQuota == nil {
		config.FlashQuota = &v1alpha1.FlashQuota{}
	}
	setTiFlashQuotasConfigDefault(config.FlashQuota)
	if config.FlashUser == nil {
		config.FlashUser = &v1alpha1.FlashUser{}
	}
	setTiFlashUsersConfigDefault(config.FlashUser)
	if config.FlashProfile == nil {
		config.FlashProfile = &v1alpha1.FlashProfile{}
	}
	setTiFlashProfilesConfigDefault(config.FlashProfile)
}

func setTiFlashFlashConfigDefault(config *v1alpha1.Flash, clusterName, ns, clusterDomain string) {
	if config.TiDBStatusAddr == nil {
		config.TiDBStatusAddr = pointer.StringPtr(fmt.Sprintf("%s.%s.svc:10080", controller.TiDBMemberName(clusterName), ns))
	}
	if config.ServiceAddr == nil {
		config.ServiceAddr = pointer.StringPtr(fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:3930", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns, controller.FormatClusterDomain(clusterDomain)))
	}
	if config.OverlapThreshold == nil {
		o := 0.6
		config.OverlapThreshold = &o
	}
	if config.CompactLogMinPeriod == nil {
		var o int32 = 200
		config.CompactLogMinPeriod = &o
	}
	if config.FlashCluster == nil {
		config.FlashCluster = &v1alpha1.FlashCluster{}
	}
	setTiFlashFlashClusterConfigDefault(config.FlashCluster)
	if config.FlashProxy == nil {
		config.FlashProxy = &v1alpha1.FlashProxy{}
	}
	setTiFlashFlashProxyConfigDefault(config.FlashProxy, clusterName, ns, clusterDomain)
}

func setTiFlashFlashProxyConfigDefault(config *v1alpha1.FlashProxy, clusterName, ns, clusterDomain string) {
	if config.Addr == nil {
		config.Addr = pointer.StringPtr("0.0.0.0:20170")
	}
	if config.AdvertiseAddr == nil {
		config.AdvertiseAddr = pointer.StringPtr(fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:20170", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns, controller.FormatClusterDomain(clusterDomain)))
	}
	if config.DataDir == nil {
		config.DataDir = pointer.StringPtr("/data0/proxy")
	}
	if config.Config == nil {
		config.Config = pointer.StringPtr("/data0/proxy.toml")
	}
	if config.LogFile == nil {
		config.LogFile = pointer.StringPtr(defaultProxyLog)
	}
}

func setTiFlashFlashClusterConfigDefault(config *v1alpha1.FlashCluster) {
	if config.ClusterManagerPath == nil {
		config.ClusterManagerPath = pointer.StringPtr("/tiflash/flash_cluster_manager")
	}
	if config.ClusterLog == nil {
		config.ClusterLog = pointer.StringPtr(defaultClusterLog)
	}
	if config.RefreshInterval == nil {
		var r int32 = 20
		config.RefreshInterval = &r
	}
	if config.UpdateRuleInterval == nil {
		var r int32 = 10
		config.UpdateRuleInterval = &r
	}
	if config.MasterTTL == nil {
		var r int32 = 60
		config.MasterTTL = &r
	}
}

func setTiFlashLoggerConfigDefault(config *v1alpha1.FlashLogger) {
	if config.ErrorLog == nil {
		config.ErrorLog = pointer.StringPtr(defaultErrorLog)
	}
	if config.Size == nil {
		config.Size = pointer.StringPtr("100M")
	}
	if config.ServerLog == nil {
		config.ServerLog = pointer.StringPtr(defaultServerLog)
	}
	if config.Level == nil {
		config.Level = pointer.StringPtr("information")
	}
	if config.Count == nil {
		var c int32 = 10
		config.Count = &c
	}
}

func setTiFlashApplicationConfigDefault(config *v1alpha1.FlashApplication) {
	if config.RunAsDaemon == nil {
		r := true
		config.RunAsDaemon = &r
	}
}

func setTiFlashRaftConfigDefault(config *v1alpha1.FlashRaft, clusterName, ns string) {
	if config.PDAddr == nil {
		config.PDAddr = pointer.StringPtr(fmt.Sprintf("%s.%s.svc:2379", controller.PDMemberName(clusterName), ns))
	}
	if config.KVStorePath == nil {
		config.KVStorePath = pointer.StringPtr("/data0/kvstore")
	}
	if config.StorageEngine == nil {
		config.StorageEngine = pointer.StringPtr("dt")
	}
}

func setTiFlashStatusConfigDefault(config *v1alpha1.FlashStatus) {
	if config.MetricsPort == nil {
		var d int32 = 8234
		config.MetricsPort = &d
	}
}

func setTiFlashQuotasConfigDefault(config *v1alpha1.FlashQuota) {
	if config.Default == nil {
		config.Default = &v1alpha1.Quota{}
	}
	if config.Default.Interval == nil {
		config.Default.Interval = &v1alpha1.Interval{}
	}
	if config.Default.Interval.Duration == nil {
		var d int32 = 3600
		config.Default.Interval.Duration = &d
	}
	if config.Default.Interval.Queries == nil {
		var d int32 = 0
		config.Default.Interval.Queries = &d
	}
	if config.Default.Interval.Errors == nil {
		var d int32 = 0
		config.Default.Interval.Errors = &d
	}
	if config.Default.Interval.ResultRows == nil {
		var d int32 = 0
		config.Default.Interval.ResultRows = &d
	}
	if config.Default.Interval.ReadRows == nil {
		var d int32 = 0
		config.Default.Interval.ReadRows = &d
	}
	if config.Default.Interval.ExecutionTime == nil {
		var d int32 = 0
		config.Default.Interval.ExecutionTime = &d
	}
}

func setTiFlashNetworksConfigDefault(config *v1alpha1.Networks) {
	if config.IP == nil {
		config.IP = pointer.StringPtr("::/0")
	}
}

func setTiFlashUsersConfigDefault(config *v1alpha1.FlashUser) {
	if config.Readonly == nil {
		config.Readonly = &v1alpha1.User{}
	}
	if config.Readonly.Profile == nil {
		config.Readonly.Profile = pointer.StringPtr("readonly")
	}
	if config.Readonly.Quota == nil {
		config.Readonly.Quota = pointer.StringPtr("default")
	}
	if config.Readonly.Networks == nil {
		config.Readonly.Networks = &v1alpha1.Networks{}
	}
	setTiFlashNetworksConfigDefault(config.Readonly.Networks)

	if config.Default == nil {
		config.Default = &v1alpha1.User{}
	}
	if config.Default.Profile == nil {
		config.Default.Profile = pointer.StringPtr("default")
	}
	if config.Default.Quota == nil {
		config.Default.Quota = pointer.StringPtr("default")
	}
	if config.Default.Networks == nil {
		config.Default.Networks = &v1alpha1.Networks{}
	}
	setTiFlashNetworksConfigDefault(config.Default.Networks)
}

func setTiFlashProfilesConfigDefault(config *v1alpha1.FlashProfile) {
	if config.Readonly == nil {
		config.Readonly = &v1alpha1.Profile{}
	}
	if config.Readonly.Readonly == nil {
		var r int32 = 1
		config.Readonly.Readonly = &r
	}
	if config.Default == nil {
		config.Default = &v1alpha1.Profile{}
	}
	if config.Default.MaxMemoryUsage == nil {
		var m int64 = 10000000000
		config.Default.MaxMemoryUsage = &m
	}
	if config.Default.UseUncompressedCache == nil {
		var u int32 = 0
		config.Default.UseUncompressedCache = &u
	}
	if config.Default.LoadBalancing == nil {
		l := "random"
		config.Default.LoadBalancing = &l
	}
}
