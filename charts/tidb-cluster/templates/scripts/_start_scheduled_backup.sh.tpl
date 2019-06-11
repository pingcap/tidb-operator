set -euo pipefail

host_env=`echo {{ template "cluster.name" . }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`
host=`eval echo '${'$host_env'}'`

dirname=/data/scheduled-backup-`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
echo "making dir ${dirname}"
mkdir -p ${dirname}

echo "getting savepoint from pd"
chmod +x /shared-dir/binlogctl
/shared-dir/binlogctl \
  -pd-urls=http://{{ template "cluster.name" . }}-pd:2379 \
  -cmd=generate_meta \
  -data-dir=${dirname}

# the content of savepoint file is:
# commitTS = 408824443621605409
savepoint=`cat ${dirname}/savepoint | cut -d "=" -f2 | sed 's/ *//g'`
cat ${dirname}/savepoint

echo "Increase TiKV GC life time to 3h"
/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "update mysql.tidb set variable_value='3h' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

/mydumper \
  --outputdir=${dirname} \
  --host=${host} \
  --port=4000 \
  --user={{ .Values.scheduledBackup.user }} \
  --password=${TIDB_PASSWORD} \
  --tidb-snapshot=${savepoint} \
  --long-query-guard=3600 \
  --tidb-force-priority=LOW_PRIORITY \
  {{ .Values.scheduledBackup.options }}

echo "Reset TiKV GC life time to 10m"
/usr/bin/mysql -h${host} -P4000 -u{{ .Values.scheduledBackup.user }} -p${TIDB_PASSWORD} -Nse "update mysql.tidb set variable_value='10m' where variable_name='tikv_gc_life_time';"
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
