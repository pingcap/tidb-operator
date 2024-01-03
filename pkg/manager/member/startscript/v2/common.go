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
	"bytes"
	"fmt"
	"net/url"
	"text/template"
)

const (
	componentCommonScript = `#!/bin/sh

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
`
	dnsAwaitPart = "<<dns-await-part>>"

	pdEnableMicroService = "<<pd-enable-micro-service>>"

	componentCommonWaitForDnsIpMatchScript = `
elapseTime=0
period=1
while true; do
    sleep ${period}
    elapseTime=$(( elapseTime+period ))

    if [[ ${elapseTime} -ge ${waitThreshold} ]]; then
        echo "waiting for cluster ready timeout" >&2
        exit 1
    fi

    digRes=$(eval "$nsLookupCmd")
    if [ $? -ne 0  ]; then
        echo "domain resolve ${componentDomain} failed"
        echo "$digRes"
        continue
    fi

    if [ -z "${digRes}" ]
    then
        echo "domain resolve ${componentDomain} no record return"
    else
        echo "domain resolve ${componentDomain} success"
        echo "$digRes"

        # now compare resolved IPs with host IPs
        hostnameIRes=($(hostname -I))
        hostIps=()
        while IFS= read -r line; do
            hostIps+=("$line")
        done <<< "$hostnameIRes"
        echo "hostIps: ${hostIps[@]}"

        resolvedIps=()
        while IFS= read -r line; do
            resolvedIps+=("$line")
        done <<< "$digRes"
        echo "resolvedIps: ${resolvedIps[@]}"

        foundIp=false
        for element in "${resolvedIps[@]}"
        do
            if [[ " ${hostIps[@]} " =~ " ${element} " ]]; then
                foundIp=true
                break
            fi
        done
        if [ "$foundIp" = true ]; then
            echo "Success: Resolved IP matches one of podIPs"
            break
        else
            echo "Resolved IP does not match any of podIPs"
        fi
    fi
done
`
)

// AcrossK8sScriptModel contain fields for rendering subscript
type AcrossK8sScriptModel struct {
	// DiscoveryAddr is the address of the discovery service.
	//
	// When cluster is deployed across k8s, all components except pd will get the pd addr from discovery.
	DiscoveryAddr string

	// PDAddr is the address used by discovery to get the actual pd addr.
	PDAddr string
}

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}

func addressesWithSchemeAndPort(addresses []string, scheme string, port int32) []string {
	res := make([]string, len(addresses))
	for i, a := range addresses {
		u, err := url.Parse(a)
		if err != nil {
			res[i] = fmt.Sprintf("%s%s:%d", scheme, a, port)
		} else if u.Hostname() != "" {
			res[i] = fmt.Sprintf("%s%s:%d", scheme, u.Hostname(), port)
		} else {
			res[i] = fmt.Sprintf("%s%s:%d", scheme, u.Path, port)
		}

	}
	return res
}
