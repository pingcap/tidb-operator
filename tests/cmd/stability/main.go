// Copyright 2018 PingCAP, Inc.
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

package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	"github.com/pingcap/tidb-operator/tests/slack"
	"github.com/robfig/cron"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/logs"
	glog "k8s.io/klog"
)

var cfg *tests.Config
var certCtx *apimachinery.CertContext
var upgradeVersions []string

func init() {
	client.RegisterFlags()
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	go func() {
		glog.Info(http.ListenAndServe(":6060", nil))
	}()
	metrics.StartServer()
	cfg = tests.ParseConfigOrDie()
	upgradeVersions = cfg.GetUpgradeTidbVersionsOrDie()
	ns := os.Getenv("NAMESPACE")

	var err error
	certCtx, err = apimachinery.SetupServerCert(ns, tests.WebhookServiceName)
	if err != nil {
		panic(err)
	}
	go tests.StartValidatingAdmissionWebhookServerOrDie(certCtx)

	c := cron.New()
	if err := c.AddFunc("0 0 10 * * *", func() {
		slack.NotifyAndCompletedf("Succeed %d times in the past 24 hours.", slack.SuccessCount)
		slack.SuccessCount = 0
	}); err != nil {
		panic(err)
	}
	go c.Start()

	wait.Forever(run, 5*time.Minute)
}

