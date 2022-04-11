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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/klog/v2"
)

// PumpStartScriptModel contain fields for rendering Pump start script
type PumpStartScriptModel struct {
	PDAddr        string
	LogLevel      string
	AdvertiseAddr string
	ExtraArgs     string

	AcrossK8s *AcrossK8sScriptModel
}

// RenderPumpStartScript renders Pump start script from TidbCluster
func RenderPumpStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &PumpStartScriptModel{}

	m.PDAddr = fmt.Sprintf("%s://%s:2379", tc.Scheme(), controller.PDMemberName(tc.Name))
	if tc.AcrossK8s() {
		m.AcrossK8s = &AcrossK8sScriptModel{
			PDAddr:        fmt.Sprintf("%s://%s:2379", tc.Scheme(), controller.PDMemberName(tc.Name)),
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tc.Name, tc.Namespace),
		}
		m.PDAddr = "${result}" // get pd addr in subscript
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		m.PDAddr = fmt.Sprintf("%s://%s:2379", tc.Scheme(), controller.PDMemberName(tc.Spec.Cluster.Name)) // use pd of reference cluster
	}

	m.LogLevel = "info"
	if cfg := tc.Spec.Pump.Config; cfg != nil {
		if v := cfg.Get("log-level"); v != nil {
			logLevel, err := v.AsString()
			if err == nil {
				m.LogLevel = logLevel
			} else {
				klog.Warningf("error log-level for pump: %s", err)
			}
		}
	}

	advertiseAddr := fmt.Sprintf("${PUMP_POD_NAME}.%s-pump", tc.Name)
	if tc.Spec.ClusterDomain != "" {
		advertiseAddr = advertiseAddr + fmt.Sprintf(".%s.svc.%s", tc.Namespace, tc.Spec.ClusterDomain)
	} else if tc.Spec.ClusterDomain == "" && tc.AcrossK8s() {
		advertiseAddr = advertiseAddr + fmt.Sprintf(".%s.svc", tc.Namespace)
	}
	m.AdvertiseAddr = advertiseAddr + ":8250"

	m.ExtraArgs = ""

	return renderTemplateFunc(pumpStartScriptTpl, m)
}

const (
	pumpStartSubScript = `
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

	pumpStartScript = `
PUMP_POD_NAME=$HOSTNAME
{{- if .AcrossK8s -}} {{ template "AcrossK8sSubscript" . }} {{- end }}

ARGS="-pd-urls={{ .PDAddr }} \
    -L {{ .LogLevel }} \
    -log-file= \
    -advertise-addr={{ .AdvertiseAddr }} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

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
		template.New("pump-start-script").Parse(pumpStartSubScript),
	).Parse(componentCommonScript + pumpStartScript),
)
