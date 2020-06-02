// Copyright 2019 PingCAP, Inc.
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

// TODO(aylei): it is hard to maintain script in go literal, we should figure out a better solution
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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="--store=tikv \
--advertise-address=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc \
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

// pdStartScriptTpl is the pd start script
// Note: changing this will cause a rolling-update of pd cluster
var pdStartScriptTpl = template.Must(template.New("pd-start-script").Parse(`#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
# the general form of variable PEER_SERVICE_NAME is: "<clusterName>-pd-peer"
cluster_name=` + "`" + `echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'` + "`" +
	`
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=` + "`" + `echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"` + "`" +
	`
elapseTime=0
period=1
threshold=30
while true; do
sleep ${period}
elapseTime=$(( elapseTime+period ))

if [[ ${elapseTime} -ge ${threshold} ]]
then
echo "waiting for pd cluster ready timeout" >&2
exit 1
fi

if nslookup ${domain} 2>/dev/null
then
echo "nslookup domain ${domain}.svc success"
break
else
echo "nslookup domain ${domain} failed" >&2
fi
done

ARGS="--data-dir=/var/lib/pd \
--name=${POD_NAME} \
--peer-urls={{ .Scheme }}://0.0.0.0:2380 \
--advertise-peer-urls={{ .Scheme }}://${domain}:2380 \
--client-urls={{ .Scheme }}://0.0.0.0:2379 \
--advertise-client-urls={{ .Scheme }}://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ -f /var/lib/pd/join ]]
then
# The content of the join file is:
#   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
# The --join args must be:
#   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
join=` + "`" + `cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]
then
until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
echo "waiting for discovery service to return start args ..."
sleep $((RANDOM % 5))
done
ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`))

type PDStartScriptModel struct {
	Scheme string
}

func RenderPDStartScript(model *PDStartScriptModel) (string, error) {
	return renderTemplateFunc(pdStartScriptTpl, model)
}

var tikvStartScriptTpl = template.Must(template.New("tikv-start-script").Parse(`#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="--pd={{ .Scheme }}://${CLUSTER_NAME}-pd:2379 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--advertise-status-addr={{ .StatusHost }}:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`))

type TiKVStartScriptModel struct {
	Scheme     string
	StatusHost string
}

func RenderTiKVStartScript(model *TiKVStartScriptModel) (string, error) {
	return renderTemplateFunc(tikvStartScriptTpl, model)
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

// tidbInitStartScriptTpl is the template string of tidb initializer start script
var tidbInitStartScriptTpl = template.Must(template.New("tidb-init-start-script").Parse(`import os, MySQLdb
host = '{{ .ClusterName }}-tidb'
permit_host = '{{ .PermitHost }}'
port = 4000
{{- if .TLS }}
conn = MySQLdb.connect(host=host, port=port, user='root', connect_timeout=5, ssl={'ca': '{{ .CAPath }}', 'cert': '{{ .CertPath }}', 'key': '{{ .KeyPath }}'})
{{- else }}
conn = MySQLdb.connect(host=host, port=port, user='root', connect_timeout=5)
{{- end }}
{{- if .PasswordSet }}
password_dir = '/etc/tidb/password'
for file in os.listdir(password_dir):
    if file.startswith('.'):
        continue
    user = file
    with open(os.path.join(password_dir, file), 'r') as f:
        password = f.read()
    if user == 'root':
        conn.cursor().execute("set password for 'root'@'%%' = %s;", (password,))
    else:
        conn.cursor().execute("create user %s@%s identified by %s;", (user, permit_host, password,))
{{- end }}
{{- if .InitSQL }}
with open('/data/init.sql', 'r') as sql:
    for line in sql.readlines():
        conn.cursor().execute(line)
        conn.commit()
{{- end }}
if permit_host != '%%':
    conn.cursor().execute("update mysql.user set Host=%s where User='root';", (permit_host,))
conn.cursor().execute("flush privileges;")
conn.commit()
conn.close()
`))

type TiDBInitStartScriptModel struct {
	ClusterName string
	PermitHost  string
	PasswordSet bool
	InitSQL     bool
	TLS         bool
	CAPath      string
	CertPath    string
	KeyPath     string
}

func RenderTiDBInitStartScript(model *TiDBInitStartScriptModel) (string, error) {
	return renderTemplateFunc(tidbInitStartScriptTpl, model)
}

// tidbInitInitStartScriptTpl is the template string of tidb initializer init container start script
var tidbInitInitStartScriptTpl = template.Must(template.New("tidb-init-init-start-script").Parse(`trap exit TERM
host={{ .ClusterName }}-tidb
port=4000
while true; do
  nc -zv -w 3 $host $port
  if [ $? -eq 0 ]; then
	break
  else
	echo "info: failed to connect to $host:$port, sleep 1 second then retry"
	sleep 1
  fi
done
echo "info: successfully connected to $host:$port, able to initialize TiDB now"
`))

type TiDBInitInitStartScriptModel struct {
	ClusterName string
}

func RenderTiDBInitInitStartScript(model *TiDBInitInitStartScriptModel) (string, error) {
	return renderTemplateFunc(tidbInitInitStartScriptTpl, model)
}

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
