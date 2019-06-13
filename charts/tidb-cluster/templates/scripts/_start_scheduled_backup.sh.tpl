#!/bin/sh

set -euo pipefail

timestamp=$(echo ${POD_NAME}|awk -F- '{print $(NF-1)}')
## use UTC time zone to resolve timestamp, avoiding different parsing results due to different default time zones
backupName=scheduled-backup-`date -u -d @${timestamp}  "+%Y%m%d-%H%M%S"`
backupPath=/data/${backupName}
host=`echo {{ template "cluster.name" . }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

mkdir -p ${backupPath}
cp /savepoint-dir/savepoint ${backupPath}

# the content of savepoint file is:
# commitTS = 408824443621605409
savepoint=`cat ${backupPath}/savepoint | cut -d "=" -f2 | sed 's/ *//g'`

cat ${backupPath}/savepoint

/mydumper \
  --outputdir=${backupPath} \
  --host=`eval echo '${'$host'}'` \
  --port=4000 \
  --user=${TIDB_USER} \
  --password=${TIDB_PASSWORD} \
  --tidb-snapshot=${savepoint} \
  --regex '^(?!(mysql\.))' \
  {{ .Values.scheduledBackup.options }}

{{- if .Values.scheduledBackup.gcp }}
uploader \
  --cloud=gcp \
  --bucket={{ .Values.scheduledBackup.gcp.bucket }} \
  --backup-dir=${backupPath}
{{- end }}

{{- if .Values.scheduledBackup.ceph }}
uploader \
  --cloud=ceph \
  --bucket={{ .Values.scheduledBackup.ceph.bucket }} \
  --endpoint={{ .Values.scheduledBackup.ceph.endpoint }} \
  --backup-dir=${backupPath}
{{- end }}
