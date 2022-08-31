package config

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/cmd/collector/pkg/collect"
)

// BaseConfig contains common fields for each collector configuration.
// If BaseConfig is nil, then collect all resources by default.
type BaseConfig struct {
	// Disable is the switch for collector.
	// If set, then the collector is disabled.
	Disable bool `yaml:"disable"`
	// Label specifies the label selector for collecting resources.
	Labels map[string]string `yaml:"labels"`
	// Namespace limits the namespace for collecting resources.
	// If non-empty, then collect resources in that specific namespace.
	// Otherwise, collect resources in all namespaces if global namespace is
	// not specified.
	Namespace string `yaml:"namespace"`
}

// CollectorConfig is the configuration for collector.
type CollectorConfig struct {
	// Namespace is the global namespace for collector.
	Namespace string `yaml:"namespace"`
	// Configurations for each individual resource.
	// Corev1 group
	PV        *BaseConfig `yaml:"pv,omitempty"`
	PVC       *BaseConfig `yaml:"pvc,omitempty"`
	Pod       *BaseConfig `yaml:"pod,omitempty"`
	Svc       *BaseConfig `yaml:"service,omitempty"`
	Event     *BaseConfig `yaml:"event,omitempty"`
	ConfigMap *BaseConfig `yaml:"configMap,omitempty"`
	// appsv1 group
	StatefulSet *BaseConfig `yaml:"statefulSet,omitempty"`
	// ASTS group
	ASTS *BaseConfig `yaml:"asts,omitempty"`
	// tidb-operator group
	TidbCluster *BaseConfig `yaml:"tidbCluster,omitempty"`
}

// configureCollector configures a single collector based on its configs.
func configureCollector(cli client.Reader, collectorFn func(client.Reader) collect.Collector, config *BaseConfig, globalNS string) collect.Collector {
	if config != nil && config.Disable {
		return nil
	}
	collector := collectorFn(cli)
	// config namespace
	if config != nil && config.Namespace != "" {
		collector.WithNamespace(config.Namespace)
	} else if globalNS != "" {
		collector.WithNamespace(globalNS)
	}
	// config label selector
	if config != nil && len(config.Labels) != 0 {
		collector.WithLabel(config.Labels)
	}

	return collector
}

// ConfigCollectors returns a list of collectors based on configuraitons.
func ConfigCollectors(cli client.Reader, config CollectorConfig) (res []collect.Collector) {
	if c := configureCollector(cli, collect.NewPVCollector, config.PV, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewPVCCollector, config.PVC, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewPodCollector, config.Pod, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewEventCollector, config.Event, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewServiceCollector, config.Svc, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewASTSCollector, config.ASTS, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewTidbClusterCollector, config.TidbCluster, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewSSCollector, config.StatefulSet, config.Namespace); c != nil {
		res = append(res, c)
	}
	if c := configureCollector(cli, collect.NewConfigMapCollector, config.ConfigMap, config.Namespace); c != nil {
		res = append(res, c)
	}
	return
}
