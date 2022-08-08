// Copyright 2021 PingCAP, Inc.
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

package v1

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/member/constants"

	corev1 "k8s.io/api/core/v1"
)

func RenderTiKVStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	model := &TiKVStartScriptModel{
		CommonModel: CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		EnableAdvertiseStatusAddr: false,
		DataDir:                   filepath.Join(constants.TiKVDataVolumeMountPath, tc.Spec.TiKV.DataSubDir),
	}
	if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
		model.AdvertiseStatusAddr = "${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc" + controller.FormatClusterDomain(tc.Spec.ClusterDomain)
		model.EnableAdvertiseStatusAddr = true
	}

	model.PDAddress = tc.Scheme() + "://${CLUSTER_NAME}-pd:2379"
	if tc.AcrossK8s() {
		model.PDAddress = tc.Scheme() + "://${CLUSTER_NAME}-pd:2379" // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		model.PDAddress = tc.Scheme() + "://" + controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379" // use pd of reference cluster
	}

	return renderTemplateFunc(tikvStartScriptTpl, model)
}

func RenderPDStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	model := &PDStartScriptModel{
		CommonModel: CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		Scheme:  tc.Scheme(),
		DataDir: filepath.Join(constants.PDDataVolumeMountPath, tc.Spec.PD.DataSubDir),
	}
	if tc.Spec.PD.StartUpScriptVersion == "v1" {
		model.CheckDomainScript = checkDNSV1
	}
	return renderTemplateFunc(pdStartScriptTpl, model)
}

func RenderTiDBStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	plugins := tc.Spec.TiDB.Plugins
	model := &TidbStartScriptModel{
		CommonModel: CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		EnablePlugin:    len(plugins) > 0,
		PluginDirectory: "/plugins",
		PluginList:      strings.Join(plugins, ","),
	}

	model.Path = "${CLUSTER_NAME}-pd:2379"
	if tc.AcrossK8s() {
		model.Path = "${CLUSTER_NAME}-pd:2379" // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		model.Path = controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379" // use pd of reference cluster
	}

	return renderTemplateFunc(tidbStartScriptTpl, model)
}

func RenderPumpStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	scheme := "http"
	if tc.IsTLSClusterEnabled() {
		scheme = "https"
	}

	pdDomain := controller.PDMemberName(tc.Name)
	if tc.AcrossK8s() {
		pdDomain = controller.PDMemberName(tc.Name) // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		pdDomain = controller.PDMemberName(tc.Spec.Cluster.Name) // use pd of reference cluster
	}

	pdAddr := fmt.Sprintf("%s://%s:2379", scheme, pdDomain)
	return renderTemplateFunc(pumpStartScriptTpl, &PumpStartScriptModel{
		CommonModel: CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		Scheme:      scheme,
		ClusterName: tc.Name,
		PDAddr:      pdAddr,
		LogLevel:    tc.PumpLogLevel(),
		Namespace:   tc.GetNamespace(),
	})
}

func RenderTiCDCStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	tcName := tc.GetName()

	// NB: TiCDC control relies the format.
	// TODO move advertise addr format to package controller.
	advertiseAddr := fmt.Sprintf("${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:8301",
		controller.FormatClusterDomain(tc.Spec.ClusterDomain))
	cmdArgs := []string{"/cdc server", "--addr=0.0.0.0:8301", fmt.Sprintf("--advertise-addr=%s", advertiseAddr)}
	cmdArgs = append(cmdArgs, fmt.Sprintf("--gc-ttl=%d", tc.TiCDCGCTTL()))
	cmdArgs = append(cmdArgs, fmt.Sprintf("--log-file=%s", tc.TiCDCLogFile()))
	cmdArgs = append(cmdArgs, fmt.Sprintf("--log-level=%s", tc.TiCDCLogLevel()))

	scheme := "http"
	if tc.IsTLSClusterEnabled() {
		scheme = "https"
	}
	pdAddr := fmt.Sprintf("%s://%s:2379", scheme, controller.PDMemberName(tc.Name))
	if tc.AcrossK8s() {
		pdAddr = "${result}" // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		pdAddr = fmt.Sprintf("%s://%s:2379", scheme, controller.PDMemberName(tc.Spec.Cluster.Name)) // use pd of reference cluster
	}

	if tc.IsTLSClusterEnabled() {
		ticdcCertPath := constants.TiCDCCertPath
		cmdArgs = append(cmdArgs, fmt.Sprintf("--ca=%s", path.Join(ticdcCertPath, corev1.ServiceAccountRootCAKey)))
		cmdArgs = append(cmdArgs, fmt.Sprintf("--cert=%s", path.Join(ticdcCertPath, corev1.TLSCertKey)))
		cmdArgs = append(cmdArgs, fmt.Sprintf("--key=%s", path.Join(ticdcCertPath, corev1.TLSPrivateKeyKey)))
	}
	cmdArgs = append(cmdArgs, fmt.Sprintf("--pd=%s", pdAddr))

	if tc.Spec.TiCDC.Config != nil && !tc.Spec.TiCDC.Config.OnlyOldItems() {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--config=%s", "/etc/ticdc/ticdc.toml"))
	}

	var script string

	if tc.AcrossK8s() {
		var pdAddr string
		pdDomain := controller.PDMemberName(tcName)
		if tc.IsTLSClusterEnabled() {
			pdAddr = fmt.Sprintf("https://%s:2379", pdDomain)
		} else {
			pdAddr = fmt.Sprintf("http://%s:2379", pdDomain)
		}

		str := `set -uo pipefail
pd_url="%s"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="%s-discovery.${NAMESPACE}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep 2
done
`

		script += fmt.Sprintf(str, pdAddr, tc.GetName())
		script += "\n" + strings.Join(append([]string{"exec"}, cmdArgs...), " ")
	} else {
		script = strings.Join(cmdArgs, " ")
	}

	return script, nil
}

func RenderTiFlashStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	return "/tiflash/tiflash server --config-file /data0/config.toml", nil
}

func RenderTiFlashInitScript(tc *v1alpha1.TidbCluster) (string, error) {
	tcName := tc.GetName()

	script := "set -ex;ordinal=`echo ${POD_NAME} | awk -F- '{print $NF}'`;sed s/POD_NUM/${ordinal}/g /etc/tiflash/config_templ.toml > /data0/config.toml;sed s/POD_NUM/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data0/proxy.toml"
	if tc.AcrossK8s() {
		var pdAddr string
		if tc.IsTLSClusterEnabled() {
			pdAddr = fmt.Sprintf("https://%s-pd:2379", tcName)
		} else {
			pdAddr = fmt.Sprintf("http://%s-pd:2379", tcName)
		}
		str := `pd_url="%s"
set +e
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="%s-discovery.%s:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g' | sed 's/https:\/\///g'); do
echo "waiting for the verification of PD endpoints ..."
sleep 2
done


sed -i s/PD_ADDR/${result}/g /data0/config.toml
sed -i s/PD_ADDR/${result}/g /data0/proxy.toml
`
		script += "\n"
		script += fmt.Sprintf(str, pdAddr, tc.GetName(), tc.GetNamespace())
	}

	return script, nil
}
