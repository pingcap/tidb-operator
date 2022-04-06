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

type TiKVStartScriptModel struct {
	CommonModel

	DataDir             string
	PDAddr              string
	AdvertiseStatusAddr string
}

func RenderTiKVStartScript(m *TiKVStartScriptModel) (string, error) {
	return renderTemplateFunc(tikvStartScriptTpl, m)
}

const (
	tikvStartScriptVar = `
{{ define "TIKV_POD_NAME" -}}
TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
{{- end }}


{{ define "TIKV_PD_ADDR" -}}
    {{- if .AcrossK8s -}}
pd_url="{{ .PDAddr }}"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="{{ .ClusterName }}-discovery.{{ .ClusterNamespace }}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIKV_PD_ADDR=${result}
    {{- else -}}
TIKV_PD_ADDR={{ .PDAddr }}
    {{- end -}}
{{- end }}


{{ define "TIKV_ADVERTISE_ADDR" -}}
TIKV_ADVERTISE_ADDR=${TIKV_POD_NAME}.{{ .PeerServiceName }}.{{ .ClusterNamespace }}.svc{{ .FormatClusterDomain }}:20160
{{- end }}


{{ define "TIKV_DATA_DIR" -}}
TIKV_DATA_DIR={{ .DataDir }}
{{- end}}


{{ define "TIKV_CAPACITY" -}}
TIKV_CAPACITY=${CAPACITY}
{{- end}}


{{ define "TIKV_EXTRA_ARGS" -}}
TIKV_EXTRA_ARGS=

{{- if .AdvertiseStatusAddr }}
TIKV_EXTRA_ARGS="--advertise-status-addr={{ .AdvertiseStatusAddr }}:20180"
{{- end }}

if [ ! -z "${STORE_LABELS:-}" ]; then
    LABELS="--labels ${STORE_LABELS} "
    TIKV_EXTRA_ARGS="${TIKV_EXTRA_ARGS} ${LABELS}"
fi
{{- end}}
`

	tikvStartScript = `
{{ template "TIKV_POD_NAME" . }}
{{ template "TIKV_PD_ADDR" . }}
{{ template "TIKV_ADVERTISE_ADDR" . }}
{{ template "TIKV_DATA_DIR" . }}
{{ template "TIKV_CAPACITY" . }}
{{ template "TIKV_EXTRA_ARGS" . }}

set | grep TIKV_

ARGS="--pd=${TIKV_PD_ADDR} \
    --advertise-addr=${TIKV_ADVERTISE_ADDR} \
    --addr=0.0.0.0:20160 \
    --status-addr=0.0.0.0:20180 \
    --data-dir=${TIKV_DATA_DIR} \
    --capacity=${TIKV_CAPACITY} \
    --config=/etc/tikv/tikv.toml

if [[ -n "${TIKV_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIKV_EXTRA_ARGS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`
)

var tikvStartScriptTpl = template.Must(
	template.Must(
		template.New("tikv-start-script").Parse(tikvStartScriptVar),
	).Parse(componentCommonScript + tikvStartScript),
)
