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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func TestRenderPDStartScriptWithWaitForIpMatch(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyTC     func(tc *v1alpha1.TidbCluster)
		expectScript string
	}

	cases := []testcase{
		{
			name:     "basic",
			modifyTC: func(tc *v1alpha1.TidbCluster) {},
			expectScript: `#!/bin/sh

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

PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PD_POD_NAME}.start-script-test-pd-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS="--data-dir=/var/lib/pd \
--name=${PD_POD_NAME} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${PD_DOMAIN}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${PD_DOMAIN}:2379 \
--config=/etc/pd/pd.toml"

if [[ -f /var/lib/pd/join ]]; then
    join=$(cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://start-script-test-discovery.start-script-test-ns:10261/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS} ${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "enable tls",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			},
			expectScript: `#!/bin/sh

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

PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PD_POD_NAME}.start-script-test-pd-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS="--data-dir=/var/lib/pd \
--name=${PD_POD_NAME} \
--peer-urls=https://0.0.0.0:2380 \
--advertise-peer-urls=https://${PD_DOMAIN}:2380 \
--client-urls=https://0.0.0.0:2379 \
--advertise-client-urls=https://${PD_DOMAIN}:2379 \
--config=/etc/pd/pd.toml"

if [[ -f /var/lib/pd/join ]]; then
    join=$(cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://start-script-test-discovery.start-script-test-ns:10261/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS} ${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "set data sub dir",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.DataSubDir = "pd-data"
			},
			expectScript: `#!/bin/sh

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

PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PD_POD_NAME}.start-script-test-pd-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS="--data-dir=/var/lib/pd/pd-data \
--name=${PD_POD_NAME} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${PD_DOMAIN}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${PD_DOMAIN}:2379 \
--config=/etc/pd/pd.toml"

if [[ -f /var/lib/pd/pd-data/join ]]; then
    join=$(cat /var/lib/pd/pd-data/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/pd-data/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://start-script-test-discovery.start-script-test-ns:10261/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS} ${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "set cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster-1.com"
			},
			expectScript: `#!/bin/sh

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

PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PD_POD_NAME}.start-script-test-pd-peer.start-script-test-ns.svc.cluster-1.com
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS="--data-dir=/var/lib/pd \
--name=${PD_DOMAIN} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${PD_DOMAIN}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${PD_DOMAIN}:2379 \
--config=/etc/pd/pd.toml"

if [[ -f /var/lib/pd/join ]]; then
    join=$(cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://start-script-test-discovery.start-script-test-ns:10261/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS} ${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = ""
				tc.Spec.AcrossK8s = true
			},
			expectScript: `#!/bin/sh

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

PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PD_POD_NAME}.start-script-test-pd-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS="--data-dir=/var/lib/pd \
--name=${PD_DOMAIN} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${PD_DOMAIN}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${PD_DOMAIN}:2379 \
--config=/etc/pd/pd.toml"

if [[ -f /var/lib/pd/join ]]; then
    join=$(cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://start-script-test-discovery.start-script-test-ns:10261/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS} ${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster-1.com"
				tc.Spec.AcrossK8s = true
			},
			expectScript: `#!/bin/sh

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

PD_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PD_POD_NAME}.start-script-test-pd-peer.start-script-test-ns.svc.cluster-1.com
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS="--data-dir=/var/lib/pd \
--name=${PD_DOMAIN} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${PD_DOMAIN}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${PD_DOMAIN}:2379 \
--config=/etc/pd/pd.toml"

if [[ -f /var/lib/pd/join ]]; then
    join=$(cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://start-script-test-discovery.start-script-test-ns:10261/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS} ${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				PD: &v1alpha1.PDSpec{},
			},
		}
		tc.Name = "start-script-test"
		tc.Namespace = "start-script-test-ns"
		tc.Spec.StartScriptV2FeatureFlags = []v1alpha1.StartScriptV2FeatureFlag{
			v1alpha1.StartScriptV2FeatureFlagWaitForDnsNameIpMatch,
		}
		if c.modifyTC != nil {
			c.modifyTC(tc)
		}

		script, err := RenderPDStartScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}

