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
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	"time"
)

var (
	truePattern, _     = config.NewRegexp("true")
	allMatchPattern, _ = config.NewRegexp("(.+)")
	portPattern, _     = config.NewRegexp("([^:]+)(?::\\d+)?;(\\d+)")
	dashBoardConfig    = `{
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
)

type MonitorConfigModel struct {
	AlertmanagerURL    string
	ReleaseNamespaces  []string
	ReleaseTargetRegex *config.Regexp
	EnableTLSCluster   bool
}

func newPrometheusConfig(cmodel *MonitorConfigModel) *config.Config {
	var c = config.Config{
		AlertingConfig: config.AlertingConfig{
			AlertRelabelConfigs: nil,
			AlertmanagerConfigs: nil,
			XXX:                 nil,
		},
		GlobalConfig: config.GlobalConfig{
			ScrapeInterval:     model.Duration(15 * time.Second),
			EvaluationInterval: model.Duration(15 * time.Second),
		},
		RuleFiles: []string{
			"/prometheus-rules/rules/*.rules.yml",
		},
		ScrapeConfigs: []*config.ScrapeConfig{
			{
				JobName:        "tidb-cluster",
				ScrapeInterval: model.Duration(15 * time.Second),
				HonorLabels:    true,
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					KubernetesSDConfigs: []*config.KubernetesSDConfig{
						{
							Role: "pod",
							NamespaceDiscovery: config.KubernetesNamespaceDiscovery{
								Names: cmodel.ReleaseNamespaces,
							},
						},
					},
				},
				HTTPClientConfig: config.HTTPClientConfig{
					TLSConfig: config.TLSConfig{
						InsecureSkipVerify: true,
					},
					XXX: map[string]interface{}{
						"scheme": "http",
					},
				},
				RelabelConfigs: []*config.RelabelConfig{
					{
						SourceLabels: model.LabelNames{
							"__meta_kubernetes_pod_label_app_kubernetes_io_instance",
						},
						Action: "keep",
						Regex:  *cmodel.ReleaseTargetRegex,
					},
					{
						SourceLabels: model.LabelNames{
							"__meta_kubernetes_pod_annotation_prometheus_io_scrape",
						},
						Action: "keep",
						Regex:  truePattern,
					},
					{
						SourceLabels: model.LabelNames{
							"__meta_kubernetes_pod_annotation_prometheus_io_path",
						},
						Action:      "replace",
						TargetLabel: "__metrics_path__",
						Regex:       allMatchPattern,
					},
					{
						SourceLabels: model.LabelNames{
							"__meta_kubernetes_namespace",
						},
						Action:      "replace",
						TargetLabel: "kubernetes_pod_ip",
					},
					{
						SourceLabels: model.LabelNames{
							"__meta_kubernetes_pod_name",
						},
						Action:      "replace",
						TargetLabel: "instance",
					},
					{
						SourceLabels: model.LabelNames{
							model.LabelName("__meta_kubernetes_pod_label_app_kubernetes_io_instance"),
						},
						Action:      "replace",
						TargetLabel: "cluster",
					},
				},
			},
		},
	}
	return &c
}

func addAlertManagerUrl(pc *config.Config, cmodel *MonitorConfigModel) {
	pc.AlertingConfig = config.AlertingConfig{
		AlertmanagerConfigs: []*config.AlertmanagerConfig{
			{
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.TargetGroup{
						{
							Targets: []model.LabelSet{
								map[model.LabelName]model.LabelValue{
									"targets": model.LabelValue(cmodel.AlertmanagerURL),
								},
							},
						},
					},
				},
			},
		},
	}
}

func addTlsConfig(pc *config.Config, cmodel *MonitorConfigModel) {

	r, _ := config.NewRegexp(".*\\-tikv\\-\\d*$")
	for id, sconfig := range pc.ScrapeConfigs {
		if sconfig.JobName == "tidb-cluster" {
			sconfig.HTTPClientConfig.TLSConfig = config.TLSConfig{
				CAFile:   "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
				CertFile: "/var/lib/pd-client-tls/cert",
				KeyFile:  "/var/lib/pd-client-tls/key",
			}
			sconfig.RelabelConfigs = append(sconfig.RelabelConfigs, &config.RelabelConfig{
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_name",
				},
				Action: "drop",
				Regex:  r,
			})
			pc.ScrapeConfigs[id] = sconfig
			sconfig.HTTPClientConfig.XXX["scheme"] = "https"
			break
		}
	}

	// This is a workaround of https://github.com/tikv/tikv/issues/5340 and should
	// be removed after TiKV fix this issue
	pc.ScrapeConfigs = append(pc.ScrapeConfigs, &config.ScrapeConfig{
		JobName:        "tidb-cluster-tikv",
		ScrapeInterval: model.Duration(15 * time.Second),
		HonorLabels:    true,
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			KubernetesSDConfigs: []*config.KubernetesSDConfig{
				{
					Role: "pod",
					NamespaceDiscovery: config.KubernetesNamespaceDiscovery{
						Names: cmodel.ReleaseNamespaces,
					},
				},
			},
		},
		HTTPClientConfig: config.HTTPClientConfig{
			TLSConfig: config.TLSConfig{
				InsecureSkipVerify: true,
			},
			XXX: map[string]interface{}{
				"scheme": "http",
			},
		},
		RelabelConfigs: []*config.RelabelConfig{
			{
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_label_app_kubernetes_io_instance",
				},
				Action: "keep",
				Regex:  *cmodel.ReleaseTargetRegex,
			},
			{
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_annotation_prometheus_io_scrape]",
				},
				Action: "keep",
				Regex:  truePattern,
			},
			{
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_annotation_prometheus_io_path",
				},
				Action:      "replace",
				TargetLabel: "__metrics_path__",
				Regex:       allMatchPattern,
			},
			{
				SourceLabels: model.LabelNames{
					"__address__, __meta_kubernetes_pod_annotation_prometheus_io_port",
				},
				Action:      "replace",
				Regex:       portPattern,
				Replacement: "$1:$2",
				TargetLabel: "__address__",
			},
			{
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_namespace",
				},
				Action:      "replace",
				TargetLabel: "kubernetes_namespace",
			},
			{
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_node_name",
				},
				Action:      "replace",
				TargetLabel: "kubernetes_node",
			},
			{
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_ip",
				},
				Action:      "replace",
				TargetLabel: "kubernetes_pod_ip",
			},
		},
	})
}

func RenderPrometheusConfig(model *MonitorConfigModel) (string, error) {
	pc := newPrometheusConfig(model)
	if model.EnableTLSCluster {
		addTlsConfig(pc, model)
	}
	if len(model.AlertmanagerURL) > 0 {
		addAlertManagerUrl(pc, model)
	}
	bs, err := yaml.Marshal(pc)
	if err != nil {
		return "", err
	}
	// remove character "'"
	return string(bs), nil
}
