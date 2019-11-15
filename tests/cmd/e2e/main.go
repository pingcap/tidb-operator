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
	"sync"

	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/apiserver"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"golang.org/x/mod/semver"
	v1 "k8s.io/api/core/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/component-base/logs"
	glog "k8s.io/klog"
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
	certCtx, err = apimachinery.SetupServerCert("tidb-operator-e2e", tests.WebhookServiceName)
	if err != nil {
		panic(err)
	}
	go tests.StartValidatingAdmissionWebhookServerOrDie(certCtx)

	restConfig := client.GetConfigOrDie()
	cli, kubeCli := client.NewCliOrDie()
	ocfg := newOperatorConfig()

	cluster1 := newTidbClusterConfig(ns, "cluster1", "", "")
	cluster2 := newTidbClusterConfig(ns, "cluster2", "admin", "")
	cluster2.Resources["pd.replicas"] = "1"
	// TLS only works with PD >= v3.0.5
	if semver.Compare(cfg.GetTiDBVersionOrDie(), "v3.0.5") >= 0 {
		cluster2.Resources["enableTLSCluster"] = "true"
	}
	cluster3 := newTidbClusterConfig(ns, "cluster3", "admin", "")
	cluster4 := newTidbClusterConfig(ns, "cluster4", "admin", "")
	cluster5 := newTidbClusterConfig(ns, "cluster5", "", "v2.1.16") // for v2.1.x series
	cluster5.Resources["tikv.resources.limits.storage"] = "1G"

	oa := tests.NewOperatorActions(cli, kubeCli, tests.DefaultPollInterval, cfg, nil)
	oa.CleanCRDOrDie()
	oa.InstallCRDOrDie()
	oa.LabelNodesOrDie()
	oa.CleanOperatorOrDie(ocfg)
	oa.DeployOperatorOrDie(ocfg)

	/**
	 * This test case covers basic operators of a single cluster.
	 * - deployment
	 * - scaling
	 * - check reclaim pv success
	 * - update configuration
	 */
	testBasic := func(wg *sync.WaitGroup, cluster *tests.TidbClusterConfig) {
		oa.CleanTidbClusterOrDie(cluster)

		// support reclaim pv when scale in tikv or pd component
		cluster1.EnablePVReclaim = true
		oa.DeployTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)
		oa.CheckDisasterToleranceOrDie(cluster)

		// scale
		cluster.ScaleTiDB(3).ScaleTiKV(5).ScalePD(5)
		oa.ScaleTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)
		oa.CheckDisasterToleranceOrDie(cluster)

		cluster.ScaleTiDB(2).ScaleTiKV(4).ScalePD(3)
		oa.ScaleTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)
		oa.CheckDisasterToleranceOrDie(cluster)

		// configuration change
		cluster.EnableConfigMapRollout = true
		// moved to stability test because these two cases need too many times
		// bad conf
		//cluster.TiDBPreStartScript = strconv.Quote("exit 1")
		//cluster.TiKVPreStartScript = strconv.Quote("exit 1")
		//cluster.PDPreStartScript = strconv.Quote("exit 1")
		//oa.UpgradeTidbClusterOrDie(cluster)
		//time.Sleep(30 * time.Second)
		//oa.CheckTidbClustersAvailableOrDie([]*tests.TidbClusterConfig{cluster})
		// rollback conf
		//cluster.PDPreStartScript = strconv.Quote("")
		//cluster.TiKVPreStartScript = strconv.Quote("")
		//cluster.TiDBPreStartScript = strconv.Quote("")
		//oa.UpgradeTidbClusterOrDie(cluster)
		//oa.CheckTidbClusterStatusOrDie(cluster)
		// correct conf
		cluster.UpdatePdMaxReplicas(cfg.PDMaxReplicas).
			UpdateTiKVGrpcConcurrency(cfg.TiKVGrpcConcurrency).
			UpdateTiDBTokenLimit(cfg.TiDBTokenLimit)
		oa.UpgradeTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)

	}

	/**
	 * This test case covers upgrading TiDB version.
	 */
	testUpgrade := func(wg *sync.WaitGroup, cluster *tests.TidbClusterConfig) {
		// deploy
		oa.CleanTidbClusterOrDie(cluster)
		oa.DeployTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)
		oa.CheckDisasterToleranceOrDie(cluster)

		cluster.ScalePD(3)
		oa.ScaleTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)

		// upgrade
		oa.RegisterWebHookAndServiceOrDie(certCtx, ocfg)
		ctx, cancel := context.WithCancel(context.Background())
		assignedNodes := oa.GetTidbMemberAssignedNodesOrDie(cluster)
		cluster.UpgradeAll(upgradeVersions[0])
		oa.UpgradeTidbClusterOrDie(cluster)
		oa.CheckUpgradeOrDie(ctx, cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)
		oa.CheckTidbMemberAssignedNodesOrDie(cluster, assignedNodes)
		cancel()

		oa.CleanWebHookAndServiceOrDie(ocfg)
	}

	/**
	 * This test case covers backup and restore between two clusters.
	 */
	testBackupAndRestore := func(wg *sync.WaitGroup, clusterA, clusterB *tests.TidbClusterConfig) {
		oa.CleanTidbClusterOrDie(clusterA)
		oa.CleanTidbClusterOrDie(clusterB)
		oa.DeployTidbClusterOrDie(clusterA)
		oa.DeployTidbClusterOrDie(clusterB)
		oa.CheckTidbClusterStatusOrDie(clusterA)
		oa.CheckTidbClusterStatusOrDie(clusterB)
		oa.CheckDisasterToleranceOrDie(clusterA)
		oa.CheckDisasterToleranceOrDie(clusterB)

		go oa.BeginInsertDataToOrDie(clusterA)

		// backup and restore
		oa.BackupRestoreOrDie(clusterA, clusterB)

		oa.StopInsertDataTo(clusterA)
	}

	/**
	 * This test case switches back and forth between pod network and host network of a single cluster.
	 * Note that only one cluster can run in host network mode at the same time.
	 */
	testHostNetwork := func(wg *sync.WaitGroup, cluster *tests.TidbClusterConfig) {
		serverVersion, err := kubeCli.Discovery().ServerVersion()
		if err != nil {
			panic(err)
		}
		sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
		glog.Infof("ServerVersion: %v", serverVersion.String())
		if sv.LessThan(utilversion.MustParseSemantic("v1.13.11")) || // < v1.13.11
			(sv.AtLeast(utilversion.MustParseSemantic("v1.14.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.14.7"))) || // >= v1.14.0 but < v1.14.7
			(sv.AtLeast(utilversion.MustParseSemantic("v1.15.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.15.4"))) { // >= v1.15.0 but < v1.15.4
			// https://github.com/pingcap/tidb-operator/issues/1042#issuecomment-547742565
			glog.Infof("Skipping HostNetwork test. Kubernetes %v has a bug that StatefulSet may apply revision incorrectly, HostNetwork cannot work well in this cluster", serverVersion)
			return
		}

		// switch to host network
		cluster.RunInHost(true)
		oa.UpgradeTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)

		// switch to pod network
		cluster.RunInHost(false)
		oa.UpgradeTidbClusterOrDie(cluster)
		oa.CheckTidbClusterStatusOrDie(cluster)
	}

	/**
	 * This test case covers the aggregated apiserver framework
	 */
	testAggregatedApiserver := func() {
		aaCtx := apiserver.NewE2eContext("aa", restConfig, cfg.TestApiserverImage)
		aaCtx.Do()
	}

	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		testAggregatedApiserver()
	}()
	go func() {
		defer wg.Done()
		testBasic(&wg, cluster1)
		testHostNetwork(&wg, cluster1)
		oa.CleanTidbClusterOrDie(cluster1)
	}()
	go func() {
		defer wg.Done()
		testBasic(&wg, cluster5)
		oa.CleanTidbClusterOrDie(cluster5)
	}()
	go func() {
		defer wg.Done()
		testUpgrade(&wg, cluster2)
		oa.CleanTidbClusterOrDie(cluster2)
	}()
	go func() {
		defer wg.Done()
		testBackupAndRestore(&wg, cluster3, cluster4)
		oa.CleanTidbClusterOrDie(cluster3)
		oa.CleanTidbClusterOrDie(cluster4)
	}()
	wg.Wait()

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

func newTidbClusterConfig(ns, clusterName, password, tidbVersion string) *tests.TidbClusterConfig {
	if tidbVersion == "" {
		tidbVersion = cfg.GetTiDBVersionOrDie()
	}
	topologyKey := "rack"
	return &tests.TidbClusterConfig{
		Namespace:        ns,
		ClusterName:      clusterName,
		EnablePVReclaim:  false,
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
		ClusterVersion:         tidbVersion,
	}
}
