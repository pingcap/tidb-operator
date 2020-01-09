// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
	"bytes"
	"html/template"
)

var prometheusConfigTpl = template.Must(template.New("monitor-configmap").Parse(`global:
  scrape_interval: 15s
  evaluation_interval: 15s
{{- if .AlertmanagerURL }}
alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - {{ .AlertmanagerURL }}
{{- end }}
scrape_configs:
  - job_name: 'tidb-cluster'
    scrape_interval: 15s
    honor_labels: true
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
        {{- range .ReleaseNamespaces }}
        - {{ . }}
        {{- end}}
    tls_config:
      insecure_skip_verify: true
    {{- if .EnableTLSCluster }}
      ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      cert_file: /var/lib/pd-client-tls/cert
      key_file: /var/lib/pd-client-tls/key

    scheme: https
    {{- end }}
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
      action: keep
      regex: {{ .ReleaseTargetRegex }}
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
    {{- if .EnableTLSCluster }}
    # This is a workaround of https://github.com/tikv/tikv/issues/5340 and should
    # be removed after TiKV fix this issue
    - source_labels: [__meta_kubernetes_pod_name]
      action: drop
      regex: .*\-tikv\-\d*$
    {{- end }}
  {{- if .EnableTLSCluster }}
  # This is a workaround of https://github.com/tikv/tikv/issues/5340 and should
  # be removed after TiKV fix this issue
  - job_name: 'tidb-cluster-tikv'
    scrape_interval: 15s
    honor_labels: true
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
		{{- range .ReleaseNamespaces }}
		- {{ . }}
  		{{- end }}
    tls_config:
      insecure_skip_verify: true
    scheme: http
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
      action: keep
      regex: {{ .ReleaseTargetRegex }}
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
      action: keep
      regex: .*\-tikv\-\d*$
    - source_labels: [__meta_kubernetes_pod_name]
      action: replace
      target_label: instance
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
      action: replace
      target_label: cluster
  {{- end }}
rule_files:
  - '/prometheus-rules/rules/*.rules.yml'`))

var dashBoardConfig = `{
    "apiVersion": 1,
    "providers": [
        {
            "folder": "",
            "name": "0",
            "options": {
                "path": "/grafana-dashboard-definitions/tidb"
            },
            "orgId": 1,
            "type": "file"
        }
    ]
}`

type MonitorConfigModel struct {
	AlertmanagerURL    string
	ReleaseNamespaces  []string
	ReleaseTargetRegex string
	EnableTLSCluster   bool
}

func RenderPrometheusConfig(model *MonitorConfigModel) (string, error) {
	return renderTemplateFunc(prometheusConfigTpl, model)
}

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
