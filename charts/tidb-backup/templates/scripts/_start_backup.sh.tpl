set -euo pipefail
dirname=backup-`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

mkdir -p /data/${dirname}/
cp /savepoint-dir/savepoint /data/${dirname}/

/mydumper \
  --outputdir=/data/${dirname} \
  --host=`eval echo '${'$host'}'` \
  --port=4000 \
  --user={{ .Values.backup.user }} \
  --password=${TIDB_PASSWORD} \
  {{ .Values.backup.options }}

{{- if .Values.backup.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.backup.gcp.bucket }} \
  --backup-dir=/data/${dirname}
{{- end }}
