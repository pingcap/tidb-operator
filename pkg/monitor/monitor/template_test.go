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
	"time"

	"github.com/docker/docker/client"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
)

func TestRenderPrometheusConfig(t *testing.T) {
	g := NewGomegaWithT(t)
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
- job_name: ns1-target-pd
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pd-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tidb
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tidb-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tikv
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tikv-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tiflash
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tiflash-proxy
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_tiflash_proxy_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-pump
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pump.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-drainer
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-ticdc
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-ticdc-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-importer
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-importer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-lightning
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $2.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
<<<<<<< HEAD
=======
- job_name: ns1-target-dm-worker
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: dm-worker
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-dm-worker-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-dm-master
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: dm-master
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-dm-master-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
remote_write:
- url: http://localhost:1234
  remote_timeout: 15s
  write_relabel_configs:
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    separator: ;
    regex: (.+)
    target_label: node
    replacement: $1
    action: replace
>>>>>>> d4f51454... TidbMonitor add remotewrite configuration (#3679)
`
	url, _ := client.ParseHostURL("http://localhost:1234")
	regex, _ := config.NewRegexp("(.+)")
	model := &MonitorConfigModel{
		ClusterInfos: []ClusterRegexInfo{
			{Name: "target", Namespace: "ns1"},
		},
		DMClusterInfos: []ClusterRegexInfo{
			{Name: "target", Namespace: "ns1"},
		},
		EnableTLSCluster:   false,
		EnableTLSDMCluster: false,
		AlertmanagerURL:    "alert-url",
		RemoteWriteConfigs: []*config.RemoteWriteConfig{
			{
				URL:           &config.URL{URL: url},
				RemoteTimeout: model.Duration(15 * time.Second),
				WriteRelabelConfigs: []*config.RelabelConfig{
					{
						SourceLabels: model.LabelNames{
							"__address__",
							portLabel,
						},
						Separator:   ";",
						Regex:       regex,
						TargetLabel: "node",
						Replacement: "$1",
						Action:      "replace",
					},
				},
			},
		},
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).Should(Equal(expectedContent))
}

func TestRenderPrometheusConfigTLSEnabled(t *testing.T) {
	g := NewGomegaWithT(t)
	expectedContent := `global:
  scrape_interval: 15s
  evaluation_interval: 15s
rule_files:
- /prometheus-rules/rules/*.rules.yml
scrape_configs:
- job_name: ns1-target-pd
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pd-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tidb
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tidb-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tikv
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tikv-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tiflash
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-tiflash-proxy
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_tiflash_proxy_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-pump
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pump.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-drainer
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-ticdc
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-ticdc-peer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-importer
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    ca_file: /var/lib/cluster-client-tls/ca.crt
    cert_file: /var/lib/cluster-client-tls/tls.crt
    key_file: /var/lib/cluster-client-tls/tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-importer.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
- job_name: ns1-target-lightning
  honor_labels: true
  scrape_interval: 15s
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
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
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $2.$3:$4
    action: replace
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    target_label: cluster
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: instance
    action: replace
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    target_label: component
    action: replace
`
	model := &MonitorConfigModel{
		ClusterInfos: []ClusterRegexInfo{
			{Name: "target", Namespace: "ns1"},
		},
		EnableTLSCluster: true,
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(content).Should(Equal(expectedContent))
}

func TestBuildAddressRelabelConfigByComponent(t *testing.T) {
	g := NewGomegaWithT(t)
	c := buildAddressRelabelConfigByComponent("non-exist-kind")
	expectedContent := `source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
regex: ([^:]+)(?::\d+)?;(\d+)
target_label: __address__
replacement: $1:$2
action: replace
`

	bs, err := yaml.Marshal(c)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(bs)).Should(Equal(expectedContent))
}

func TestMultipleClusterConfigRender(t *testing.T) {
	g := NewGomegaWithT(t)
	model := &MonitorConfigModel{
		ClusterInfos: []ClusterRegexInfo{
			{Name: "ns1", Namespace: "ns1"},
			{Name: "ns2", Namespace: "ns2"},
		},
		EnableTLSCluster: false,
		AlertmanagerURL:  "alert-url",
	}
	// firsrt validate json generate normally
	_, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	// check scrapeJob number
	pc := newPrometheusConfig(model)
	g.Expect(len(pc.ScrapeConfigs)).Should(Equal(20))
}

func TestMultipleClusterTlsConfigRender(t *testing.T) {
	g := NewGomegaWithT(t)
	model := &MonitorConfigModel{
		ClusterInfos: []ClusterRegexInfo{
			{Name: "ns1", Namespace: "ns1"},
			{Name: "ns2", Namespace: "ns2"},
		},
		EnableTLSCluster: true,
		AlertmanagerURL:  "alert-url",
	}
	// firsrt validate json generate normally
	_, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	// check scrapeJob number
	pc := newPrometheusConfig(model)
	g.Expect(pc.ScrapeConfigs[0].Scheme).Should(Equal("https"))
}
