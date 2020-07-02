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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/prometheus/prometheus/config"
)

func TestRenderPrometheusConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	target, _ := config.NewRegexp("target")
	expectedContent := `global:
  scrape_interval: 15s
  evaluation_interval: 15s
alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - alert-url
rule_files:
- /prometheus-rules/rules/*.rules.yml
scrape_configs:
- job_name: pd
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pd
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pd-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tidb
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tidb-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tikv
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tikv
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tikv-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tiflash
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tiflash-proxy
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_tiflash_proxy_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: pump
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pump
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pump:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: drainer
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: drainer
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_name,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: ticdc
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: ticdc
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-ticdc-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: importer
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: importer
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-importer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: lightning
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb-lightning
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_name,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $2:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
`
	model := &MonitorConfigModel{
		ReleaseTargetRegex: &target,
		ReleaseNamespaces: []string{
			"ns1",
			"ns2",
		},
		EnableTLSCluster: false,
		AlertmanagerURL:  "alert-url",
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).Should(Equal(expectedContent))
}

func TestRenderPrometheusConfigTLSEnabled(t *testing.T) {
	g := NewGomegaWithT(t)
	target, _ := config.NewRegexp("target")
	expectedContent := `global:
  scrape_interval: 15s
  evaluation_interval: 15s
rule_files:
- /prometheus-rules/rules/*.rules.yml
scrape_configs:
- job_name: pd
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pd
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pd-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tidb
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tidb-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tikv
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tikv
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tikv-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tiflash
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: tiflash-proxy
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_tiflash_proxy_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: pump
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pump
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pump:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: drainer
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: drainer
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_name,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: ticdc
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: ticdc
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-ticdc-peer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: importer
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: importer
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-importer:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
- job_name: lightning
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
      - ns2
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb-lightning
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_name,
      __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+)
    target_label: __address__
    replacement: $2:$3
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
`
	model := &MonitorConfigModel{
		ReleaseTargetRegex: &target,
		ReleaseNamespaces: []string{
			"ns1",
			"ns2",
		},
		EnableTLSCluster: true,
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).Should(Equal(expectedContent))
}
