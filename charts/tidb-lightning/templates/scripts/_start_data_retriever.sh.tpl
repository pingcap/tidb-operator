set -euo pipefail
{{ if .Values.dataSource.remote.directory }}
# rclone sync skip identical files automatically
rclone --config /etc/rclone/rclone.conf sync -P {{ .Values.dataSource.remote.directory}} /data
{{- else -}}
filename=$(basename {{ .Values.dataSource.remote.path }})
if find /data -name metadata | egrep '.*'; then
    echo "data already exist"
    exit 0
else
    rclone --config /etc/rclone/rclone.conf copy -P {{ .Values.dataSource.remote.path }} /data
    cd /data && tar xzvf ${filename}
fi
{{- end -}}
