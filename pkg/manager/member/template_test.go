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

package member

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRenderTiKVStartScript(t *testing.T) {
	tests := []struct {
		name                string
		enableAdvertiseAddr bool
		advertiseAddr       string
		dataSubDir          string
		result              string
	}{
		{
			name:                "disable AdvertiseAddr",
			enableAdvertiseAddr: false,
			advertiseAddr:       "",
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
			name:                "enable AdvertiseAddr",
			enableAdvertiseAddr: true,
			advertiseAddr:       "test-tikv-1.test-tikv-peer.namespace.svc",
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
--advertise-status-addr=test-tikv-1.test-tikv-peer.namespace.svc:20180 \
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
			name:                "enable AdvertiseAddr and non-empty dataSubDir",
			enableAdvertiseAddr: true,
			advertiseAddr:       "test-tikv-1.test-tikv-peer.namespace.svc",
			dataSubDir:          "data",
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
--advertise-status-addr=test-tikv-1.test-tikv-peer.namespace.svc:20180 \
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := TiKVStartScriptModel{
				PdAddress:                 "http://${CLUSTER_NAME}-pd:2379",
				EnableAdvertiseStatusAddr: tt.enableAdvertiseAddr,
				AdvertiseStatusAddr:       tt.advertiseAddr,
				DataDir:                   filepath.Join(tikvDataVolumeMountPath, tt.dataSubDir),
			}
			script, err := RenderTiKVStartScript(&model)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.result, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}

func TestRenderPDStartScript(t *testing.T) {
	tests := []struct {
		name       string
		scheme     string
		dataSubDir string
		result     string
	}{
		{
			name:   "https scheme",
			scheme: "https",
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
--peer-urls=://0.0.0.0:2380 \
--advertise-peer-urls=://${domain}:2380 \
--client-urls=://0.0.0.0:2379 \
--advertise-client-urls=://${domain}:2379 \
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
			name:       "non-empty dataSubDir",
			scheme:     "http",
			dataSubDir: "data",
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
--peer-urls=://0.0.0.0:2380 \
--advertise-peer-urls=://${domain}:2380 \
--client-urls=://0.0.0.0:2379 \
--advertise-client-urls=://${domain}:2379 \
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := PDStartScriptModel{
				DataDir: filepath.Join(pdDataVolumeMountPath, tt.dataSubDir),
			}
			script, err := RenderPDStartScript(&model)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.result, script); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}
