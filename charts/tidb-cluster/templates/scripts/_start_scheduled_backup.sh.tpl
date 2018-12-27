set -euo pipefail
dirname=scheduled-backup-`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

mkdir -p /data/${dirname}/
cp /savepoint-dir/savepoint /data/${dirname}/

/mydumper \
  --outputdir=/data/${dirname} \
  --host=`eval echo '${'$host'}'` \
  --port=4000 \
  --user={{ .Values.scheduledBackup.user }} \
  --password=${TIDB_PASSWORD} \
  {{ .Values.scheduledBackup.options }}

{{- if .Values.scheduledBackup.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.scheduledBackup.gcp.bucket }} \
  --backup-dir=/data/${dirname}
{{- end }}
