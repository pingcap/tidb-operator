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
	"text/template"
)

type PumpStartScriptModel struct {
	CommonModel

	PDAddr   string
	LogLevel string
}

func (m PumpStartScriptModel) FormatPumpZone() string {
	if m.ClusterDomain != "" {
		return fmt.Sprintf(".%s.svc.%s", m.ClusterNamespace, m.ClusterDomain)
	}
	if m.ClusterDomain == "" && m.AcrossK8s {
		return fmt.Sprintf(".%s.svc", m.ClusterNamespace)
	}
	return ""
}
func RenderPumpStartScript(m *PumpStartScriptModel) (string, error) {
	return renderTemplateFunc(pumpStartScriptTpl, m)
}

const (
	pumpStartScriptVar = `
{{ define "PUMP_POD_NAME" -}}
PUMP_POD_NAME=$HOSTNAME
{{- end }}


{{ define "PUMP_PD_ADDR" -}}
    {{- if .AcrossK8s -}}
pd_url="{{ .PDAddr }}"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="{{ .ClusterName }}-discovery.{{ .ClusterNamespace }}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PUMP_PD_ADDR=${result}
    {{- else -}}
PUMP_PD_ADDR={{ .PDAddr }}
    {{- end -}}
{{- end}}


{{ define "PUMP_LOG_LEVEL" -}}
PUMP_LOG_LEVEL={{ .LogLevel }}
{{- end }}


{{ define "PUMP_ADVERTISE_ADDR" -}}
PUMP_ADVERTISE_ADDR=${PUMP_POD_NAME}.{{ .ClusterName }}-pump{{ .FormatPumpZone }}:8250
{{- end }}


{{ define "PUMP_EXTRA_ARGS" -}}
PUMP_EXTRA_ARGS=
{{- end}}
`

	pumpStartScript = `
{{ template "PUMP_POD_NAME" . }}
{{ template "PUMP_PD_ADDR" . }}
{{ template "PUMP_LOG_LEVEL" . }}
{{ template "PUMP_ADVERTISE_ADDR" . }}
{{ template "PUMP_EXTRA_ARGS" . }}

set | grep PUMP_

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml

if [[ -n "${PUMP_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${PUMP_EXTRA_ARGS}"
fi

echo "start pump-server ..."
echo "/pump ${ARGS}"
exec /pump ${ARGS}

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi
`
)

var pumpStartScriptTpl = template.Must(
	template.Must(
		template.New("pump-start-script").Parse(pumpStartScriptVar),
	).Parse(componentCommonScript + pumpStartScript),
)
