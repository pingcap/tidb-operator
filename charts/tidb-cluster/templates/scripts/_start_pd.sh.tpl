#!/bin/sh

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

cluster_name=`echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'`
domain="${HOSTNAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=`echo ${domain}:2380 | base64`

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
--name=${HOSTNAME} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${domain}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ ! -d /var/lib/pd/member/wal ]]
then
    until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
        echo "waiting for discovery service returns start args ..."
        sleep 2
    done
    ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
echo "/pd-server ${ARGS}"
exec /pd-server ${ARGS}
