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

	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
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
	truePattern      = "true"
	allMatchPattern  = "(.+)"
	portPattern      = "([^:]+)(?::\\d+)?;(\\d+)"
	tikvPattern      = "tikv"
	tiproxyPattern   = "tiproxy"
	pdPattern        = "pd"
	tidbPattern      = "tidb"
	addressPattern   = "(.+);(.+);(.+);(.+)"
	tiflashPattern   = "tiflash"
	pumpPattern      = "pump"
	drainerPattern   = "drainer"
	cdcPattern       = "ticdc"
	importerPattern  = "importer"
	lightningPattern = "tidb-lightning"
	dmWorkerPattern  = dmWorker
	dmMasterPattern  = dmMaster
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

type MonitorConfigModel struct {
	AlertmanagerURL           string
	ClusterInfos              []ClusterRegexInfo
	DMClusterInfos            []ClusterRegexInfo
	ExternalLabels            model.LabelSet
	RemoteWriteCfg            *yaml.MapItem
	EnableAlertRules          bool
	EnableExternalRuleConfigs bool
	shards                    int32
}

// ClusterRegexInfo is the monitor cluster info
type ClusterRegexInfo struct {
	Name      string
	Namespace string
	enableTLS bool
}

func newPrometheusConfig(cmodel *MonitorConfigModel) yaml.MapSlice {
	var scrapeJobs []yaml.MapSlice
	scrapeJobs = append(scrapeJobs, scrapeJob("pd", pdPattern, cmodel, buildAddressRelabelConfigByComponent("pd"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tidb", tidbPattern, cmodel, buildAddressRelabelConfigByComponent("tidb"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tikv", tikvPattern, cmodel, buildAddressRelabelConfigByComponent("tikv"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tiproxy", tiproxyPattern, cmodel, buildAddressRelabelConfigByComponent("tiproxy"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tiflash", tiflashPattern, cmodel, buildAddressRelabelConfigByComponent("tiflash"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("tiflash-proxy", tiflashPattern, cmodel, buildAddressRelabelConfigByComponent("tiflash-proxy"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("pump", pumpPattern, cmodel, buildAddressRelabelConfigByComponent("pump"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("drainer", drainerPattern, cmodel, buildAddressRelabelConfigByComponent("drainer"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("ticdc", cdcPattern, cmodel, buildAddressRelabelConfigByComponent("ticdc"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("importer", importerPattern, cmodel, buildAddressRelabelConfigByComponent("importer"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob("lightning", lightningPattern, cmodel, buildAddressRelabelConfigByComponent("lightning"))...)
	scrapeJobs = append(scrapeJobs, scrapeJob(dmWorker, dmWorkerPattern, cmodel, buildAddressRelabelConfigByComponent(dmWorker))...)
	scrapeJobs = append(scrapeJobs, scrapeJob(dmMaster, dmMasterPattern, cmodel, buildAddressRelabelConfigByComponent(dmMaster))...)
	cfg := yaml.MapSlice{}
	globalItems := yaml.MapSlice{
		{Key: "evaluation_interval", Value: "15s"},
		{Key: "scrape_interval", Value: "15s"},
		{Key: "external_labels", Value: cmodel.ExternalLabels},
	}
	cfg = append(cfg, yaml.MapItem{Key: "global", Value: globalItems})
	cfg = append(cfg, yaml.MapItem{Key: "scrape_configs", Value: scrapeJobs})
	if cmodel.RemoteWriteCfg != nil {
		cfg = append(cfg, *cmodel.RemoteWriteCfg)
	}

	return cfg
}

func buildAddressRelabelConfigByComponent(kind string) yaml.MapSlice {
	kind = strings.ToLower(kind)
	replacement := fmt.Sprintf("$1.$2-%s-peer.$3:$4", kind)
	f := func() yaml.MapSlice {
		return yaml.MapSlice{
			{Key: "action", Value: "replace"},
			{Key: "regex", Value: addressPattern},
			{Key: "replacement", Value: replacement},
			{Key: "target_label", Value: "__address__"},
			{Key: "source_labels", Value: []string{podNameLabel,
				instanceLabel,
				namespaceLabel,
				portLabel}},
		}
	}

	switch strings.ToLower(kind) {
	case "pd":
		return f()
	case "tidb":
		return f()
	case "tikv":
		return f()
	case "tiproxy":
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
		return yaml.MapSlice{
			{Key: "action", Value: "replace"},
			{Key: "regex", Value: addressPattern},
			{Key: "replacement", Value: "$1.$2-tiflash-peer.$3:$4"},
			{Key: "target_label", Value: "__address__"},
			{Key: "source_labels", Value: []string{podNameLabel,
				instanceLabel,
				namespaceLabel,
				fmt.Sprintf(additionalPortLabelPattern, "tiflash_proxy")}},
		}

	case "pump":
		return yaml.MapSlice{
			{Key: "action", Value: "replace"},
			{Key: "regex", Value: addressPattern},
			{Key: "replacement", Value: "$1.$2-pump.$3:$4"},
			{Key: "target_label", Value: "__address__"},
			{Key: "source_labels", Value: []string{
				podNameLabel,
				instanceLabel,
				namespaceLabel,
				portLabel,
			}},
		}
	case "importer":
		return yaml.MapSlice{
			{Key: "action", Value: "replace"},
			{Key: "regex", Value: addressPattern},
			{Key: "replacement", Value: "$1.$2-importer.$3:$4"},
			{Key: "target_label", Value: "__address__"},
			{Key: "source_labels", Value: []string{
				podNameLabel,
				instanceLabel,
				namespaceLabel,
				portLabel,
			}},
		}
	case "drainer":
		return yaml.MapSlice{
			{Key: "action", Value: "replace"},
			{Key: "regex", Value: addressPattern},
			{Key: "replacement", Value: "$1.$2.$3:$4"},
			{Key: "target_label", Value: "__address__"},
			{Key: "source_labels", Value: []string{
				podNameLabel,
				nameLabel,
				namespaceLabel,
				portLabel,
			}},
		}
	case "lightning":
		return yaml.MapSlice{
			{Key: "action", Value: "replace"},
			{Key: "regex", Value: addressPattern},
			{Key: "replacement", Value: "$2.$3:$4"},
			{Key: "target_label", Value: "__address__"},
			{Key: "source_labels", Value: []string{
				podNameLabel,
				nameLabel,
				namespaceLabel,
				portLabel,
			}},
		}
	default:
		return yaml.MapSlice{
			{Key: "source_labels", Value: []string{
				"__address__",
				portLabel,
			}},
			{Key: "action", Value: "replace"},
			{Key: "regex", Value: portPattern},
			{Key: "replacement", Value: "$1:$2"},
			{Key: "target_label", Value: "__address__"},
		}

	}
}

func scrapeJob(jobName string, componentPattern string, cmodel *MonitorConfigModel, addressRelabelConfig yaml.MapSlice) []yaml.MapSlice {
	var scrapeJobs []yaml.MapSlice
	var currCluster []ClusterRegexInfo

	if isDMJob(jobName) {
		currCluster = cmodel.DMClusterInfos
	} else {
		currCluster = cmodel.ClusterInfos
	}

	for _, cluster := range currCluster {
		clusterTargetPattern := cluster.Name

		nsTargetPattern := cluster.Namespace

		schemeRelabelConfig := yaml.MapItem{
			Key:   "scheme",
			Value: "http",
		}
		tlsConfigRelabelConfig := yaml.MapSlice{
			{
				Key:   "insecure_skip_verify",
				Value: true,
			},
		}

		if cluster.enableTLS && !isDMJob(jobName) {
			schemeRelabelConfig.Value = "https"
			// lightning does not need to authenticate the access of other components,
			// so there is no need to enable mtls for the time being.
			if jobName != "lightning" {
				tcTlsSecretName := util.ClusterClientTLSSecretName(cluster.Name)
				tlsConfigRelabelConfig = yaml.MapSlice{
					yaml.MapItem{
						Key:   "ca_file",
						Value: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, tcTlsSecretName, corev1.ServiceAccountRootCAKey}.String()),
					},
					yaml.MapItem{
						Key:   "cert_file",
						Value: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, tcTlsSecretName, corev1.TLSCertKey}.String()),
					},
					yaml.MapItem{
						Key:   "key_file",
						Value: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, tcTlsSecretName, corev1.TLSPrivateKeyKey}.String()),
					},
				}
			}
		}

		if cluster.enableTLS && isDMJob(jobName) {
			schemeRelabelConfig.Value = "https"
			dmTlsSecretName := util.DMClientTLSSecretName(cluster.Name)
			tlsConfigRelabelConfig = yaml.MapSlice{
				yaml.MapItem{
					Key:   "ca_file",
					Value: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, dmTlsSecretName, corev1.ServiceAccountRootCAKey}.String()),
				},
				yaml.MapItem{
					Key:   "cert_file",
					Value: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, dmTlsSecretName, corev1.TLSCertKey}.String()),
				},
				yaml.MapItem{
					Key:   "key_file",
					Value: path.Join(util.ClusterAssetsTLSPath, TLSAssetKey{"secret", cluster.Namespace, dmTlsSecretName, corev1.TLSPrivateKeyKey}.String()),
				},
			}

		}

		scrapeConfig := yaml.MapSlice{
			{Key: "job_name", Value: fmt.Sprintf("%s-%s-%s", cluster.Namespace, cluster.Name, jobName)},
			{Key: "honor_labels", Value: true},
			{Key: "scrape_interval", Value: "15s"},
			schemeRelabelConfig,
			{Key: "kubernetes_sd_configs", Value: []yaml.MapSlice{
				{
					{
						Key:   "api_server",
						Value: nil,
					},
					{
						Key:   "role",
						Value: "pod",
					},
					{
						Key: "namespaces",
						Value: yaml.MapSlice{
							{
								Key:   "names",
								Value: []string{cluster.Namespace},
							},
						},
					},
				},
			}},
			{Key: "tls_config", Value: tlsConfigRelabelConfig},
		}

		relabelConfigs := []yaml.MapSlice{}
		relabelConfigs = append(relabelConfigs, yaml.MapSlice{
			{Key: "source_labels", Value: []string{instanceLabel}},
			{Key: "action", Value: "keep"},
			{Key: "regex", Value: clusterTargetPattern},
		},
			yaml.MapSlice{
				{Key: "source_labels", Value: []string{namespaceLabel}},
				{Key: "action", Value: "keep"},
				{Key: "regex", Value: nsTargetPattern},
			},
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{scrapeLabel},
				},
				{
					Key: "action", Value: "keep",
				},
				{
					Key: "regex", Value: truePattern,
				},
			},
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{componentLabel},
				},
				{
					Key: "action", Value: "keep",
				},
				{
					Key: "regex", Value: componentPattern,
				},
			},
			addressRelabelConfig,
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{namespaceLabel},
				},
				{
					Key: "action", Value: "replace",
				},
				{
					Key: "target_label", Value: "kubernetes_namespace",
				},
			},
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{instanceLabel},
				},
				{
					Key: "action", Value: "replace",
				},
				{
					Key: "target_label", Value: "cluster",
				},
			},
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{podNameLabel},
				},
				{
					Key: "action", Value: "replace",
				},
				{
					Key: "target_label", Value: "instance",
				},
			},
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{componentLabel},
				},
				{
					Key: "action", Value: "replace",
				},
				{
					Key: "target_label", Value: "component",
				},
			},
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{
						namespaceLabel,
						instanceLabel,
					},
				},
				{
					Key: "separator", Value: "-",
				},
				{
					Key: "target_label", Value: "tidb_cluster",
				},
			},
			yaml.MapSlice{
				{
					Key: "source_labels", Value: []string{metricsPathLabel},
				},
				{
					Key: "action", Value: "replace",
				},
				{
					Key: "target_label", Value: "__metrics_path__",
				},
				{
					Key: "regex", Value: allMatchPattern,
				},
			},
		)

		relabelConfigs = appendShardingRelabelConfigRules(relabelConfigs, uint64(cmodel.shards))
		scrapeConfig = append(scrapeConfig, yaml.MapItem{Key: "relabel_configs", Value: relabelConfigs})
		scrapeJobs = append(scrapeJobs, scrapeConfig)

	}
	return scrapeJobs

}

