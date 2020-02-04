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
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

const (
	instanceLabel    = "__meta_kubernetes_pod_label_app_kubernetes_io_instance"
	componentLabel   = "__meta_kubernetes_pod_label_app_kubernetes_io_component"
	scrapeLabel      = "__meta_kubernetes_pod_annotation_prometheus_io_scrape"
	metricsPathLabel = "__meta_kubernetes_pod_annotation_prometheus_io_path"
	ioPortLabel      = "__meta_kubernetes_pod_annotation_prometheus_io_port"
	namespaceLabel   = "__meta_kubernetes_namespace"
	podNameLabel     = "__meta_kubernetes_pod_name"
	nodeNameLabel    = "__meta_kubernetes_pod_node_name"
	podIPLabel       = "__meta_kubernetes_pod_ip"
	caFilePath       = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	certFilePath     = "/var/lib/pd-client-tls/cert"
	keyFilePath      = "/var/lib/pd-client-tls/key"
)

var (
	truePattern     config.Regexp
	allMatchPattern config.Regexp
	portPattern     config.Regexp
	tikvPattern     config.Regexp
	pdPattern       config.Regexp
	tidbPattern     config.Regexp
	dashBoardConfig = `{
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

func init() {
	var err error
	truePattern, err = config.NewRegexp("true")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	allMatchPattern, err = config.NewRegexp("(.+)")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	portPattern, err = config.NewRegexp("([^:]+)(?::\\d+)?;(\\d+)")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	tikvPattern, err = config.NewRegexp(".*\\-tikv\\-\\d*$")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	pdPattern, err = config.NewRegexp("pd")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	tidbPattern, err = config.NewRegexp("tidb")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
}

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
			scrapeJob("pd", pdPattern, cmodel),
			scrapeJob("tidb", tidbPattern, cmodel),
			scrapeJob("tikv", tikvPattern, cmodel),
		},
	}
	return &c
}

func scrapeJob(name string, componentPattern config.Regexp, cmodel *MonitorConfigModel) *config.ScrapeConfig {
	return &config.ScrapeConfig{

		JobName:        name,
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
					instanceLabel,
				},
				Action: config.RelabelKeep,
				Regex:  *cmodel.ReleaseTargetRegex,
			},
			{
				SourceLabels: model.LabelNames{
					componentLabel,
				},
				Action: config.RelabelKeep,
				Regex:  componentPattern,
			},
			{
				SourceLabels: model.LabelNames{
					scrapeLabel,
				},
				Action: config.RelabelKeep,
				Regex:  truePattern,
			},
			{
				SourceLabels: model.LabelNames{
					metricsPathLabel,
				},
				Action:      config.RelabelReplace,
				TargetLabel: "__metrics_path__",
				Regex:       allMatchPattern,
			},
			{
				SourceLabels: model.LabelNames{
					"__address__",
					ioPortLabel,
				},
				Action:      config.RelabelReplace,
				Regex:       portPattern,
				Replacement: "$1:$2",
				TargetLabel: "__address__",
			},
			{
				SourceLabels: model.LabelNames{
					namespaceLabel,
				},
				Action:      config.RelabelReplace,
				TargetLabel: "kubernetes_namespace",
			},
			{
				SourceLabels: model.LabelNames{
					podNameLabel,
				},
				Action:      config.RelabelReplace,
				TargetLabel: "instance",
			},
			{
				SourceLabels: model.LabelNames{
					instanceLabel,
				},
				Action:      config.RelabelReplace,
				TargetLabel: "cluster",
			},
			{
				SourceLabels: model.LabelNames{
					podNameLabel,
				},
				Action:      config.RelabelReplace,
				TargetLabel: "instance",
			},
			{
				SourceLabels: model.LabelNames{
					instanceLabel,
				},
				Action:      config.RelabelReplace,
				TargetLabel: "cluster",
			},
		},
	}

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

func addTlsConfig(pc *config.Config) {

	for id, sconfig := range pc.ScrapeConfigs {
		// TiKV doesn't support scheme https for now.
		// And we should fix it after TiKV fix this issue: https://github.com/tikv/tikv/issues/5340
		if sconfig.JobName == "pd" || sconfig.JobName == "tidb" {
			sconfig.HTTPClientConfig.TLSConfig = config.TLSConfig{
				CAFile:   caFilePath,
				CertFile: certFilePath,
				KeyFile:  keyFilePath,
			}
			pc.ScrapeConfigs[id] = sconfig
			sconfig.HTTPClientConfig.XXX["scheme"] = "https"
		}
	}
}

func RenderPrometheusConfig(model *MonitorConfigModel) (string, error) {
	pc := newPrometheusConfig(model)
	if model.EnableTLSCluster {
		addTlsConfig(pc)
	}
	if len(model.AlertmanagerURL) > 0 {
		addAlertManagerUrl(pc, model)
	}
	bs, err := yaml.Marshal(pc)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
