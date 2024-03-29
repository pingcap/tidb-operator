{{- if and .Values.dataSource.local.hostPath .Values.dataSource.local.nodeName -}}
data_dir={{ .Values.dataSource.local.hostPath }}
{{- else if .Values.dataSource.adhoc.pvcName -}}
data_dir=/var/lib/tidb-lightning/{{ .Values.dataSource.adhoc.backupName | default .Values.dataSource.adhoc.pvcName }}
{{- else if .Values.dataSource.remote.directory -}}
data_dir=/var/lib/tidb-lightning
if [ -z "$(ls -A ${data_dir})" ]; then
    if [ ! -z ${FAIL_FAST} ]; then
        exit 1
    else
        echo "No files in data dir, please exec into my container to diagnose"
        tail -f /dev/null
    fi
fi
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

sed "s#CHECKPOINT_USE_DATA_DIR#$data_dir#g" /etc/tidb-lightning/tidb-lightning.toml > /tidb-lightning.toml

/tidb-lightning \
    --pd-urls={{ .Values.targetTidbCluster.name }}-pd.{{ .Values.targetTidbCluster.namespace | default .Release.Namespace }}:{{ .Values.targetTidbCluster.pdClientPort | default 2379 }} \
    --status-addr=0.0.0.0:8289 \
{{- if eq .Values.backend "tidb" }}
    --backend=tidb \
    --tidb-port={{ .Values.targetTidbCluster.port | default 4000 }} \
{{- else if eq .Values.backend "local" }}
    --backend=local \
    --sorted-kv-dir=/var/lib/sorted-kv \
    --tidb-status={{ .Values.targetTidbCluster.statusPort | default 10080 }} \
{{- end }}
    --server-mode=false \
{{- if and .Values.targetTidbCluster.secretName .Values.targetTidbCluster.secretUserKey -}}
    --tidb-user=${TIDB_USER} \
{{- else }}
    --tidb-user={{ .Values.targetTidbCluster.user | default "root" }} \
{{- end }}
{{- if and .Values.targetTidbCluster.secretName .Values.targetTidbCluster.secretPwdKey -}}
    --tidb-password=${TIDB_PASSWORD} \
{{- end }}
    --tidb-host={{ .Values.targetTidbCluster.name }}-tidb.{{ .Values.targetTidbCluster.namespace | default .Release.Namespace }} \
    --d=${data_dir} \
    --config=/tidb-lightning.toml \
    --log-file=""

if [ $? != 0 ]; then
    if [ ! -z ${FAIL_FAST} ]; then
        exit 1
    else
        echo $(date -u +"[%Y/%m/%d %H:%M:%S.%3N %:z]") "tidb-lightning exits abnormally, please exec into my container to do manual intervention"
        tail -f /dev/null
    fi
fi
