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
	"slices"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util/cmpver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	defaultClusterLog = "/data0/logs/flash_cluster_manager.log"
	defaultErrorLog   = "/data0/logs/error.log"
	defaultServerLog  = "/data0/logs/server.log"
	listenHostForIPv4 = "0.0.0.0"
	listenHostForIPv6 = "[::]"
)

var (
	// the first version that tiflash change default config
	tiflashEqualOrGreaterThanV540, _ = cmpver.NewConstraint(cmpver.GreaterOrEqual, "v5.4.0")
	// the first version that tiflash discards http and tcp ports.
	tiflashEqualOrGreaterThanV710, _ = cmpver.NewConstraint(cmpver.GreaterOrEqual, "v7.1.0")
)

func buildTiFlashSidecarContainers(tc *v1alpha1.TidbCluster) ([]corev1.Container, error) {
	spec := tc.Spec.TiFlash
	config := spec.Config.DeepCopy()
	image := tc.HelperImage()
	pullPolicy := tc.HelperImagePullPolicy()
	var containers []corev1.Container
	var resource corev1.ResourceRequirements
	var useSidecar bool
	if spec.LogTailer != nil {
		resource = controller.ContainerResource(spec.LogTailer.ResourceRequirements)
		useSidecar = spec.LogTailer.UseSidecar
	}
	if config == nil {
		config = v1alpha1.NewTiFlashConfig()
	}
	setTiFlashLogConfigDefault(config)

	path, err := config.Common.Get("logger.log").AsString()
	if err != nil {
		return nil, err
	}
	containers = append(containers, buildSidecarContainer("serverlog", path, image, pullPolicy, resource, useSidecar))
	path, err = config.Common.Get("logger.errorlog").AsString()
	if err != nil {
		return nil, err
	}
	containers = append(containers, buildSidecarContainer("errorlog", path, image, pullPolicy, resource, useSidecar))
	path, err = config.Common.Get("flash.flash_cluster.log").AsString()
	if err != nil {
		return nil, err
	}
	containers = append(containers, buildSidecarContainer("clusterlog", path, image, pullPolicy, resource, useSidecar))
	return containers, nil
}

