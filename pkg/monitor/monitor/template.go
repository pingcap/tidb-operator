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
	"github.com/ghodss/yaml"
	"html/template"
)

type PrometheusConfig struct {
	Global        GlobalConfig       `json:"global,omitempty"`
	Alerting      []AlertingSpec     `json:"alerting,omitempty"`
	RuleFiles     []string           `json:"rule_files,omitempty"`
	ScrapeConfigs []ScrapeConfigSpec `json:"scrape_configs,omitempty"`
}

type GlobalConfig struct {
	ScrapeInterval     string `json:"scrape_interval,omitempty"`
	EvaluationInterval string `json:"evaluation_interval,omitempty"`
}

type AlertingSpec struct {
	AlertManagers []AlertmanagerSpec `json:"alertmanagers,omitempty"`
}

type AlertmanagerSpec struct {
	StaticConfigs []StaticConfig `json:"static_configs,omitempty"`
}

type StaticConfig struct {
	Targets []string `json:"targets,omitempty"`
}

type ScrapeConfigSpec struct {
	JobName             string               `json:"job_name"`
	ScrapeInterval      string               `json:"scrape_interval,omitempty"`
	HonorLabels         bool                 `json:"honor_labels,omitempty"`
	KubernetesSDConfigs []KubernetesSDConfig `json:"kubernetes_sd_configs,omitempty"`
	TlsConfig           TlsConfig            `json:"tls_config,omitempty"`
	Schema              string               `json:"schema,omitempty"`
	RelabelConfigs      []RelabelConfig      `json:"relabel_configs,omitempty"`
}

type KubernetesSDConfig struct {
	Role       string           `json:"role,omitempty"`
	Namespaces NamespacesConfig `json:"namespaces,omitempty"`
}

type NamespacesConfig struct {
	Names []string `json:"names,omitempty"`
}

type RelabelConfig struct {
	SourceLabels string `json:"source_labels"`
	Action       string `json:"action"`
	Regex        string `json:"regex,omitempty"`
	TargetLabel  string `json:"target_label,omitempty"`
	Replacement  string `json:"replacement,omitempty"`
}

