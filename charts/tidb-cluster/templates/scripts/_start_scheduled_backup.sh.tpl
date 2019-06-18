set -euo pipefail

host=$(getent hosts {{ template "cluster.name" . }}-tidb | head | awk '{print $1}')

dirname=/data/scheduled-backup-`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
echo "making dir ${dirname}"
mkdir -p ${dirname}

gc_life_time=`/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "select variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"`
echo "Old TiKV GC life time is ${gc_life_time}"

echo "Increase TiKV GC life time to 3h"
/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "update mysql.tidb set variable_value='3h' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

/mydumper \
  --outputdir=${dirname} \
  --host=${host} \
  --port=4000 \
  --user={{ .Values.scheduledBackup.user }} \
  --password=${TIDB_PASSWORD} \
  --long-query-guard=3600 \
  --tidb-force-priority=LOW_PRIORITY \
  {{ .Values.scheduledBackup.options }}

echo "Reset TiKV GC life time to ${gc_life_time}"
/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "update mysql.tidb set variable_value='${gc_life_time}' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

{{- if .Values.scheduledBackup.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.scheduledBackup.gcp.bucket }} \
  --backup-dir=${dirname}
{{- end }}

{{- if .Values.scheduledBackup.ceph }}
uploader \
  --cloud=ceph \
  --bucket={{ .Values.scheduledBackup.ceph.bucket }} \
  --endpoint={{ .Values.scheduledBackup.ceph.endpoint }} \
  --backup-dir=${dirname}
{{- end }}
