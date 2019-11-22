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
// limitations under the License.package spec

package tidbcluster

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"strconv"

	"github.com/onsi/ginkgo"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/apiserver"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"golang.org/x/mod/semver"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = ginkgo.Describe("[tidb-operator] TiDBCluster", func() {
	f := framework.NewDefaultFramework("tidb-cluster")

	var ns string
	var c clientset.Interface
	var oa tests.OperatorActions
	var cfg *tests.Config
	var config *restclient.Config
	var ocfg *tests.OperatorConfig

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		config, err := framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err := versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		asCli, err := asclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		oa = tests.NewOperatorActions(cli, c, asCli, tests.DefaultPollInterval, e2econfig.TestConfig, nil)
		cfg = e2econfig.TestConfig
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
	})

	ginkgo.Context("Basic: Deploying, Scaling, Update Configuration", func() {
		clusters := map[string]string{}
		clusters["v3.0.5"] = "cluster1"
		clusters["v2.1.16"] = "cluster5" // for v2.1.x series

		for version, name := range clusters {
			localVersion := version
			localName := name
			ginkgo.It(fmt.Sprintf("[TiDB Version: %s] %s", localVersion, localName), func() {
				cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, localName, "", localVersion)
				if name == "cluster5" {
					// verify v2.1.x configuration compatibility
					// https://github.com/pingcap/tidb-operator/pull/950
					cluster.Resources["tikv.resources.limits.storage"] = "1G"
				}

				// support reclaim pv when scale in tikv or pd component
				cluster.EnablePVReclaim = true
				oa.DeployTidbClusterOrDie(&cluster)
				oa.CheckTidbClusterStatusOrDie(&cluster)
				oa.CheckDisasterToleranceOrDie(&cluster)
				oa.CheckInitSQLOrDie(&cluster)

				// scale
				cluster.ScaleTiDB(3).ScaleTiKV(5).ScalePD(5)
				oa.ScaleTidbClusterOrDie(&cluster)
				oa.CheckTidbClusterStatusOrDie(&cluster)
				oa.CheckDisasterToleranceOrDie(&cluster)

				cluster.ScaleTiDB(2).ScaleTiKV(4).ScalePD(3)
				oa.ScaleTidbClusterOrDie(&cluster)
				oa.CheckTidbClusterStatusOrDie(&cluster)
				oa.CheckDisasterToleranceOrDie(&cluster)

				// configuration change
				cluster.EnableConfigMapRollout = true
				cluster.UpdatePdMaxReplicas(cfg.PDMaxReplicas).
					UpdateTiKVGrpcConcurrency(cfg.TiKVGrpcConcurrency).
					UpdateTiDBTokenLimit(cfg.TiDBTokenLimit)
				oa.UpgradeTidbClusterOrDie(&cluster)
				oa.CheckTidbClusterStatusOrDie(&cluster)
			})
		}
	})

	/**
	 * This test case switches back and forth between pod network and host network of a single cluster.
	 * Note that only one cluster can run in host network mode at the same time.
	 */
	ginkgo.It("Switching back and forth between pod network and host network", func() {
		// TODO do not skip this if AdvancedStatefulSet feature is enabled
		serverVersion, err := c.Discovery().ServerVersion()
		if err != nil {
			panic(err)
		}
		sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
		klog.Infof("ServerVersion: %v", serverVersion.String())
		if sv.LessThan(utilversion.MustParseSemantic("v1.13.11")) || // < v1.13.11
			(sv.AtLeast(utilversion.MustParseSemantic("v1.14.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.14.7"))) || // >= v1.14.0 but < v1.14.7
			(sv.AtLeast(utilversion.MustParseSemantic("v1.15.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.15.4"))) { // >= v1.15.0 but < v1.15.4
			// https://github.com/pingcap/tidb-operator/issues/1042#issuecomment-547742565
			framework.Skipf("Skipping HostNetwork test. Kubernetes %v has a bug that StatefulSet may apply revision incorrectly, HostNetwork cannot work well in this cluster", serverVersion)
		}

		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster1", "", "")
		oa.DeployTidbClusterOrDie(&cluster)

		// switch to host network
		cluster.RunInHost(true)
		oa.UpgradeTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		// switch to pod network
		cluster.RunInHost(false)
		oa.UpgradeTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)
	})

	ginkgo.It("Upgrading TiDB Cluster", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster", "admin", "")
		cluster.Resources["pd.replicas"] = "1"
		// TLS only works with PD >= v3.0.5
		if semver.Compare(cfg.GetTiDBVersionOrDie(), "v3.0.5") >= 0 {
			cluster.Resources["enableTLSCluster"] = "true"
		}
		// deploy
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)
		oa.CheckDisasterToleranceOrDie(&cluster)

		cluster.ScalePD(3)
		oa.ScaleTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		// upgrade
		upgradeVersions := cfg.GetUpgradeTidbVersionsOrDie()
		certCtx, err := apimachinery.SetupServerCert("tidb-operator-e2e", tests.WebhookServiceName)
		if err != nil {
			panic(err)
		}
		go tests.StartValidatingAdmissionWebhookServerOrDie(certCtx, fmt.Sprintf("%s/%s", cluster.Namespace, cluster.ClusterName))
		oa.RegisterWebHookAndServiceOrDie(certCtx, ocfg)
		ctx, cancel := context.WithCancel(context.Background())
		assignedNodes := oa.GetTidbMemberAssignedNodesOrDie(&cluster)
		cluster.UpgradeAll(upgradeVersions[0])
		oa.UpgradeTidbClusterOrDie(&cluster)
		oa.CheckUpgradeOrDie(ctx, &cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)
		oa.CheckTidbMemberAssignedNodesOrDie(&cluster, assignedNodes)
		cancel()

		oa.CleanWebHookAndServiceOrDie(ocfg)
	})

	ginkgo.It("Backup and restore TiDB Cluster", func() {
		clusterA := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster3", "admin", "")
		clusterB := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster4", "admin", "")
		oa.DeployTidbClusterOrDie(&clusterA)
		oa.DeployTidbClusterOrDie(&clusterB)
		oa.CheckTidbClusterStatusOrDie(&clusterA)
		oa.CheckTidbClusterStatusOrDie(&clusterB)
		oa.CheckDisasterToleranceOrDie(&clusterA)
		oa.CheckDisasterToleranceOrDie(&clusterB)

		go oa.BeginInsertDataToOrDie(&clusterA)

		// backup and restore
		oa.BackupRestoreOrDie(&clusterA, &clusterB)

		oa.StopInsertDataTo(&clusterA)
	})

	ginkgo.It("Test aggregated apiserver", func() {
		ginkgo.By(fmt.Sprintf("Starting to test apiserver, test apiserver image: %s", cfg.TestApiserverImage))
		framework.Logf("config: %v", config)
		aaCtx := apiserver.NewE2eContext("aa", config, cfg.TestApiserverImage)
		aaCtx.Do()
	})

})

func newTidbClusterConfig(cfg *tests.Config, ns, clusterName, password, tidbVersion string) tests.TidbClusterConfig {
	if tidbVersion == "" {
		tidbVersion = cfg.GetTiDBVersionOrDie()
	}
	topologyKey := "rack"
	return tests.TidbClusterConfig{
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
		ClusterVersion:         tidbVersion,
	}
}