func buildSidecarContainer(name, path, image string,
	pullPolicy corev1.PullPolicy,
	resource corev1.ResourceRequirements,
	// If true we use native sidecar feature
	// See https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/
	useSidecar bool,
) corev1.Container {
	c := corev1.Container{
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
	splitPath := strings.Split(path, string(os.PathSeparator))
	// The log path should be at least /dir/base.log
	if len(splitPath) >= 3 {
		serverLogVolumeName := splitPath[1]
		serverLogMountDir := "/" + serverLogVolumeName
		c.VolumeMounts = []corev1.VolumeMount{
			{Name: serverLogVolumeName, MountPath: serverLogMountDir},
		}
	}

	if useSidecar {
		c.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
		// NOTE: tail cannot hanle sig TERM when it's PID is 1
		c.Command = []string{
			"sh",
			"-c",
			fmt.Sprintf(`trap "exit 0" TERM; touch %s; tail -n0 -F %s & wait $!`, path, path),
		}
	}
	return c
}

func GetTiFlashConfig(tc *v1alpha1.TidbCluster) *v1alpha1.TiFlashConfigWraper {
	version := tc.TiFlashVersion()
	if ok, err := tiflashEqualOrGreaterThanV540.Check(version); err == nil && ok {
		return getTiFlashConfigV2(tc)
	}
	return getTiFlashConfig(tc)
}

func getTiFlashConfigV2(tc *v1alpha1.TidbCluster) *v1alpha1.TiFlashConfigWraper {
	config := tc.Spec.TiFlash.Config.DeepCopy()
	if config == nil {
		config = v1alpha1.NewTiFlashConfig()
	}
	if config.Common == nil {
		config.Common = v1alpha1.NewTiFlashCommonConfig()
	}
	if config.Proxy == nil {
		config.Proxy = v1alpha1.NewTiFlashProxyConfig()
	}
	mountCMInTiflashContainer := tc.Spec.TiFlash.DoesMountCMInTiflashContainer()
	common := config.Common
	proxy := config.Proxy

	name := tc.Name
	ns := tc.Namespace
	clusterDomain := tc.Spec.ClusterDomain
	ref := tc.Spec.Cluster.DeepCopy()
	listenHost := listenHostForIPv4
	if tc.Spec.PreferIPv6 {
		listenHost = listenHostForIPv6
	}
	version := tc.TiFlashVersion()

	// common
	{
		// storage
		// check "path" to be compatible with old version
		if common.Get("path") == nil && common.Get("storage.main.dir") == nil {
			paths := []string{}
			for i := range tc.Spec.TiFlash.StorageClaims {
				paths = append(paths, fmt.Sprintf("/data%d/db", i))
			}
			if len(paths) == 0 {
				paths = []string{"/data0/db"}
			}
			common.Set("storage.main.dir", paths)
		}
		// check "raft.kvstore_path" to be compatible with old version
		if common.Get("raft.kvstore_path") == nil {
			common.SetIfNil("storage.raft.dir", []string{"/data0/kvstore"})
		}
		// workaround for issue #4091 about v5.4.0 TiFlash
		common.SetIfNil("tmp_path", "/data0/tmp")

		// port
		if ok, err := tiflashEqualOrGreaterThanV710.Check(version); err == nil && !ok || tc.KeepTiFlash710Ports() {
			common.SetIfNil("tcp_port", int64(v1alpha1.DefaultTiFlashTcpPort))
			common.SetIfNil("http_port", int64(v1alpha1.DefaultTiFlashHttpPort))
		}

		// flash
		tidbStatusAddr := fmt.Sprintf("%s.%s.svc:%d", controller.TiDBMemberName(name), ns, v1alpha1.DefaultTiDBStatusPort)
		if tc.WithoutLocalTiDB() {
			// TODO: support first cluster which don't contain TiDB when deploy cluster across mutli Kubernete clusters
			if tc.Heterogeneous() {
				if tc.AcrossK8s() {
					// use headless service of TiDB in reference cluster
					tidbStatusAddr = fmt.Sprintf("%s.%s.svc%s:%d",
						controller.TiDBPeerMemberName(ref.Name), ref.Namespace, controller.FormatClusterDomain(ref.ClusterDomain), v1alpha1.DefaultTiDBStatusPort)
				} else {
					// use service of TiDB in reference cluster
					tidbStatusAddr = fmt.Sprintf("%s.%s.svc%s:%d",
						controller.TiDBMemberName(ref.Name), ref.Namespace, controller.FormatClusterDomain(ref.ClusterDomain), v1alpha1.DefaultTiDBStatusPort)
				}
			}
		}
		common.SetIfNil("flash.tidb_status_addr", tidbStatusAddr)
		if !mountCMInTiflashContainer {
			common.SetIfNil("flash.service_addr", fmt.Sprintf("%s:%d", listenHost, v1alpha1.DefaultTiFlashFlashPort))
		}
		common.SetIfNil("flash.flash_cluster.log", defaultClusterLog)
		common.SetIfNil("flash.proxy.addr", fmt.Sprintf("%s:%d", listenHost, v1alpha1.DefaultTiFlashProxyPort))
		if !mountCMInTiflashContainer {
			common.SetIfNil("flash.proxy.advertise-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:%d", controller.TiFlashMemberName(name),
				controller.TiFlashPeerMemberName(name), ns, controller.FormatClusterDomain(clusterDomain), v1alpha1.DefaultTiFlashProxyPort))
		}
		common.SetIfNil("flash.proxy.data-dir", "/data0/proxy")
		if !mountCMInTiflashContainer {
			common.SetIfNil("flash.proxy.config", "/data0/proxy.toml")
		} else {
			common.SetIfNil("flash.proxy.config", "/etc/tiflash/proxy_templ.toml")
		}

		// logger
		common.SetIfNil("logger.errorlog", defaultErrorLog)
		common.SetIfNil("logger.log", defaultServerLog)

		// raft
		pdAddr := fmt.Sprintf("%s.%s.svc:%d", controller.PDMemberName(name), ns, v1alpha1.DefaultPDClientPort)
		if tc.AcrossK8s() {
			pdAddr = "PD_ADDR" // get pd addr from discovery in startup script
		} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
			pdAddr = fmt.Sprintf("%s.%s.svc%s:%d", controller.PDMemberName(ref.Name), ref.Namespace,
				controller.FormatClusterDomain(ref.ClusterDomain), v1alpha1.DefaultPDClientPort) // use pd of reference cluster
		}

		preferPDAddressesOverDiscovery := slices.Contains(
			tc.Spec.StartScriptV2FeatureFlags, v1alpha1.StartScriptV2FeatureFlagPreferPDAddressesOverDiscovery)
		if preferPDAddressesOverDiscovery && tc.Spec.StartScriptVersion == v1alpha1.StartScriptV2 {
			pdAddr = strings.Join(tc.Spec.PDAddresses, ",")
		}
		// tiflash require at least one configuration item in ["raft"] config group, otherwise
		// tiflash with version less than v7.1.0 may encounter schema sync problems. So we keep this item
		// even if this item is configured via command line args.
		common.SetIfNil("raft.pd_addr", pdAddr)

		if listenHost == listenHostForIPv6 {
			common.SetIfNil("listen_host", "::") // listen host must be "::" not "[::]"
			common.SetIfNil("status.metrics_port", int64(v1alpha1.DefaultTiFlashMetricsPort))
		}
	}

	// proxy
	{
		proxy.SetIfNil("log-level", "info")
		if !mountCMInTiflashContainer {
			proxy.SetIfNil("server.engine-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:%d", controller.TiFlashMemberName(name), controller.TiFlashPeerMemberName(name), ns,
				controller.FormatClusterDomain(clusterDomain), v1alpha1.DefaultTiFlashFlashPort))
			proxy.SetIfNil("server.advertise-status-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:%d", controller.TiFlashMemberName(name), controller.TiFlashPeerMemberName(name), ns,
				controller.FormatClusterDomain(clusterDomain), v1alpha1.DefaultTiFlashProxyStatusPort))
		}
		proxy.SetIfNil("server.status-addr", fmt.Sprintf("%s:%d", listenHost, v1alpha1.DefaultTiFlashProxyStatusPort))
	}

	// Note the config of tiflash use "_" by convention, others(proxy) use "-".
	if tc.IsTLSClusterEnabled() {
		common.Set("security.ca_path", path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey))
		common.Set("security.cert_path", path.Join(tiflashCertPath, corev1.TLSCertKey))
		common.Set("security.key_path", path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey))
		if ok, err := tiflashEqualOrGreaterThanV710.Check(version); err == nil && !ok || tc.KeepTiFlash710Ports() {
			common.SetIfNil("tcp_port_secure", int64(v1alpha1.DefaultTiFlashTcpPort))
			common.SetIfNil("https_port", int64(v1alpha1.DefaultTiFlashHttpPort))
		}
		common.Del("http_port")
		common.Del("tcp_port")

		proxy.Set("security.ca-path", path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey))
		proxy.Set("security.cert-path", path.Join(tiflashCertPath, corev1.TLSCertKey))
		proxy.Set("security.key-path", path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey))

		if commonVal, proxyVal := common.Get("security.cert_allowed_cn"), proxy.Get("security.cert-allowed-cn"); commonVal != nil && proxyVal == nil {
			proxy.Set("security.cert-allowed-cn", commonVal.Interface())
		}
	}

	return config
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

	ref := tc.Spec.Cluster.DeepCopy()
	noLocalPD := tc.WithoutLocalPD()
	acrossK8s := tc.AcrossK8s()
	noLocalTiDB := tc.WithoutLocalTiDB()
	listenHost := listenHostForIPv4
	if tc.Spec.PreferIPv6 {
		listenHost = listenHostForIPv6
	}

	setTiFlashConfigDefault(config, ref, tc.Name, tc.Namespace, tc.Spec.ClusterDomain, listenHost, noLocalPD, noLocalTiDB, acrossK8s)

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
func setTiFlashConfigDefault(config *v1alpha1.TiFlashConfigWraper, ref *v1alpha1.TidbClusterRef,
	clusterName, ns, clusterDomain, listenHost string, noLocalPD bool, noLocalTiDB bool, acrossK8s bool,
) {
	if config.Common == nil {
		config.Common = v1alpha1.NewTiFlashCommonConfig()
	}
	setTiFlashCommonConfigDefault(config.Common, ref, clusterName, ns, clusterDomain, listenHost, noLocalPD, noLocalTiDB, acrossK8s)

	if config.Proxy == nil {
		config.Proxy = v1alpha1.NewTiFlashProxyConfig()
	}
	setTiFlashProxyConfigDefault(config.Proxy, clusterName, ns, clusterDomain, listenHost)
}

