#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

set -uo pipefail
ANNOTATIONS="/etc/podinfo/annotations"
# get statefulset ordinal index for current pod
ORDINAL=$(echo ${HOSTNAME} | awk -F '-' '{print $NF}')

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null

PEER_SERVICE_DOMAIN="${HOSTNAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc"
SERVICE_DOMAIN="${SERVICE_NAME}.${NAMESPACE}.svc"

runmode=${runmode:-normal}
if [[ X${runmode} == Xdebug ]]
then
    echo "entering debug mode."
    tail -f /dev/null
fi

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

    source ${ANNOTATIONS} 2>/dev/null
    if nslookup ${PEER_SERVICE_DOMAIN} 2>/dev/null
    then
        echo "nslookup domain ${HOSTNAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc success"

        if [[ ${ORDINAL} -eq 0 ]]
        then
            [[ -z ${bootstrapping:-} ]] && continue
            [[ ${bootstrapping} == "true" ]] && break
        fi

        [[ -d /var/lib/pd/member/wal ]] && break
        wget -qO- ${SERVICE_DOMAIN}:2379/pd/api/v1/members 2>/dev/null
        [[ $? -eq 0 ]] && break
        echo "pd cluster is not ready now: ${SERVICE_DOMAIN}"
    else
        echo "nslookup domain ${PEER_SERVICE_DOMAIN} failed" >&2
    fi
done

ARGS="--data-dir=/var/lib/pd \
--name=${HOSTNAME} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${HOSTNAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${HOSTNAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc:2379 \
--config=/etc/pd/pd.toml \
"

replicas=${replicas:-3}
if [[ ${ORDINAL} -eq 0 && ${bootstrapping:-} == "true" ]]
then
    ARGS="${ARGS}--initial-cluster=${HOSTNAME}=http://${HOSTNAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc:2380"
else
    if [[ ${ORDINAL} -eq 0 ]]
    then
      TOP=$((replicas-1))
    else
      TOP=$((ORDINAL-1))
    fi

    ARGS="${ARGS}--join="
    for i in $(seq 0 ${TOP});
    do
        [[ ${i} -eq ${ORDINAL} ]] && continue
        ARGS="${ARGS}http://${SET_NAME}-${i}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc:2380"
        if [[ ${i} -lt ${TOP} ]]
        then
            ARGS="${ARGS},"
        fi
    done
fi

echo "starting pd-server ..."
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
