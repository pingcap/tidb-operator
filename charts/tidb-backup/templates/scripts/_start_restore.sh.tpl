set -euo pipefail

dirname=/data/${BACKUP_NAME}
mkdir -p ${dirname}
host=`echo {{ .Values.clusterName }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

{{- if .Values.gcp }}
downloader \
  --cloud=gcp \
  {{- if .Values.gcp.prefix }}
  --bucket={{ .Values.gcp.bucket }}/{{ .Values.gcp.prefix }} \
  {{- else }}
  --bucket={{ .Values.gcp.bucket }} \
  {{- end }}
  --srcDir=${BACKUP_NAME} \
  --destDir=/data
{{- end }}

{{- if .Values.ceph }}
downloader \
  --cloud=ceph \
  {{- if .Values.ceph.prefix }}
  --bucket={{ .Values.ceph.bucket }}/{{ .Values.ceph.prefix }} \
  {{- else }}
  --bucket={{ .Values.ceph.bucket }} \
  {{- end }}
  --endpoint={{ .Values.ceph.endpoint }} \
  --srcDir=${BACKUP_NAME} \
  --destDir=/data
{{- end }}

{{- if .Values.s3 }}
downloader \
  --cloud=aws \
  --region={{ .Values.s3.region }} \
  {{- if .Values.s3.prefix }}
  --bucket={{ .Values.s3.bucket }}/{{ .Values.s3.prefix }} \
  {{- else }}
  --bucket={{ .Values.s3.bucket }} \
  {{- end }}
  --srcDir=${BACKUP_NAME} \
  --destDir=/data
{{- end }}

password_str=""
if [ -n "${TIDB_PASSWORD}" ];
then
    password_str="-p${TIDB_PASSWORD}"
fi

count=1
while ! mysql -u ${TIDB_USER} -h `eval echo '${'$host'}'` -P 4000 ${password_str} -e 'select version();'
do
  echo "waiting for tidb, retry ${count} times ..."
  sleep 10
  if [ ${count} -ge 180 ];then
    echo "30 minutes timeout"
    exit 1
  fi
  let "count++"
done

/loader \
  -d=${dirname} \
  -h=`eval echo '${'$host'}'` \
  -u=${TIDB_USER} \
  -p=${TIDB_PASSWORD} \
  -P=4000 \
  {{ .Values.restoreOptions }}
