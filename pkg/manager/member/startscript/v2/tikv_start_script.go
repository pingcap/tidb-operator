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
	"github.com/pingcap/tidb-operator/pkg/manager/member/constants"
)

// TiKVStartScriptModel contain fields for rendering TiKV start script
type TiKVStartScriptModel struct {
	PDAddr        string
	AdvertiseAddr string
	DataDir       string
	Capacity      string
	ExtraArgs     string

	AcrossK8s *AcrossK8sScriptModel
}

// RenderTiKVStartScript renders TiKV start script from TidbCluster
func RenderTiKVStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiKVStartScriptModel{}
	tcName := tc.Name
	tcNS := tc.Namespace
	peerServiceName := controller.TiKVPeerMemberName(tcName)

	m.PDAddr = fmt.Sprintf("%s:2379", controller.PDMemberName(tcName))
	if tc.AcrossK8s() {
		m.AcrossK8s = &AcrossK8sScriptModel{
			PDAddr:        fmt.Sprintf("%s:2379", controller.PDMemberName(tcName)),
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tcName, tcNS),
		}
		m.PDAddr = "${result}" // get pd addr in subscript
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		m.PDAddr = fmt.Sprintf("%s:2379", controller.PDMemberName(tc.Spec.Cluster.Name)) // use pd of reference cluster
	}

	advertiseAddr := fmt.Sprintf("${TIKV_POD_NAME}.%s.%s.svc", peerServiceName, tcNS)
	if tc.Spec.ClusterDomain != "" {
		advertiseAddr = advertiseAddr + "." + tc.Spec.ClusterDomain
	}
	m.AdvertiseAddr = advertiseAddr + ":20160"

	m.DataDir = filepath.Join(constants.TiKVDataVolumeMountPath, tc.Spec.TiKV.DataSubDir)

	m.Capacity = "${CAPACITY}"

	extraArgs := []string{}
	if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
		advertiseStatusAddr := fmt.Sprintf("${TIKV_POD_NAME}.%s.%s.svc", peerServiceName, tcNS)
		if tc.Spec.ClusterDomain != "" {
			advertiseStatusAddr = advertiseStatusAddr + "." + tc.Spec.ClusterDomain
		}
		extraArgs = append(extraArgs, fmt.Sprintf("--advertise-status-addr=%s:20180", advertiseStatusAddr))
	}
	if len(extraArgs) > 0 {
		m.ExtraArgs = strings.Join(extraArgs, " ")
	}

	return renderTemplateFunc(tikvStartScriptTpl, m)
}

const (
	// tikvStartSubScript contains optional subscripts used in start script.
	tikvStartSubScript = `
{{ define "AcrossK8sSubscript" }}
pd_url={{ .AcrossK8s.PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
{{- end }}
`

	// tikvStartScript is the template of start script.
	tikvStartScript = `
TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
{{- if .AcrossK8s -}} {{ template "AcrossK8sSubscript" . }} {{- end }}

ARGS="--pd={{ .PDAddr }} \
--advertise-addr={{ .AdvertiseAddr }} \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir={{ .DataDir }} \
--capacity={{ .Capacity }} \
--config=/etc/tikv/tikv.toml"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

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
