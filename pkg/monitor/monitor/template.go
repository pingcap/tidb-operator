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
	"fmt"
	"path"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	instanceLabel              = "__meta_kubernetes_pod_label_app_kubernetes_io_instance"
	componentLabel             = "__meta_kubernetes_pod_label_app_kubernetes_io_component"
	scrapeLabel                = "__meta_kubernetes_pod_annotation_prometheus_io_scrape"
	metricsPathLabel           = "__meta_kubernetes_pod_annotation_prometheus_io_path"
	portLabel                  = "__meta_kubernetes_pod_annotation_prometheus_io_port"
	namespaceLabel             = "__meta_kubernetes_namespace"
	podNameLabel               = "__meta_kubernetes_pod_name"
	additionalPortLabelPattern = "__meta_kubernetes_pod_annotation_%s_prometheus_io_port"
)

var (
	truePattern     config.Regexp
	allMatchPattern config.Regexp
	portPattern     config.Regexp
	tikvPattern     config.Regexp
	pdPattern       config.Regexp
	tidbPattern     config.Regexp
	addressPattern  config.Regexp
	tiflashPattern  config.Regexp
	pumpPattern     config.Regexp
	drainerPattern  config.Regexp
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
	tikvPattern, err = config.NewRegexp("tikv")
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
	addressPattern, err = config.NewRegexp("(.+);(.+);(.+)")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	tiflashPattern, err = config.NewRegexp("tiflash")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	pumpPattern, err = config.NewRegexp("pump")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	drainerPattern, err = config.NewRegexp("drainer")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
}

type MonitorConfigModel struct {
	AlertmanagerURL    string
	ReleaseNamespaces  []string
	ReleaseTargetRegex *config.Regexp
	EnableTLSCluster   bool
	Drainers           []v1alpha1.DrainerRef
}

func newPrometheusConfig(cmodel *MonitorConfigModel) *config.Config {
	pdReplacement := fmt.Sprintf("$1.$2-%s-peer:$3", "pd")
	tikvReplacement := fmt.Sprintf("$1.$2-%s-peer:$3", "tikv")
	tidbReplacement := fmt.Sprintf("$1.$2-%s-peer:$3", "tidb")
	tiflashReplacement := fmt.Sprintf("$1.$2-%s-peer:$3", "tiflash")
	tiflashProxyReplacement := fmt.Sprintf("$1.$2-%s-peer:$3", "tiflash")
	tiflashProxyPortLabel := fmt.Sprintf(additionalPortLabelPattern, "tiflash_proxy")
	pumpReplacement := fmt.Sprintf("$1.$2-%s:$3", "pump")

	var c = config.Config{
		GlobalConfig: config.GlobalConfig{
			ScrapeInterval:     model.Duration(15 * time.Second),
			EvaluationInterval: model.Duration(15 * time.Second),
		},
		RuleFiles: []string{
			"/prometheus-rules/rules/*.rules.yml",
		},
		ScrapeConfigs: []*config.ScrapeConfig{
			scrapeJob("pd", pdPattern, cmodel, buildAddressRelabelConfig(portLabel, pdReplacement, true), cmodel.ReleaseNamespaces),
			scrapeJob("tidb", tidbPattern, cmodel, buildAddressRelabelConfig(portLabel, tidbReplacement, true), cmodel.ReleaseNamespaces),
			scrapeJob("tikv", tikvPattern, cmodel, buildAddressRelabelConfig(portLabel, tikvReplacement, true), cmodel.ReleaseNamespaces),
			scrapeJob("tiflash", tiflashPattern, cmodel, buildAddressRelabelConfig(portLabel, tiflashReplacement, true), cmodel.ReleaseNamespaces),
			scrapeJob("tiflash-proxy", tiflashPattern, cmodel, buildAddressRelabelConfig(tiflashProxyPortLabel, tiflashProxyReplacement, true), cmodel.ReleaseNamespaces),
			scrapeJob("pump", pumpPattern, cmodel, buildAddressRelabelConfig(portLabel, pumpReplacement, true), cmodel.ReleaseNamespaces),
		},
	}
	if len(cmodel.Drainers) > 0 {
		c.ScrapeConfigs = append(c.ScrapeConfigs, generateDrainerJobs(cmodel)...)
	}
	return &c
}

