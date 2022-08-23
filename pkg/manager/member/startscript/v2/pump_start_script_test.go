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
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
)

func TestRenderPumpStartScript(t *testing.T) {
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

PUMP_POD_NAME=$HOSTNAME

ARGS="-pd-urls=http://start-script-test-pd:2379 \
-L info \
-log-file= \
-advertise-addr=${PUMP_POD_NAME}.start-script-test-pump:8250 \
-data-dir=/data \
--config=/etc/pump/pump.toml"

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
			name: "set log level",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.Pump.Config = config.New(map[string]interface{}{
					"log-level": "debug",
				})
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

PUMP_POD_NAME=$HOSTNAME

ARGS="-pd-urls=http://start-script-test-pd:2379 \
-L debug \
-log-file= \
-advertise-addr=${PUMP_POD_NAME}.start-script-test-pump:8250 \
-data-dir=/data \
--config=/etc/pump/pump.toml"

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
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "demo.com"
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

PUMP_POD_NAME=$HOSTNAME

ARGS="-pd-urls=http://start-script-test-pd:2379 \
-L info \
-log-file= \
-advertise-addr=${PUMP_POD_NAME}.start-script-test-pump.start-script-test-ns.svc.demo.com:8250 \
-data-dir=/data \
--config=/etc/pump/pump.toml"

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
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "demo.com"
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

PUMP_POD_NAME=$HOSTNAME
pd_url=http://start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS="-pd-urls=${result} \
-L info \
-log-file= \
-advertise-addr=${PUMP_POD_NAME}.start-script-test-pump.start-script-test-ns.svc.demo.com:8250 \
-data-dir=/data \
--config=/etc/pump/pump.toml"

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

PUMP_POD_NAME=$HOSTNAME
pd_url=http://start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS="-pd-urls=${result} \
-L info \
-log-file= \
-advertise-addr=${PUMP_POD_NAME}.start-script-test-pump.start-script-test-ns.svc:8250 \
-data-dir=/data \
--config=/etc/pump/pump.toml"

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
			name: "heterogeneous cluster without local pd",
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

PUMP_POD_NAME=$HOSTNAME

ARGS="-pd-urls=http://target-cluster-pd:2379 \
-L info \
-log-file= \
-advertise-addr=${PUMP_POD_NAME}.start-script-test-pump:8250 \
-data-dir=/data \
--config=/etc/pump/pump.toml"

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
			name: "heterogeneous cluster when across k8s",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD = nil
				tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "target-cluster"}
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

PUMP_POD_NAME=$HOSTNAME
pd_url=http://start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done

ARGS="-pd-urls=${result} \
-L info \
-log-file= \
-advertise-addr=${PUMP_POD_NAME}.start-script-test-pump.start-script-test-ns.svc:8250 \
-data-dir=/data \
--config=/etc/pump/pump.toml"

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

		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				Pump: &v1alpha1.PumpSpec{},
			},
		}
		tc.Name = "start-script-test"
		tc.Namespace = "start-script-test-ns"
		if c.modifyTC != nil {
			c.modifyTC(tc)
		}

		script, err := RenderPumpStartScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}
