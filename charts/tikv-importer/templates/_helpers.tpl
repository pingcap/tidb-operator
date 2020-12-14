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
    {{- if and .Values.tlsCluster .Values.tlsCluster.enabled }}
  [security]
  ca-path="/etc/lib/importer-tls/ca.crt"
  cert-path="/etc/lib/importer-tls/tls.crt"
  key-path="/etc/lib/importer-tls/tls.key"
    {{- end }}
{{- end -}}

{{- define "importer-configmap.data-digest" -}}
{{ include "importer-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}
