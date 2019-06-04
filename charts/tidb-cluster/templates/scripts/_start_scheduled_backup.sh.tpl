set -euo pipefail
dirname=scheduled-backup-`date +%Y-%m-%dT%H%M%S`-${MY_POD_NAME}
host=`echo {{ template "cluster.name" . }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

mkdir -p /data/${dirname}/
cp /savepoint-dir/savepoint /data/${dirname}/

# the content of savepoint file is:
# commitTS = 408824443621605409
savepoint=`cat /data/${dirname}/savepoint | cut -d "=" -f2 | sed 's/ *//g'`

cat /data/${dirname}/savepoint

/mydumper \
  --outputdir=/data/${dirname} \
  --host=`eval echo '${'$host'}'` \
  --port=4000 \
  --user={{ .Values.scheduledBackup.user }} \
  --password=${TIDB_PASSWORD} \
  --tidb-snapshot=${savepoint} \
  {{ .Values.scheduledBackup.options }}

{{- if .Values.scheduledBackup.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.scheduledBackup.gcp.bucket }} \
  --backup-dir=/data/${dirname}
{{- end }}

{{- if .Values.scheduledBackup.ceph }}
uploader \
  --cloud=ceph \
  --bucket={{ .Values.scheduledBackup.ceph.bucket }} \
  --endpoint={{ .Values.scheduledBackup.ceph.endpoint }} \
  --backup-dir=/data/${dirname}
{{- end }}
