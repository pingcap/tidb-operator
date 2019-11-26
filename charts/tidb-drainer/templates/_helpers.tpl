{{- define "drainer.name" -}}
{{ .Values.clusterName }}-{{ .Release.Name }}-drainer
{{- end -}}

{{/*
Encapsulate config data for consistent digest calculation
*/}}
{{- define "drainer-configmap.data" -}}
config-file: |-
    {{- if .Values.config }}
{{ .Values.config | indent 2 }}
    {{- end -}}
{{- end -}}

{{- define "drainer-configmap.name" -}}
{{ include "drainer.name" . }}-{{ include "drainer-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}

{{- define "helm-toolkit.utils.template" -}}
{{- $name := index . 0 -}}
{{- $context := index . 1 -}}
{{- $last := base $context.Template.Name }}
{{- $wtf := $context.Template.Name | replace $last $name -}}
{{ include $wtf $context }}
{{- end -}}
