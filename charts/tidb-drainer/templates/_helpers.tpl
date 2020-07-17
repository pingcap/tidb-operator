{{- define "drainer.name" -}}
{{- if .Values.drainerName -}}
{{ .Values.drainerName }}
{{- else -}}
{{ .Values.clusterName }}-{{ .Release.Name }}-drainer
{{- end -}}
{{- end -}}

{{- define "drainer.tlsSecretName" -}}
{{ .Values.clusterName }}-drainer-cluster-secret
{{- end -}}

{{/*
Encapsulate config data for consistent digest calculation
*/}}
{{- define "drainer-configmap.data" -}}
config-file: |-
    {{- if .Values.config }}
{{ .Values.config | indent 2 }}
    {{- end -}}
    {{- if and .Values.tlsCluster .Values.tlsCluster.enabled }}
  [security]
  ssl-ca = "/var/lib/drainer-tls/ca.crt"
  ssl-cert = "/var/lib/drainer-tls/tls.crt"
  ssl-key = "/var/lib/drainer-tls/tls.key"
  {{- if .Values.tlsCluster.certAllowedCN }}
  cert-allowed-cn = {{ .Values.tlsCluster.certAllowedCN | toJson }}
  {{- end -}}
    {{- end -}}
    {{- if and .Values.tlsSyncer .Values.tlsSyncer.tlsClientSecretName }}
  [syncer.to.security]
  ssl-ca = "/var/lib/drainer-syncer-tls/ca.crt"
  ssl-cert = "/var/lib/drainer-syncer-tls/tls.crt"
  ssl-key = "/var/lib/drainer-syncer-tls/tls.key"
  {{- if .Values.tlsSyncer.certAllowedCN }}
  cert-allowed-cn = {{ .Values.tlsSyncer.certAllowedCN | toJson }}
  {{- end -}}
    {{- end -}}
{{- end -}}

{{- define "drainer-configmap.name" -}}
{{ include "drainer.name" . }}-{{ include "drainer-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}

{{- define "cluster.scheme" -}}
{{ if and .Values.tlsCluster .Values.tlsCluster.enabled }}https{{ else }}http{{ end }}
{{- end -}}

{{- define "helm-toolkit.utils.template" -}}
{{- $name := index . 0 -}}
{{- $context := index . 1 -}}
{{- $last := base $context.Template.Name }}
{{- $wtf := $context.Template.Name | replace $last $name -}}
{{ include $wtf $context }}
{{- end -}}
