// Copyright 2024 PingCAP, Inc.
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

func TestRenderTiProxyStartScript(t *testing.T) {
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

TIPROXY_POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="--config=/etc/proxy/proxy.toml"
if [[ "$(/bin/tiproxy --help)" == *"advertise-addr"* ]]; then
  ARGS="${ARGS} --advertise-addr=${TIPROXY_POD_NAME}.start-script-test-tiproxy-peer.start-script-test-ns.svc"
fi

echo "starting: tiproxy ${ARGS}"
exec /bin/tiproxy ${ARGS}
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

TIPROXY_POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="--config=/etc/proxy/proxy.toml"
if [[ "$(/bin/tiproxy --help)" == *"advertise-addr"* ]]; then
  ARGS="${ARGS} --advertise-addr=${TIPROXY_POD_NAME}.start-script-test-tiproxy-peer.start-script-test-ns.svc.cluster.local"
fi

echo "starting: tiproxy ${ARGS}"
exec /bin/tiproxy ${ARGS}
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

		script, err := RenderTiProxyStartScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}
