package main

import (
	"fmt"

	"github.com/pingcap/tidb-operator/tests"
	v1 "k8s.io/api/core/v1"
)

func newOperatorConfig() *tests.OperatorConfig {
	return &tests.OperatorConfig{
		Namespace:      "pingcap",
		ReleaseName:    "operator",
		Image:          cfg.OperatorImage,
		Tag:            cfg.OperatorTag,
		SchedulerImage: "gcr.io/google-containers/hyperkube",
		SchedulerFeatures: []string{
			"StableScheduling=true",
		},
		LogLevel:           "2",
		WebhookServiceName: tests.WebhookServiceName,
		WebhookSecretName:  "webhook-secret",
		WebhookConfigName:  "webhook-config",
		ImagePullPolicy:    v1.PullAlways,
		TestMode:           true,
	}
}

func newTidbClusterConfig(ns, clusterName string) *tests.TidbClusterConfig {
	tidbVersion := cfg.GetTiDBVersionOrDie()
	topologyKey := "rack"
	return &tests.TidbClusterConfig{
		Namespace:        ns,
		ClusterName:      clusterName,
		OperatorTag:      cfg.OperatorTag,
		PDImage:          fmt.Sprintf("pingcap/pd:%s", tidbVersion),
		TiKVImage:        fmt.Sprintf("pingcap/tikv:%s", tidbVersion),
		TiDBImage:        fmt.Sprintf("pingcap/tidb:%s", tidbVersion),
		StorageClassName: "local-storage",
		UserName:         "root",
		Password:         "admin",
		InitSecretName:   fmt.Sprintf("%s-set-secret", clusterName),
		BackupSecretName: fmt.Sprintf("%s-backup-secret", clusterName),
		BackupName:       "backup",
		Resources: map[string]string{
			"pd.resources.limits.cpu":        "1000m",
			"pd.resources.limits.memory":     "2Gi",
			"pd.resources.requests.cpu":      "200m",
			"pd.resources.requests.memory":   "1Gi",
			"tikv.resources.limits.cpu":      "8000m",
			"tikv.resources.limits.memory":   "16Gi",
			"tikv.resources.requests.cpu":    "1000m",
			"tikv.resources.requests.memory": "2Gi",
			"tidb.resources.limits.cpu":      "8000m",
			"tidb.resources.limits.memory":   "8Gi",
			"tidb.resources.requests.cpu":    "500m",
			"tidb.resources.requests.memory": "1Gi",
			"monitor.persistent":             "true",
			"discovery.image":                cfg.OperatorImage,
			"tikv.defaultcfBlockCacheSize":   "8GB",
			"tikv.writecfBlockCacheSize":     "2GB",
		},
		Args: map[string]string{
			"binlog.drainer.workerCount": "1024",
			"binlog.drainer.txnBatch":    "512",
		},
		Monitor:          true,
		BlockWriteConfig: cfg.BlockWriter,
		TopologyKey:      topologyKey,
	}
}
