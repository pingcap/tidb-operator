set -euo pipefail
{{- if and .Values.dataSource.local.hostPath .Values.dataSource.local.nodeName -}}
data_dir={{ .Values.dataSource.local.hostPath }}
{{- else if .Values.dataSource.adhoc.pvcName -}}
data_dir=/var/lib/tidb-lightning/{{ .Values.dataSource.adhoc.backupName | default .Values.dataSource.adhoc.pvcName }}
{{- else -}}
data_dir=$(dirname $(find /var/lib/tidb-lightning -name metadata 2>/dev/null) 2>/dev/null)
if [ -z $data_dir ]; then
    if [ ! -z ${FAIL_FAST} ]; then
        exit 1
    else
        echo "No mydumper files are found, please exec into my container to diagnose"
        tail -f /dev/null
    fi
fi
{{ end }}
/tidb-lightning \
    --pd-urls={{ .Values.targetTidbCluster.name }}-pd.{{ .Values.targetTidbCluster.namespace | default .Release.Namespace }}:2379 \
    --status-addr=0.0.0.0:8289 \
{{- if eq .Values.backend "importer" }}
    --importer={{ .Values.targetTidbCluster.name }}-importer.{{ .Values.targetTidbCluster.namespace | default .Release.Namespace }}:8287 \
{{- else if eq .Values.backend "tidb" }}
    --backend tidb \
{{- end }}
    --server-mode=false \
    --tidb-user={{ .Values.targetTidbCluster.user | default "root" }} \
    --tidb-host={{ .Values.targetTidbCluster.name }}-tidb.{{ .Values.targetTidbCluster.namespace | default .Release.Namespace }} \
    --d=${data_dir} \
    --config=/etc/tidb-lightning/tidb-lightning.toml
