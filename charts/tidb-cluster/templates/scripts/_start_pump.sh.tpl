set -euo pipefail
/pump \
-L={{ .Values.binlog.pump.logLevel | default "info" }} \
-advertise-addr=`echo ${HOSTNAME}`.{{ .Values.clusterName }}-pump:8250 \
-config=/etc/pump/pump.toml \
-log-file=
