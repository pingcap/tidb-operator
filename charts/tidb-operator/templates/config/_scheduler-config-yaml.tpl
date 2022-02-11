{{- if semverCompare ">=1.23.0-0" .Capabilities.KubeVersion.GitVersion }}
apiVersion: kubescheduler.config.k8s.io/v1beta2
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
  resourceNamespace: {{ .Release.Namespace }}
  {{- if eq .Values.appendReleaseSuffix true}}
  resourceName: {{ .Values.scheduler.schedulerName }}-{{.Release.Name}}
  {{- else }}
  resourceName: {{ .Values.scheduler.schedulerName }}
  {{- end }}
profiles:
  - schedulerName: tidb-scheduler
extenders:
  - urlPrefix: http://127.0.0.1:10262/scheduler
    filterVerb: filter
    preemptVerb: preempt
    weight: 1
    enableHTTPS: false
    httpTimeout: 30s
{{- else }}
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
  resourceNamespace: {{ .Release.Namespace }}
  {{- if eq .Values.appendReleaseSuffix true}}
  resourceName: {{ .Values.scheduler.schedulerName }}-{{.Release.Name}}
  {{- else }}
  resourceName: {{ .Values.scheduler.schedulerName }}
  {{- end }}
healthzBindAddress: 0.0.0.0:10261
metricsBindAddress: 0.0.0.0:10261
profiles:
  - schedulerName: tidb-scheduler
extenders:
  - urlPrefix: http://127.0.0.1:10262/scheduler
    filterVerb: filter
    preemptVerb: preempt
    weight: 1
    enableHTTPS: false
    httpTimeout: 30s
{{- end }}
