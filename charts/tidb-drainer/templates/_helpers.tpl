{{- define "drainer.name" -}}
{{- if .Values.drainerName }}
{{ .Values.drainerName }}
{{- else -}}
{{ .Values.clusterName }}-{{ .Release.Name }}-drainer
{{- end -}}
{{- end -}}

{{- define "drainer.tlsSecretName" -}}
{{ .Values.clusterName }}-drainer
{{- end -}}

{{/*
Encapsulate config data for consistent digest calculation
*/}}
{{- define "drainer-configmap.data" -}}
config-file: |-
    {{- if .Values.config }}
{{ .Values.config | indent 2 }}
    {{- end -}}
    {{- if .Values.enableTLSCluster }}
  [security]
  ssl-ca = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
  ssl-cert = "/var/lib/drainer-tls/tls.crt"
  ssl-key = "/var/lib/drainer-tls/tls.key"
    {{- end -}}
{{- end -}}

{{- define "drainer-configmap.name" -}}
{{ include "drainer.name" . }}-{{ include "drainer-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}

{{- define "cluster.scheme" -}}
{{ if .Values.enableTLSCluster }}https{{ else }}http{{ end }}
{{- end -}}

{{- define "helm-toolkit.utils.template" -}}
{{- $name := index . 0 -}}
{{- $context := index . 1 -}}
{{- $last := base $context.Template.Name }}
{{- $wtf := $context.Template.Name | replace $last $name -}}
{{ include $wtf $context }}
{{- end -}}