func TestRenderPDMSStartScriptWithWaitForIpMatch(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyTC     func(tc *v1alpha1.TidbCluster)
		expectScript string
	}

	cases := []testcase{
		{
			name:     "basic",
			modifyTC: func(tc *v1alpha1.TidbCluster) {},
			expectScript: `#!/bin/sh

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

PDMS_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PDMS_POD_NAME}.start-script-test-tso-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS=" services tso --listen-addr=http://0.0.0.0:2379 \
--advertise-listen-addr=http://${PD_DOMAIN}:2379 \
--backend-endpoints=http://start-script-test-pd:2379 \
--config=/etc/pd/pd.toml \
"

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
exit 0
`,
		},
		{
			name: "enable tls",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			},
			expectScript: `#!/bin/sh

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

PDMS_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PDMS_POD_NAME}.start-script-test-tso-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS=" services tso --listen-addr=https://0.0.0.0:2379 \
--advertise-listen-addr=https://${PD_DOMAIN}:2379 \
--backend-endpoints=https://start-script-test-pd:2379 \
--config=/etc/pd/pd.toml \
"

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
exit 0
`,
		},
		{
			name: "set data sub dir",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.DataSubDir = "pd-data"
			},
			expectScript: `#!/bin/sh

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

PDMS_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PDMS_POD_NAME}.start-script-test-tso-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS=" services tso --listen-addr=http://0.0.0.0:2379 \
--advertise-listen-addr=http://${PD_DOMAIN}:2379 \
--backend-endpoints=http://start-script-test-pd:2379 \
--config=/etc/pd/pd.toml \
"

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
exit 0
`,
		},
		{
			name: "set cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster-1.com"
			},
			expectScript: `#!/bin/sh

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

PDMS_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PDMS_POD_NAME}.start-script-test-tso-peer.start-script-test-ns.svc.cluster-1.com
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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

ARGS=" services tso --listen-addr=http://0.0.0.0:2379 \
--advertise-listen-addr=http://${PD_DOMAIN}:2379 \
--backend-endpoints=http://start-script-test-pd:2379 \
--config=/etc/pd/pd.toml \
"

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
exit 0
`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = ""
				tc.Spec.AcrossK8s = true
			},
			expectScript: `#!/bin/sh

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

PDMS_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PDMS_POD_NAME}.start-script-test-tso-peer.start-script-test-ns.svc
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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
pd_url=http://start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS=" services tso --listen-addr=http://0.0.0.0:2379 \
--advertise-listen-addr=http://${PD_DOMAIN}:2379 \
--backend-endpoints=${result} \
--config=/etc/pd/pd.toml \
"

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
exit 0
`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster-1.com"
				tc.Spec.AcrossK8s = true
			},
			expectScript: `#!/bin/sh

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

PDMS_POD_NAME=${POD_NAME:-$HOSTNAME}
PD_DOMAIN=${PDMS_POD_NAME}.start-script-test-tso-peer.start-script-test-ns.svc.cluster-1.com
componentDomain=${PD_DOMAIN}
waitThreshold=30
initWaitTime=0
sleep initWaitTime
nsLookupCmd="dig ${componentDomain} A ${componentDomain} AAAA +search +short"

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
pd_url=http://start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS=" services tso --listen-addr=http://0.0.0.0:2379 \
--advertise-listen-addr=http://${PD_DOMAIN}:2379 \
--backend-endpoints=${result} \
--config=/etc/pd/pd.toml \
"

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
exit 0
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				PD: &v1alpha1.PDSpec{},
			},
		}
		tc.Name = "start-script-test"
		tc.Namespace = "start-script-test-ns"
		tc.Spec.StartScriptV2FeatureFlags = []v1alpha1.StartScriptV2FeatureFlag{
			v1alpha1.StartScriptV2FeatureFlagWaitForDnsNameIpMatch,
		}
		if c.modifyTC != nil {
			c.modifyTC(tc)
		}

		script, err := RenderPDTSOStartScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}
