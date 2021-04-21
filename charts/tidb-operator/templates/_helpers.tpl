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

{{/*
Define the default value for cluster permissions variables. They should be `true` by default for back compatibility.
It seems `ternary` is not short-circuit evaluation, so we can't combine multiple `ternary` into one line here and need to use `if` instead.
*/}}
{{- define "controller-manager.cluster-permissions.nodes" -}}
{{- if hasKey .Values.controllerManager "clusterPermissions" }}
{{ hasKey .Values.controllerManager.clusterPermissions "nodes" | ternary .Values.controllerManager.clusterPermissions.nodes true }}
{{- else }}
true
{{- end }}
{{- end }}

{{- define "controller-manager.cluster-permissions.persistentvolumes" -}}
{{- if hasKey .Values.controllerManager "clusterPermissions" }}
{{ hasKey .Values.controllerManager.clusterPermissions "persistentvolumes" | ternary .Values.controllerManager.clusterPermissions.persistentvolumes true }}
{{- else }}
true
{{- end }}
{{- end }}

{{- define "controller-manager.cluster-permissions.storageclasses" -}}
{{- if hasKey .Values.controllerManager "clusterPermissions" }}
{{ hasKey .Values.controllerManager.clusterPermissions "storageclasses" | ternary .Values.controllerManager.clusterPermissions.storageclasses true }}
{{- else }}
true
{{- end }}
{{- end }}

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
