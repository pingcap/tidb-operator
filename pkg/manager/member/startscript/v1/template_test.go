// Copyright 2020 PingCAP, Inc.
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

package v1

import (
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	"mvdan.cc/sh/v3/syntax"
)

func TestRenderTiDBStartScript(t *testing.T) {
	tests := []struct {
		name     string
		modifyTC func(tc *v1alpha1.TidbCluster)
		result   string
	}{
		{
			name:     "basic",
			modifyTC: func(tc *v1alpha1.TidbCluster) {},
			result: `#!/bin/sh

# This script is used to start tidb containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#
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
ARGS="--store=tikv \
--advertise-address=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc \
--host=0.0.0.0 \
--path=${CLUSTER_NAME}-pd:2379 \
--config=/etc/tidb/tidb.toml
"

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    ARGS="${ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    ARGS="${ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
		{
			name: "non-empty cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "test.com"
			},
			result: `#!/bin/sh

# This script is used to start tidb containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#
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
ARGS="--store=tikv \
--advertise-address=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.test.com \
--host=0.0.0.0 \
--path=${CLUSTER_NAME}-pd:2379 \
--config=/etc/tidb/tidb.toml
"

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    ARGS="${ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    ARGS="${ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "test.com"
				tc.Spec.AcrossK8s = true
			},
			result: `#!/bin/sh

# This script is used to start tidb containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#
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
pd_url="${CLUSTER_NAME}-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="${CLUSTER_NAME}-discovery.${NAMESPACE}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
echo "waiting for the verification of PD endpoints ..."
sleep $((RANDOM % 5))
done

ARGS="--store=tikv \
--advertise-address=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.test.com \
--host=0.0.0.0 \
--path=${result} \

--config=/etc/tidb/tidb.toml
"

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    ARGS="${ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    ARGS="${ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = ""
				tc.Spec.AcrossK8s = true
			},
			result: `#!/bin/sh

# This script is used to start tidb containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#
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
pd_url="${CLUSTER_NAME}-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="${CLUSTER_NAME}-discovery.${NAMESPACE}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g'); do
echo "waiting for the verification of PD endpoints ..."
sleep $((RANDOM % 5))
done

ARGS="--store=tikv \
--advertise-address=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc \
--host=0.0.0.0 \
--path=${result} \

--config=/etc/tidb/tidb.toml
"

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    ARGS="${ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    ARGS="${ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
exec /tidb-server ${ARGS}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			tc := &v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{},
				},
			}
			tc.Name = "test-tidb"
			if tt.modifyTC != nil {
				tt.modifyTC(tc)
			}

			script, err := RenderTiDBStartScript(tc)
			g.Expect(err).Should(gomega.Succeed())
			if diff := cmp.Diff(tt.result, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
			g.Expect(validateScript(script)).Should(gomega.Succeed())
		})
	}
}

func TestRenderTiKVStartScript(t *testing.T) {
	tests := []struct {
		name     string
		modifyTC func(tc *v1alpha1.TidbCluster)

		result string
	}{
		{
			name: "basic",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
			},
			result: `#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
ARGS="--pd=http://${CLUSTER_NAME}-pd:2379 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "enable AdvertiseAddr",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				enable := true
				tc.Spec.EnableDynamicConfiguration = &enable
			},
			result: `#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
ARGS="--pd=http://${CLUSTER_NAME}-pd:2379 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--advertise-status-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
		{
			name: "enable AdvertiseAddr and non-empty dataSubDir",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				enable := true
				tc.Spec.EnableDynamicConfiguration = &enable
				tc.Spec.TiKV.DataSubDir = "data"
			},
			result: `#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
ARGS="--pd=http://${CLUSTER_NAME}-pd:2379 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--advertise-status-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20180 \
--data-dir=/var/lib/tikv/data \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
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
			},
			result: `#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
ARGS="--pd=http://${CLUSTER_NAME}-pd:2379 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
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
			result: `#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
pd_url="http://${CLUSTER_NAME}-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="${CLUSTER_NAME}-discovery.${NAMESPACE}:10261"

until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep $((RANDOM % 5))
done

ARGS="--pd=${result} \

--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
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
			result: `#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
pd_url="http://${CLUSTER_NAME}-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="${CLUSTER_NAME}-discovery.${NAMESPACE}:10261"

until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep $((RANDOM % 5))
done

ARGS="--pd=${result} \

--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			tc := &v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{},
				},
			}
			tc.Name = "test-tikv"
			if tt.modifyTC != nil {
				tt.modifyTC(tc)
			}

			script, err := RenderTiKVStartScript(tc)
			g.Expect(err).Should(gomega.Succeed())
			if diff := cmp.Diff(tt.result, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
			g.Expect(validateScript(script)).Should(gomega.Succeed())
		})
	}
}

