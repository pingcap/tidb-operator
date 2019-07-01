set -euo pipefail

host=$(getent hosts {{ .Values.clusterName }}-tidb | head | awk '{print $1}')

dirname=/data/${BACKUP_NAME}
echo "making dir ${dirname}"
mkdir -p ${dirname}

gc_life_time=`/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "select variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"`
echo "Old TiKV GC life time is ${gc_life_time}"

echo "Increase TiKV GC life time to 3h"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "update mysql.tidb set variable_value='3h' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} -p${TIDB_PASSWORD} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

if [ -n "{{ .Values.initialCommitTs }}" ];
then
    snapshot_args="--tidb-snapshot={{ .Values.initialCommitTs }}"
    echo "commitTS = {{ .Values.initialCommitTs }}" > ${dirname}/savepoint
    cat ${dirname}/savepoint
fi

/mydumper \
  --outputdir=${dirname} \
  --host=${host} \
  --port=4000 \
  --user=${TIDB_USER} \
  --password=${TIDB_PASSWORD} \
  --long-query-guard=3600 \
  --tidb-force-priority=LOW_PRIORITY \
  {{ .Values.backupOptions }} ${snapshot_args:-}

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
