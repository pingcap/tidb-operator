set -euo pipefail

dirname=restore-`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
dataDir=/data/${dirname}
mkdir -p ${dataDir}/
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]'`

{{- if .Values.restore.gcp }}
downloader \
  --cloud=gcp \
  --bucket={{ .Values.restore.gcp.bucket }} \
  --srcDir={{ .Values.restore.gcp.srcDir }} \
  --destDir=${dataDir}

/loader \
  -d ${dataDir}/{{ .Values.restore.gcp.srcDir }} \
  -h `eval echo '${'$host'}'` \
  -u {{ .Values.restore.user }} \
  -p ${TIDB_PASSWORD} \
  -P 4000 \
  {{ .Values.restore.options }}
{{- end }}

