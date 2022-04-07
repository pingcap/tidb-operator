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
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

type TiKVStartScriptModel struct {
	PDAddr        string
	AdvertiseAddr string
	DataDir       string
	Capacity      string
	ExtraArgs     string

	AcrossK8s *ComponentAcrossK8s
}

func RenderTiKVStartScript(tc *v1alpha1.TidbCluster, tikvDataVolumeMountPath string) (string, error) {
	m := &TiKVStartScriptModel{}

	pdDomain := controller.PDMemberName(tc.Name)
	if tc.AcrossK8s() {
		pdDomain = controller.PDMemberName(tc.Name) // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		pdDomain = controller.PDMemberName(tc.Spec.Cluster.Name) // use pd of reference cluster
	}
	m.PDAddr = fmt.Sprintf("%s:2379", pdDomain)

	// FIXME: not use HEADLESS_SERVICE_NAME
	advertiseAddr := fmt.Sprintf("${TIKV_POD_NAME}.${HEADLESS_SERVICE_NAME}.%s.svc", tc.Namespace)
	if tc.Spec.ClusterDomain != "" {
		advertiseAddr = advertiseAddr + "." + tc.Spec.ClusterDomain
	}
	m.AdvertiseAddr = advertiseAddr + ":20160"

	m.DataDir = filepath.Join(tikvDataVolumeMountPath, tc.Spec.TiKV.DataSubDir)

	m.Capacity = "${CAPACITY}"

	extraArgs := []string{}
	if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
		advertiseStatusAddr := fmt.Sprintf("${TIKV_POD_NAME}.${HEADLESS_SERVICE_NAME}.%s.svc", tc.Namespace)
		if tc.Spec.ClusterDomain != "" {
			advertiseStatusAddr = advertiseStatusAddr + "." + tc.Spec.ClusterDomain
		}
		extraArgs = append(extraArgs, fmt.Sprintf("--advertise-status-addr=%s:20180", advertiseStatusAddr))
	}
	if len(extraArgs) > 0 {
		m.ExtraArgs = fmt.Sprintf("\"%s\"", strings.Join(extraArgs, " "))
	}

	if tc.AcrossK8s() {
		m.AcrossK8s = &ComponentAcrossK8s{
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tc.Name, tc.Namespace),
		}
	}

	return renderTemplateFunc(tikvStartScriptTpl, m)
}

const (
	tikvStartSubScript = `
{{ define "TIKV_PD_ADDR" -}}
    {{- if .AcrossK8s -}}
pd_url={{ .PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIKV_PD_ADDR=${result}
    {{- else -}}
TIKV_PD_ADDR={{ .PDAddr }}
    {{- end -}}
{{- end }}
`

	tikvStartScript = `
TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
{{ template "TIKV_PD_ADDR" . }}
TIKV_ADVERTISE_ADDR={{ .AdvertiseAddr }}
TIKV_DATA_DIR={{ .DataDir }}
TIKV_CAPACITY={{ .Capacity }}
TIKV_EXTRA_ARGS={{ .ExtraArgs }}

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

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`
)

var tikvStartScriptTpl = template.Must(
	template.Must(
		template.New("tikv-start-script").Parse(tikvStartSubScript),
	).Parse(componentCommonScript + tikvStartScript),
)
