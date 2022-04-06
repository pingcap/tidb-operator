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

import "text/template"

type TiDBPlugin struct {
	PluginDirectory string
	PluginList      string
}

type TiDBStartScriptModel struct {
	CommonModel

	PDAddr string
	Plugin *TiDBPlugin
}

func RenderTiDBStartScript(m *TiDBStartScriptModel) (string, error) {
	return renderTemplateFunc(tidbStartScriptTpl, m)
}

const (
	tidbStartScriptVar = `
{{ define "TIDB_POD_NAME" -}}
TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
{{- end }}


{{ define "TIDB_ADVERTISE_ADDR" -}}
TIDB_ADVERTISE_ADDR=${TIDB_POD_NAME}.{{ .PeerServiceName }}.{{ .ClusterNamespace }}.svc{{ .FormatClusterDomain }}
{{- end }}


{{ define "TIDB_PD_ADDR" -}}
    {{- if .AcrossK8s -}}
pd_url="{{ .PDAddr }}"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="{{ .ClusterName }}-discovery.{{ .ClusterNamespace }}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIDB_PD_ADDR=${result}
    {{- else -}}
TIDB_PD_ADDR={{ .PDAddr }}
    {{- end -}}
{{- end}}


{{ define "TIDB_EXTRA_ARGS" -}}
TIDB_EXTRA_ARGS=

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

    {{- if .Plugin }}
TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS}  --plugin-dir  {{ .Plugin.PluginDirectory  }} --plugin-load {{ .Plugin.PluginList }}"
    {{- end }}
{{- end}}
`

	tidbStartScript = `
{{ template "TIDB_POD_NAME" . }}
{{ template "TIDB_ADVERTISE_ADDR" . }}
{{ template "TIDB_PD_ADDR" . }}
{{ template "TIDB_EXTRA_ARGS" . }}

set | grep TIDB_

ARGS="--store=tikv \
    --advertise-address=${TIDB_ADVERTISE_ADDR} \
    --host=0.0.0.0 \
    --path=${TIDB_PD_ADDR} \
    --config=/etc/tidb/tidb.toml

if [[ -n "${TIDB_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIDB_EXTRA_ARGS}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`
)

var tidbStartScriptTpl = template.Must(
	template.Must(
		template.New("tidb-start-script").Parse(tidbStartScriptVar),
	).Parse(componentCommonScript + tidbStartScript),
)
