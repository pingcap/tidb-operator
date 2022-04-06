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

func TestRenderTiKVStartScript(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyModel  func(m *TiKVStartScriptModel)
		expectScript string
	}

	cases := []testcase{
		{
			name:        "basic",
			modifyModel: func(m *TiKVStartScriptModel) {},
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
TIKV_PD_ADDR=cluster01-pd:2379
TIKV_ADVERTISE_ADDR=${TIKV_POD_NAME}.tikv-peer.tikv-test.svc:20160
TIKV_DATA_DIR=/var/lib/tikv
TIKV_CAPACITY=${CAPACITY}
TIKV_EXTRA_ARGS=

if [ ! -z "${STORE_LABELS:-}" ]; then
    LABELS="--labels ${STORE_LABELS} "
    TIKV_EXTRA_ARGS="${TIKV_EXTRA_ARGS} ${LABELS}"
fi

set | grep TIKV_

ARGS="--pd=${TIKV_PD_ADDR} \
    --advertise-addr=${TIKV_ADVERTISE_ADDR} \
    --addr=0.0.0.0:20160 \
    --status-addr=0.0.0.0:20180 \
    --data-dir=${TIKV_DATA_DIR} \
    --capacity=${TIKV_CAPACITY} \
    --config=/etc/tikv/tikv.toml

if [[ -n "${TIKV_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIKV_EXTRA_ARGS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "enable AdvertiseAddr",
			modifyModel: func(m *TiKVStartScriptModel) {
				m.AdvertiseStatusAddr = "test-tikv-1.test-tikv-peer.namespace.svc"
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
TIKV_PD_ADDR=cluster01-pd:2379
TIKV_ADVERTISE_ADDR=${TIKV_POD_NAME}.tikv-peer.tikv-test.svc:20160
TIKV_DATA_DIR=/var/lib/tikv
TIKV_CAPACITY=${CAPACITY}
TIKV_EXTRA_ARGS=
TIKV_EXTRA_ARGS="--advertise-status-addr=test-tikv-1.test-tikv-peer.namespace.svc:20180"

if [ ! -z "${STORE_LABELS:-}" ]; then
    LABELS="--labels ${STORE_LABELS} "
    TIKV_EXTRA_ARGS="${TIKV_EXTRA_ARGS} ${LABELS}"
fi

set | grep TIKV_

ARGS="--pd=${TIKV_PD_ADDR} \
    --advertise-addr=${TIKV_ADVERTISE_ADDR} \
    --addr=0.0.0.0:20160 \
    --status-addr=0.0.0.0:20180 \
    --data-dir=${TIKV_DATA_DIR} \
    --capacity=${TIKV_CAPACITY} \
    --config=/etc/tikv/tikv.toml

if [[ -n "${TIKV_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIKV_EXTRA_ARGS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "set data dir",
			modifyModel: func(m *TiKVStartScriptModel) {
				m.DataDir = "/data/tikv"
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
TIKV_PD_ADDR=cluster01-pd:2379
TIKV_ADVERTISE_ADDR=${TIKV_POD_NAME}.tikv-peer.tikv-test.svc:20160
TIKV_DATA_DIR=/data/tikv
TIKV_CAPACITY=${CAPACITY}
TIKV_EXTRA_ARGS=

if [ ! -z "${STORE_LABELS:-}" ]; then
    LABELS="--labels ${STORE_LABELS} "
    TIKV_EXTRA_ARGS="${TIKV_EXTRA_ARGS} ${LABELS}"
fi

set | grep TIKV_

ARGS="--pd=${TIKV_PD_ADDR} \
    --advertise-addr=${TIKV_ADVERTISE_ADDR} \
    --addr=0.0.0.0:20160 \
    --status-addr=0.0.0.0:20180 \
    --data-dir=${TIKV_DATA_DIR} \
    --capacity=${TIKV_CAPACITY} \
    --config=/etc/tikv/tikv.toml

if [[ -n "${TIKV_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIKV_EXTRA_ARGS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "set cluster domain",
			modifyModel: func(m *TiKVStartScriptModel) {
				m.ClusterDomain = "cluster.local"
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
TIKV_PD_ADDR=cluster01-pd:2379
TIKV_ADVERTISE_ADDR=${TIKV_POD_NAME}.tikv-peer.tikv-test.svc.cluster.local:20160
TIKV_DATA_DIR=/var/lib/tikv
TIKV_CAPACITY=${CAPACITY}
TIKV_EXTRA_ARGS=

if [ ! -z "${STORE_LABELS:-}" ]; then
    LABELS="--labels ${STORE_LABELS} "
    TIKV_EXTRA_ARGS="${TIKV_EXTRA_ARGS} ${LABELS}"
fi

set | grep TIKV_

ARGS="--pd=${TIKV_PD_ADDR} \
    --advertise-addr=${TIKV_ADVERTISE_ADDR} \
    --addr=0.0.0.0:20160 \
    --status-addr=0.0.0.0:20180 \
    --data-dir=${TIKV_DATA_DIR} \
    --capacity=${TIKV_CAPACITY} \
    --config=/etc/tikv/tikv.toml

if [[ -n "${TIKV_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIKV_EXTRA_ARGS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyModel: func(m *TiKVStartScriptModel) {
				m.ClusterDomain = "cluster.local"
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
pd_url="cluster01-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="tikv-start-script-test-discovery.tikv-test:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIKV_PD_ADDR=${result}
TIKV_ADVERTISE_ADDR=${TIKV_POD_NAME}.tikv-peer.tikv-test.svc.cluster.local:20160
TIKV_DATA_DIR=/var/lib/tikv
TIKV_CAPACITY=${CAPACITY}
TIKV_EXTRA_ARGS=

if [ ! -z "${STORE_LABELS:-}" ]; then
    LABELS="--labels ${STORE_LABELS} "
    TIKV_EXTRA_ARGS="${TIKV_EXTRA_ARGS} ${LABELS}"
fi

set | grep TIKV_

ARGS="--pd=${TIKV_PD_ADDR} \
    --advertise-addr=${TIKV_ADVERTISE_ADDR} \
    --addr=0.0.0.0:20160 \
    --status-addr=0.0.0.0:20180 \
    --data-dir=${TIKV_DATA_DIR} \
    --capacity=${TIKV_CAPACITY} \
    --config=/etc/tikv/tikv.toml

if [[ -n "${TIKV_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIKV_EXTRA_ARGS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyModel: func(m *TiKVStartScriptModel) {
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
pd_url="cluster01-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="tikv-start-script-test-discovery.tikv-test:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
TIKV_PD_ADDR=${result}
TIKV_ADVERTISE_ADDR=${TIKV_POD_NAME}.tikv-peer.tikv-test.svc:20160
TIKV_DATA_DIR=/var/lib/tikv
TIKV_CAPACITY=${CAPACITY}
TIKV_EXTRA_ARGS=

if [ ! -z "${STORE_LABELS:-}" ]; then
    LABELS="--labels ${STORE_LABELS} "
    TIKV_EXTRA_ARGS="${TIKV_EXTRA_ARGS} ${LABELS}"
fi

set | grep TIKV_

ARGS="--pd=${TIKV_PD_ADDR} \
    --advertise-addr=${TIKV_ADVERTISE_ADDR} \
    --addr=0.0.0.0:20160 \
    --status-addr=0.0.0.0:20180 \
    --data-dir=${TIKV_DATA_DIR} \
    --capacity=${TIKV_CAPACITY} \
    --config=/etc/tikv/tikv.toml

if [[ -n "${TIKV_EXTRA_ARGS}" ]]; then
    ARGS="${ARGS} ${TIKV_EXTRA_ARGS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		model := &TiKVStartScriptModel{
			CommonModel: CommonModel{
				ClusterName:      "tikv-start-script-test",
				ClusterNamespace: "tikv-test",
				ClusterDomain:    "",
				PeerServiceName:  "tikv-peer",
				AcrossK8s:        false,
			},
			DataDir:             "/var/lib/tikv",
			PDAddr:              "cluster01-pd:2379",
			AdvertiseStatusAddr: "",
		}
		if c.modifyModel != nil {
			c.modifyModel(model)
		}

		script, err := RenderTiKVStartScript(model)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
	}
}
