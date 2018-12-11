set -euo pipefail
dirname=`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]'`

mkdir -p /data/${dirname}/
cp /savepoint-dir/savepoint /data/${dirname}/

/mydumper \
  --outputdir=/data/${dirname} \
  --host=`eval echo '${'$host'}'` \
  --port=4000 \
  --user={{ .Values.fullbackup.user }} \
  --password=${TIDB_PASSWORD} \
  {{ .Values.fullbackup.options }}

{{- if .Values.fullbackup.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.fullbackup.gcp.bucket }} \
  --backup-dir=/data/${dirname}
{{- end }}