func isDMJob(jobName string) bool {
	if jobName == dmMaster || jobName == dmWorker {
		return true
	}
	return false
}

func addAlertManagerUrl(cfg yaml.MapSlice, cmodel *MonitorConfigModel) yaml.MapSlice {
	cfg = append(cfg, yaml.MapItem{
		Key: "alerting",
		Value: yaml.MapSlice{
			{
				Key: "alertmanagers",
				Value: []yaml.MapSlice{
					{
						{
							Key: "static_configs", Value: []yaml.MapSlice{
								{
									{
										Key:   "targets",
										Value: []string{cmodel.AlertmanagerURL},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	return cfg
}

func RenderPrometheusConfig(model *MonitorConfigModel) (yaml.MapSlice, error) {
	cfg := newPrometheusConfig(model)
	var rulesPath []string
	if len(model.AlertmanagerURL) > 0 {
		cfg = addAlertManagerUrl(cfg, model)
		rulesPath = []string{
			"/prometheus-rules/rules/*.rules.yml",
		}
	}
	if model.EnableAlertRules {
		// Add alert rules when `EnableAlertRules` enabled even if AlertManager not configured.
		rulesPath = []string{
			"/prometheus-rules/rules/*.rules.yml",
		}
	}
	if model.EnableExternalRuleConfigs {
		rulesPath = []string{
			"/prometheus-external-rules/*.rules.yml",
		}
	}
	if rulesPath != nil {
		cfg = append(cfg, yaml.MapItem{
			Key:   "rule_files",
			Value: rulesPath,
		})
	}
	return cfg, nil
}

func appendShardingRelabelConfigRules(relabelConfigs []yaml.MapSlice, shard uint64) []yaml.MapSlice {
	shardsPattern := "$(SHARD)"
	return append(relabelConfigs, yaml.MapSlice{
		{Key: "source_labels", Value: []string{"__address__"}},
		{Key: "action", Value: "hashmod"},
		{Key: "target_label", Value: "__tmp_hash"},
		{Key: "modulus", Value: shard},
	}, yaml.MapSlice{
		{Key: "source_labels", Value: []string{"__tmp_hash"}},
		{Key: "regex", Value: shardsPattern},
		{Key: "action", Value: "keep"},
	},
	)
}
