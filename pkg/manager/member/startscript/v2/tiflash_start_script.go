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

package v2

import (
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

// TiFlashStartScriptModel contain fields for rendering TiFlash start script
type TiFlashStartScriptModel struct {
	ExtraArgs string
}

// RenderTiFlashStartScript renders TiFlash start script from TidbCluster
func RenderTiFlashStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	if tc.Spec.TiFlash.DoesMountCMInTiflashContainer() {
		return RenderTiFlashStartScriptWithStartArgs(tc)
	}

	m := &TiFlashStartScriptModel{}

	m.ExtraArgs = ""

	return renderTemplateFunc(tiflashStartScriptTpl, m)
}

// TiFlashStartScriptWithStartArgsModel is an alternative of TiFlashStartScriptModel that enable tiflash pod run without
// initContainer
type TiFlashStartScriptWithStartArgsModel struct {
	AdvertiseAddr       string
	AdvertiseStatusAddr string
	Addr                string
	PDAddresses         string
	ExtraArgs           string

	AcrossK8s *AcrossK8sScriptModel
}

func RenderTiFlashStartScriptWithStartArgs(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiFlashStartScriptWithStartArgsModel{
		AdvertiseAddr: fmt.Sprintf("${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:%d", controller.FormatClusterDomain(tc.Spec.ClusterDomain), v1alpha1.DefaultTiFlashProxyPort),
		ExtraArgs:     "",
	}
	tcName := tc.Name
	tcNS := tc.Namespace

	// only the tiflash learner supports dynamic configuration
	if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
		m.AdvertiseStatusAddr = fmt.Sprintf("${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:%d", controller.FormatClusterDomain(tc.Spec.ClusterDomain), v1alpha1.DefaultTiFlashProxyStatusPort)
	}

	preferPDAddressesOverDiscovery := slices.Contains(
		tc.Spec.StartScriptV2FeatureFlags, v1alpha1.StartScriptV2FeatureFlagPreferPDAddressesOverDiscovery)
	if preferPDAddressesOverDiscovery {
		pdAddressesWithSchemeAndPort := addressesWithSchemeAndPort(tc.Spec.PDAddresses, "", v1alpha1.DefaultPDClientPort)
		m.PDAddresses = strings.Join(pdAddressesWithSchemeAndPort, ",")
	}
	if len(m.PDAddresses) == 0 {
		if tc.AcrossK8s() {
			m.AcrossK8s = &AcrossK8sScriptModel{
				PDAddr:        fmt.Sprintf("%s:%d", controller.PDMemberName(tcName), v1alpha1.DefaultPDClientPort),
				DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tcName, tcNS),
			}
			m.PDAddresses = "${result}" // get pd addr in subscript
		} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
			m.PDAddresses = fmt.Sprintf("%s:%d", controller.PDMemberName(tc.Spec.Cluster.Name), v1alpha1.DefaultPDClientPort) // use pd of reference cluster
		} else {
			m.PDAddresses = fmt.Sprintf("%s:%d", controller.PDMemberName(tcName), v1alpha1.DefaultPDClientPort)
		}
	}

	m.Addr = fmt.Sprintf("${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:%d", controller.FormatClusterDomain(tc.Spec.ClusterDomain), v1alpha1.DefaultTiFlashFlashPort)

	return renderTemplateFunc(tiflashStartScriptWithStartArgsTpl, m)
}

const (
	// tiflashStartSubScript contains optional subscripts used in start script.
	tiflashStartSubScript = ``

	// tiflashStartScript is the template of start script.
	//
	// Because init container of tiflash have core start script, so just to start tiflash there.
	tiflashStartScript = `
ARGS="--config-file /data0/config.toml"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash server ${ARGS}
`

	tiflashStartSubScriptWithStartArgs = `
{{ define "AcrossK8sSubscript" }}
pd_url={{ .AcrossK8s.PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g' | sed 's/https:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PD_ADDRESS=${result}
{{- end}}
`

	// tiflashStartScriptWithStartArgs is the new way to launch tiflash, that enable tiflash pod run without init container.
	tiflashStartScriptWithStartArgs = `
# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
PD_ADDRESS="{{ .PDAddresses }}"

{{- if .AcrossK8s -}} {{ template "AcrossK8sSubscript" . }} {{- end }}
ARGS="server --config-file /etc/tiflash/config_templ.toml \
-- \
--flash.proxy.advertise-addr={{ .AdvertiseAddr }} \{{- if .AdvertiseStatusAddr }}
--flash.proxy.advertise-status-addr={{ .AdvertiseStatusAddr }} \
{{- end }}
--flash.service_addr={{ .Addr }} \
--raft.pd_addr=${PD_ADDRESS}
"

{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash ${ARGS}
`
)

var tiflashStartScriptTpl = template.Must(
	template.Must(
		template.New("tiflash-start-script").Parse(tiflashStartSubScript),
	).Parse(componentCommonScript + tiflashStartScript),
)

var tiflashStartScriptWithStartArgsTpl = template.Must(
	template.Must(
		template.New("tiflash-start-script-with-start-args").Parse(tiflashStartSubScriptWithStartArgs),
	).Parse(componentCommonScript + tiflashStartScriptWithStartArgs),
)
