set -euo pipefail

dirname=restore-`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
dataDir=/data/${dirname}
mkdir -p ${dataDir}/
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

{{- if .Values.restore.gcp }}
downloader \
  --cloud=gcp \
  --bucket={{ .Values.restore.gcp.bucket }} \
  --srcDir={{ .Values.restore.srcDir }} \
  --destDir=${dataDir}
{{- end }}

{{- if .Values.restore.ceph }}
downloader \
  --cloud=ceph \
  --bucket={{ .Values.restore.ceph.bucket }} \
  --endpoint={{ .Values.restore.ceph.endpoint }} \
  --srcDir={{ .Values.restore.srcDir }} \
  --destDir=${dataDir}
{{- end }}

/loader \
  -d ${dataDir}/{{ .Values.restore.srcDir }} \
  -h `eval echo '${'$host'}'` \
  -u {{ .Values.restore.user }} \
  -p ${TIDB_PASSWORD} \
  -P 4000 \
  {{ .Values.restore.options }}
