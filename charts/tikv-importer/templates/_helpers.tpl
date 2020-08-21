{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Encapsulate tikv-importer configmap data for consistent digest calculation
*/}}
{{- define "importer-configmap.data" -}}
config-file: |-
    {{- if .Values.config }}
{{ .Values.config | indent 2 }}
    {{- end -}}
{{- end -}}

{{- define "importer-configmap.data-digest" -}}
{{ include "importer-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}
