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
)

func TestPDStartScript(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyModel  func(m *PDStartScriptModel)
		expectScript string
	}

	cases := []testcase{
		{
			name:        "basic",
			modifyModel: func(m *PDStartScriptModel) {},
			expectScript: `#!/bin/sh

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"
OPERATOR_ENV="/etc/operator.env"

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
PD_DOMAIN=${POD_NAME}.pd-peer.pd-test.svc
PD_COMPONENT_NAME=${PD_POD_NAME}
PD_DATA_DIR=/var/lib/pd
PD_PEER_URL=http://0.0.0.0:2380
PD_ADVERTISE_PEER_URL=http://${PD_DOMAIN}:2380
PD_CLIENT_URL=http://0.0.0.0:2379
PD_ADVERTISE_CLIENT_URL=http://${PD_DOMAIN}:2379
PD_DISCOVERY_ADDR=pd-start-script-test-discovery.pd-peer.pd-test.svc:10261

set | grep PD_

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

ARGS="--data-dir=${PD_DATA_DIR} \
    --name=${PD_COMPONENT_NAME} \
    --peer-urls=${PD_PEER_URL} \
    --advertise-peer-urls=${PD_ADVERTISE_PEER_URL} \
    --client-urls=${PD_CLIENT_URL} \
    --advertise-client-urls=${PD_ADVERTISE_CLIENT_URL} \
    --config=/etc/pd/pd.toml"

if [[ -f ${PD_DATA_DIR}/join ]]; then
    join=$(cat ${PD_DATA_DIR}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d ${PD_DATA_DIR}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://${PD_DISCOVERY_ADDR}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "https scheme",
			modifyModel: func(m *PDStartScriptModel) {
				m.Scheme = "https"
			},
			expectScript: `#!/bin/sh

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"
OPERATOR_ENV="/etc/operator.env"

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
PD_DOMAIN=${POD_NAME}.pd-peer.pd-test.svc
PD_COMPONENT_NAME=${PD_POD_NAME}
PD_DATA_DIR=/var/lib/pd
PD_PEER_URL=https://0.0.0.0:2380
PD_ADVERTISE_PEER_URL=https://${PD_DOMAIN}:2380
PD_CLIENT_URL=https://0.0.0.0:2379
PD_ADVERTISE_CLIENT_URL=https://${PD_DOMAIN}:2379
PD_DISCOVERY_ADDR=pd-start-script-test-discovery.pd-peer.pd-test.svc:10261

set | grep PD_

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

ARGS="--data-dir=${PD_DATA_DIR} \
    --name=${PD_COMPONENT_NAME} \
    --peer-urls=${PD_PEER_URL} \
    --advertise-peer-urls=${PD_ADVERTISE_PEER_URL} \
    --client-urls=${PD_CLIENT_URL} \
    --advertise-client-urls=${PD_ADVERTISE_CLIENT_URL} \
    --config=/etc/pd/pd.toml"

if [[ -f ${PD_DATA_DIR}/join ]]; then
    join=$(cat ${PD_DATA_DIR}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d ${PD_DATA_DIR}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://${PD_DISCOVERY_ADDR}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "set data dir",
			modifyModel: func(m *PDStartScriptModel) {
				m.DataDir = "/var/lib/pd/data"
			},
			expectScript: `#!/bin/sh

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"
OPERATOR_ENV="/etc/operator.env"

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
PD_DOMAIN=${POD_NAME}.pd-peer.pd-test.svc
PD_COMPONENT_NAME=${PD_POD_NAME}
PD_DATA_DIR=/var/lib/pd/data
PD_PEER_URL=http://0.0.0.0:2380
PD_ADVERTISE_PEER_URL=http://${PD_DOMAIN}:2380
PD_CLIENT_URL=http://0.0.0.0:2379
PD_ADVERTISE_CLIENT_URL=http://${PD_DOMAIN}:2379
PD_DISCOVERY_ADDR=pd-start-script-test-discovery.pd-peer.pd-test.svc:10261

set | grep PD_

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

ARGS="--data-dir=${PD_DATA_DIR} \
    --name=${PD_COMPONENT_NAME} \
    --peer-urls=${PD_PEER_URL} \
    --advertise-peer-urls=${PD_ADVERTISE_PEER_URL} \
    --client-urls=${PD_CLIENT_URL} \
    --advertise-client-urls=${PD_ADVERTISE_CLIENT_URL} \
    --config=/etc/pd/pd.toml"

if [[ -f ${PD_DATA_DIR}/join ]]; then
    join=$(cat ${PD_DATA_DIR}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d ${PD_DATA_DIR}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://${PD_DISCOVERY_ADDR}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "set cluster domain",
			modifyModel: func(m *PDStartScriptModel) {
				m.ClusterDomain = "cluster-1.com"
			},
			expectScript: `#!/bin/sh

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"
OPERATOR_ENV="/etc/operator.env"

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
PD_DOMAIN=${POD_NAME}.pd-peer.pd-test.svc.cluster-1.com
PD_COMPONENT_NAME=${PD_DOMAIN}
PD_DATA_DIR=/var/lib/pd
PD_PEER_URL=http://0.0.0.0:2380
PD_ADVERTISE_PEER_URL=http://${PD_DOMAIN}:2380
PD_CLIENT_URL=http://0.0.0.0:2379
PD_ADVERTISE_CLIENT_URL=http://${PD_DOMAIN}:2379
PD_DISCOVERY_ADDR=pd-start-script-test-discovery.pd-peer.pd-test.svc:10261

set | grep PD_

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

ARGS="--data-dir=${PD_DATA_DIR} \
    --name=${PD_COMPONENT_NAME} \
    --peer-urls=${PD_PEER_URL} \
    --advertise-peer-urls=${PD_ADVERTISE_PEER_URL} \
    --client-urls=${PD_CLIENT_URL} \
    --advertise-client-urls=${PD_ADVERTISE_CLIENT_URL} \
    --config=/etc/pd/pd.toml"

if [[ -f ${PD_DATA_DIR}/join ]]; then
    join=$(cat ${PD_DATA_DIR}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d ${PD_DATA_DIR}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://${PD_DISCOVERY_ADDR}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "across k8s but not set cluster domain",
			modifyModel: func(m *PDStartScriptModel) {
				m.AcrossK8s = true
			},
			expectScript: `#!/bin/sh

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"
OPERATOR_ENV="/etc/operator.env"

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
PD_DOMAIN=${POD_NAME}.pd-peer.pd-test.svc
PD_COMPONENT_NAME=${PD_DOMAIN}
PD_DATA_DIR=/var/lib/pd
PD_PEER_URL=http://0.0.0.0:2380
PD_ADVERTISE_PEER_URL=http://${PD_DOMAIN}:2380
PD_CLIENT_URL=http://0.0.0.0:2379
PD_ADVERTISE_CLIENT_URL=http://${PD_DOMAIN}:2379
PD_DISCOVERY_ADDR=pd-start-script-test-discovery.pd-peer.pd-test.svc:10261

set | grep PD_

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

ARGS="--data-dir=${PD_DATA_DIR} \
    --name=${PD_COMPONENT_NAME} \
    --peer-urls=${PD_PEER_URL} \
    --advertise-peer-urls=${PD_ADVERTISE_PEER_URL} \
    --client-urls=${PD_CLIENT_URL} \
    --advertise-client-urls=${PD_ADVERTISE_CLIENT_URL} \
    --config=/etc/pd/pd.toml"

if [[ -f ${PD_DATA_DIR}/join ]]; then
    join=$(cat ${PD_DATA_DIR}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d ${PD_DATA_DIR}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://${PD_DISCOVERY_ADDR}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
`,
		},
		{
			name: "across k8s and set cluster domain",
			modifyModel: func(m *PDStartScriptModel) {
				m.ClusterDomain = "cluster-1.com"
				m.AcrossK8s = true
			},
			expectScript: `#!/bin/sh

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"
OPERATOR_ENV="/etc/operator.env"

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
PD_DOMAIN=${POD_NAME}.pd-peer.pd-test.svc.cluster-1.com
PD_COMPONENT_NAME=${PD_DOMAIN}
PD_DATA_DIR=/var/lib/pd
PD_PEER_URL=http://0.0.0.0:2380
PD_ADVERTISE_PEER_URL=http://${PD_DOMAIN}:2380
PD_CLIENT_URL=http://0.0.0.0:2379
PD_ADVERTISE_CLIENT_URL=http://${PD_DOMAIN}:2379
PD_DISCOVERY_ADDR=pd-start-script-test-discovery.pd-peer.pd-test.svc:10261

set | grep PD_

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

ARGS="--data-dir=${PD_DATA_DIR} \
    --name=${PD_COMPONENT_NAME} \
    --peer-urls=${PD_PEER_URL} \
    --advertise-peer-urls=${PD_ADVERTISE_PEER_URL} \
    --client-urls=${PD_CLIENT_URL} \
    --advertise-client-urls=${PD_ADVERTISE_CLIENT_URL} \
    --config=/etc/pd/pd.toml"

if [[ -f ${PD_DATA_DIR}/join ]]; then
    join=$(cat ${PD_DATA_DIR}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ",")
    join=${join%,}
    ARGS="${ARGS} --join=${join}"
elif [[ ! -d ${PD_DATA_DIR}/member/wal ]]; then
    encoded_domain_url=$(echo ${PD_DOMAIN}:2380 | base64 | tr "\n" " " | sed "s/ //g")

    until result=$(wget -qO- -T 3 http://${PD_DISCOVERY_ADDR}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service to return start args ..."
        sleep $((RANDOM % 5))
    done
    ARGS="${ARGS}${result}"
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

		model := &PDStartScriptModel{
			CommonModel: CommonModel{
				ClusterName:      "pd-start-script-test",
				ClusterNamespace: "pd-test",
				ClusterDomain:    "",
				PeerServiceName:  "pd-peer",
				AcrossK8s:        false,
			},
			DataDir: "/var/lib/pd",
			Scheme:  "http",
		}
		if c.modifyModel != nil {
			c.modifyModel(model)
		}

		script, err := RenderPDStartScript(model)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
	}
}
