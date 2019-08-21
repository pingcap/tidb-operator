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
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"sync"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/util/logs"
)

var cfg *tests.Config
var certCtx *apimachinery.CertContext
var upgradeVersions []string

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	go func() {
		glog.Info(http.ListenAndServe(":6060", nil))
	}()

	cfg = tests.ParseConfigOrDie()
	cfg.ManifestDir = "/manifests"
	upgradeVersions = cfg.GetUpgradeTidbVersionsOrDie()
	ns := "e2e"

	var err error
	certCtx, err = apimachinery.SetupServerCert(ns, tests.WebhookServiceName)
	if err != nil {
		panic(err)
	}
	go tests.StartValidatingAdmissionWebhookServerOrDie(certCtx)

	cli, kubeCli := client.NewCliOrDie()
	ocfg := newOperatorConfig()

	cluster1 := newTidbClusterConfig(ns, "cluster1", "")
	cluster2 := newTidbClusterConfig(ns, "cluster2", "admin")
	cluster2.Resources["pd.replicas"] = "1"
	cluster3 := newTidbClusterConfig(ns, "cluster3", "admin")
	cluster4 := newTidbClusterConfig(ns, "cluster4", "admin")

	allClusters := []*tests.TidbClusterConfig{
		cluster1,
		cluster2,
	}

	oa := tests.NewOperatorActions(cli, kubeCli, tests.DefaultPollInterval, cfg, nil)
	oa.CleanCRDOrDie()
	oa.InstallCRDOrDie()
	oa.LabelNodesOrDie()
	oa.CleanOperatorOrDie(ocfg)
	oa.DeployOperatorOrDie(ocfg)

	fn1 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		oa.CleanTidbClusterOrDie(cluster1)
		oa.DeployTidbClusterOrDie(cluster1)
		oa.CheckTidbClusterStatusOrDie(cluster1)
		oa.CheckDisasterToleranceOrDie(cluster1)
		oa.CheckInitSQLOrDie(cluster1)

		// scale
		cluster1.ScaleTiDB(3).ScaleTiKV(5).ScalePD(5)
		oa.ScaleTidbClusterOrDie(cluster1)
		oa.CheckTidbClusterStatusOrDie(cluster1)
		oa.CheckDisasterToleranceOrDie(cluster1)

		cluster1.ScaleTiDB(2).ScaleTiKV(4).ScalePD(3)
		oa.ScaleTidbClusterOrDie(cluster1)
		oa.CheckTidbClusterStatusOrDie(cluster1)
		oa.CheckDisasterToleranceOrDie(cluster1)

		// configuration change
		cluster1.EnableConfigMapRollout = true
		// moved to stability test because these two cases need too many times
		// bad conf
		//cluster1.TiDBPreStartScript = strconv.Quote("exit 1")
		//cluster1.TiKVPreStartScript = strconv.Quote("exit 1")
		//cluster1.PDPreStartScript = strconv.Quote("exit 1")
		//oa.UpgradeTidbClusterOrDie(cluster1)
		//time.Sleep(30 * time.Second)
		//oa.CheckTidbClustersAvailableOrDie([]*tests.TidbClusterConfig{cluster1})
		// rollback conf
		//cluster1.PDPreStartScript = strconv.Quote("")
		//cluster1.TiKVPreStartScript = strconv.Quote("")
		//cluster1.TiDBPreStartScript = strconv.Quote("")
		//oa.UpgradeTidbClusterOrDie(cluster1)
		//oa.CheckTidbClusterStatusOrDie(cluster1)
		// correct conf
		cluster1.UpdatePdMaxReplicas(cfg.PDMaxReplicas).
			UpdateTiKVGrpcConcurrency(cfg.TiKVGrpcConcurrency).
			UpdateTiDBTokenLimit(cfg.TiDBTokenLimit)
		oa.UpgradeTidbClusterOrDie(cluster1)
		oa.CheckTidbClusterStatusOrDie(cluster1)

		// switch to host network
		cluster1.RunInHost(true)
		oa.UpgradeTidbClusterOrDie(cluster1)
		oa.CheckTidbClusterStatusOrDie(cluster1)

		// switch to pod network
		cluster1.RunInHost(false)
		oa.UpgradeTidbClusterOrDie(cluster1)
		oa.CheckTidbClusterStatusOrDie(cluster1)
	}
	fn2 := func(wg *sync.WaitGroup) {
		defer wg.Done()

		// deploy
		oa.CleanTidbClusterOrDie(cluster2)
		oa.DeployTidbClusterOrDie(cluster2)
		oa.CheckTidbClusterStatusOrDie(cluster2)
		oa.CheckDisasterToleranceOrDie(cluster2)

		cluster2.ScalePD(3)
		oa.ScaleTidbClusterOrDie(cluster2)
		oa.CheckTidbClusterStatusOrDie(cluster2)

		// upgrade
		oa.RegisterWebHookAndServiceOrDie(certCtx, ocfg)
		ctx, cancel := context.WithCancel(context.Background())
		assignedNodes := oa.GetTidbMemberAssignedNodesOrDie(cluster2)
		cluster2.UpgradeAll(upgradeVersions[0])
		oa.UpgradeTidbClusterOrDie(cluster2)
		oa.CheckUpgradeOrDie(ctx, cluster2)
		oa.CheckManualPauseTiDBOrDie(cluster2)
		oa.CheckTidbClusterStatusOrDie(cluster2)
		oa.CheckTidbMemberAssignedNodesOrDie(cluster2, assignedNodes)
		cancel()

		oa.CleanWebHookAndServiceOrDie(ocfg)
	}
	fn3 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		oa.CleanTidbClusterOrDie(cluster3)
		oa.CleanTidbClusterOrDie(cluster4)
		oa.DeployTidbClusterOrDie(cluster3)
		oa.DeployTidbClusterOrDie(cluster4)
		oa.CheckTidbClusterStatusOrDie(cluster3)
		oa.CheckTidbClusterStatusOrDie(cluster4)
		go oa.BeginInsertDataToOrDie(cluster3)

		// backup and restore
		oa.BackupRestoreOrDie(cluster3, cluster4)
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go fn1(&wg)
	go fn2(&wg)
	go fn3(&wg)
	wg.Wait()

	// check data regions disaster tolerance
	for _, clusterInfo := range allClusters {
		oa.CheckDataRegionDisasterToleranceOrDie(clusterInfo)
	}

	glog.Infof("\nFinished.")
}

