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
- job_name: 'pd'
  scrape_interval: 15s
  honor_labels: true
  kubernetes_sd_configs:
  - role: pod
  {{- if not .Values.rbac.crossNamespace }}
    namespaces:
      names:
      - {{ .Release.Namespace }}
  {{- end }}
  {{- if and .Values.tlsCluster .Values.tlsCluster.enabled }}
  scheme: https
  tls_config:
    insecure_skip_verify: false
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
  {{- else }}
  scheme: http
  tls_config:
    insecure_skip_verify: true
  {{- end }}
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    action: keep
    regex: {{ .Release.Name }}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    action: keep
    regex: pd
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pd-peer:$3
    action: replace
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
- job_name: 'tidb'
  scrape_interval: 15s
  honor_labels: true
  kubernetes_sd_configs:
  - role: pod
  {{- if not .Values.rbac.crossNamespace }}
    namespaces:
      names:
      - {{ .Release.Namespace }}
  {{- end }}
  {{- if and .Values.tlsCluster .Values.tlsCluster.enabled }}
  scheme: https
  tls_config:
    insecure_skip_verify: false
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
  {{- else }}
  scheme: http
  tls_config:
    insecure_skip_verify: true
  {{- end }}
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    action: keep
    regex: {{ .Release.Name }}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    action: keep
    regex: tidb
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tidb-peer:$3
    action: replace
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
- job_name: 'tikv'
  scrape_interval: 15s
  honor_labels: true
  kubernetes_sd_configs:
  - role: pod
  {{- if not .Values.rbac.crossNamespace }}
    namespaces:
      names:
      - {{ .Release.Namespace }}
  {{- end }}
  scheme: http
  tls_config:
    insecure_skip_verify: true
  # TiKV doesn't support scheme https for now.
  # And we should fix it after TiKV fix this issue: https://github.com/tikv/tikv/issues/5340
  # {{- if and .Values.tlsCluster .Values.tlsCluster.enabled }}
  # scheme: https
  # tls_config:
  #   insecure_skip_verify: false
  #   ca_file: /var/lib/cluster-client-tls/ca.crt
  #   cert_file: /var/lib/cluster-client-tls/tls.crt
  #   key_file: /var/lib/cluster-client-tls/tls.key
  # {{- else }}
  # scheme: http
  # tls_config:
  #   insecure_skip_verify: true
  # {{- end }}
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    action: keep
    regex: {{ .Release.Name }}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    action: keep
    regex: tikv
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tikv-peer:$3
    action: replace
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
- job_name: 'tiproxy'
  scrape_interval: 15s
  honor_labels: true
  kubernetes_sd_configs:
  - role: pod
  {{- if not .Values.rbac.crossNamespace }}
    namespaces:
      names:
      - {{ .Release.Namespace }}
  {{- end }}
  {{- if and .Values.tlsCluster .Values.tlsCluster.enabled }}
  scheme: https
  tls_config:
    insecure_skip_verify: false
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
  {{- else }}
  scheme: http
  tls_config:
    insecure_skip_verify: true
  {{- end }}
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    action: keep
    regex: {{ .Release.Name }}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    action: keep
    regex: tiproxy
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiproxy-peer:$3
    action: replace
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