func TestRenderPDStartScript(t *testing.T) {
	tests := []struct {
		name     string
		modifyTC func(tc *v1alpha1.TidbCluster)
		result   string
	}{
		{
			name: "https scheme",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			},
			result: `#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
# the general form of variable PEER_SERVICE_NAME is: "<clusterName>-pd-peer"
cluster_name=` + "`" + `echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'` + "`" + `
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=` + "`" + `echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"` + "`" + `
elapseTime=0
period=1
threshold=30
while true; do
sleep ${period}
elapseTime=$(( elapseTime+period ))

if [[ ${elapseTime} -ge ${threshold} ]]
then
echo "waiting for pd cluster ready timeout" >&2
exit 1
fi

if nslookup ${domain} 2>/dev/null
then
echo "nslookup domain ${domain}.svc success"
break
else
echo "nslookup domain ${domain} failed" >&2
fi
done

ARGS="--data-dir=/var/lib/pd \
--name=${POD_NAME} \
--peer-urls=https://0.0.0.0:2380 \
--advertise-peer-urls=https://${domain}:2380 \
--client-urls=https://0.0.0.0:2379 \
--advertise-client-urls=https://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ -f /var/lib/pd/join ]]
then
# The content of the join file is:
#   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
# The --join args must be:
#   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
join=` + "`" + `cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]
then
until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
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
			name: "non-empty dataSubDir",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.DataSubDir = "data"
			},
			result: `#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
# the general form of variable PEER_SERVICE_NAME is: "<clusterName>-pd-peer"
cluster_name=` + "`" + `echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'` + "`" + `
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=` + "`" + `echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"` + "`" + `
elapseTime=0
period=1
threshold=30
while true; do
sleep ${period}
elapseTime=$(( elapseTime+period ))

if [[ ${elapseTime} -ge ${threshold} ]]
then
echo "waiting for pd cluster ready timeout" >&2
exit 1
fi

if nslookup ${domain} 2>/dev/null
then
echo "nslookup domain ${domain}.svc success"
break
else
echo "nslookup domain ${domain} failed" >&2
fi
done

ARGS="--data-dir=/var/lib/pd/data \
--name=${POD_NAME} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${domain}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ -f /var/lib/pd/data/join ]]
then
# The content of the join file is:
#   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
# The --join args must be:
#   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
join=` + "`" + `cat /var/lib/pd/data/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/data/member/wal ]]
then
until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
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
			name: "non-empty clusterDomain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster.local"
			},
			result: `#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
# the general form of variable PEER_SERVICE_NAME is: "<clusterName>-pd-peer"
cluster_name=` + "`" + `echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'` + "`" + `
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=` + "`" + `echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"` + "`" + `
elapseTime=0
period=1
threshold=30
while true; do
sleep ${period}
elapseTime=$(( elapseTime+period ))

if [[ ${elapseTime} -ge ${threshold} ]]
then
echo "waiting for pd cluster ready timeout" >&2
exit 1
fi

if nslookup ${domain} 2>/dev/null
then
echo "nslookup domain ${domain}.svc success"
break
else
echo "nslookup domain ${domain} failed" >&2
fi
done

ARGS="--data-dir=/var/lib/pd \
--name=${domain} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${domain}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ -f /var/lib/pd/join ]]
then
# The content of the join file is:
#   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
# The --join args must be:
#   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
join=` + "`" + `cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]
then
until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
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
			name: "across k8s without setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = ""
				tc.Spec.AcrossK8s = true
			},
			result: `#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
# the general form of variable PEER_SERVICE_NAME is: "<clusterName>-pd-peer"
cluster_name=` + "`" + `echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'` + "`" + `
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=` + "`" + `echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"` + "`" + `
elapseTime=0
period=1
threshold=30
while true; do
sleep ${period}
elapseTime=$(( elapseTime+period ))

if [[ ${elapseTime} -ge ${threshold} ]]
then
echo "waiting for pd cluster ready timeout" >&2
exit 1
fi

if nslookup ${domain} 2>/dev/null
then
echo "nslookup domain ${domain}.svc success"
break
else
echo "nslookup domain ${domain} failed" >&2
fi
done

ARGS="--data-dir=/var/lib/pd \
--name=${domain} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${domain}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ -f /var/lib/pd/join ]]
then
# The content of the join file is:
#   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
# The --join args must be:
#   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
join=` + "`" + `cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]
then
until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
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
			name: "across k8s with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster.local"
				tc.Spec.AcrossK8s = true
			},
			result: `#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

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
# the general form of variable PEER_SERVICE_NAME is: "<clusterName>-pd-peer"
cluster_name=` + "`" + `echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'` + "`" + `
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=` + "`" + `echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"` + "`" + `
elapseTime=0
period=1
threshold=30
while true; do
sleep ${period}
elapseTime=$(( elapseTime+period ))

if [[ ${elapseTime} -ge ${threshold} ]]
then
echo "waiting for pd cluster ready timeout" >&2
exit 1
fi

if nslookup ${domain} 2>/dev/null
then
echo "nslookup domain ${domain}.svc success"
break
else
echo "nslookup domain ${domain} failed" >&2
fi
done

ARGS="--data-dir=/var/lib/pd \
--name=${domain} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${domain}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ -f /var/lib/pd/join ]]
then
# The content of the join file is:
#   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
# The --join args must be:
#   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
join=` + "`" + `cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d /var/lib/pd/member/wal ]]
then
until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			tc := &v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{},
				},
			}
			tc.Name = "test-pd"
			if tt.modifyTC != nil {
				tt.modifyTC(tc)
			}

			script, err := RenderPDStartScript(tc)
			g.Expect(err).Should(gomega.Succeed())
			if diff := cmp.Diff(tt.result, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}

func TestRenderPumpStartScript(t *testing.T) {
	tests := []struct {
		name string

		modifyTC func(tc *v1alpha1.TidbCluster)
		result   string
	}{
		{
			name:     "basic",
			modifyTC: func(tc *v1alpha1.TidbCluster) {},
			result: `set -euo pipefail

