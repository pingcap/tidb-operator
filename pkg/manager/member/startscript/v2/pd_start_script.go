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
	"text/template"
)

type PDStartScriptModel struct {
	CommonModel

	DataDir string
	Scheme  string
}

func RenderPDStartScript(m *PDStartScriptModel) (string, error) {
	return renderTemplateFunc(pdStartScriptTpl, m)
}

const (
	pdStartScriptVar = `
{{ define "PD_POD_NAME" -}}
    ${POD_NAME:-$HOSTNAME}
{{- end }}

{{ define "PD_DOMAIN" -}}
    {{- if .ClusterDomain -}}
        ${POD_NAME}.{{ .PeerServiceName }}.{{ .ClusterNamespace }}.svc.{{ .ClusterDomain }}
    {{- else -}}
        ${POD_NAME}.{{ .PeerServiceName }}.{{ .ClusterNamespace }}.svc
    {{- end -}}
{{- end }}

{{ define "PD_COMPONENT_NAME" -}}
    {{- if or .AcrossK8s .ClusterDomain -}}
        ${PD_DOMAIN}
    {{- else -}}
        ${PD_POD_NAME}
    {{- end -}}
{{- end }}

{{ define "PD_DATA_DIR" -}}
    {{ .DataDir }}
{{- end }}

{{ define "PD_PEER_URL" -}}
    {{ .Scheme }}://0.0.0.0:2380
{{- end }}

{{ define "PD_ADVERTISE_PEER_URL" -}}
    {{ .Scheme }}://${PD_DOMAIN}:2380
{{- end }}

{{ define "PD_CLIENT_URL" -}}
    {{ .Scheme }}://0.0.0.0:2379
{{- end }}

{{ define "PD_ADVERTISE_CLIENT_URL" -}}
    {{ .Scheme }}://${PD_DOMAIN}:2379
{{- end }}

{{ define "PD_DISCOVERY_ADDR" -}}
    {{ .ClusterName }}-discovery.{{ .PeerServiceName }}.{{ .ClusterNamespace }}.svc:10261
{{- end}}
`
	pdStartScript = `
PD_POD_NAME={{ template "PD_POD_NAME" . }}
PD_DOMAIN={{ template "PD_DOMAIN" . }}
PD_COMPONENT_NAME={{ template "PD_COMPONENT_NAME" . }}
PD_DATA_DIR={{ template "PD_DATA_DIR" . }}
PD_PEER_URL={{ template "PD_PEER_URL" . }}
PD_ADVERTISE_PEER_URL={{ template "PD_ADVERTISE_PEER_URL" . }}
PD_CLIENT_URL={{ template "PD_CLIENT_URL" . }}
PD_ADVERTISE_CLIENT_URL={{ template "PD_ADVERTISE_CLIENT_URL" . }}
PD_DISCOVERY_ADDR={{ template "PD_DISCOVERY_ADDR" . }}

set | grep PD_

elapseTime=0
period=1
threshold=30
while true; do
    sleep ${period}
    elapseTime=$(( elapseTime+period ))

    if [[ ${elapseTime} -ge ${threshold} ]]; then
        echo "waiting for pd cluster ready timeout" >&2
        exit 1
    fi

    digRes=$(dig ${PD_DOMAIN} A ${PD_DOMAIN} AAAA +search +short)
    if [ $? -ne 0  ]; then
        echo "domain resolve ${PD_DOMAIN} failed"
        echo "$digRes"
        continue
    fi

    if [ -z "${digRes}" ]
    then
        echo "domain resolve ${PD_DOMAIN} no record return"
    else
        echo "domain resolve ${PD_DOMAIN} success"
        echo "$digRes"
        break
    fi
done

ARGS="--data-dir=${PD_DATA_DIR} \
    --name=${PD_COMPONENT_NAME} \
    --peer-urls=${PD_PEER_URL} \
    --advertise-peer-urls=${PD_ADVERTISE_PEER_URL} \
    --client-urls=${PD_CLIENT_URL} \
    --advertise-client-urls=${PD_ADVERTISE_CLIENT_URL} \
    --config=/etc/pd/pd.toml"

if [[ -f ${PD_DATA_DIR}/join ]]; then
    join=$(cat ${PD_DATA_DIR}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d ${PD_DATA_DIR}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://${PD_DISCOVERY_ADDR}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`
)

var pdStartScriptTpl = template.Must(
	template.Must(
		template.New("pd-start-script").Parse(pdStartScriptVar),
	).Parse(componentCommonScript + pdStartScript),
)
