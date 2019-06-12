set -euo pipefail

host_env=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`
host=`eval echo '${'$host_env'}'`

dirname=/data/${BACKUP_NAME}
echo "making dir ${dirname}"
mkdir -p ${dirname}

echo "getting savepoint from pd"
chmod +x /shared-dir/binlogctl
/shared-dir/binlogctl \
  -pd-urls=http://{{ .Values.clusterName }}-pd:2379 \
  -cmd=generate_meta \
  -data-dir=${dirname}

# the content of savepoint file is:
# commitTS = 408824443621605409
savepoint=`cat ${dirname}/savepoint | cut -d "=" -f2 | sed 's/ *//g'`
cat ${dirname}/savepoint

gc_life_time=`/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "select variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"`
echo "Old TiKV GC life time is ${gc_life_time}"

echo "Increase TiKV GC life time to 3h"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "update mysql.tidb set variable_value='3h' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

/mydumper \
  --outputdir=${dirname} \
  --host=${host} \
  --port=4000 \
  --user=${TIDB_USER} \
  --password=${TIDB_PASSWORD} \
  --tidb-snapshot=${savepoint} \
  --long-query-guard=3600 \
  --tidb-force-priority=LOW_PRIORITY \
  {{ .Values.backupOptions }}

echo "Reset TiKV GC life time to ${gc_life_time}"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "update mysql.tidb set variable_value='${gc_life_time}' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

{{- if .Values.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.gcp.bucket }} \
  --backup-dir=${dirname}
{{- end }}

{{- if .Values.ceph }}
uploader \
  --cloud=ceph \
  --bucket={{ .Values.ceph.bucket }} \
  --endpoint={{ .Values.ceph.endpoint }} \
  --backup-dir=${dirname}
{{- end }}
