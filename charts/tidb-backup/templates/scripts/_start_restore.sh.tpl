set -euo pipefail

dirname=/data/${BACKUP_NAME}
mkdir -p ${dirname}
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

{{- if .Values.gcp }}
downloader \
  --cloud=gcp \
  --bucket={{ .Values.gcp.bucket }} \
  --srcDir=${BACKUP_NAME} \
  --destDir=${dirname}
{{- end }}

{{- if .Values.ceph }}
downloader \
  --cloud=ceph \
  --bucket={{ .Values.ceph.bucket }} \
  --endpoint={{ .Values.ceph.endpoint }} \
  --srcDir=${BACKUP_NAME} \
  --destDir=${dirname}
{{- end }}

/loader \
  -d ${dirname} \
  -h `eval echo '${'$host'}'` \
  -u ${TIDB_USER} \
  -p ${TIDB_PASSWORD} \
  -P 4000 \
  {{ .Values.restoreOptions }}
