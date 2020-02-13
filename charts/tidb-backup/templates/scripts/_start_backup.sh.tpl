set -euo pipefail

{{- if .Values.host }}
host={{ .Values.host }}
{{- else }}
host=$(getent hosts {{ .Values.clusterName }}-tidb | head | awk '{print $1}')
{{- end }}

dirname=/data/${BACKUP_NAME}
echo "making dir ${dirname}"
mkdir -p ${dirname}

password_str=""
if [ -n "${TIDB_PASSWORD}" ];
then
    password_str="-p${TIDB_PASSWORD}"
fi

gc_life_time=`/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "select variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"`
echo "Old TiKV GC life time is ${gc_life_time}"

function reset_gc_lifetime() {
echo "Reset TiKV GC life time to ${gc_life_time}"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "update mysql.tidb set variable_value='${gc_life_time}' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"
}
trap "reset_gc_lifetime" EXIT

echo "Increase TiKV GC life time to {{ .Values.tikvGCLifeTime | default "720h" }}"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "update mysql.tidb set variable_value='{{ .Values.tikvGCLifeTime | default "720h" }}' where variable_name='tikv_gc_life_time';"
/usr/bin/mysql -h${host} -P4000 -u${TIDB_USER} ${password_str} -Nse "select variable_name,variable_value from mysql.tidb where variable_name='tikv_gc_life_time';"

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


backup_name="$(basename "${dirname}")"
backup_base_dir="$(dirname "${dirname}")"
{{- if .Values.gcp }}
# Once we know there are no more credentials that will be logged we can run with -x
set -x
bucket={{ .Values.gcp.bucket }}
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

cd "${backup_base_dir}"
{{- if .Values.gcp.prefix }}
tar -cf - "${backup_name}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat gcp:${bucket}/{{ .Values.gcp.prefix }}/${backup_name}/${backup_name}.tgz
{{- else }}
tar -cf - "${backup_name}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat gcp:${bucket}/${backup_name}/${backup_name}.tgz
{{- end }}
{{- end }}

{{- if .Values.ceph }}
uploader \
  --cloud=ceph \
  {{- if .Values.ceph.prefix }}
  --bucket={{ .Values.ceph.bucket }}/{{ .Values.ceph.prefix }} \
  {{- else }}
  --bucket={{ .Values.ceph.bucket }} \
  {{- end }}
  --endpoint={{ .Values.ceph.endpoint }} \
  --backup-dir=${dirname}
{{- end }}

{{- if .Values.s3 }}
# Once we know there are no more credentials that will be logged we can run with -x
set -x
bucket={{ .Values.s3.bucket }}

cat <<EOF > /tmp/rclone.conf
[s3]
type = s3
provider = AWS
env_auth = true
region = {{ .Values.s3.region }}
EOF

cd "${backup_base_dir}"
{{- if .Values.s3.prefix }}
tar -cf - "${backup_name}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat s3:${bucket}/{{ .Values.s3.prefix }}/${backup_name}/${backup_name}.tgz
{{- else }}
tar -cf - "${backup_name}" | pigz -p 16 \
  | rclone --config /tmp/rclone.conf rcat s3:${bucket}/${backup_name}/${backup_name}.tgz
{{- end }}
{{- end }}
