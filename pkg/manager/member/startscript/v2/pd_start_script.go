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
	"text/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/member/constants"
)

// PDStartScriptModel contain fields for rendering PD start script
type PDStartScriptModel struct {
	PDDomain           string
	PDName             string
	DataDir            string
	PeerURL            string
	AdvertisePeerURL   string
	ClientURL          string
	AdvertiseClientURL string
	DiscoveryAddr      string
	ExtraArgs          string
}

// RenderPDStartScript renders PD start script from TidbCluster
func RenderPDStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &PDStartScriptModel{}
	tcName := tc.Name
	tcNS := tc.Namespace
	peerServiceName := controller.PDPeerMemberName(tcName)

	m.PDDomain = fmt.Sprintf("${PD_POD_NAME}.%s.%s.svc", peerServiceName, tcNS)
	if tc.Spec.ClusterDomain != "" {
		m.PDDomain = m.PDDomain + "." + tc.Spec.ClusterDomain
	}

	m.PDName = "${PD_POD_NAME}"
	if tc.AcrossK8s() || tc.Spec.ClusterDomain != "" {
		m.PDName = "${PD_DOMAIN}"
	}

	m.DataDir = filepath.Join(constants.PDDataVolumeMountPath, tc.Spec.PD.DataSubDir)

	m.PeerURL = fmt.Sprintf("%s://0.0.0.0:2380", tc.Scheme())

	m.AdvertisePeerURL = fmt.Sprintf("%s://${PD_DOMAIN}:2380", tc.Scheme())

	m.ClientURL = fmt.Sprintf("%s://0.0.0.0:2379", tc.Scheme())

	m.AdvertiseClientURL = fmt.Sprintf("%s://${PD_DOMAIN}:2379", tc.Scheme())

	m.DiscoveryAddr = fmt.Sprintf("%s-discovery.%s:10261", tcName, tcNS)

	return renderTemplateFunc(pdStartScriptTpl, m)
}

const (
	// pdStartSubScript contains optional subscripts used in start script.
	pdStartSubScript = ``

	// pdStartScript is the template of start script.
	pdStartScript = `
PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN={{ .PDDomain }}

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

ARGS="--data-dir={{ .DataDir }} \
--name={{ .PDName }} \
--peer-urls={{ .PeerURL }} \
--advertise-peer-urls={{ .AdvertisePeerURL }} \
--client-urls={{ .ClientURL }} \
--advertise-client-urls={{ .AdvertiseClientURL }} \
--config=/etc/pd/pd.toml"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

if [[ -f {{ .DataDir }}/join ]]; then
    join=$(cat {{ .DataDir }}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d {{ .DataDir }}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://{{ .DiscoveryAddr }}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS} ${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`
)

var pdStartScriptTpl = template.Must(
	template.Must(
		template.New("pd-start-script").Parse(pdStartSubScript),
	).Parse(componentCommonScript + pdStartScript),
)
