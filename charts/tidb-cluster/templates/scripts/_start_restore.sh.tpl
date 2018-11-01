set -euo pipefail

dirname=`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
mkdir -p /data/${dirname}/
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]'`

{{- if .Values.restore.gcp }}
downloader \
  --cloud=gcp \
  --bucket={{ .Values.restore.gcp.bucket }} \
  --srcDir={{ .Values.restore.gcp.srcDir }} \
  --destDir=/data/${dirname}

dataDir=/data/${dirname}/{{ .Values.restore.gcp.srcDir }}
./loader \
  -d ${dataDir} \
  -h `eval echo '${'$host'}'` \
  -u {{ .Values.restore.user }} \
  -p ${TIDB_PASSWORD} \
  -P 4000 \
  {{ .Values.restore.options }}
{{- end }}