type TlsConfig struct {
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	CaFile             string `json:"ca_file,omitempty"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
}

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
    {{- if .EnableTLSCluster }}
    # This is a workaround of https://github.com/tikv/tikv/issues/5340 and should
    # be removed after TiKV fix this issue
    - source_labels: [__meta_kubernetes_pod_name]
      action: drop
      regex: .*\-tikv\-\d*$
    {{- end }}
    - source_labels: [__meta_kubernetes_pod_name]
      action: replace
      target_label: instance
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
      action: replace
      target_label: cluster
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

func newPrometheusConfig(model *MonitorConfigModel) *PrometheusConfig {
	c := PrometheusConfig{
		Global: GlobalConfig{
			ScrapeInterval:     "15s",
			EvaluationInterval: "15s",
		},
		Alerting: []AlertingSpec{
			{
				AlertManagers: []AlertmanagerSpec{
					{
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{
									model.AlertmanagerURL,
								},
							},
						},
					},
				},
			},
		},
		RuleFiles: []string{
			"/prometheus-rules/rules/*.rules.yml",
		},
		ScrapeConfigs: []ScrapeConfigSpec{
			{
				JobName:        "tidb-cluster",
				ScrapeInterval: "15s",
				HonorLabels:    true,
				KubernetesSDConfigs: []KubernetesSDConfig{
					{
						Role: "pod",
					},
				},
				TlsConfig: TlsConfig{
					InsecureSkipVerify: true,
				},
				RelabelConfigs: []RelabelConfig{
					{
						SourceLabels: "[__meta_kubernetes_pod_label_app_kubernetes_io_instance]",
						Action:       "keep",
						Regex:        model.ReleaseTargetRegex,
					},
					{
						SourceLabels: "[__meta_kubernetes_pod_annotation_prometheus_io_scrape]",
						Action:       "keep",
						Regex:        "true",
					},
					{
						SourceLabels: "[__meta_kubernetes_pod_annotation_prometheus_io_path]",
						Action:       "replace",
						TargetLabel:  "__metrics_path__",
						Regex:        "(.+)",
					},
					{
						SourceLabels: "[__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]",
						Action:       "replace",
						Regex:        "([^:]+)(?::\\d+)?;(\\d+)",
						Replacement:  "$1:$2",
						TargetLabel:  "__address__",
					},
					{
						SourceLabels: "[__meta_kubernetes_namespace]",
						Action:       "replace",
						TargetLabel:  "kubernetes_pod_ip",
					},
					{
						SourceLabels: "[__meta_kubernetes_pod_name]",
						Action:       "replace",
						TargetLabel:  "instance",
					},
					{
						SourceLabels: "[__meta_kubernetes_pod_label_app_kubernetes_io_instance]",
						Action:       "replace",
						TargetLabel:  "cluster",
					},
				},
			},
		},
	}
	return &c
}

func addTlsConfig(pc *PrometheusConfig, model *MonitorConfigModel) {
	for id, config := range pc.ScrapeConfigs {
		if config.JobName == "tidb-cluster" {
			config.TlsConfig.CaFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
			config.TlsConfig.CertFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
			config.TlsConfig.KeyFile = "/var/lib/pd-client-tls/key"
			config.Schema = "https"
			// This is a workaround of https://github.com/tikv/tikv/issues/5340 and should
			// be removed after TiKV fix this issue
			config.RelabelConfigs = append(config.RelabelConfigs, RelabelConfig{
				SourceLabels: "[__meta_kubernetes_pod_name]",
				Action:       "drop",
				Regex:        ".*\\-tikv\\-\\d*$",
			})
		}
		pc.ScrapeConfigs[id] = config
	}
	// This is a workaround of https://github.com/tikv/tikv/issues/5340 and should
	// be removed after TiKV fix this issue
	pc.ScrapeConfigs = append(pc.ScrapeConfigs, ScrapeConfigSpec{
		JobName:        "tidb-cluster-tikv",
		ScrapeInterval: "15s",
		HonorLabels:    true,
		KubernetesSDConfigs: []KubernetesSDConfig{
			{
				Role: "pod",
				Namespaces: NamespacesConfig{
					Names: model.ReleaseNamespaces,
				},
			},
		},
		TlsConfig: TlsConfig{
			InsecureSkipVerify: true,
		},
		Schema: "http",
		RelabelConfigs: []RelabelConfig{
			{
				SourceLabels: "[__meta_kubernetes_pod_label_app_kubernetes_io_instance]",
				Action:       "keep",
				Regex:        model.ReleaseTargetRegex,
			},
			{
				SourceLabels: "[__meta_kubernetes_pod_annotation_prometheus_io_scrape]",
				Action:       "keep",
				Regex:        "true",
			},
			{
				SourceLabels: "[__meta_kubernetes_pod_annotation_prometheus_io_path]",
				Action:       "replace",
				TargetLabel:  "__metrics_path__",
				Regex:        "(.+)",
			},
			{
				SourceLabels: "[__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]",
				Action:       "replace",
				Regex:        "([^:]+)(?::\\d+)?;(\\d+)",
				Replacement:  "$1:$2",
				TargetLabel:  "__address__",
			},
			{
				SourceLabels: "[__meta_kubernetes_namespace]",
				Action:       "replace",
				TargetLabel:  "kubernetes_namespace",
			},
			{
				SourceLabels: "[__meta_kubernetes_pod_node_name]",
				Action:       "replace",
				TargetLabel:  "kubernetes_node",
			},
			{
				SourceLabels: "[__meta_kubernetes_pod_ip]",
				Action:       "replace",
				TargetLabel:  "kubernetes_pod_ip",
			},
		},
	})
}

func RenderPrometheusConfig(model *MonitorConfigModel) (string, error) {
	pc := newPrometheusConfig(model)
	if model.EnableTLSCluster {
		addTlsConfig(pc, model)
	}
	bytes, err := yaml.Marshal(pc)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
