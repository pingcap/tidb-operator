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
    [security]
    {{- if and .Values.secretName .Values.secretSSLCA }}
  ca-path="/etc/tikv-importer-tls/{{ .Values.secretSSLCA }}"
    {{- end }}
    {{- if and .Values.secretName .Values.secretSSLCert }}
  cert-path="/etc/tikv-importer-tls/{{ .Values.secretSSLCert }}"
    {{- end }}
    {{- if and .Values.secretName .Values.secretSSLKey }}
  key-path="/etc/tikv-importer-tls/{{ .Values.secretSSLKey }}"
    {{- end }}
{{- end -}}

{{- define "importer-configmap.data-digest" -}}
{{ include "importer-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}