/pump \
-pd-urls=http://demo-pd:2379 \
-L=info \
-advertise-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.demo-pump:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi`,
		},
		{
			name: "non-empty cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "demo.com"
			},
			result: `set -euo pipefail

/pump \
-pd-urls=http://demo-pd:2379 \
-L=info \
-advertise-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.demo-pump.demo-ns.svc.demo.com:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "demo.com"
				tc.Spec.AcrossK8s = true
			},
			result: `
pd_url="http://demo-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="demo-discovery.demo-ns:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep $((RANDOM % 5))
done

pd_url=$result

set -euo pipefail

/pump \
-pd-urls=$pd_url \
-L=info \
-advertise-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.demo-pump.demo-ns.svc.demo.com:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = ""
				tc.Spec.AcrossK8s = true
			},
			result: `
pd_url="http://demo-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="demo-discovery.demo-ns:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep $((RANDOM % 5))
done

pd_url=$result

set -euo pipefail

/pump \
-pd-urls=$pd_url \
-L=info \
-advertise-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.demo-pump.demo-ns.svc:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi`,
		},
		{
			name: "heterogeneous cluster without local pd",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD = nil
				tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "target"}
			},
			result: `set -euo pipefail

/pump \
-pd-urls=http://target-pd:2379 \
-L=info \
-advertise-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.demo-pump:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi`,
		},
		{
			name: "heterogeneous cluster when across k8s",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD = nil
				tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "target"}
				tc.Spec.ClusterDomain = "demo.com"
				tc.Spec.AcrossK8s = true
			},
			result: `
pd_url="http://demo-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="demo-discovery.demo-ns:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep $((RANDOM % 5))
done

pd_url=$result

set -euo pipefail

