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
	"strings"
	"text/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

// TiDBStartScriptModel contain some fields for rendering TiDB start script
type TiDBStartScriptModel struct {
	PDAddr        string
	AdvertiseAddr string
	ExtraArgs     string

	AcrossK8s *AcrossK8sScriptModel
}

// RenderTiDBStartScript renders TiDB start script from TidbCluster
func RenderTiDBStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiDBStartScriptModel{}

	pdDomain := controller.PDMemberName(tc.Name)
	if tc.AcrossK8s() {
		pdDomain = controller.PDMemberName(tc.Name) // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		pdDomain = controller.PDMemberName(tc.Spec.Cluster.Name) // use pd of reference cluster
	}
	m.PDAddr = fmt.Sprintf("%s:2379", pdDomain)

	m.AdvertiseAddr = fmt.Sprintf("${TIDB_POD_NAME}.${HEADLESS_SERVICE_NAME}.%s.svc", tc.Namespace)
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
		m.ExtraArgs = fmt.Sprintf("\"%s\"", strings.Join(extraArgs, " "))
	}

	if tc.AcrossK8s() {
		m.AcrossK8s = &AcrossK8sScriptModel{
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tc.Name, tc.Namespace),
		}
	}

	return renderTemplateFunc(tidbStartScriptTpl, m)
}

const (
	tidbStartSubScript = `
{{ define "TIDB_PD_ADDR" -}}
    {{- if .AcrossK8s -}}
pd_url={{ .PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIDB_PD_ADDR=${result}
    {{- else -}}
TIDB_PD_ADDR={{ .PDAddr }}
    {{- end -}}
{{- end}}
`

	tidbStartScript = `
TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
TIDB_ADVERTISE_ADDR={{ .AdvertiseAddr }}
{{ template "TIDB_PD_ADDR" . }}
TIDB_EXTRA_ARGS={{ .ExtraArgs }}

ARGS="--store=tikv \
    --advertise-address=${TIDB_ADVERTISE_ADDR} \
    --host=0.0.0.0 \
    --path=${TIDB_PD_ADDR} \
    --config=/etc/tidb/tidb.toml"

if [[ -n "${TIDB_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIDB_EXTRA_ARGS}"
fi

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
