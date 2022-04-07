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
	"path"
	"strings"
	"text/template"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

// TiCDCStartScriptModel contain fields for rendering TiCDC start script
type TiCDCStartScriptModel struct {
	AdvertiseAddr string
	GCTTL         int32
	LogFile       string
	LogLevel      string
	PDAddr        string
	ExtraArgs     string

	AcrossK8s *AcrossK8sScriptModel
}

func RenderTiCDCStartScript(tc *v1alpha1.TidbCluster, ticdcCertPath string) (string, error) {
	m := &TiCDCStartScriptModel{}

	advertiseAddr := fmt.Sprintf("${POD_NAME}.${HEADLESS_SERVICE_NAME}.%s.svc", tc.Namespace)
	if tc.Spec.ClusterDomain != "" {
		advertiseAddr = advertiseAddr + "." + tc.Spec.ClusterDomain
	}
	m.AdvertiseAddr = advertiseAddr + ":8301"

	m.GCTTL = tc.TiCDCGCTTL()

	m.LogFile = tc.TiCDCLogFile()

	m.LogLevel = tc.TiCDCLogLevel()

	pdDomain := controller.PDMemberName(tc.Name)
	if tc.AcrossK8s() {
		pdDomain = controller.PDMemberName(tc.Name) // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		pdDomain = controller.PDMemberName(tc.Spec.Cluster.Name) // use pd of reference cluster
	}
	m.PDAddr = fmt.Sprintf("%s://%s:2379", tc.Scheme(), pdDomain)

	extraArgs := []string{}
	if tc.IsTLSClusterEnabled() {
		extraArgs = append(extraArgs, fmt.Sprintf("--ca=%s", path.Join(ticdcCertPath, corev1.ServiceAccountRootCAKey)))
		extraArgs = append(extraArgs, fmt.Sprintf("--cert=%s", path.Join(ticdcCertPath, corev1.TLSCertKey)))
		extraArgs = append(extraArgs, fmt.Sprintf("--key=%s", path.Join(ticdcCertPath, corev1.TLSPrivateKeyKey)))
	}
	if tc.Spec.TiCDC.Config != nil && !tc.Spec.TiCDC.Config.OnlyOldItems() {
		extraArgs = append(extraArgs, fmt.Sprintf("--config=%s", "/etc/ticdc/ticdc.toml"))
	}
	if len(extraArgs) > 0 {
		m.ExtraArgs = fmt.Sprintf("\"%s\"", strings.Join(extraArgs, " "))
	}

	if tc.AcrossK8s() {
		m.AcrossK8s = &AcrossK8sScriptModel{
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tc.Name, tc.Namespace),
		}
	}

	return renderTemplateFunc(ticdcStartScriptTpl, m)
}

const (
	ticdcStartSubScript = `
{{ define "TICDC_PD_ADDR" -}}
    {{- if .AcrossK8s -}}
pd_url={{ .PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TICDC_PD_ADDR=${result}
    {{- else -}}
TICDC_PD_ADDR={{ .PDAddr }}
    {{- end -}}
{{- end}}
`

	ticdcStartScript = `
TICDC_POD_NAME=${POD_NAME}
TICDC_ADVERTISE_ADDR={{ .AdvertiseAddr }}
TICDC_GC_TTL={{ .GCTTL }}
TICDC_LOG_FILE={{ .LogFile }}
TICDC_LOG_LEVEL={{ .LogLevel }}
{{ template "TICDC_PD_ADDR" . }}
TICDC_EXTRA_ARGS={{ .ExtraArgs }}

ARGS="--addr=0.0.0.0:8301 \
    --advertise-addr=${TICDC_ADVERTISE_ADDR} \
    --gc-ttl=${TICDC_GC_TTL} \
    --log-file=${TICDC_LOG_FILE} \
    --log-level=${TICDC_LOG_LEVEL} \
    --pd=${TICDC_PD_ADDR}"

if [[ -n "${TICDC_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TICDC_EXTRA_ARGS}"
fi

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
`
)

var ticdcStartScriptTpl = template.Must(
	template.Must(
		template.New("ticdc-start-script").Parse(ticdcStartSubScript),
	).Parse(componentCommonScript + ticdcStartScript),
)
