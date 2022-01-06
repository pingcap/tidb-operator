// Copyright 2022 PingCAP, Inc.
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
	"path"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util/cmpver"
	corev1 "k8s.io/api/core/v1"
)

var (
	// the first version that tiflash change default config
	tiflashEqualOrGreaterThanV540, _ = cmpver.NewConstraint(cmpver.GreaterOrEqual, "v5.4.0")
)

func GetTiFlashConfig(tc *v1alpha1.TidbCluster) *v1alpha1.TiFlashConfigWraper {
	version := tc.TiFlashVersion()
	if ok, err := tiflashEqualOrGreaterThanV540.Check(version); err == nil && ok {

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
	common := config.Common
	proxy := config.Proxy

	name := tc.Name
	ns := tc.Namespace
	clusterDomain := tc.Spec.ClusterDomain
	ref := tc.Spec.Cluster.DeepCopy()
	noLocalPD := tc.HeterogeneousWithoutLocalPD()
	noLocalTiDB := ref != nil && ref.Name != "" && tc.Spec.TiDB == nil

	// common
	{
		// TODO:storage
		// raft.kvstore_path

		// port
		common.SetIfNil("tcp_port", int64(9000))
		common.SetIfNil("http_port", int64(8123))

		// flash
		tidbStatusAddr := fmt.Sprintf("%s.%s.svc:10080", controller.TiDBMemberName(name), ns)
		if noLocalTiDB {
			tidbStatusAddr = fmt.Sprintf("%s.%s.svc%s:10080",
				controller.TiDBMemberName(ref.Name), ref.Namespace, controller.FormatClusterDomain(ref.ClusterDomain))
		}
		common.SetIfNil("flash.tidb_status_addr", tidbStatusAddr)
		common.SetIfNil("flash.service_addr", "0.0.0.0:3930")
		common.SetIfNil("flash.flash_cluster.log", defaultClusterLog)
		common.SetIfNil("flash.proxy.addr", "0.0.0.0:20170")
		common.SetIfNil("flash.proxy.advertise-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:20170", controller.TiFlashMemberName(name), controller.TiFlashPeerMemberName(name), ns, controller.FormatClusterDomain(clusterDomain)))
		common.SetIfNil("flash.proxy.data-dir", "/data0/proxy")
		common.SetIfNil("flash.proxy.config", "/data0/proxy.toml")

		// logger
		common.SetIfNil("logger.errorlog", defaultErrorLog)
		common.SetIfNil("logger.log", defaultServerLog)

		// raft
		pdAddr := fmt.Sprintf("%s.%s.svc:2379", controller.PDMemberName(name), ns)
		if len(clusterDomain) > 0 {
			pdAddr = "PD_ADDR"
		} else if noLocalPD {
			pdAddr = fmt.Sprintf("%s.%s.svc%s:2379", controller.PDMemberName(ref.Name), ref.Namespace, controller.FormatClusterDomain(ref.ClusterDomain))
		} 
		common.SetIfNil("raft.pd_addr", pdAddr)

		// TODO: server
	}

	// proxy
	{
		proxy.SetIfNil("log-level", "info")
		proxy.SetIfNil("server.engine-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:3930", controller.TiFlashMemberName(name), controller.TiFlashPeerMemberName(name), ns, controller.FormatClusterDomain(clusterDomain)))
		proxy.SetIfNil("server.status-addr", "0.0.0.0:20292")
		proxy.SetIfNil("server.advertise-status-addr", fmt.Sprintf("%s-POD_NUM.%s.%s.svc%s:20292", controller.TiFlashMemberName(name), controller.TiFlashPeerMemberName(name), ns, controller.FormatClusterDomain(clusterDomain)))
	}

	// Note the config of tiflash use "_" by convention, others(proxy) use "-".
	if tc.IsTLSClusterEnabled() {
		common.Set("security.ca_path", path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey))
		common.Set("security.cert_path", path.Join(tiflashCertPath, corev1.TLSCertKey))
		common.Set("security.key_path", path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey))
		common.SetIfNil("tcp_port_secure", int64(9000))
		common.SetIfNil("https_port", int64(8123))
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
