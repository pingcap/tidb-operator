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
	"path"
	"testing"
	"text/template"
	"time"

	"github.com/docker/docker/client"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type promConfigsModel struct {
	KubeSDConfigs string
	KeepLabels    string
	ReplaceLabels string
}

var promCfgModel = promConfigsModel{
	KubeSDConfigs: `kubernetes_sd_configs:
  - api_server: null
    role: pod
    namespaces:
      names:
      - ns1`,
	KeepLabels: `- source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep`,
	ReplaceLabels: `- source_labels: [__meta_kubernetes_namespace]
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace`,
}

func TestRenderPrometheusConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	expectedContentTpl := `global:
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
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pd
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pd-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-tidb
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tidb-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-tikv
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tikv
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tikv-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-tiflash
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-tiflash-proxy
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_tiflash_proxy_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-tiflash-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-pump
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pump
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-pump.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-drainer
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: drainer
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_name,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-ticdc
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: ticdc
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-ticdc-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-importer
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: importer
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-importer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-lightning
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb-lightning
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_name,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $2.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-dm-worker
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: dm-worker
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-dm-worker-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
- job_name: ns1-target-dm-master
  honor_labels: true
  scrape_interval: 15s
  scheme: http
  {{.KubeSDConfigs}}
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  {{.KeepLabels}}
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: dm-master
    action: keep
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_label_app_kubernetes_io_instance,
      __meta_kubernetes_namespace, __meta_kubernetes_pod_annotation_prometheus_io_port]
    regex: (.+);(.+);(.+);(.+)
    target_label: __address__
    replacement: $1.$2-dm-master-peer.$3:$4
    action: replace
  {{.ReplaceLabels}}
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
		AlertmanagerURL: "alert-url",
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
	expectedContentParsed := template.Must(template.New("relabelConfig").Parse(expectedContentTpl))
	var expectedContentBytes bytes.Buffer
	expectedContentParsed.Execute(&expectedContentBytes, promCfgModel)
	g.Expect(content).Should(Equal(expectedContentBytes.String()))
}

func TestRenderPrometheusConfigTLSEnabled(t *testing.T) {
	g := NewGomegaWithT(t)
	expectedContentTpl := `global:
  scrape_interval: 15s
  evaluation_interval: 15s
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pd
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tikv
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tiflash
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: pump
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: drainer
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: ticdc
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-cluster-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: importer
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
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
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: tidb-lightning
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
- job_name: ns1-target-dm-worker
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-dm-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-dm-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-dm-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: dm-worker
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
- job_name: ns1-target-dm-master
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
    ca_file: /var/lib/cluster-assets-tls/secret_ns1_target-dm-client-secret_ca.crt
    cert_file: /var/lib/cluster-assets-tls/secret_ns1_target-dm-client-secret_tls.crt
    key_file: /var/lib/cluster-assets-tls/secret_ns1_target-dm-client-secret_tls.key
    insecure_skip_verify: false
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_instance]
    regex: target
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    regex: ns1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    regex: "true"
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
    regex: dm-master
    action: keep
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
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_label_app_kubernetes_io_instance]
    separator: '-'
    target_label: tidb_cluster
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    regex: (.+)
    target_label: __metrics_path__
    action: replace
`
	model := &MonitorConfigModel{
		ClusterInfos: []ClusterRegexInfo{
			{Name: "target", Namespace: "ns1", enableTLS: true},
		},
		DMClusterInfos: []ClusterRegexInfo{
			{Name: "target", Namespace: "ns1", enableTLS: true},
		},
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	expectedContentParsed := template.Must(template.New("relabelConfig").Parse(expectedContentTpl))
	var expectedContentBytes bytes.Buffer
	expectedContentParsed.Execute(&expectedContentBytes, promCfgModel)
	g.Expect(content).Should(Equal(expectedContentBytes.String()))
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
		DMClusterInfos: []ClusterRegexInfo{
			{Name: "ns1", Namespace: "ns1"},
			{Name: "ns2", Namespace: "ns2"},
		},
		AlertmanagerURL: "alert-url",
	}
	// firsrt validate json generate normally
	_, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	// check scrapeJob number
	pc := newPrometheusConfig(model)
	g.Expect(len(pc.ScrapeConfigs)).Should(Equal(24))
}

func TestMultipleClusterTlsConfigRender(t *testing.T) {
	g := NewGomegaWithT(t)
	model := &MonitorConfigModel{
		ClusterInfos: []ClusterRegexInfo{
			{Name: "ns1", Namespace: "ns1", enableTLS: true},
			{Name: "ns2", Namespace: "ns2", enableTLS: true},
		},
		DMClusterInfos: []ClusterRegexInfo{
			{Name: "ns1", Namespace: "ns1", enableTLS: true},
			{Name: "ns2", Namespace: "ns2", enableTLS: true},
		},
		AlertmanagerURL: "alert-url",
	}
	// firsrt validate json generate normally
	_, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	// check scrapeJob number
	pc := newPrometheusConfig(model)
	g.Expect(pc.ScrapeConfigs[0].Scheme).Should(Equal("https"))
}

func TestScrapeJob(t *testing.T) {
	g := NewGomegaWithT(t)
	name := "ns1"
	ns := "ns1"
	ClusterInfos := []ClusterRegexInfo{
		{Name: name, Namespace: ns, enableTLS: true},
	}

	tm := &v1alpha1.TidbMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbMonitorSpec{
			Clusters: []v1alpha1.TidbClusterRef{
				{Name: ""},
			},
			Prometheus: v1alpha1.PrometheusSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage: "hub.pingcap.net",
					Version:   "latest",
				},
				Config: &v1alpha1.PrometheusConfiguration{
					CommandOptions: []string{
						"--web.external-url=https://www.example.com/prometheus/",
					},
				},
			},
		},
	}

	model := &MonitorConfigModel{
		AlertmanagerURL: "",
		ClusterInfos:    ClusterInfos,
		DMClusterInfos:  nil,
		ExternalLabels:  buildExternalLabels(tm),
	}
	scrapeJobs := scrapeJob("pd", pdPattern, model, buildAddressRelabelConfigByComponent("pd"))
	tcTlsSecretName := util.ClusterClientTLSSecretName(name)
	g.Expect(scrapeJobs[0].HTTPClientConfig.TLSConfig).Should(Equal(config.TLSConfig{
		CAFile:   path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", ns, tcTlsSecretName, corev1.ServiceAccountRootCAKey}.String()),
		CertFile: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", ns, tcTlsSecretName, corev1.TLSCertKey}.String()),
		KeyFile:  path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", ns, tcTlsSecretName, corev1.TLSPrivateKeyKey}.String()),
	}))
}
