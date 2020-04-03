// Copyright 2019 PingCAP, Inc.
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

package defaulting

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

func setTiFlashSpecDefault(tc *v1alpha1.TidbCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.TiFlash.Version != nil {
		if tc.Spec.TiFlash.BaseImage == "" {
			tc.Spec.TiFlash.BaseImage = defaultTiFlashImage
		}
	}
	if tc.Spec.TiFlash.Config == nil {
		tc.Spec.TiFlash.Config = &v1alpha1.TiFlashConfig{}
	}
	setTiFlashConfigDefault(tc.Spec.TiFlash.Config, tc.Name, tc.Namespace)
}

func setTiFlashConfigDefault(config *v1alpha1.TiFlashConfig, clusterName, ns string) {
	if config.CommonConfig == nil {
		config.CommonConfig = &v1alpha1.CommonConfig{}
	}
	setTiFlashCommonConfigDefault(config.CommonConfig, clusterName, ns)
	if config.ProxyConfig == nil {
		config.ProxyConfig = &v1alpha1.ProxyConfig{}
	}
	setTiFlashProxyConfigDefault(config.ProxyConfig, clusterName, ns)
}

func setTiFlashProxyConfigDefault(config *v1alpha1.ProxyConfig, clusterName, ns string) {
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.Server == nil {
		config.Server = &v1alpha1.FlashServerConfig{}
	}
	if config.Server.EngineAddr == "" {
		config.Server.EngineAddr = fmt.Sprintf("%s-POD_NUM.%s.%s.svc:3930", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns)
	}
}
func setTiFlashCommonConfigDefault(config *v1alpha1.CommonConfig, clusterName, ns string) {
	if config.TmpPath == "" {
		config.TmpPath = "/data/tmp"
	}
	if config.DisplayName == "" {
		config.DisplayName = "TiFlash"
	}
	if config.DefaultProfile == "" {
		config.DefaultProfile = "default"
	}
	if config.Path == "" {
		config.Path = "/data/db"
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
	if config.ListenHost == "" {
		config.ListenHost = "0.0.0.0"
	}
	if config.TCPPort == nil {
		var p int32 = 9000
		config.TCPPort = &p
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
	setTiFlashFlashConfigDefault(config.Flash, clusterName, ns)
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
	setTiFlashRaftConfigDefault(config.FlashRaft, clusterName, ns)
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

func setTiFlashFlashConfigDefault(config *v1alpha1.Flash, clusterName, ns string) {
	if config.TiDBStatusAddr == "" {
		config.TiDBStatusAddr = fmt.Sprintf("%s.%s.svc:10080", controller.TiDBMemberName(clusterName), ns)
	}
	if config.ServiceAddr == "" {
		config.ServiceAddr = fmt.Sprintf("%s-POD_NUM.%s.%s.svc:3930", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns)
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
	setTiFlashFlashProxyConfigDefault(config.FlashProxy, clusterName, ns)
}

func setTiFlashFlashProxyConfigDefault(config *v1alpha1.FlashProxy, clusterName, ns string) {
	if config.Addr == "" {
		config.Addr = "0.0.0.0:20170"
	}
	if config.AdvertiseAddr == "" {
		config.AdvertiseAddr = fmt.Sprintf("%s-POD_NUM.%s.%s.svc:20170", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns)
	}
	if config.DataDir == "" {
		config.DataDir = "/data/proxy"
	}
	if config.Config == "" {
		config.Config = "/data/proxy.toml"
	}
	if config.LogFile == "" {
		config.LogFile = "/data/logs/proxy.log"
	}
}

func setTiFlashFlashClusterConfigDefault(config *v1alpha1.FlashCluster) {
	if config.ClusterManagerPath == "" {
		config.ClusterManagerPath = "/tiflash/flash_cluster_manager"
	}
	if config.ClusterLog == "" {
		config.ClusterLog = "/data/logs/flash_cluster_manager.log"
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
	if config.Errorlog == "" {
		config.Errorlog = "/data/logs/error.log"
	}
	if config.Size == "" {
		config.Size = "100M"
	}
	if config.ServerLog == "" {
		config.ServerLog = "/data/logs/server.log"
	}
	if config.Level == "" {
		config.Level = "information"
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
	if config.PDAddr == "" {
		config.PDAddr = fmt.Sprintf("%s.%s.svc:2379", controller.PDMemberName(clusterName), ns)
	}
	if config.KVStorePath == "" {
		config.KVStorePath = "/data/kvstore"
	}
	if config.StorageEngine == "" {
		config.StorageEngine = "dt"
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
	if config.IP == "" {
		config.IP = "::/0"
	}
}

func setTiFlashUsersConfigDefault(config *v1alpha1.FlashUser) {
	if config.Readonly == nil {
		config.Readonly = &v1alpha1.User{}
	}
	if config.Readonly.Profile == "" {
		config.Readonly.Profile = "readonly"
	}
	if config.Readonly.Quota == "" {
		config.Readonly.Quota = "default"
	}
	if config.Readonly.Networks == nil {
		config.Readonly.Networks = &v1alpha1.Networks{}
	}
	setTiFlashNetworksConfigDefault(config.Readonly.Networks)

	if config.Default == nil {
		config.Default = &v1alpha1.User{}
	}
	if config.Default.Profile == "" {
		config.Default.Profile = "default"
	}
	if config.Default.Quota == "" {
		config.Default.Quota = "default"
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