/pump \
-pd-urls=$pd_url \
-L=info \
-advertise-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.demo-pump.demo-ns.svc.demo.com:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			tc := &v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					Pump: &v1alpha1.PumpSpec{},
				},
			}
			tc.Name = "demo"
			tc.Namespace = "demo-ns"
			if tt.modifyTC != nil {
				tt.modifyTC(tc)
			}

			script, err := RenderPumpStartScript(tc)
			g.Expect(err).Should(gomega.Succeed())
			if diff := cmp.Diff(tt.result, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}

func TestRenderTiCDCStartScript(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyTC     func(tc *v1alpha1.TidbCluster)
		expectScript string
	}

	cases := []testcase{
		{
			name:         "basic",
			modifyTC:     func(tc *v1alpha1.TidbCluster) {},
			expectScript: `/cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301 --gc-ttl=86400 --log-file= --log-level=info --pd=http://start-script-test-pd:2379`,
		},
		{
			name: "set cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster.local"
			},
			expectScript: `/cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:8301 --gc-ttl=86400 --log-file= --log-level=info --pd=http://start-script-test-pd:2379`,
		},
		{
			name: "set gc ttl",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				cfg := v1alpha1.NewCDCConfig()
				cfg.Set("gc-ttl", 3600)
				tc.Spec.TiCDC.Config = cfg
			},
			expectScript: `/cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301 --gc-ttl=3600 --log-file= --log-level=info --pd=http://start-script-test-pd:2379`,
		},
		{
			name: "set log file and log level",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				cfg := v1alpha1.NewCDCConfig()
				cfg.Set("log-file", "/tmp/ticdc.log")
				cfg.Set("log-level", "debug")
				tc.Spec.TiCDC.Config = cfg
			},
			expectScript: `/cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301 --gc-ttl=86400 --log-file=/tmp/ticdc.log --log-level=debug --pd=http://start-script-test-pd:2379`,
		},
		{
			name: "across k8s with setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = "cluster.local"
				tc.Spec.AcrossK8s = true
			},
			expectScript: `set -uo pipefail
pd_url="http://start-script-test-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="start-script-test-discovery.${NAMESPACE}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep 2
done

exec /cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local:8301 --gc-ttl=86400 --log-file= --log-level=info --pd=${result}`,
		},
		{
			name: "across k8s without setting cluster domain",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.ClusterDomain = ""
				tc.Spec.AcrossK8s = true
			},
			expectScript: `set -uo pipefail
pd_url="http://start-script-test-pd:2379"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="start-script-test-discovery.${NAMESPACE}:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep 2
done

exec /cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301 --gc-ttl=86400 --log-file= --log-level=info --pd=${result}`,
		},
		{
			name: "heterogeneous without local pd",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD = nil
				tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "target-cluster"}
			},
			expectScript: `/cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301 --gc-ttl=86400 --log-file= --log-level=info --pd=http://target-cluster-pd:2379`,
		},
		{
			name: "enable tls",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			},
			expectScript: `/cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301 --gc-ttl=86400 --log-file= --log-level=info --ca=/var/lib/ticdc-tls/ca.crt --cert=/var/lib/ticdc-tls/tls.crt --key=/var/lib/ticdc-tls/tls.key --pd=https://start-script-test-pd:2379`,
		},
		{
			name: "set config",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				cfg := v1alpha1.NewCDCConfig()
				cfg.Set("log-backup", 4)
				tc.Spec.TiCDC.Config = cfg
			},
			expectScript: `/cdc server --addr=0.0.0.0:8301 --advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301 --gc-ttl=86400 --log-file= --log-level=info --pd=http://start-script-test-pd:2379 --config=/etc/ticdc/ticdc.toml`,
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

