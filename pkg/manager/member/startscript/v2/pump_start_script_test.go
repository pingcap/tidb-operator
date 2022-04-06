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

func TestRenderPumpStartScript(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyModel  func(m *PumpStartScriptModel)
		expectScript string
	}

	cases := []testcase{
		{
			name:        "basic",
			modifyModel: func(m *PumpStartScriptModel) {},
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

PUMP_POD_NAME=$HOSTNAME
PUMP_PD_ADDR=http://demo-pd:2379
PUMP_LOG_LEVEL=INFO
PUMP_ADVERTISE_ADDR=${PUMP_POD_NAME}.pump-start-script-test-pump:8250
PUMP_EXTRA_ARGS=

set | grep PUMP_

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml

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
`,
		},
		{
			name: "non-empty cluster domain",
			modifyModel: func(m *PumpStartScriptModel) {
				m.ClusterDomain = "demo.com"
				m.AcrossK8s = false
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

PUMP_POD_NAME=$HOSTNAME
PUMP_PD_ADDR=http://demo-pd:2379
PUMP_LOG_LEVEL=INFO
PUMP_ADVERTISE_ADDR=${PUMP_POD_NAME}.pump-start-script-test-pump.pump-test.svc.demo.com:8250
PUMP_EXTRA_ARGS=

set | grep PUMP_

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml

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
`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyModel: func(m *PumpStartScriptModel) {
				m.ClusterDomain = "demo.com"
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

PUMP_POD_NAME=$HOSTNAME
pd_url="http://demo-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="pump-start-script-test-discovery.pump-test:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PUMP_PD_ADDR=${result}
PUMP_LOG_LEVEL=INFO
PUMP_ADVERTISE_ADDR=${PUMP_POD_NAME}.pump-start-script-test-pump.pump-test.svc.demo.com:8250
PUMP_EXTRA_ARGS=

set | grep PUMP_

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml

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
`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyModel: func(m *PumpStartScriptModel) {
				m.ClusterDomain = ""
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

PUMP_POD_NAME=$HOSTNAME
pd_url="http://demo-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="pump-start-script-test-discovery.pump-test:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PUMP_PD_ADDR=${result}
PUMP_LOG_LEVEL=INFO
PUMP_ADVERTISE_ADDR=${PUMP_POD_NAME}.pump-start-script-test-pump.pump-test.svc:8250
PUMP_EXTRA_ARGS=

set | grep PUMP_

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml

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
`,
		},
		{
			name: "specify pd addr",
			modifyModel: func(m *PumpStartScriptModel) {
				m.PDAddr = "http://target-pd:2379"
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

PUMP_POD_NAME=$HOSTNAME
PUMP_PD_ADDR=http://target-pd:2379
PUMP_LOG_LEVEL=INFO
PUMP_ADVERTISE_ADDR=${PUMP_POD_NAME}.pump-start-script-test-pump:8250
PUMP_EXTRA_ARGS=

set | grep PUMP_

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml

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
`,
		},
		{
			name: "specify pd addr when heterogeneous across k8s",
			modifyModel: func(m *PumpStartScriptModel) {
				m.PDAddr = "http://target-pd:2379"
				m.ClusterDomain = "demo.com"
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

PUMP_POD_NAME=$HOSTNAME
pd_url="http://target-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="pump-start-script-test-discovery.pump-test:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PUMP_PD_ADDR=${result}
PUMP_LOG_LEVEL=INFO
PUMP_ADVERTISE_ADDR=${PUMP_POD_NAME}.pump-start-script-test-pump.pump-test.svc.demo.com:8250
PUMP_EXTRA_ARGS=

set | grep PUMP_

ARGS="-pd-urls=${PUMP_PD_ADDR} \
    -L ${PUMP_LOG_LEVEL} \
    -log-file= \
    -advertise-addr=${PUMP_ADVERTISE_ADDR} \
    -data-dir=/data \
    --config=/etc/pump/pump.toml

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
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		model := &PumpStartScriptModel{
			CommonModel: CommonModel{
				ClusterName:      "pump-start-script-test",
				ClusterNamespace: "pump-test",
				ClusterDomain:    "",
				PeerServiceName:  "",
				AcrossK8s:        false,
			},
			PDAddr:   "http://demo-pd:2379",
			LogLevel: "INFO",
		}
		if c.modifyModel != nil {
			c.modifyModel(model)
		}

		script, err := RenderPumpStartScript(model)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
	}
}
