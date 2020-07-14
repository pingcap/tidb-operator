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
{{- define "tidb-operator.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "helm-toolkit.utils.template" -}}
{{- $name := index . 0 -}}
{{- $context := index . 1 -}}
{{- $last := base $context.Template.Name }}
{{- $wtf := $context.Template.Name | replace $last $name -}}
{{ include $wtf $context }}
{{- end }}

{{/*
By default, we extract the v<major>.<minor>.<patch> part from the KubeVersion, e.g.
- v1.17.1+3f6f40d -> v1.17.1 (OpenShift)
- v1.15.11-gke.15 -> v1.15.11 (GKE)
*/}}
{{- define "kube-scheduler.image_tag" -}}
{{- default (regexFind "^v\\d+\\.\\d+\\.\\d+" .Capabilities.KubeVersion.GitVersion) .Values.scheduler.kubeSchedulerImageTag -}}
{{- end -}}
