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

func TestRenderTiFlashStartScript(t *testing.T) {
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

ARGS="--config-file /data0/config.toml"

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash server ${ARGS}
`,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				TiFlash: &v1alpha1.TiFlashSpec{},
			},
		}
		tc.Name = "start-script-test"
		tc.Namespace = "start-script-test-ns"

		if c.modifyTC != nil {
			c.modifyTC(tc)
		}

		script, err := RenderTiFlashStartScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}

func TestRenderTiFlashStartScriptWithStartArgs(t *testing.T) {
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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
PD_ADDRESS="start-script-test-pd:2379"
ARGS="server --config-file /etc/tiflash/config_templ.toml \
-- \
--flash.proxy.advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20170 \
--flash.service_addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:3930 \
--raft.pd_addr=${PD_ADDRESS}
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash ${ARGS}
`,
		},
		{
			name: "enable AdvertiseAddr",
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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
PD_ADDRESS="start-script-test-pd:2379"
ARGS="server --config-file /etc/tiflash/config_templ.toml \
-- \
--flash.proxy.advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20170 \
--flash.proxy.advertise-status-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20292 \
--flash.service_addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:3930 \
--raft.pd_addr=${PD_ADDRESS}
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash ${ARGS}
`,
		},
		{
			name: "across k8s",
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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
PD_ADDRESS="${result}"
pd_url=start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g' | sed 's/https:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PD_ADDRESS=${result}
ARGS="server --config-file /etc/tiflash/config_templ.toml \
-- \
--flash.proxy.advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:20170 \
--flash.service_addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:3930 \
--raft.pd_addr=${PD_ADDRESS}
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash ${ARGS}
`,
		},
		{
			name: "across k8s with tls enabled",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster.local"
				tc.Spec.AcrossK8s = true
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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
PD_ADDRESS="${result}"
pd_url=start-script-test-pd:2379
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url=start-script-test-discovery.start-script-test-ns:10261
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g' | sed 's/https:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep $((RANDOM % 5))
done
PD_ADDRESS=${result}
ARGS="server --config-file /etc/tiflash/config_templ.toml \
-- \
--flash.proxy.advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:20170 \
--flash.service_addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:3930 \
--raft.pd_addr=${PD_ADDRESS}
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash ${ARGS}
`,
		},
		{
			name: "basic with ipv6",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PreferIPv6 = true
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

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
PD_ADDRESS="start-script-test-pd:2379"
ARGS="server --config-file /etc/tiflash/config_templ.toml \
-- \
--flash.proxy.advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20170 \
--flash.service_addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:3930 \
--raft.pd_addr=${PD_ADDRESS}
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash ${ARGS}
`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Logf("test case: %s", c.name)

			tc := &v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{},
				},
			}
			tc.Name = "start-script-test"
			tc.Namespace = "start-script-test-ns"

			if c.modifyTC != nil {
				c.modifyTC(tc)
			}

			script, err := RenderTiFlashStartScriptWithStartArgs(tc)
			g.Expect(err).Should(gomega.Succeed())
			if diff := cmp.Diff(c.expectScript, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
				t.Logf("got: %s", script)
			}
			g.Expect(validateScript(script)).Should(gomega.Succeed())
		})
	}
}
