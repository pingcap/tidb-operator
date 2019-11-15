#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

set -uo pipefail

{{ .Values.pd.preStartScript }}

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
cluster_name=`echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'`
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc"
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=`echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"`

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
--peer-urls={{ template "cluster.scheme" . }}://0.0.0.0:2380 \
--advertise-peer-urls={{ template "cluster.scheme" . }}://${domain}:2380 \
--client-urls={{ template "cluster.scheme" . }}://0.0.0.0:2379 \
--advertise-client-urls={{ template "cluster.scheme" . }}://${domain}:2379 \
--config=/etc/pd/pd.toml \
"

if [[ -f /var/lib/pd/join ]]
then
    # The content of the join file is:
    #   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
    # The --join args must be:
    #   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
    join=`cat /var/lib/pd/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","`
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
