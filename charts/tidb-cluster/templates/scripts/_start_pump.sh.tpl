set -euo pipefail
/pump \
-pd-urls={{ template "cluster.name" . }}-pd:2379 \
-L={{ .Values.binlog.pump.logLevel | default "info" }} \
-advertise-addr=`echo ${HOSTNAME}`.{{ template "cluster.name" . }}-pump:8250 \
-config=/etc/pump/pump.toml \
-log-file=
