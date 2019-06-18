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
// limitations under the License.package spec

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"github.com/pingcap/tidb-operator/tests/slack"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
)

var successCount int
var cfg *tests.Config
var context *apimachinery.CertContext
var upgradeVersions []string

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	go func() {
		glog.Info(http.ListenAndServe(":6060", nil))
	}()
	cfg = tests.ParseConfigOrDie()
	upgradeVersions = cfg.GetUpgradeTidbVersionsOrDie()
	ns := os.Getenv("NAMESPACE")

	var err error
	context, err = apimachinery.SetupServerCert(ns, tests.WebhookServiceName)
	if err != nil {
		panic(err)
	}
	go tests.StartValidatingAdmissionWebhookServerOrDie(context)

	c := cron.New()
	c.AddFunc("0 0 10 * * *", func() {
		slack.NotifyAndCompletedf("Succeed %d times in the past 24 hours.", successCount)
		successCount = 0
	})
	go c.Start()

	wait.Forever(run, 5*time.Minute)
}

func run() {
	cli, kubeCli := client.NewCliOrDie()

	ocfg := newOperatorConfig()

	cluster1 := newTidbClusterConfig("ns1", "cluster1")
	cluster2 := newTidbClusterConfig("ns2", "cluster2")
	cluster3 := newTidbClusterConfig("ns1", "cluster3")

	restoreCluster1 := newTidbClusterConfig("ns1", "restore1")
	restoreCluster2 := newTidbClusterConfig("ns2", "restore2")

	onePDCluster1 := newTidbClusterConfig("ns1", "one-pd-cluster-1")
	onePDCluster2 := newTidbClusterConfig("ns2", "one-pd-cluster-2")
	onePDCluster1.Resources["pd.replicas"] = "1"
	onePDCluster2.Resources["pd.replicas"] = "1"

	allClusters := []*tests.TidbClusterConfig{
		cluster1,
		cluster2,
		cluster3,
		restoreCluster1,
		restoreCluster2,
		onePDCluster1,
		onePDCluster2,
	}
	deployedClusers := make([]*tests.TidbClusterConfig, 0)

	fta := tests.NewFaultTriggerAction(cli, kubeCli, cfg)

	oa := tests.NewOperatorActions(cli, kubeCli, tests.DefaultPollInterval, cfg, allClusters)
	oa.LabelNodesOrDie()

	go wait.Forever(oa.EventWorker, 10*time.Second)

	oa.CleanOperatorOrDie(ocfg)
	oa.DeployOperatorOrDie(ocfg)

	for _, cluster := range allClusters {
		oa.CleanTidbClusterOrDie(cluster)
	}

	caseFn := func(clusters []*tests.TidbClusterConfig, restoreCluster *tests.TidbClusterConfig, upgradeVersion string) {
		// check env
		fta.CheckAndRecoverEnvOrDie()
		oa.CheckK8sAvailableOrDie(nil, nil)

		// deploy
		for _, cluster := range clusters {
			oa.DeployTidbClusterOrDie(cluster)
			deployedClusers = append(deployedClusers, cluster)
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
		oa.RegisterWebHookAndServiceOrDie(context, ocfg)
		for idx, cluster := range clusters {
			assignedNodes := oa.GetTidbMemberAssignedNodesOrDie(cluster)
			cluster.UpgradeAll(upgradeVersion)
			oa.UpgradeTidbClusterOrDie(cluster)
			if idx == 0 {
				oa.CheckManualPauseTiDBOrDie(cluster)
			}
			oa.CheckTidbClusterStatusOrDie(cluster)
			oa.CheckTidbMemberAssignedNodesOrDie(cluster, assignedNodes)
		}

		// configuration change
		for _, cluster := range clusters {
			cluster.EnableConfigMapRollout = true

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
			oa.CheckTidbClusterStatusOrDie(cluster)

			cluster.UpdatePdMaxReplicas(cfg.PDMaxReplicas).
				UpdateTiKVGrpcConcurrency(cfg.TiKVGrpcConcurrency).
				UpdateTiDBTokenLimit(cfg.TiDBTokenLimit)
			oa.UpgradeTidbClusterOrDie(cluster)
			oa.CheckTidbClusterStatusOrDie(cluster)
		}
		oa.CleanWebHookAndServiceOrDie(ocfg)

		for _, cluster := range clusters {
			oa.CheckDataRegionDisasterToleranceOrDie(cluster)
		}

		// backup and restore
		oa.DeployTidbClusterOrDie(restoreCluster)
		deployedClusers = append(deployedClusers, restoreCluster)
		oa.CheckTidbClusterStatusOrDie(restoreCluster)
		oa.BackupRestoreOrDie(clusters[0], restoreCluster)

		// delete operator
		oa.CleanOperatorOrDie(ocfg)
		oa.CheckOperatorDownOrDie(deployedClusers)
		oa.DeployOperatorOrDie(ocfg)

		// stop node
		physicalNode, node, faultTime := fta.StopNodeOrDie()
		oa.EmitEvent(nil, fmt.Sprintf("StopNode: %s on %s", node, physicalNode))
		oa.CheckFailoverPendingOrDie(deployedClusers, node, &faultTime)
		oa.CheckFailoverOrDie(deployedClusers, node)
		time.Sleep(3 * time.Minute)
		fta.StartNodeOrDie(physicalNode, node)
		oa.EmitEvent(nil, fmt.Sprintf("StartNode: %s on %s", node, physicalNode))
		oa.CheckRecoverOrDie(deployedClusers)
		for _, cluster := range deployedClusers {
			oa.CheckTidbClusterStatusOrDie(cluster)
		}

		// truncate tikv sst file
		oa.TruncateSSTFileThenCheckFailoverOrDie(clusters[0], 5*time.Minute)

		// stop etcd
		faultEtcd := tests.SelectNode(cfg.ETCDs)
		fta.StopETCDOrDie(faultEtcd)
		defer fta.StartETCDOrDie(faultEtcd)
		time.Sleep(3 * time.Minute)
		oa.CheckOneEtcdDownOrDie(ocfg, deployedClusers, faultEtcd)
		fta.StartETCDOrDie(faultEtcd)

		// stop all kube-proxy and k8s/operator/tidbcluster is available
		fta.StopKubeProxyOrDie()
		oa.CheckKubeProxyDownOrDie(ocfg, clusters)
		fta.StartKubeProxyOrDie()

		// stop all kube-scheduler pods
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StopKubeSchedulerOrDie(vNode)
			}
		}
		oa.CheckKubeSchedulerDownOrDie(ocfg, clusters)
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StartKubeSchedulerOrDie(vNode)
			}
		}

		// stop all kube-controller-manager pods
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StopKubeControllerManagerOrDie(vNode)
			}
		}
		oa.CheckKubeControllerManagerDownOrDie(ocfg, clusters)
		for _, physicalNode := range cfg.APIServers {
			for _, vNode := range physicalNode.Nodes {
				fta.StartKubeControllerManagerOrDie(vNode)
			}
		}
	}

	// before operator upgrade
	preUpgrade := []*tests.TidbClusterConfig{
		cluster1,
		cluster2,
		onePDCluster1,
	}
	caseFn(preUpgrade, restoreCluster1, upgradeVersions[0])

	// after operator upgrade
	if cfg.UpgradeOperatorImage != "" && cfg.UpgradeOperatorTag != "" {
		ocfg.Image = cfg.UpgradeOperatorImage
		ocfg.Tag = cfg.UpgradeOperatorTag
		oa.UpgradeOperatorOrDie(ocfg)
		time.Sleep(5 * time.Minute)
		postUpgrade := []*tests.TidbClusterConfig{
			cluster3,
			onePDCluster2,
			cluster1,
			cluster2,
			onePDCluster1,
		}
		var v string
		if len(upgradeVersions) == 2 {
			v = upgradeVersions[1]
		}
		// caseFn(postUpgrade, restoreCluster2, tidbUpgradeVersion)
		caseFn(postUpgrade, restoreCluster2, v)
	}

	successCount++
	glog.Infof("################## Stability test finished at: %v\n\n\n\n", time.Now().Format(time.RFC3339))
}
