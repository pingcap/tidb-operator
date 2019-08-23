{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "tidb-lightning.name" -}}
{{ .Release.Name }}-tidb-lightning
{{- end -}}

{{/*
Encapsulate config data for consistent digest calculation
*/}}
{{- define "lightning-configmap.data" -}}
config-file: |-
    {{- if .Values.config }}
{{ .Values.config | indent 2 }}
    {{- end -}}
{{- end -}}

{{- define "lightning-configmap.data-digest" -}}
{{ include "lightning-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "tidb-lightning.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
