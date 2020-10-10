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
)

const (
	defaultClusterLog = "/data0/logs/flash_cluster_manager.log"
	defaultErrorLog   = "/data0/logs/error.log"
	defaultServerLog  = "/data0/logs/server.log"
)

func buildTiFlashSidecarContainers(tc *v1alpha1.TidbCluster) ([]corev1.Container, error) {
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
		config = v1alpha1.NewTiFlashConfig()
	}
	setTiFlashLogConfigDefault(config)

	path, err := config.Common.Get("logger.log").AsString()
	if err != nil {
		return nil, err
	}
	containers = append(containers, buildSidecarContainer("serverlog", path, image, pullPolicy, resource))
	path, err = config.Common.Get("logger.errorlog").AsString()
	if err != nil {
		return nil, err
	}
	containers = append(containers, buildSidecarContainer("errorlog", path, image, pullPolicy, resource))
	path, err = config.Common.Get("flash.flash_cluster.log").AsString()
	if err != nil {
		return nil, err
	}
	containers = append(containers, buildSidecarContainer("clusterlog", path, image, pullPolicy, resource))
	return containers, nil
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

func getTiFlashConfig(tc *v1alpha1.TidbCluster) *v1alpha1.TiFlashConfigWraper {
	config := tc.Spec.TiFlash.Config.DeepCopy()
	if config == nil {
		config = v1alpha1.NewTiFlashConfig()
	}

	if config.Common == nil {
		config.Common = v1alpha1.NewTiFlashCommonConfig()
	}

	if config.Common.Get("path") == nil {
		var paths []string
		for k := range tc.Spec.TiFlash.StorageClaims {
			paths = append(paths, fmt.Sprintf("/data%d/db", k))
		}
		if len(paths) > 0 {
			dataPath := strings.Join(paths, ",")
			config.Common.Set("path", dataPath)
		}
	}

	if tc.IsHeterogeneous() {
		setTiFlashConfigDefault(config, tc.Spec.Cluster.Name, tc.Name, tc.Namespace)
	} else {
		setTiFlashConfigDefault(config, "", tc.Name, tc.Namespace)
	}

	// Note the config of tiflash use "_" by convention, others(proxy) use "-".
	if tc.IsTLSClusterEnabled() {
		config.Proxy.Set("security.ca-path", path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey))
		config.Proxy.Set("security.cert-path", path.Join(tiflashCertPath, corev1.TLSCertKey))
		config.Proxy.Set("security.key-path", path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey))
		config.Common.Set("security.ca_path", path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey))
		config.Common.Set("security.cert_path", path.Join(tiflashCertPath, corev1.TLSCertKey))
		config.Common.Set("security.key_path", path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey))

		common := config.Common.Get("security.cert_allowed_cn")
		proxy := config.Proxy.Get("security.cert-allowed-cn")
		if common != nil && proxy == nil {
			config.Proxy.Set("security.cert-allowed-cn", common.Interface())
		}

		// unset the http ports
		config.Common.Del("http_port")
		config.Common.Del("tcp_port")
	} else {
		// unset the https ports
		config.Common.Del("https_port")
		config.Common.Del("tcp_port_secure")
	}

	return config
}

func setTiFlashLogConfigDefault(config *v1alpha1.TiFlashConfigWraper) {
	if config.Common == nil {
		config.Common = v1alpha1.NewTiFlashCommonConfig()
	}
	config.Common.SetIfNil("flash.flash_cluster.log", defaultClusterLog)
	config.Common.SetIfNil("logger.errorlog", defaultErrorLog)
	config.Common.SetIfNil("logger.log", defaultServerLog)
}

// setTiFlashConfigDefault sets default configs for TiFlash
func setTiFlashConfigDefault(config *v1alpha1.TiFlashConfigWraper, externalClusterName, clusterName, ns string) {
	if config.Common == nil {
		config.Common = v1alpha1.NewTiFlashCommonConfig()
	}
	setTiFlashCommonConfigDefault(config.Common, externalClusterName, clusterName, ns)

	if config.Proxy == nil {
		config.Proxy = v1alpha1.NewTiFlashProxyConfig()
	}
	setTiFlashProxyConfigDefault(config.Proxy, clusterName, ns)
}

func setTiFlashProxyConfigDefault(config *v1alpha1.TiFlashProxyConfigWraper, clusterName, ns string) {
	config.SetIfNil("log-level", "info")
	config.SetIfNil("server.engine-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc:3930", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns))
	config.SetIfNil("server.status-addr", "0.0.0.0:20292")
	config.SetIfNil("server.advertise-status-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc:20292", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns))
}

func setTiFlashCommonConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper, externalClusterName string, clusterName, ns string) {
	config.SetIfNil("tmp_path", "/data0/tmp")
	config.SetIfNil("display_name", "TiFlash")
	config.SetIfNil("default_profile", "default")
	config.SetIfNil("path", "/data0/db")
	config.SetIfNil("path_realtime_mode", false)
	config.SetIfNil("mark_cache_size", int64(5368709120))
	config.SetIfNil("minmax_index_cache_size", int64(5368709120))
	config.SetIfNil("listen_host", "0.0.0.0")
	config.SetIfNil("tcp_port", int64(9000))
	config.SetIfNil("tcp_port_secure", int64(9000))
	config.SetIfNil("https_port", int64(8123))
	config.SetIfNil("http_port", int64(8123))
	config.SetIfNil("interserver_http_port", int64(9009))
	setTiFlashFlashConfigDefault(config, externalClusterName, clusterName, ns)
	setTiFlashLoggerConfigDefault(config)
	setTiFlashApplicationConfigDefault(config)
	if len(externalClusterName) > 0 {
		setTiFlashRaftConfigDefault(config, externalClusterName, ns)
	} else {
		setTiFlashRaftConfigDefault(config, clusterName, ns)
	}

	config.SetIfNil("status.metrics_port", int64(8234))

	config.SetIfNil("quotas.default.interval.duration", int64(3600))
	config.SetIfNil("quotas.default.interval.queries", int64(0))
	config.SetIfNil("quotas.default.interval.errors", int64(0))
	config.SetIfNil("quotas.default.interval.result_rows", int64(0))
	config.SetIfNil("quotas.default.interval.read_rows", int64(0))
	config.SetIfNil("quotas.default.interval.execution_time", int64(0))

	config.SetIfNil("users.readonly.profile", "readonly")
	config.SetIfNil("users.readonly.quota", "default")
	config.SetIfNil("users.readonly.networks.ip", "::/0")
	config.SetIfNil("users.readonly.password", "")
	config.SetIfNil("users.default.profile", "default")
	config.SetIfNil("users.default.quota", "default")
	config.SetIfNil("users.default.networks.ip", "::/0")
	config.SetIfNil("users.default.password", "")

	config.SetIfNil("profiles.readonly.readonly", int64(1))
	config.SetIfNil("profiles.default.max_memory_usage", int64(10000000000))
	config.SetIfNil("profiles.default.load_balancing", "random")
	config.SetIfNil("profiles.default.use_uncompressed_cache", int64(0))
}

func setTiFlashFlashConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper, externalClusterName string, clusterName, ns string) {
	var tidbStatusAddr string
	if len(externalClusterName) > 0 {
		tidbStatusAddr = fmt.Sprintf("%s.%s.svc:10080", controller.TiDBMemberName(externalClusterName), ns)
	} else {
		tidbStatusAddr = fmt.Sprintf("%s.%s.svc:10080", controller.TiDBMemberName(clusterName), ns)
	}
	config.SetIfNil("flash.tidb_status_addr", tidbStatusAddr)
	config.SetIfNil("flash.service_addr", "0.0.0.0:3930")
	config.SetIfNil("flash.overlap_threshold", 0.6)
	config.SetIfNil("flash.compact_log_min_period", int64(200))

	// set flash_cluster
	config.SetIfNil("flash.flash_cluster.cluster_manager_path", "/tiflash/flash_cluster_manager")
	config.SetIfNil("flash.flash_cluster.log", defaultClusterLog)
	config.SetIfNil("flash.flash_cluster.refresh_interval", int64(20))
	config.SetIfNil("flash.flash_cluster.update_rule_interval", int64(10))
	config.SetIfNil("flash.flash_cluster.master_ttl", int64(60))

	// set proxy
	config.SetIfNil("flash.proxy.addr", "0.0.0.0:20170")
	config.SetIfNil("flash.proxy.advertise-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc:20170", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns))
	config.SetIfNil("flash.proxy.data-dir", "/data0/proxy")
	config.SetIfNil("flash.proxy.config", "/data0/proxy.toml")
}

func setTiFlashLoggerConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper) {
	// "logger"
	config.SetIfNil("logger.errorlog", defaultErrorLog)
	config.SetIfNil("logger.size", "100M")
	config.SetIfNil("logger.log", defaultServerLog)
	config.SetIfNil("logger.level", "information")
	config.SetIfNil("logger.count", int64(10))
}

func setTiFlashApplicationConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper) {
	config.SetIfNil("application.runAsDaemon", true)
}

func setTiFlashRaftConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper, clusterName, ns string) {
	config.SetIfNil("raft.pd_addr", fmt.Sprintf("%s.%s.svc:2379", controller.PDMemberName(clusterName), ns))
	config.SetIfNil("raft.kvstore_path", "/data0/kvstore")
	config.SetIfNil("raft.storage_engine", "dt")
}
