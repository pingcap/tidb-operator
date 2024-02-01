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
	"slices"
	"strings"
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
	PDAddresses        string
	PDStartTimeout     int
}

// PDMSStartScriptModel contain fields for rendering PD Micro Service start script
type PDMSStartScriptModel struct {
	PDMSDomain string
	PDMSName   string

	DataDir string

	ListenAddr          string
	AdvertiseListenAddr string
	BackendEndpoints    string

	DiscoveryAddr string
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

	preferPDAddressesOverDiscovery := slices.Contains(
		tc.Spec.StartScriptV2FeatureFlags, v1alpha1.StartScriptV2FeatureFlagPreferPDAddressesOverDiscovery)
	if preferPDAddressesOverDiscovery {
		pdAddressesWithSchemeAndPort := addressesWithSchemeAndPort(tc.Spec.PDAddresses, tc.Scheme()+"://", v1alpha1.DefaultPDPeerPort)
		m.PDAddresses = strings.Join(pdAddressesWithSchemeAndPort, ",")
	}

	m.DataDir = filepath.Join(constants.PDDataVolumeMountPath, tc.Spec.PD.DataSubDir)

	m.PeerURL = fmt.Sprintf("%s://0.0.0.0:%d", tc.Scheme(), v1alpha1.DefaultPDPeerPort)

	m.AdvertisePeerURL = fmt.Sprintf("%s://${PD_DOMAIN}:%d", tc.Scheme(), v1alpha1.DefaultPDPeerPort)

	m.ClientURL = fmt.Sprintf("%s://0.0.0.0:%d", tc.Scheme(), v1alpha1.DefaultPDClientPort)

	m.AdvertiseClientURL = fmt.Sprintf("%s://${PD_DOMAIN}:%d", tc.Scheme(), v1alpha1.DefaultPDClientPort)

	m.DiscoveryAddr = fmt.Sprintf("%s-discovery.%s:10261", tcName, tcNS)

	m.PDStartTimeout = tc.PDStartTimeout()

	waitForDnsNameIpMatchOnStartup := slices.Contains(
		tc.Spec.StartScriptV2FeatureFlags, v1alpha1.StartScriptV2FeatureFlagWaitForDnsNameIpMatch)

	mode := ""
	if tc.Spec.PD.Mode == "ms" {
		mode = "api"
	}
	pdStartScriptTpl := template.Must(
		template.Must(
			template.New("pd-start-script").Parse(pdStartSubScript),
		).Parse(
			componentCommonScript +
				replacePdStartScriptCustomPorts(
					replacePdStartScriptDnsAwaitPart(waitForDnsNameIpMatchOnStartup,
						enableMicroServiceModeDynamic(mode, pdStartScript)))),
	)

	return renderTemplateFunc(pdStartScriptTpl, m)
}

func RenderPDTSOStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	return renderPDMSStartScript(tc, "tso")
}

func RenderPDSchedulingStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	return renderPDMSStartScript(tc, "scheduling")
}

// RenderPDMCSStartScript renders TSO start script from TidbCluster
func renderPDMSStartScript(tc *v1alpha1.TidbCluster, name string) (string, error) {
	m := &PDMSStartScriptModel{}
	tcName := tc.Name
	tcNS := tc.Namespace
	peerServiceName := controller.PDMSMemberName(tcName, name)

	m.DataDir = constants.PDMSDataVolumeMountPath
	m.PDMSDomain = fmt.Sprintf("${PDMS_POD_NAME}.%s.%s.svc", peerServiceName, tcNS)
	if tc.Spec.ClusterDomain != "" {
		m.PDMSDomain = m.PDMSDomain + "." + tc.Spec.ClusterDomain
	}

	m.PDMSName = "${PDMS_POD_NAME}"
	if tc.AcrossK8s() || tc.Spec.ClusterDomain != "" {
		m.PDMSName = "${PDMS_DOMAIN}"
	}

	m.ListenAddr = fmt.Sprintf("%s://0.0.0.0:%d", tc.Scheme(), v1alpha1.DefaultPDClientPort)

	m.AdvertiseListenAddr = fmt.Sprintf("%s://${PDMS_DOMAIN}:%d", tc.Scheme(), v1alpha1.DefaultPDClientPort)

	m.DiscoveryAddr = fmt.Sprintf("%s-discovery.%s:10261", tcName, tcNS)

	m.BackendEndpoints = fmt.Sprintf("%s://${PDMS_DOMAIN}:%d", tc.Scheme(), v1alpha1.DefaultPDPeerPort)

	msStartSubScript := ``
	msStartScriptTpl := template.Must(
		template.Must(
			template.New("pdms-start-script").Parse(msStartSubScript),
		).Parse(
			componentCommonScript +
				enableMicroServiceModeDynamic(name, pdmsStartScriptTplText)))

	return renderTemplateFunc(msStartScriptTpl, m)
}

const (
	// pdStartSubScript contains optional subscripts used in start script.
	pdStartSubScript = ``

	pdWaitForDnsIpMatchSubScript = `
componentDomain=${PD_DOMAIN}
waitThreshold={{ .PDStartTimeout }}
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"
` + componentCommonWaitForDnsIpMatchScript

	pdEnableMicroServiceSubScript = "services"

	pdWaitForDnsOnlySubScript = `

elapseTime=0
period=1
threshold={{ .PDStartTimeout }}
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
`

	// pdStartScript is the template of start script.
	pdStartScript = `
PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN={{ .PDDomain }}` +
		dnsAwaitPart + `
ARGS="` + pdEnableMicroService + `--data-dir={{ .DataDir }} \
--name={{ .PDName }} \
--peer-urls={{ .PeerURL }} \
--advertise-peer-urls={{ .AdvertisePeerURL }} \
--client-urls={{ .ClientURL }} \
--advertise-client-urls={{ .AdvertiseClientURL }} \
--config=/etc/pd/pd.toml"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}
{{ if .PDAddresses }}
ARGS="${ARGS} --join={{ .PDAddresses }}"
{{- else }}
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
{{- end }}

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`

	// pdmsStartScriptTplText is the template of pd microservices start script.
	pdmsStartScriptTplText = `
ARGS="` + pdEnableMicroService + `--listen-addr={{ .ListenAddr }} \
--advertise-listen-addr={{ .AdvertiseListenAddr }} \
--backend-endpoints={{ .BackendEndpoints }} \
--config=/etc/pd/pd.toml \
"

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
exit 0
`
)

func replacePdStartScriptCustomPorts(startScript string) string {
	// `DefaultPDPeerPort` may be changed when building the binary
	if v1alpha1.DefaultPDPeerPort != 2380 {
		startScript = strings.ReplaceAll(startScript, ":2380", fmt.Sprintf(":%d", v1alpha1.DefaultPDPeerPort))
	}
	return startScript
}

func replacePdStartScriptDnsAwaitPart(withLocalIpMatch bool, startScript string) string {
	if withLocalIpMatch {
		return strings.ReplaceAll(startScript, dnsAwaitPart, pdWaitForDnsIpMatchSubScript)
	} else {
		return strings.ReplaceAll(startScript, dnsAwaitPart, pdWaitForDnsOnlySubScript)
	}
}

func enableMicroServiceModeDynamic(ms string, startScript string) string {
	if ms != "" {
		return strings.ReplaceAll(startScript, pdEnableMicroService, fmt.Sprintf(" %s %s ", pdEnableMicroServiceSubScript, ms))
	} else {
		return strings.ReplaceAll(startScript, pdEnableMicroService, "")
	}
}
