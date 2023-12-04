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

// TiDBStartScriptModel contain some fields for rendering TiDB start script
type TiDBStartScriptModel struct {
	AdvertiseAddr string
	ExtraArgs     string
	PDAddresses   string

	AcrossK8s *AcrossK8sScriptModel
}

// RenderTiDBStartScript renders TiDB start script from TidbCluster
func RenderTiDBStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiDBStartScriptModel{}
	tcName := tc.Name
	tcNS := tc.Namespace
	peerServiceName := controller.TiDBPeerMemberName(tcName)

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

	m.AdvertiseAddr = fmt.Sprintf("${TIDB_POD_NAME}.%s.%s.svc", peerServiceName, tcNS)
	if tc.Spec.ClusterDomain != "" {
		m.AdvertiseAddr = m.AdvertiseAddr + "." + tc.Spec.ClusterDomain
	}

	extraArgs := []string{}
	if tc.IsTiDBBinlogEnabled() {
		extraArgs = append(extraArgs, "--enable-binlog=true")
	}
	if plugins := tc.Spec.TiDB.Plugins; len(plugins) > 0 {
		extraArgs = append(extraArgs, "--plugin-dir=/plugins")
		extraArgs = append(extraArgs, fmt.Sprintf("--plugin-load=%s", strings.Join(plugins, ",")))
	}
	if len(extraArgs) > 0 {
		m.ExtraArgs = strings.Join(extraArgs, " ")
	}

	return renderTemplateFunc(tidbStartScriptTpl, m)
}

const (
	// tidbStartSubScript contains optional subscripts used in start script.
	tidbStartSubScript = `
{{ define "AcrossK8sSubscript" }}
pd_url={{ .AcrossK8s.PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
{{- end}}
`

	// tidbStartScript is the template of start script.
	tidbStartScript = `
TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
{{- if .AcrossK8s -}} {{ template "AcrossK8sSubscript" . }} {{- end }}

ARGS="--store=tikv \
--advertise-address={{ .AdvertiseAddr }} \
--host=0.0.0.0 \
--path={{ .PDAddresses }} \
--config=/etc/tidb/tidb.toml"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    ARGS="${ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`
)

var tidbStartScriptTpl = template.Must(
	template.Must(
		template.New("tidb-start-script").Parse(tidbStartSubScript),
	).Parse(componentCommonScript + tidbStartScript),
)
