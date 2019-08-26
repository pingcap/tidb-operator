set -euo pipefail

SCHEME={{ if .Values.enableTLSCluster }}"https"{{ else }}"http"{{ end }}

/pump \
-pd-urls=$SCHEME://{{ template "cluster.name" . }}-pd:2379 \
-L={{ .Values.binlog.pump.logLevel | default "info" }} \
-advertise-addr=`echo ${HOSTNAME}`.{{ template "cluster.name" . }}-pump:8250 \
-config=/etc/pump/pump.toml \
-data-dir=/data \
-log-file=

if [ $? == 0 ]; then
    echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "pump offline, please delete my pod"
    tail -f /dev/null
fi