func newOperatorConfig() *tests.OperatorConfig {
	return &tests.OperatorConfig{
		Namespace:      "pingcap",
		ReleaseName:    "operator",
		Image:          cfg.OperatorImage,
		Tag:            cfg.OperatorTag,
		SchedulerImage: "mirantis/hypokube",
		SchedulerTag:   "final",
		SchedulerFeatures: []string{
			"StableScheduling=true",
		},
		LogLevel:           "2",
		WebhookServiceName: "webhook-service",
		WebhookSecretName:  "webhook-secret",
		WebhookConfigName:  "webhook-config",
		ImagePullPolicy:    v1.PullIfNotPresent,
		TestMode:           true,
	}
}

func newTidbClusterConfig(ns, clusterName, password string) *tests.TidbClusterConfig {
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
		Password:         password,
		UserName:         "root",
		InitSecretName:   fmt.Sprintf("%s-set-secret", clusterName),
		BackupSecretName: fmt.Sprintf("%s-backup-secret", clusterName),
		BackupName:       "backup",
		Resources: map[string]string{
			"pd.resources.limits.cpu":        "1000m",
			"pd.resources.limits.memory":     "2Gi",
			"pd.resources.requests.cpu":      "200m",
			"pd.resources.requests.memory":   "200Mi",
			"tikv.resources.limits.cpu":      "2000m",
			"tikv.resources.limits.memory":   "4Gi",
			"tikv.resources.requests.cpu":    "200m",
			"tikv.resources.requests.memory": "200Mi",
			"tidb.resources.limits.cpu":      "2000m",
			"tidb.resources.limits.memory":   "4Gi",
			"tidb.resources.requests.cpu":    "200m",
			"tidb.resources.requests.memory": "200Mi",
			"tidb.initSql":                   strconv.Quote("create database e2e;"),
			"discovery.image":                cfg.OperatorImage,
		},
		Args:    map[string]string{},
		Monitor: true,
		BlockWriteConfig: blockwriter.Config{
			TableNum:    1,
			Concurrency: 1,
			BatchSize:   1,
			RawSize:     1,
		},
		TopologyKey:            topologyKey,
		EnableConfigMapRollout: true,
	}
}
