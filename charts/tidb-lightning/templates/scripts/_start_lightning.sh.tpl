{{- if and .Values.dataSource.local.hostPath .Values.dataSource.local.nodeName -}}
data_dir={{ .Values.dataSource.local.hostPath }}
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
    --pd-urls={{ .Values.targetTidbCluster.name }}-pd.{{ .Release.Namespace }}:2379 \
    --status-addr=0.0.0.0:8289 \
    --importer={{ .Values.targetTidbCluster.name }}-importer.{{ .Release.Namespace }}:8287 \
    --server-mode=false \
    --tidb-user={{ .Values.targetTidbCluster.user | default "root" }} \
    --tidb-host={{ .Values.targetTidbCluster.name }}-tidb.{{ .Release.Namespace }} \
    --d=${data_dir} \
    --config=/etc/tidb-lightning/tidb-lightning.toml

if [ $? != 0 ]; then
    if [ ! -z ${FAIL_FAST} ]; then
        exit 1
    else
        echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "tidb-lightning exits abnormally, please exec into my container to do manual intervention"
        tail -f /dev/null
    fi
fi
