// Copyright 2019. PingCAP, Inc.
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
	"bytes"
	"text/template"
)

// tidbStartScriptTpl is the template string of tidb start script
// Note: changing this will cause a rolling-update of tidb-servers
var tidbStartScriptTpl = template.Must(template.New("tidb-start-script").Parse(`#!/bin/sh

# This script is used to start tidb containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#
set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null
runmode=${runmode:-normal}
if [[ X${runmode} == Xdebug ]]
then
    echo "entering debug mode."
    tail -f /dev/null
fi

ARGS="--store=tikv \
--host=0.0.0.0 \
--path=${CLUSTER_NAME}-pd:2379 \
--config=/etc/tidb/tidb.toml
"

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    ARGS="${ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    ARGS="${ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

{{- if .EnablePlugin }}
ARGS="${ARGS}  --plugin-dir  {{ .PluginDirectory  }} --plugin-load {{ .PluginList }}  "
{{- end }}

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`))

type TidbStartScriptModel struct {
	ClusterName     string
	EnablePlugin    bool
	PluginDirectory string
	PluginList      string
}

func RenderTiDBStartScript(model *TidbStartScriptModel) (string, error) {
	return renderTemplateFunc(tidbStartScriptTpl, model)
}

// pumpStartScriptTpl is the template string of pump start script
// Note: changing this will cause a rolling-update of pump cluster
var pumpStartScriptTpl = template.Must(template.New("pump-start-script").Parse(`set -euo pipefail

/pump \
-pd-urls={{ .Scheme }}://{{ .ClusterName }}-pd:2379 \
-L={{ .LogLevel }} \
-advertise-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.{{ .ClusterName }}-pump:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi`))

type PumpStartScriptModel struct {
	Scheme      string
	ClusterName string
	LogLevel    string
}

func RenderPumpStartScript(model *PumpStartScriptModel) (string, error) {
	return renderTemplateFunc(pumpStartScriptTpl, model)
}

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
