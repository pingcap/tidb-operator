set -euo pipefail

domain=`echo ${HOSTNAME}`.{{ include "drainer.name" . }}

elapseTime=0
period=1
threshold=30
while true; do
    sleep ${period}
    elapseTime=$(( elapseTime+period ))

    if [[ ${elapseTime} -ge ${threshold} ]]
    then
        echo "waiting for drainer domain ready timeout" >&2
        exit 1
    fi

    if nslookup ${domain} 2>/dev/null
    then
        echo "nslookup domain ${domain} success"
        break
    else
        echo "nslookup domain ${domain} failed" >&2
    fi
done

/drainer \
-L={{ .Values.logLevel | default "info" }} \
-pd-urls={{ include "cluster.scheme" . }}://{{ .Values.clusterName }}-pd:2379 \
-addr=`echo ${HOSTNAME}`.{{ include "drainer.name" . }}:8249 \
-config=/etc/drainer/drainer.toml \
-disable-detect={{ .Values.disableDetect | default false }} \
-initial-commit-ts={{ .Values.initialCommitTs | default -1 }} \
-data-dir=/data \
-log-file=""
