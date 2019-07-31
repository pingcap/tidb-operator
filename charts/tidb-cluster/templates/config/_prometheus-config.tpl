global:
  scrape_interval: 15s
  evaluation_interval: 15s
{{- if .Values.monitor.prometheus.alertmanagerURL }}
alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - {{ .Values.monitor.prometheus.alertmanagerURL }}
{{- end }}
scrape_configs:
  - job_name: 'tidb-cluster'
    scrape_interval: 15s
    honor_labels: true
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
        - {{ .Release.Namespace }}
    tls_config:
      insecure_skip_verify: true
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
      action: keep
      regex: {{ .Release.Name }}
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
      action: keep
      regex: true
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
      action: replace
      target_label: __metrics_path__
      regex: (.+)
    - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
      action: replace
      regex: ([^:]+)(?::\d+)?;(\d+)
      replacement: $1:$2
      target_label: __address__
    - source_labels: [__meta_kubernetes_namespace]
      action: replace
      target_label: kubernetes_namespace
    - source_labels: [__meta_kubernetes_pod_node_name]
      action: replace
      target_label: kubernetes_node
    - source_labels: [__meta_kubernetes_pod_ip]
      action: replace
      target_label: kubernetes_pod_ip
    - source_labels: [__meta_kubernetes_pod_name]
      action: replace
      target_label: instance
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
      action: replace
      target_label: cluster
rule_files:
  - '/prometheus-rules/rules/*.rules.yml'
