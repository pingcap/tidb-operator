set -euo pipefail

host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

dirname=/data/${BACKUP_NAME}
mkdir -p ${dirname}
cp /savepoint-dir/savepoint ${dirname}/
savepoint=`cat ${dirname}/savepoint | cut -d "=" -f2`

/mydumper \
  --outputdir=${dirname} \
  --host=`eval echo '${'$host'}'` \
  --port=4000 \
  --user=${TIDB_USER} \
  --password=${TIDB_PASSWORD} \
  --tidb-snapshot=${savepoint} \
  {{ .Values.backupOptions }}

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
