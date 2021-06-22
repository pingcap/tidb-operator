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
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	nameLabel                  = "__meta_kubernetes_pod_label_app_kubernetes_io_name"
	instanceLabel              = "__meta_kubernetes_pod_label_app_kubernetes_io_instance"
	componentLabel             = "__meta_kubernetes_pod_label_app_kubernetes_io_component"
	scrapeLabel                = "__meta_kubernetes_pod_annotation_prometheus_io_scrape"
	metricsPathLabel           = "__meta_kubernetes_pod_annotation_prometheus_io_path"
	portLabel                  = "__meta_kubernetes_pod_annotation_prometheus_io_port"
	namespaceLabel             = "__meta_kubernetes_namespace"
	podNameLabel               = "__meta_kubernetes_pod_name"
	additionalPortLabelPattern = "__meta_kubernetes_pod_annotation_%s_prometheus_io_port"
	dmWorker                   = "dm-worker"
	dmMaster                   = "dm-master"
)

var (
	truePattern      config.Regexp
	allMatchPattern  config.Regexp
	portPattern      config.Regexp
	tikvPattern      config.Regexp
	pdPattern        config.Regexp
	tidbPattern      config.Regexp
	addressPattern   config.Regexp
	tiflashPattern   config.Regexp
	pumpPattern      config.Regexp
	drainerPattern   config.Regexp
	cdcPattern       config.Regexp
	importerPattern  config.Regexp
	lightningPattern config.Regexp
	dmWorkerPattern  config.Regexp
	dmMasterPattern  config.Regexp
	dashBoardConfig  = `{
    "apiVersion": 1,
    "providers": [
        {
            "folder": "",
            "name": "0",
            "options": {
                "path": "/grafana-dashboard-definitions/tidb"
            },
			"allowUiUpdates":true,
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
	addressPattern, err = config.NewRegexp("(.+);(.+);(.+);(.+)")
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
	cdcPattern, err = config.NewRegexp("ticdc")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	importerPattern, err = config.NewRegexp("importer")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	lightningPattern, err = config.NewRegexp("tidb-lightning")
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	dmWorkerPattern, err = config.NewRegexp(dmWorker)
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
	dmMasterPattern, err = config.NewRegexp(dmMaster)
	if err != nil {
		klog.Fatalf("monitor regex template parse error,%v", err)
	}
}

type MonitorConfigModel struct {
	AlertmanagerURL    string
	ClusterInfos       []ClusterRegexInfo
	DMClusterInfos     []ClusterRegexInfo
	ExternalLabels     model.LabelSet
	RemoteWriteConfigs []*config.RemoteWriteConfig
}

// ClusterRegexInfo is the monitor cluster info
type ClusterRegexInfo struct {
	Name      string
	Namespace string
	enableTLS bool
}

func newPrometheusConfig(cmodel *MonitorConfigModel) *config.Config {
	var scrapeJobs []*config.ScrapeConfig
	scrapeJobs = append(scrapeJobs, scrapeJob("pd", pdPattern, cmodel, buildAddressRelabelConfigByComponent("pd"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tidb", tidbPattern, cmodel, buildAddressRelabelConfigByComponent("tidb"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tikv", tikvPattern, cmodel, buildAddressRelabelConfigByComponent("tikv"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tiflash", tiflashPattern, cmodel, buildAddressRelabelConfigByComponent("tiflash"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tiflash-proxy", tiflashPattern, cmodel, buildAddressRelabelConfigByComponent("tiflash-proxy"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("pump", pumpPattern, cmodel, buildAddressRelabelConfigByComponent("pump"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("drainer", drainerPattern, cmodel, buildAddressRelabelConfigByComponent("drainer"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("ticdc", cdcPattern, cmodel, buildAddressRelabelConfigByComponent("ticdc"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("importer", importerPattern, cmodel, buildAddressRelabelConfigByComponent("importer"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("lightning", lightningPattern, cmodel, buildAddressRelabelConfigByComponent("lightning"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob(dmWorker, dmWorkerPattern, cmodel, buildAddressRelabelConfigByComponent(dmWorker))...)
	scrapeJobs = append(scrapeJobs, scrapeJob(dmMaster, dmMasterPattern, cmodel, buildAddressRelabelConfigByComponent(dmMaster))...)
	var c = config.Config{
		GlobalConfig: config.GlobalConfig{
			ScrapeInterval:     model.Duration(15 * time.Second),
			EvaluationInterval: model.Duration(15 * time.Second),
			ExternalLabels:     cmodel.ExternalLabels,
		},
		ScrapeConfigs:      scrapeJobs,
		RemoteWriteConfigs: cmodel.RemoteWriteConfigs,
	}
	return &c
}

func buildAddressRelabelConfigByComponent(kind string) *config.RelabelConfig {
	kind = strings.ToLower(kind)
	replacement := fmt.Sprintf("$1.$2-%s-peer.$3:$4", kind)
	f := func() *config.RelabelConfig {
		return &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: replacement,
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				instanceLabel,
				namespaceLabel,
				portLabel,
			},
		}
	}
	switch strings.ToLower(kind) {
	case "pd":
		return f()
	case "tidb":
		return f()
	case "tikv":
		return f()
	case "tiflash":
		return f()
	case "ticdc":
		return f()
	case dmWorker:
		return f()
	case dmMaster:
		return f()
	case "tiflash-proxy":
		return &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: "$1.$2-tiflash-peer.$3:$4",
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				instanceLabel,
				namespaceLabel,
				model.LabelName(fmt.Sprintf(additionalPortLabelPattern, "tiflash_proxy")),
			},
		}
	case "pump":
		return &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: "$1.$2-pump.$3:$4",
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				instanceLabel,
				namespaceLabel,
				portLabel,
			},
		}
	case "importer":
		return &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: "$1.$2-importer.$3:$4",
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				instanceLabel,
				namespaceLabel,
				portLabel,
			},
		}
	case "drainer":
		return &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: "$1.$2.$3:$4",
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				nameLabel,
				namespaceLabel,
				portLabel,
			},
		}
	case "lightning":
		return &config.RelabelConfig{
			Action:      config.RelabelReplace,
			Regex:       addressPattern,
			Replacement: "$2.$3:$4",
			TargetLabel: "__address__",
			SourceLabels: model.LabelNames{
				podNameLabel,
				nameLabel,
				namespaceLabel,
				portLabel,
			},
		}
	default:
		return &config.RelabelConfig{
			SourceLabels: model.LabelNames{
				"__address__",
				portLabel,
			},
			Action:      config.RelabelReplace,
			Regex:       portPattern,
			Replacement: "$1:$2",
			TargetLabel: "__address__",
		}
	}
}

func scrapeJob(jobName string, componentPattern config.Regexp, cmodel *MonitorConfigModel, addressRelabelConfig *config.RelabelConfig) []*config.ScrapeConfig {
	var scrapeJobs []*config.ScrapeConfig
	var currCluster []ClusterRegexInfo

	if isDMJob(jobName) {
		currCluster = cmodel.DMClusterInfos
	} else {
		currCluster = cmodel.ClusterInfos
	}

	for _, cluster := range currCluster {
		clusterTargetPattern, err := config.NewRegexp(cluster.Name)
		if err != nil {
			klog.Errorf("generate scrapeJob[%s] clusterName:%s error:%v", jobName, cluster.Name, err)
			continue
		}
		nsTargetPattern, err := config.NewRegexp(cluster.Namespace)
		if err != nil {
			klog.Errorf("generate scrapeJob[%s] clusterName:%s namespace:%s error:%v", jobName, cluster.Name, cluster.Namespace, err)
			continue
		}

		scrapeconfig := &config.ScrapeConfig{
			JobName:        fmt.Sprintf("%s-%s-%s", cluster.Namespace, cluster.Name, jobName),
			ScrapeInterval: model.Duration(15 * time.Second),
			Scheme:         "http",
			HonorLabels:    true,
			ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
				KubernetesSDConfigs: []*config.KubernetesSDConfig{
					{
						Role: "pod",
						NamespaceDiscovery: config.KubernetesNamespaceDiscovery{
							Names: []string{cluster.Namespace},
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
					Regex:  clusterTargetPattern,
				},
				{
					SourceLabels: model.LabelNames{
						namespaceLabel,
					},
					Action: config.RelabelKeep,
					Regex:  nsTargetPattern,
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
						componentLabel,
					},
					Action: config.RelabelKeep,
					Regex:  componentPattern,
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
						componentLabel,
					},
					Action:      config.RelabelReplace,
					TargetLabel: "component",
				},
				{
					SourceLabels: model.LabelNames{
						namespaceLabel,
						instanceLabel,
					},
					Separator:   "-",
					TargetLabel: "tidb_cluster",
				},
				{
					SourceLabels: model.LabelNames{
						metricsPathLabel,
					},
					Action:      config.RelabelReplace,
					TargetLabel: "__metrics_path__",
					Regex:       allMatchPattern,
				},
			},
		}

		if cluster.enableTLS && !isDMJob(jobName) {
			scrapeconfig.Scheme = "https"
			// lightning does not need to authenticate the access of other components,
			// so there is no need to enable mtls for the time being.
			if jobName != "lightning" {
				tcTlsSecretName := util.ClusterClientTLSSecretName(cluster.Name)
				scrapeconfig.HTTPClientConfig.TLSConfig = config.TLSConfig{
					CAFile:   path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, tcTlsSecretName, corev1.ServiceAccountRootCAKey}.String()),
					CertFile: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, tcTlsSecretName, corev1.TLSCertKey}.String()),
					KeyFile:  path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, tcTlsSecretName, corev1.TLSPrivateKeyKey}.String()),
				}
			}
		}

		if cluster.enableTLS && isDMJob(jobName) {
			scrapeconfig.Scheme = "https"
			dmTlsSecretName := util.DMClientTLSSecretName(cluster.Name)
			scrapeconfig.HTTPClientConfig.TLSConfig = config.TLSConfig{
				CAFile:   path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, dmTlsSecretName, corev1.ServiceAccountRootCAKey}.String()),
				CertFile: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, dmTlsSecretName, corev1.TLSCertKey}.String()),
				KeyFile:  path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, dmTlsSecretName, corev1.TLSPrivateKeyKey}.String()),
			}
		}
		scrapeJobs = append(scrapeJobs, scrapeconfig)

	}
	return scrapeJobs

}

func isDMJob(jobName string) bool {
	if jobName == dmMaster || jobName == dmWorker {
		return true
	}
	return false
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

func RenderPrometheusConfig(model *MonitorConfigModel) (string, error) {
	pc := newPrometheusConfig(model)
	if len(model.AlertmanagerURL) > 0 {
		addAlertManagerUrl(pc, model)
		pc.RuleFiles = []string{
			"/prometheus-rules/rules/*.rules.yml",
		}
	}
	bs, err := yaml.Marshal(pc)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
