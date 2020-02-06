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
	. "github.com/onsi/gomega"
	"github.com/prometheus/prometheus/config"
	"testing"
)

func TestRenderPrometheusConfig(t *testing.T) {
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
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: ([^:]+)(?::\d+)?;(\d+)
    target_label: __address__
    replacement: $1:$2
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
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: ([^:]+)(?::\d+)?;(\d+)
    target_label: __address__
    replacement: $1:$2
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
    regex: .*\-tikv\-\d*$
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: ([^:]+)(?::\d+)?;(\d+)
    target_label: __address__
    replacement: $1:$2
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
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).Should(Equal(expectedContent))
}
