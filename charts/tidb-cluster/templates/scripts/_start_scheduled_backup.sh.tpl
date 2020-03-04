set -euo pipefail

host=$(getent hosts {{ template "cluster.name" . }}-tidb | head | awk '{print $1}')

backupName=scheduled-backup-`date "+%Y%m%d-%H%M%S"`
backupBase=/data
backupPath=${backupBase}/${backupName}

echo "making dir ${backupPath}"
mkdir -p ${backupPath}

password_str=""
if [ -n "${TIDB_PASSWORD}" ];
then
    password_str="-p${TIDB_PASSWORD}"
fi

gc_life_time=`/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "select variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"`
echo "Old TiKV GC life time is ${gc_life_time}"

echo "Increase TiKV GC life time to {{ .Values.scheduledBackup.tikvGCLifeTime | default "720h" }}"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "update mysql.tidb set variable_value='{{ .Values.scheduledBackup.tikvGCLifeTime | default "720h" }}' where variable_name='tikv_gc_life_time';"
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
# Once we know there are no more credentials that will be logged we can run with -x
set -x
bucket={{ .Values.scheduledBackup.gcp.bucket }}
creds=${GOOGLE_APPLICATION_CREDENTIALS:-""}
if ! [[ -z $creds ]] ; then
creds="service_account_file = ${creds}"
fi

cat <<EOF > /tmp/rclone.conf
[gcp]
type = google cloud storage
bucket_policy_only = true
$creds
EOF

cd "${backupBase}"
{{- if .Values.scheduledBackup.gcp.prefix }}
tar -cf - "${backupName}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat gcp:${bucket}/{{ .Values.scheduledBackup.gcp.prefix }}/${backupName}/${backupName}.tgz
{{- else }}
tar -cf - "${backupName}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat gcp:${bucket}/${backupName}/${backupName}.tgz
{{- end }}
{{- end }}

{{- if .Values.scheduledBackup.ceph }}
uploader \
  --cloud=ceph \
  --bucket={{ .Values.scheduledBackup.ceph.bucket }} \
  --endpoint={{ .Values.scheduledBackup.ceph.endpoint }} \
  --backup-dir=${backupPath}
{{- end }}

{{- if .Values.scheduledBackup.s3 }}
# Once we know there are no more credentials that will be logged we can run with -x
set -x
bucket={{ .Values.scheduledBackup.s3.bucket }}

cat <<EOF > /tmp/rclone.conf
[s3]
type = s3
provider = AWS
env_auth = true
region = {{ .Values.scheduledBackup.s3.region }}
EOF

cd "${backupBase}"
{{- if .Values.scheduledBackup.s3.prefix }}
tar -cf - "${backupName}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat s3:${bucket}/{{ .Values.scheduledBackup.s3.prefix }}/${backupName}/${backupName}.tgz
{{- else }}
tar -cf - "${backupName}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat s3:${bucket}/${backupName}/${backupName}.tgz
{{- end }}
{{- end }}

{{- if and (.Values.scheduledBackup.cleanupAfterUpload) (or (.Values.scheduledBackup.gcp) (or .Values.scheduledBackup.ceph .Values.scheduledBackup.s3)) }}
if [ $? -eq 0 ]; then
    if [ -d "${backupPath}" -a -n "${backupName}" ]; then
        echo "Removing the directory ${backupPath}"
        rm -rf "${backupPath}"
    fi
fi
{{- end }}
