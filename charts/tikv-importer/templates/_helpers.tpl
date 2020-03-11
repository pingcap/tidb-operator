{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "cluster.name" -}}
{{- default .Release.Name .Values.clusterName }}
{{- end -}}

{{/*
Encapsulate tikv-importer configmap data for consistent digest calculation
*/}}
{{- define "importer-configmap.data" -}}
config-file: |-
    {{- if .Values.importer.config }}
{{ .Values.importer.config | indent 2 }}
    {{- end -}}
{{- end -}}

{{- define "importer-configmap.data-digest" -}}
{{ include "importer-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}