func buildAddressRelabelConfig(portLabelName, replacement string, isTidbClusterComponent bool) *config.RelabelConfig {
	addressRelabelConfig := &config.RelabelConfig{
		SourceLabels: model.LabelNames{
			"__address__",
			model.LabelName(portLabelName),
		},
		Action:      config.RelabelReplace,
		Regex:       portPattern,
		Replacement: "$1:$2",
		TargetLabel: "__address__",
	}
	if isTidbClusterComponent {
		addressRelabelConfig = &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: replacement,
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				instanceLabel,
				model.LabelName(portLabelName),
			},
		}
	}
	return addressRelabelConfig
}

func scrapeJob(jobName string, componentPattern config.Regexp, cmodel *MonitorConfigModel, addressRelabelConfig *config.RelabelConfig, sdNamespaces []string) *config.ScrapeConfig {
	return &config.ScrapeConfig{

		JobName:        jobName,
		ScrapeInterval: model.Duration(15 * time.Second),
		Scheme:         "http",
		HonorLabels:    true,
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			KubernetesSDConfigs: []*config.KubernetesSDConfig{
				{
					Role: "pod",
					NamespaceDiscovery: config.KubernetesNamespaceDiscovery{
						Names: sdNamespaces,
					},
				},
			},
		},
		HTTPClientConfig: config.HTTPClientConfig{
			TLSConfig: config.TLSConfig{
				InsecureSkipVerify: true,
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
			addressRelabelConfig,
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

func generateDrainerJobs(cmodel *MonitorConfigModel) []*config.ScrapeConfig {
	if len(cmodel.Drainers) < 1 {
		return nil
	}
	var jobs []*config.ScrapeConfig
	for _, drainer := range cmodel.Drainers {
		jobName := fmt.Sprintf("%s-drainer", drainer.ReleaseName)
		replacement := ""
		if drainer.DrainerName != nil {
			replacement = fmt.Sprintf("$1.%s:$2", *drainer.DrainerName)
		} else {
			replacement = fmt.Sprintf("$1.%s-%s-drainer:$2", drainer.ClusterName, drainer.ReleaseName)
		}
		addressRelabelConfig := &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: replacement,
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				model.LabelName(portLabel),
			},
		}
		var namespaces []string
		namespaces = append(namespaces, drainer.Namespace)
		job := scrapeJob(jobName, drainerPattern, cmodel, addressRelabelConfig, namespaces)
		jobs = append(jobs, job)
	}
	return jobs
}

func addAlertManagerUrl(pc *config.Config, cmodel *MonitorConfigModel) {
	pc.AlertingConfig = config.AlertingConfig{
		AlertmanagerConfigs: []*config.AlertmanagerConfig{
			{
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.TargetGroup{
						{
							Targets: []model.LabelSet{
								{model.AddressLabel: model.LabelValue(cmodel.AlertmanagerURL)},
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
		// TODO support tiflash tls when it gets ready
		if sconfig.JobName == "pd" || sconfig.JobName == "tidb" || sconfig.JobName == "tikv" {
			sconfig.HTTPClientConfig.TLSConfig = config.TLSConfig{
				CAFile:   path.Join(util.ClusterClientTLSPath, corev1.ServiceAccountRootCAKey),
				CertFile: path.Join(util.ClusterClientTLSPath, corev1.TLSCertKey),
				KeyFile:  path.Join(util.ClusterClientTLSPath, corev1.TLSPrivateKeyKey),
			}
			pc.ScrapeConfigs[id] = sconfig
			sconfig.Scheme = "https"
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
