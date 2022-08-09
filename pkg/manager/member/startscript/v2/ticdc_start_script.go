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
	"github.com/pingcap/tidb-operator/pkg/manager/member/constants"
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

func RenderTiCDCStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiCDCStartScriptModel{}
	tcName := tc.Name
	tcNS := tc.Namespace
	peerServiceName := controller.TiCDCPeerMemberName(tcName)

	// NB: TiCDC control relies the format.
	// TODO move advertise addr format to package controller.
	advertiseAddr := fmt.Sprintf("${TICDC_POD_NAME}.%s.%s.svc", peerServiceName, tcNS)
	if tc.Spec.ClusterDomain != "" {
		advertiseAddr = advertiseAddr + "." + tc.Spec.ClusterDomain
	}
	m.AdvertiseAddr = advertiseAddr + ":8301"

	m.GCTTL = tc.TiCDCGCTTL()

	m.LogFile = tc.TiCDCLogFile()

	m.LogLevel = tc.TiCDCLogLevel()

	m.PDAddr = fmt.Sprintf("%s://%s:2379", tc.Scheme(), controller.PDMemberName(tcName))
	if tc.AcrossK8s() {
		m.AcrossK8s = &AcrossK8sScriptModel{
			PDAddr:        fmt.Sprintf("%s://%s:2379", tc.Scheme(), controller.PDMemberName(tcName)),
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tcName, tcNS),
		}
		m.PDAddr = "${result}" // get pd addr in subscript
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		m.PDAddr = fmt.Sprintf("%s://%s:2379", tc.Scheme(), controller.PDMemberName(tc.Spec.Cluster.Name)) // use pd of reference cluster
	}

	extraArgs := []string{}
	if tc.IsTLSClusterEnabled() {
		ticdcCertPath := constants.TiCDCCertPath
		extraArgs = append(extraArgs, fmt.Sprintf("--ca=%s", path.Join(ticdcCertPath, corev1.ServiceAccountRootCAKey)))
		extraArgs = append(extraArgs, fmt.Sprintf("--cert=%s", path.Join(ticdcCertPath, corev1.TLSCertKey)))
		extraArgs = append(extraArgs, fmt.Sprintf("--key=%s", path.Join(ticdcCertPath, corev1.TLSPrivateKeyKey)))
	}
	if tc.Spec.TiCDC.Config != nil && !tc.Spec.TiCDC.Config.OnlyOldItems() {
		extraArgs = append(extraArgs, fmt.Sprintf("--config=%s", "/etc/ticdc/ticdc.toml"))
	}
	if len(extraArgs) > 0 {
		m.ExtraArgs = strings.Join(extraArgs, " ")
	}

	return renderTemplateFunc(ticdcStartScriptTpl, m)
}

const (
	// ticdcStartSubScript contains optional subscripts used in start script.
	ticdcStartSubScript = `
{{ define "AcrossK8sSubscript" }}
pd_url={{ .AcrossK8s.PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
{{- end}}
`

	// ticdcStartScript is the template of start script.
	ticdcStartScript = `
TICDC_POD_NAME=${POD_NAME}
{{- if .AcrossK8s -}} {{ template "AcrossK8sSubscript" . }} {{- end }}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr={{ .AdvertiseAddr }} \
--gc-ttl={{ .GCTTL }} \
--log-file={{ .LogFile }} \
--log-level={{ .LogLevel }} \
--pd={{ .PDAddr }}"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

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
