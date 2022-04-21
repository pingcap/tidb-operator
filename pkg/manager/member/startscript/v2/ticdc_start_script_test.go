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

func TestRenderTiCDCStartScript(t *testing.T) {
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

TICDC_POD_NAME=${POD_NAME}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc:8301 \
--gc-ttl=86400 \
--log-file= \
--log-level=info \
--pd=http://start-script-test-pd:2379"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
`,
		},
		{
			name: "set cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
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

TICDC_POD_NAME=${POD_NAME}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc.cluster.local:8301 \
--gc-ttl=86400 \
--log-file= \
--log-level=info \
--pd=http://start-script-test-pd:2379"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
`,
		},
		{
			name: "set gc ttl",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				cfg := v1alpha1.NewCDCConfig()
				cfg.Set("gc-ttl", 3600)
				tc.Spec.TiCDC.Config = cfg
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

TICDC_POD_NAME=${POD_NAME}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc:8301 \
--gc-ttl=3600 \
--log-file= \
--log-level=info \
--pd=http://start-script-test-pd:2379"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
`,
		},
		{
			name: "set log file and log level",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				cfg := v1alpha1.NewCDCConfig()
				cfg.Set("log-file", "/tmp/ticdc.log")
				cfg.Set("log-level", "debug")
				tc.Spec.TiCDC.Config = cfg
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

TICDC_POD_NAME=${POD_NAME}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc:8301 \
--gc-ttl=86400 \
--log-file=/tmp/ticdc.log \
--log-level=debug \
--pd=http://start-script-test-pd:2379"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
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

TICDC_POD_NAME=${POD_NAME}
pd_url=http://start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc.cluster.local:8301 \
--gc-ttl=86400 \
--log-file= \
--log-level=info \
--pd=${result}"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
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

TICDC_POD_NAME=${POD_NAME}
pd_url=http://start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc:8301 \
--gc-ttl=86400 \
--log-file= \
--log-level=info \
--pd=${result}"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
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

TICDC_POD_NAME=${POD_NAME}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc:8301 \
--gc-ttl=86400 \
--log-file= \
--log-level=info \
--pd=http://target-cluster-pd:2379"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
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

TICDC_POD_NAME=${POD_NAME}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc:8301 \
--gc-ttl=86400 \
--log-file= \
--log-level=info \
--pd=https://start-script-test-pd:2379"
ARGS="${ARGS} --ca=/var/lib/ticdc-tls/ca.crt --cert=/var/lib/ticdc-tls/tls.crt --key=/var/lib/ticdc-tls/tls.key"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
`,
		},
		{
			name: "set config",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				cfg := v1alpha1.NewCDCConfig()
				cfg.Set("log-backup", 4)
				tc.Spec.TiCDC.Config = cfg
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

TICDC_POD_NAME=${POD_NAME}

ARGS="--addr=0.0.0.0:8301 \
--advertise-addr=${TICDC_POD_NAME}.start-script-test-ticdc-peer.start-script-test-ns.svc:8301 \
--gc-ttl=86400 \
--log-file= \
--log-level=info \
--pd=http://start-script-test-pd:2379"
ARGS="${ARGS} --config=/etc/ticdc/ticdc.toml"

echo "start ticdc-server ..."
echo "/cdc server ${ARGS}"
exec /cdc server ${ARGS}
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				TiCDC: &v1alpha1.TiCDCSpec{},
			},
		}
		tc.Name = "start-script-test"
		tc.Namespace = "start-script-test-ns"
		if c.modifyTC != nil {
			c.modifyTC(tc)
		}

		script, err := RenderTiCDCStartScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}
