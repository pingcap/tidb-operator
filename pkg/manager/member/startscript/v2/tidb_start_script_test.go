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

func TestRenderTiDBStartScript(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyModel  func(m *TiDBStartScriptModel)
		expectScript string
	}

	cases := []testcase{
		{
			name:        "basic",
			modifyModel: func(m *TiDBStartScriptModel) {},
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

TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
TIDB_ADVERTISE_ADDR=${TIDB_POD_NAME}.tidb-peer.tidb-test.svc
TIDB_PD_ADDR=cluster01-pd:2379
TIDB_EXTRA_ARGS=

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

set | grep TIDB_

ARGS="--store=tikv \
    --advertise-address=${TIDB_ADVERTISE_ADDR} \
    --host=0.0.0.0 \
    --path=${TIDB_PD_ADDR} \
    --config=/etc/tidb/tidb.toml

if [[ -n "${TIDB_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIDB_EXTRA_ARGS}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
		{
			name: "non-empty cluster domain",
			modifyModel: func(m *TiDBStartScriptModel) {
				m.ClusterDomain = "test.com"
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

TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
TIDB_ADVERTISE_ADDR=${TIDB_POD_NAME}.tidb-peer.tidb-test.svc.test.com
TIDB_PD_ADDR=cluster01-pd:2379
TIDB_EXTRA_ARGS=

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

set | grep TIDB_

ARGS="--store=tikv \
    --advertise-address=${TIDB_ADVERTISE_ADDR} \
    --host=0.0.0.0 \
    --path=${TIDB_PD_ADDR} \
    --config=/etc/tidb/tidb.toml

if [[ -n "${TIDB_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIDB_EXTRA_ARGS}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyModel: func(m *TiDBStartScriptModel) {
				m.ClusterDomain = "test.com"
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

TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
TIDB_ADVERTISE_ADDR=${TIDB_POD_NAME}.tidb-peer.tidb-test.svc.test.com
pd_url="cluster01-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="tidb-start-script-test-discovery.tidb-test:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIDB_PD_ADDR=${result}
TIDB_EXTRA_ARGS=

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

set | grep TIDB_

ARGS="--store=tikv \
    --advertise-address=${TIDB_ADVERTISE_ADDR} \
    --host=0.0.0.0 \
    --path=${TIDB_PD_ADDR} \
    --config=/etc/tidb/tidb.toml

if [[ -n "${TIDB_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIDB_EXTRA_ARGS}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyModel: func(m *TiDBStartScriptModel) {
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

TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
TIDB_ADVERTISE_ADDR=${TIDB_POD_NAME}.tidb-peer.tidb-test.svc
pd_url="cluster01-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="tidb-start-script-test-discovery.tidb-test:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIDB_PD_ADDR=${result}
TIDB_EXTRA_ARGS=

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

set | grep TIDB_

ARGS="--store=tikv \
    --advertise-address=${TIDB_ADVERTISE_ADDR} \
    --host=0.0.0.0 \
    --path=${TIDB_PD_ADDR} \
    --config=/etc/tidb/tidb.toml

if [[ -n "${TIDB_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIDB_EXTRA_ARGS}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
		{
			name: "set plugin",
			modifyModel: func(m *TiDBStartScriptModel) {
				m.Plugin = &TiDBPlugin{
					PluginDirectory: "/plugins",
					PluginList:      "plugin-1,plugin-2",
				}
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

TIDB_POD_NAME=${POD_NAME:-$HOSTNAME}
TIDB_ADVERTISE_ADDR=${TIDB_POD_NAME}.tidb-peer.tidb-test.svc
TIDB_PD_ADDR=cluster01-pd:2379
TIDB_EXTRA_ARGS=

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi
TIDB_EXTRA_ARGS="${TIDB_EXTRA_ARGS}  --plugin-dir  /plugins --plugin-load plugin-1,plugin-2"

set | grep TIDB_

ARGS="--store=tikv \
    --advertise-address=${TIDB_ADVERTISE_ADDR} \
    --host=0.0.0.0 \
    --path=${TIDB_PD_ADDR} \
    --config=/etc/tidb/tidb.toml

if [[ -n "${TIDB_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIDB_EXTRA_ARGS}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		model := &TiDBStartScriptModel{
			CommonModel: CommonModel{
				ClusterName:      "tidb-start-script-test",
				ClusterNamespace: "tidb-test",
				ClusterDomain:    "",
				PeerServiceName:  "tidb-peer",
				AcrossK8s:        false,
			},
			PDAddr: "cluster01-pd:2379",
			Plugin: nil,
		}
		if c.modifyModel != nil {
			c.modifyModel(model)
		}

		script, err := RenderTiDBStartScript(model)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
	}
}
