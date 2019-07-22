{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "tidb-cluster.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "helm-toolkit.utils.template" -}}
{{- $name := index . 0 -}}
{{- $context := index . 1 -}}
{{- $last := base $context.Template.Name }}
{{- $wtf := $context.Template.Name | replace $last $name -}}
{{ include $wtf $context }}
{{- end -}}

{{- define "cluster.name" -}}
{{- default .Release.Name .Values.clusterName }}
{{- end -}}

{{/*
Encapsulate PD configmap data for consistent digest calculation
*/}}
{{- define "pd-configmap.data" -}}
startup-script: |-
{{ tuple "scripts/_start_pd.sh.tpl" . | include "helm-toolkit.utils.template" | indent 2 }}
config-file: |-
    {{- if .Values.pd.config }}
{{ .Values.pd.config | indent 2 }}
    {{- end -}}
{{- end -}}

{{- define "pd-configmap.data-digest" -}}
{{ include "pd-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}

{{/*
Encapsulate TiKV configmap data for consistent digest calculation
*/}}
{{- define "tikv-configmap.data" -}}
startup-script: |-
{{ tuple "scripts/_start_tikv.sh.tpl" . | include "helm-toolkit.utils.template" | indent 2 }}
config-file: |-
    {{- if .Values.tikv.config }}
{{ .Values.tikv.config | indent 2 }}
    {{- end -}}
{{- end -}}

{{- define "tikv-configmap.data-digest" -}}
{{ include "tikv-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}

{{/*
Encapsulate TiDB configmap data for consistent digest calculation
*/}}
{{- define "tidb-configmap.data" -}}
startup-script: |-
{{ tuple "scripts/_start_tidb.sh.tpl" . | include "helm-toolkit.utils.template" | indent 2 }}
  {{- if .Values.tidb.initSql }}
init-sql: |-
{{ .Values.tidb.initSql | indent 2 }}
  {{- end }}
config-file: |-
    {{- if .Values.tidb.config }}
{{ .Values.tidb.config | indent 2 }}
    {{- end -}}
{{- end -}}

{{- define "tidb-configmap.data-digest" -}}
{{ include "tidb-configmap.data" . | sha256sum | trunc 8 }}
{{- end -}}