func run() {
	cli, kubeCli, asCli, aggrCli, apiExtCli := client.NewCliOrDie()

	ocfg := newOperatorConfig()

	cluster1 := newTidbClusterConfig("ns1", "cluster1")
	cluster2 := newTidbClusterConfig("ns2", "cluster2")
	cluster3 := newTidbClusterConfig("ns2", "cluster3")

	directRestoreCluster1 := newTidbClusterConfig("ns1", "restore1")
	fileRestoreCluster1 := newTidbClusterConfig("ns1", "file-restore1")
	directRestoreCluster2 := newTidbClusterConfig("ns2", "restore2")
	fileRestoreCluster2 := newTidbClusterConfig("ns2", "file-restore2")

	onePDCluster1 := newTidbClusterConfig("ns1", "one-pd-cluster-1")
	onePDCluster2 := newTidbClusterConfig("ns2", "one-pd-cluster-2")
	onePDCluster1.Resources["pd.replicas"] = "1"
	onePDCluster2.Resources["pd.replicas"] = "1"

	allClusters := []*tests.TidbClusterConfig{
		cluster1,
		cluster2,
		cluster3,
		directRestoreCluster1,
		fileRestoreCluster1,
		directRestoreCluster2,
		fileRestoreCluster2,
		onePDCluster1,
		onePDCluster2,
	}
	deployedClusters := make([]*tests.TidbClusterConfig, 0)
	addDeployedClusterFn := func(cluster *tests.TidbClusterConfig) {
		for _, tc := range deployedClusters {
			if tc.Namespace == cluster.Namespace && tc.ClusterName == cluster.ClusterName {
				return
			}
		}
		deployedClusters = append(deployedClusters, cluster)
	}

	fta := tests.NewFaultTriggerAction(cli, kubeCli, cfg)
	fta.CheckAndRecoverEnvOrDie()

	oa := tests.NewOperatorActions(cli, kubeCli, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, cfg, allClusters, nil, nil)
	oa.CheckK8sAvailableOrDie(nil, nil)
	oa.LabelNodesOrDie()

	go oa.RunEventWorker()

	oa.CleanOperatorOrDie(ocfg)
	oa.DeployOperatorOrDie(ocfg)

	for _, cluster := range allClusters {
		oa.CleanTidbClusterOrDie(cluster)
	}

	caseFn := func(clusters []*tests.TidbClusterConfig, onePDClsuter *tests.TidbClusterConfig, backupTargets []tests.BackupTarget, upgradeVersion string) {
		// check env
		fta.CheckAndRecoverEnvOrDie()
		oa.CheckK8sAvailableOrDie(nil, nil)

		// deploy and clean the one-pd-cluster
		oa.DeployTidbClusterOrDie(onePDClsuter)
		oa.CheckTidbClusterStatusOrDie(onePDClsuter)
		oa.CleanTidbClusterOrDie(onePDClsuter)

		// deploy
		for _, cluster := range clusters {
			oa.DeployTidbClusterOrDie(cluster)
			addDeployedClusterFn(cluster)
		}
		for _, cluster := range clusters {
			oa.CheckTidbClusterStatusOrDie(cluster)
			oa.CheckDisasterToleranceOrDie(cluster)
			go oa.BeginInsertDataToOrDie(cluster)
		}

		// scale out
		for _, cluster := range clusters {
			cluster.ScaleTiDB(3).ScaleTiKV(5).ScalePD(5)
			oa.ScaleTidbClusterOrDie(cluster)
		}
		for _, cluster := range clusters {
			oa.CheckTidbClusterStatusOrDie(cluster)
			oa.CheckDisasterToleranceOrDie(cluster)
		}

		// scale in
		for _, cluster := range clusters {
			cluster.ScaleTiDB(2).ScaleTiKV(3).ScalePD(3)
			oa.ScaleTidbClusterOrDie(cluster)
		}
		for _, cluster := range clusters {
			oa.CheckTidbClusterStatusOrDie(cluster)
			oa.CheckDisasterToleranceOrDie(cluster)
		}

		// upgrade
		namespace := os.Getenv("NAMESPACE")
		oa.RegisterWebHookAndServiceOrDie(ocfg.WebhookConfigName, namespace, ocfg.WebhookServiceName, certCtx)
		ctx, cancel := context.WithCancel(context.Background())
		for _, cluster := range clusters {
			assignedNodes := oa.GetTidbMemberAssignedNodesOrDie(cluster)
			cluster.UpgradeAll(upgradeVersion)
			oa.UpgradeTidbClusterOrDie(cluster)
			oa.CheckUpgradeOrDie(ctx, cluster)
			oa.CheckTidbClusterStatusOrDie(cluster)
			oa.CheckTidbMemberAssignedNodesOrDie(cluster, assignedNodes)
		}

		// configuration change
		for _, cluster := range clusters {
			// bad conf
			cluster.TiDBPreStartScript = strconv.Quote("exit 1")
			cluster.TiKVPreStartScript = strconv.Quote("exit 1")
			cluster.PDPreStartScript = strconv.Quote("exit 1")
			oa.UpgradeTidbClusterOrDie(cluster)
			time.Sleep(30 * time.Second)
			oa.CheckTidbClustersAvailableOrDie([]*tests.TidbClusterConfig{cluster})
			// rollback conf
			cluster.PDPreStartScript = strconv.Quote("")
			cluster.TiKVPreStartScript = strconv.Quote("")
			cluster.TiDBPreStartScript = strconv.Quote("")
			oa.UpgradeTidbClusterOrDie(cluster)
			// wait upgrade complete
			oa.CheckUpgradeCompleteOrDie(cluster)
			oa.CheckTidbClusterStatusOrDie(cluster)

			cluster.UpdatePdMaxReplicas(cfg.PDMaxReplicas).
				UpdateTiKVGrpcConcurrency(cfg.TiKVGrpcConcurrency).
				UpdateTiDBTokenLimit(cfg.TiDBTokenLimit)
			oa.UpgradeTidbClusterOrDie(cluster)
			// wait upgrade complete
			oa.CheckUpgradeOrDie(ctx, cluster)
			oa.CheckTidbClusterStatusOrDie(cluster)
		}
		cancel()
		oa.CleanWebHookAndServiceOrDie(ocfg.WebhookConfigName)

		for _, cluster := range clusters {
			oa.CheckDisasterToleranceOrDie(cluster)
		}

		// backup and restore
		for i := range backupTargets {
			oa.DeployTidbClusterOrDie(backupTargets[i].TargetCluster)
			addDeployedClusterFn(backupTargets[i].TargetCluster)
			oa.CheckTidbClusterStatusOrDie(backupTargets[i].TargetCluster)
		}
		oa.BackupAndRestoreToMultipleClustersOrDie(clusters[0], backupTargets)

		// delete operator
		oa.CleanOperatorOrDie(ocfg)
		oa.CheckOperatorDownOrDie(deployedClusters)
		oa.DeployOperatorOrDie(ocfg)

		// stop node
		physicalNode, node, faultTime := fta.StopNodeOrDie()
		oa.EmitEvent(nil, fmt.Sprintf("StopNode: %s on %s", node, physicalNode))
		oa.CheckFailoverPendingOrDie(deployedClusters, node, &faultTime)
		oa.CheckFailoverOrDie(deployedClusters, node)
		time.Sleep(3 * time.Minute)
		fta.StartNodeOrDie(physicalNode, node)
		oa.EmitEvent(nil, fmt.Sprintf("StartNode: %s on %s", node, physicalNode))
		oa.CheckRecoverOrDie(deployedClusters)
		for _, cluster := range deployedClusters {
			oa.CheckTidbClusterStatusOrDie(cluster)
		}

		// truncate tikv sst file
		oa.TruncateSSTFileThenCheckFailoverOrDie(clusters[0], 5*time.Minute)

		// delete pd data
		oa.DeletePDDataThenCheckFailoverOrDie(clusters[0], 5*time.Minute)

		// stop one etcd
		faultEtcd := tests.SelectNode(cfg.ETCDs)
		fta.StopETCDOrDie(faultEtcd)
		defer fta.StartETCDOrDie(faultEtcd)
		time.Sleep(3 * time.Minute)
		oa.CheckEtcdDownOrDie(ocfg, deployedClusters, faultEtcd)
		fta.StartETCDOrDie(faultEtcd)

		// stop all etcds
		fta.StopETCDOrDie()
		time.Sleep(10 * time.Minute)
		fta.StartETCDOrDie()
		oa.CheckEtcdDownOrDie(ocfg, deployedClusters, "")

		// stop all kubelets
		fta.StopKubeletOrDie()
		time.Sleep(10 * time.Minute)
		fta.StartKubeletOrDie()
		oa.CheckKubeletDownOrDie(ocfg, deployedClusters, "")

		// stop all kube-proxy and k8s/operator/tidbcluster is available
		fta.StopKubeProxyOrDie()
		oa.CheckKubeProxyDownOrDie(ocfg, clusters)
		fta.StartKubeProxyOrDie()

		// stop all kube-scheduler pods
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StopKubeSchedulerOrDie(vNode.IP)
			}
		}
		oa.CheckKubeSchedulerDownOrDie(ocfg, clusters)
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StartKubeSchedulerOrDie(vNode.IP)
			}
		}

		// stop all kube-controller-manager pods
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StopKubeControllerManagerOrDie(vNode.IP)
			}
		}
		oa.CheckKubeControllerManagerDownOrDie(ocfg, clusters)
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StartKubeControllerManagerOrDie(vNode.IP)
			}
		}

		// stop one kube-apiserver pod
		faultApiServer := tests.SelectNode(cfg.APIServers)
		fta.StopKubeAPIServerOrDie(faultApiServer)
		defer fta.StartKubeAPIServerOrDie(faultApiServer)
		time.Sleep(3 * time.Minute)
		oa.CheckOneApiserverDownOrDie(ocfg, clusters, faultApiServer)
		fta.StartKubeAPIServerOrDie(faultApiServer)

		time.Sleep(time.Minute)
		// stop all kube-apiserver pods
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StopKubeAPIServerOrDie(vNode.IP)
			}
		}
		oa.CheckAllApiserverDownOrDie(ocfg, clusters)
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StartKubeAPIServerOrDie(vNode.IP)
			}
		}
		time.Sleep(time.Minute)
	}

	// before operator upgrade
	preUpgrade := []*tests.TidbClusterConfig{
		cluster1,
		cluster2,
	}
	backupTargets := []tests.BackupTarget{
		{
			TargetCluster:   directRestoreCluster1,
			IsAdditional:    false,
			IncrementalType: tests.DbTypeTiDB,
		},
	}
	if ocfg.Tag != "v1.0.0" {
		backupTargets = append(backupTargets, tests.BackupTarget{
			TargetCluster:   fileRestoreCluster1,
			IsAdditional:    true,
			IncrementalType: tests.DbTypeFile,
		})
	}
	caseFn(preUpgrade, onePDCluster1, backupTargets, upgradeVersions[0])

	// after operator upgrade
	if cfg.UpgradeOperatorImage != "" && cfg.UpgradeOperatorTag != "" {
		ocfg.Image = cfg.UpgradeOperatorImage
		ocfg.Tag = cfg.UpgradeOperatorTag
		oa.UpgradeOperatorOrDie(ocfg)
		postUpgrade := []*tests.TidbClusterConfig{
			cluster3,
			cluster1,
			cluster2,
		}
		v := upgradeVersions[0]
		if len(upgradeVersions) == 2 {
			v = upgradeVersions[1]
		}
		postUpgradeBackupTargets := []tests.BackupTarget{
			{
				TargetCluster:   directRestoreCluster2,
				IsAdditional:    false,
				IncrementalType: tests.DbTypeTiDB,
			},
		}

		if ocfg.Tag != "v1.0.0" {
			postUpgradeBackupTargets = append(postUpgradeBackupTargets, tests.BackupTarget{
				TargetCluster:   fileRestoreCluster2,
				IsAdditional:    true,
				IncrementalType: tests.DbTypeFile,
			})
		}
		// caseFn(postUpgrade, restoreCluster2, tidbUpgradeVersion)
		caseFn(postUpgrade, onePDCluster2, postUpgradeBackupTargets, v)
	}

	for _, cluster := range allClusters {
		oa.StopInsertDataTo(cluster)
	}

	slack.SuccessCount++
	glog.Infof("################## Stability test finished at: %v\n\n\n\n", time.Now().Format(time.RFC3339))
}

func newOperatorConfig() *tests.OperatorConfig {
	return &tests.OperatorConfig{
		Namespace:                 "pingcap",
		ReleaseName:               "operator",
		Image:                     cfg.OperatorImage,
		Tag:                       cfg.OperatorTag,
		ControllerManagerReplicas: tests.IntPtr(2),
		SchedulerImage:            "gcr.io/google-containers/hyperkube",
		SchedulerReplicas:         tests.IntPtr(2),
		Features: []string{
			"StableScheduling=true",
		},
		LogLevel:           "2",
		WebhookServiceName: tests.WebhookServiceName,
		WebhookSecretName:  "webhook-secret",
		WebhookConfigName:  "webhook-config",
		ImagePullPolicy:    v1.PullAlways,
		TestMode:           true,
		WebhookEnabled:     false,
		PodWebhookEnabled:  false,
		StsWebhookEnabled:  false,
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
		Monitor:                true,
		BlockWriteConfig:       cfg.BlockWriter,
		TopologyKey:            topologyKey,
		ClusterVersion:         tidbVersion,
		EnableConfigMapRollout: true,
	}
}