func setTiFlashProxyConfigDefault(config *v1alpha1.TiFlashProxyConfigWraper, clusterName, ns, clusterDomain, listenHost string) {
	config.SetIfNil("log-level", "info")
	config.SetIfNil("server.engine-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:%d", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns,
		controller.FormatClusterDomain(clusterDomain), v1alpha1.DefaultTiFlashFlashPort))
	config.SetIfNil("server.status-addr", fmt.Sprintf("%s:%d", listenHost, v1alpha1.DefaultTiFlashProxyStatusPort))
	config.SetIfNil("server.advertise-status-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:%d", controller.TiFlashMemberName(clusterName), controller.TiFlashPeerMemberName(clusterName), ns,
		controller.FormatClusterDomain(clusterDomain), v1alpha1.DefaultTiFlashProxyStatusPort))
}

func setTiFlashCommonConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper, ref *v1alpha1.TidbClusterRef, clusterName, ns, clusterDomain, listenHost string, noLocalPD bool, noLocalTiDB bool, acrossK8s bool) {
	config.SetIfNil("tmp_path", "/data0/tmp")
	config.SetIfNil("display_name", "TiFlash")
	config.SetIfNil("default_profile", "default")
	config.SetIfNil("path", "/data0/db")
	config.SetIfNil("path_realtime_mode", false)
	config.SetIfNil("mark_cache_size", int64(5368709120))
	config.SetIfNil("minmax_index_cache_size", int64(5368709120))
	config.SetIfNil("tcp_port", int64(v1alpha1.DefaultTiFlashTcpPort))
	config.SetIfNil("tcp_port_secure", int64(v1alpha1.DefaultTiFlashTcpPort))
	config.SetIfNil("https_port", int64(v1alpha1.DefaultTiFlashHttpPort))
	config.SetIfNil("http_port", int64(v1alpha1.DefaultTiFlashHttpPort))
	config.SetIfNil("interserver_http_port", int64(v1alpha1.DefaultTiFlashInternalPort))
	setTiFlashFlashConfigDefault(config, ref, clusterName, ns, clusterDomain, listenHost, noLocalTiDB, acrossK8s)
	setTiFlashLoggerConfigDefault(config)
	setTiFlashApplicationConfigDefault(config)

	setTiFlashRaftConfigDefault(config, ref, clusterName, ns, clusterDomain, noLocalPD, acrossK8s)

	if listenHost == listenHostForIPv6 {
		config.SetIfNil("listen_host", "::") // listen host must be "::" not "[::]"
	} else {
		config.SetIfNil("listen_host", listenHost)
	}
	config.SetIfNil("status.metrics_port", int64(v1alpha1.DefaultTiFlashMetricsPort))

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

func setTiFlashFlashConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper, ref *v1alpha1.TidbClusterRef, clusterName, ns, clusterDomain, listenHost string, noLocalTiDB, acrossK8s bool) {
	tidbStatusAddr := fmt.Sprintf("%s.%s.svc:%d", controller.TiDBMemberName(clusterName), ns, v1alpha1.DefaultTiDBStatusPort)
	if noLocalTiDB {
		// TODO: support first cluster without TiDB when deploy cluster across mutli Kubernete clusters
		if ref != nil {
			if acrossK8s {
				// use headless service of TiDB in reference cluster
				tidbStatusAddr = fmt.Sprintf("%s.%s.svc%s:%d", controller.TiDBPeerMemberName(ref.Name), ref.Namespace,
					controller.FormatClusterDomain(ref.ClusterDomain), v1alpha1.DefaultTiDBStatusPort)
			} else {
				// use service of TiDB in reference cluster
				tidbStatusAddr = fmt.Sprintf("%s.%s.svc%s:%d", controller.TiDBMemberName(ref.Name), ref.Namespace,
					controller.FormatClusterDomain(ref.ClusterDomain), v1alpha1.DefaultTiDBStatusPort)
			}
		}
	}

	config.SetIfNil("flash.tidb_status_addr", tidbStatusAddr)
	config.SetIfNil("flash.service_addr", fmt.Sprintf("%s:%d", listenHost, v1alpha1.DefaultTiFlashFlashPort))
	config.SetIfNil("flash.overlap_threshold", 0.6)
	config.SetIfNil("flash.compact_log_min_period", int64(200))

	// set flash_cluster
	config.SetIfNil("flash.flash_cluster.cluster_manager_path", "/tiflash/flash_cluster_manager")
	config.SetIfNil("flash.flash_cluster.log", defaultClusterLog)
	config.SetIfNil("flash.flash_cluster.refresh_interval", int64(20))
	config.SetIfNil("flash.flash_cluster.update_rule_interval", int64(10))
	config.SetIfNil("flash.flash_cluster.master_ttl", int64(60))

	// set proxy
	config.SetIfNil("flash.proxy.addr", fmt.Sprintf("%s:%d", listenHost, v1alpha1.DefaultTiFlashProxyPort))
	config.SetIfNil("flash.proxy.advertise-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:%d", controller.TiFlashMemberName(clusterName),
		controller.TiFlashPeerMemberName(clusterName), ns, controller.FormatClusterDomain(clusterDomain), v1alpha1.DefaultTiFlashProxyPort))
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

func setTiFlashRaftConfigDefault(config *v1alpha1.TiFlashCommonConfigWraper, ref *v1alpha1.TidbClusterRef, clusterName, ns string, clusterDomain string, noLocalPD bool, acrossK8s bool) {
	config.SetIfNil("raft.kvstore_path", "/data0/kvstore")
	config.SetIfNil("raft.storage_engine", "dt")

	if acrossK8s {
		config.SetIfNil("raft.pd_addr", "PD_ADDR") // get pd addr from discovery in startup script
	} else if ref != nil && noLocalPD {
		config.SetIfNil("raft.pd_addr", fmt.Sprintf("%s.%s.svc%s:%d", controller.PDMemberName(ref.Name), ref.Namespace,
			controller.FormatClusterDomain(ref.ClusterDomain), v1alpha1.DefaultPDClientPort))
	} else {
		config.SetIfNil("raft.pd_addr", fmt.Sprintf("%s.%s.svc:%d", controller.PDMemberName(clusterName), ns, v1alpha1.DefaultPDClientPort))
	}
}
