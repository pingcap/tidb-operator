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

	pdDomain := controller.PDMemberName(tc.Name)
	if tc.AcrossK8s() {
		pdDomain = controller.PDMemberName(tc.Name) // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		pdDomain = controller.PDMemberName(tc.Spec.Cluster.Name) // use pd of reference cluster
	}
	m.PDAddr = fmt.Sprintf("%s://%s:2379", tc.Scheme(), pdDomain)

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

	if tc.AcrossK8s() {
		m.AcrossK8s = &AcrossK8sScriptModel{
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tc.Name, tc.Namespace),
		}
	}

	return renderTemplateFunc(pumpStartScriptTpl, m)
}

const (
	pumpStartSubScript = `
{{ define "PUMP_PD_ADDR" -}}
    {{- if .AcrossK8s -}}
pd_url={{ .PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PUMP_PD_ADDR=${result}
    {{- else -}}
PUMP_PD_ADDR={{ .PDAddr }}
    {{- end -}}
{{- end}}
`

	pumpStartScript = `
PUMP_POD_NAME=$HOSTNAME
{{ template "PUMP_PD_ADDR" . }}
PUMP_LOG_LEVEL={{ .LogLevel }}
PUMP_ADVERTISE_ADDR={{ .AdvertiseAddr }}
PUMP_EXTRA_ARGS={{ .ExtraArgs }}

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml"

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
		template.New("pump-start-script").Parse(pumpStartSubScript),
	).Parse(componentCommonScript + pumpStartScript),
)
