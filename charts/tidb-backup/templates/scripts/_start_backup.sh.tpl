set -euo pipefail

host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

dirname=/data/${BACKUP_NAME}
mkdir -p ${dirname}

/mydumper \
  --outputdir=${dirname} \
  --host=`eval echo '${'$host'}'` \
  --port=4000 \
  --user=${TIDB_USER} \
  --password=${TIDB_PASSWORD} \
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
