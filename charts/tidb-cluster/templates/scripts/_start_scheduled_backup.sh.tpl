set -euo pipefail

host=$(getent hosts {{ template "cluster.name" . }}-tidb | head | awk '{print $1}')

timestamp=$(echo ${POD_NAME}|awk -F- '{print $(NF-1)}')
## use UTC time zone to resolve timestamp, avoiding different parsing results due to different default time zones
backupName=scheduled-backup-`date -u -d @${timestamp}  "+%Y%m%d-%H%M%S"`
backupPath=/data/${backupName}

echo "making dir ${backupPath}"
mkdir -p ${backupPath}

password_str=""
if [ -n "${TIDB_PASSWORD}" ];
then
    password_str="-p${TIDB_PASSWORD}"
fi

gc_life_time=`/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "select variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"`
echo "Old TiKV GC life time is ${gc_life_time}"

echo "Increase TiKV GC life time to 3h"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "update mysql.tidb set variable_value='3h' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

/mydumper \
  --outputdir=${backupPath} \
  --host=${host} \
  --port=4000 \
  --user=${TIDB_USER} \
  --password=${TIDB_PASSWORD} \
  --long-query-guard=3600 \
  --tidb-force-priority=LOW_PRIORITY \
  --regex '^(?!(mysql\.))' \
  {{ .Values.scheduledBackup.options }}

echo "Reset TiKV GC life time to ${gc_life_time}"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "update mysql.tidb set variable_value='${gc_life_time}' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

{{- if .Values.scheduledBackup.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.scheduledBackup.gcp.bucket }} \
  --backup-dir=${backupPath}
{{- end }}

{{- if .Values.scheduledBackup.ceph }}
uploader \
  --cloud=ceph \
  --bucket={{ .Values.scheduledBackup.ceph.bucket }} \
  --endpoint={{ .Values.scheduledBackup.ceph.endpoint }} \
  --backup-dir=${backupPath}
{{- end }}

{{- if .Values.scheduledBackup.s3 }}
uploader \
  --cloud=aws \
  --region={{ .Values.scheduledBackup.s3.region }} \
  --bucket={{ .Values.scheduledBackup.s3.bucket }} \
  --backup-dir=${backupPath}
{{- end }}
