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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
)

func TestRenderTiKVStartScript(t *testing.T) {
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--pd=start-script-test-pd:2379 \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "set data sub dir",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.DataSubDir = "tikv-data"
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--pd=start-script-test-pd:2379 \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv/tikv-data \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "enable dynamic configuration",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				enable := true
				tc.Spec.EnableDynamicConfiguration = &enable
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--pd=start-script-test-pd:2379 \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"
ARGS="${ARGS} --advertise-status-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc:20180"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "enable dynamic configuration with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				enable := true
				tc.Spec.EnableDynamicConfiguration = &enable
				tc.Spec.ClusterDomain = "cluster.local"
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--pd=start-script-test-pd:2379 \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc.cluster.local:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"
ARGS="${ARGS} --advertise-status-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc.cluster.local:20180"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "set cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster.local"
				tc.Spec.AcrossK8s = false
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--pd=start-script-test-pd:2379 \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc.cluster.local:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster.local"
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
pd_url=start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS="--pd=${result} \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc.cluster.local:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}
pd_url=start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS="--pd=${result} \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "heterogeneous without local pd",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD = nil
				tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "target-cluster"}
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

TIKV_POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--pd=target-cluster-pd:2379 \
--advertise-addr=${TIKV_POD_NAME}.start-script-test-tikv-peer.start-script-test-ns.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS="--labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				TiKV: &v1alpha1.TiKVSpec{},
			},
		}
		tc.Name = "start-script-test"
		tc.Namespace = "start-script-test-ns"
		tc.Spec.AcrossK8s = false
		tc.Spec.ClusterDomain = ""
		tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: false}

		if c.modifyTC != nil {
			c.modifyTC(tc)
		}

		script, err := RenderTiKVStartScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}
