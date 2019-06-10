#!/bin/sh

set -euo pipefail

timestamp=$(echo ${POD_NAME}|awk -F- '{print $(NF-1)}')
## use UTC time zone to resolve timestamp, avoiding different parsing results due to different default time zones
backupName=${POD_NAMESPACE}_scheduled-backup_`date -u -d @${timestamp}  "+%Y-%m-%dT%H:%M"`
backupPath=/data/${backupName}
host=`echo {{ template "cluster.name" . }}_TIDB_SERVICE_HOST | tr '[a-z]' '[A-Z]' | tr '-' '_'`

mkdir -p ${backupPath}
cp /savepoint-dir/savepoint ${backupPath}

# the content of savepoint file is:
# commitTS = 408824443621605409
savepoint=`cat /data/${dirname}/savepoint | cut -d "=" -f2 | sed 's/ *//g'`

cat /data/${dirname}/savepoint

/mydumper \
  --outputdir=${backupPath} \
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
  --backup-dir=${backupPath}
{{- end }}

{{- if .Values.scheduledBackup.ceph }}
uploader \
  --cloud=ceph \
  --bucket={{ .Values.scheduledBackup.ceph.bucket }} \
  --endpoint={{ .Values.scheduledBackup.ceph.endpoint }} \
  --backup-dir=${backupPath}
{{- end }}