func TestRenderTiFlashStartScript(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyTC     func(tc *v1alpha1.TidbCluster)
		expectScript string
	}

	cases := []testcase{
		{
			name:         "basic",
			modifyTC:     func(tc *v1alpha1.TidbCluster) {},
			expectScript: `/tiflash/tiflash server --config-file /data0/config.toml`,
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

func TestRenderTiFlashInitScript(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name string

		modifyTC     func(tc *v1alpha1.TidbCluster)
		expectScript string
	}

	cases := []testcase{
		{
			name:         "basic",
			modifyTC:     func(tc *v1alpha1.TidbCluster) {},
			expectScript: "set -ex;ordinal=`echo ${POD_NAME} | awk -F- '{print $NF}'`;sed s/POD_NUM/${ordinal}/g /etc/tiflash/config_templ.toml > /data0/config.toml;sed s/POD_NUM/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data0/proxy.toml",
		},
		{
			name: "across k8s",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.AcrossK8s = true
			},
			expectScript: "set -ex;ordinal=`echo ${POD_NAME} | awk -F- '{print $NF}'`;sed s/POD_NUM/${ordinal}/g /etc/tiflash/config_templ.toml > /data0/config.toml;sed s/POD_NUM/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data0/proxy.toml" + `
pd_url="http://start-script-test-pd:2379"
set +e
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="start-script-test-discovery.start-script-test-ns:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g' | sed 's/https:\/\///g'); do
echo "waiting for the verification of PD endpoints ..."
sleep 2
done


sed -i s/PD_ADDR/${result}/g /data0/config.toml
sed -i s/PD_ADDR/${result}/g /data0/proxy.toml
`,
		},
		{
			name: "across k8s with tls enabled",
			modifyTC: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
				tc.Spec.AcrossK8s = true
			},
			expectScript: "set -ex;ordinal=`echo ${POD_NAME} | awk -F- '{print $NF}'`;sed s/POD_NUM/${ordinal}/g /etc/tiflash/config_templ.toml > /data0/config.toml;sed s/POD_NUM/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data0/proxy.toml" + `
pd_url="https://start-script-test-pd:2379"
set +e
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="start-script-test-discovery.start-script-test-ns:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g' | sed 's/https:\/\///g'); do
echo "waiting for the verification of PD endpoints ..."
sleep 2
done


sed -i s/PD_ADDR/${result}/g /data0/config.toml
sed -i s/PD_ADDR/${result}/g /data0/proxy.toml
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

		script, err := RenderTiFlashInitScript(tc)
		g.Expect(err).Should(gomega.Succeed())
		if diff := cmp.Diff(c.expectScript, script); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
		g.Expect(validateScript(script)).Should(gomega.Succeed())
	}
}

func TestRenderTiDBInitStartScript(t *testing.T) {
	tests := []struct {
		name   string
		model  *TiDBInitStartScriptModel
		result string
	}{
		{
			name: "tls with skipca",
			model: &TiDBInitStartScriptModel{
				ClusterName:     "test",
				PermitHost:      "127.0.0.1",
				PasswordSet:     true,
				InitSQL:         true,
				TLS:             true,
				SkipCA:          true,
				CAPath:          "/var/lib/tidb-client-tls/ca.crt",
				CertPath:        "/var/lib/tidb-client-tls/tls.crt",
				KeyPath:         "/var/lib/tidb-client-tls/tls.key",
				TiDBServicePort: 4000,
			},
			result: `import os, sys, time, MySQLdb
host = 'test-tidb'
permit_host = '127.0.0.1'
port = 4000
retry_count = 0
for i in range(0, 10):
    try:
        conn = MySQLdb.connect(host=host, port=port, user='root', charset='utf8mb4',connect_timeout=5, ssl={'cert': '/var/lib/tidb-client-tls/tls.crt', 'key': '/var/lib/tidb-client-tls/tls.key'})
    except MySQLdb.OperationalError as e:
        print(e)
        retry_count += 1
        time.sleep(1)
        continue
    break
if retry_count == 10:
    sys.exit(1)
password_dir = '/etc/tidb/password'
for file in os.listdir(password_dir):
    if file.startswith('.'):
        continue
    user = file
    with open(os.path.join(password_dir, file), 'r') as f:
        lines = f.read().splitlines()
        password = lines[0] if len(lines) > 0 else ""
    if user == 'root':
        conn.cursor().execute("set password for 'root'@'%%' = %s;", (password,))
    else:
        conn.cursor().execute("create user %s@%s identified by %s;", (user, permit_host, password,))
with open('/data/init.sql', 'r') as sql:
    for line in sql.readlines():
        conn.cursor().execute(line)
        conn.commit()
if permit_host != '%%':
    conn.cursor().execute("update mysql.user set Host=%s where User='root';", (permit_host,))
conn.cursor().execute("flush privileges;")
conn.commit()
conn.close()
`,
		},
		{
			name: "tls",
			model: &TiDBInitStartScriptModel{
				ClusterName:     "test",
				PermitHost:      "127.0.0.1",
				PasswordSet:     true,
				InitSQL:         true,
				TLS:             true,
				SkipCA:          false,
				CAPath:          "/var/lib/tidb-client-tls/ca.crt",
				CertPath:        "/var/lib/tidb-client-tls/tls.crt",
				KeyPath:         "/var/lib/tidb-client-tls/tls.key",
				TiDBServicePort: 4000,
			},
			result: `import os, sys, time, MySQLdb
host = 'test-tidb'
permit_host = '127.0.0.1'
port = 4000
retry_count = 0
for i in range(0, 10):
    try:
        conn = MySQLdb.connect(host=host, port=port, user='root', charset='utf8mb4',connect_timeout=5, ssl={'ca': '/var/lib/tidb-client-tls/ca.crt', 'cert': '/var/lib/tidb-client-tls/tls.crt', 'key': '/var/lib/tidb-client-tls/tls.key'})
    except MySQLdb.OperationalError as e:
        print(e)
        retry_count += 1
        time.sleep(1)
        continue
    break
if retry_count == 10:
    sys.exit(1)
password_dir = '/etc/tidb/password'
for file in os.listdir(password_dir):
    if file.startswith('.'):
        continue
    user = file
    with open(os.path.join(password_dir, file), 'r') as f:
        lines = f.read().splitlines()
        password = lines[0] if len(lines) > 0 else ""
    if user == 'root':
        conn.cursor().execute("set password for 'root'@'%%' = %s;", (password,))
    else:
        conn.cursor().execute("create user %s@%s identified by %s;", (user, permit_host, password,))
with open('/data/init.sql', 'r') as sql:
    for line in sql.readlines():
        conn.cursor().execute(line)
        conn.commit()
if permit_host != '%%':
    conn.cursor().execute("update mysql.user set Host=%s where User='root';", (permit_host,))
conn.cursor().execute("flush privileges;")
conn.commit()
conn.close()
`,
		},
		{
			name: "confiure tidb service port",
			model: &TiDBInitStartScriptModel{
				ClusterName:     "test",
				PermitHost:      "127.0.0.1",
				PasswordSet:     true,
				InitSQL:         true,
				TLS:             true,
				SkipCA:          true,
				CAPath:          "/var/lib/tidb-client-tls/ca.crt",
				CertPath:        "/var/lib/tidb-client-tls/tls.crt",
				KeyPath:         "/var/lib/tidb-client-tls/tls.key",
				TiDBServicePort: 5000,
			},
			result: `import os, sys, time, MySQLdb
host = 'test-tidb'
permit_host = '127.0.0.1'
port = 5000
retry_count = 0
for i in range(0, 10):
    try:
        conn = MySQLdb.connect(host=host, port=port, user='root', charset='utf8mb4',connect_timeout=5, ssl={'cert': '/var/lib/tidb-client-tls/tls.crt', 'key': '/var/lib/tidb-client-tls/tls.key'})
    except MySQLdb.OperationalError as e:
        print(e)
        retry_count += 1
        time.sleep(1)
        continue
    break
if retry_count == 10:
    sys.exit(1)
password_dir = '/etc/tidb/password'
for file in os.listdir(password_dir):
    if file.startswith('.'):
        continue
    user = file
    with open(os.path.join(password_dir, file), 'r') as f:
        lines = f.read().splitlines()
        password = lines[0] if len(lines) > 0 else ""
    if user == 'root':
        conn.cursor().execute("set password for 'root'@'%%' = %s;", (password,))
    else:
        conn.cursor().execute("create user %s@%s identified by %s;", (user, permit_host, password,))
with open('/data/init.sql', 'r') as sql:
    for line in sql.readlines():
        conn.cursor().execute(line)
        conn.commit()
if permit_host != '%%':
    conn.cursor().execute("update mysql.user set Host=%s where User='root';", (permit_host,))
conn.cursor().execute("flush privileges;")
conn.commit()
conn.close()
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := RenderTiDBInitStartScript(tt.model)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.result, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}

func validateScript(script string) error {
	_, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	return err
}
