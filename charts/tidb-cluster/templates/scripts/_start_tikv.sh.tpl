#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

set -uo pipefail

{{ .Values.tikv.preStartScript }}

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
ARGS="--pd={{ template "cluster.scheme" . }}://${CLUSTER_NAME}-pd:2379 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir=/var/lib/tikv \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml
"

{{ .Values.tikv.postArgScript }}

echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
exec /tikv-server ${ARGS}
