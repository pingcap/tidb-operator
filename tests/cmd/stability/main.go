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
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	"github.com/pingcap/tidb-operator/tests/slack"
	"github.com/robfig/cron"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
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
		klog.Info(http.ListenAndServe(":6060", nil))
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
	onePDCluster1.Clustrer.Spec.PD.Replicas = 1
	onePDCluster2.Clustrer.Spec.PD.Replicas = 1

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

	crdUtil := tests.NewCrdTestUtil(cli, kubeCli, asCli, kubeCli.AppsV1())
	klog.Infof(fmt.Sprintf("allclusters: %v", allClusters))
	crdUtil.CleanResourcesOrDie("tc", "ns1")
	crdUtil.CleanResourcesOrDie("tc", "ns2")
	crdUtil.CleanResourcesOrDie("pvc", "ns1")
	crdUtil.CleanResourcesOrDie("pvc", "ns2")
	crdUtil.CleanResourcesOrDie("secret", "ns1")
	crdUtil.CleanResourcesOrDie("secret", "ns2")
	crdUtil.CleanResourcesOrDie("pod", "ns1")
	crdUtil.CleanResourcesOrDie("pod", "ns2")

	caseFn := func(clusters []*tests.TidbClusterConfig, onePDClsuter *tests.TidbClusterConfig, backupTargets []tests.BackupTarget, upgradeVersion string) {
		// check env
		fta.CheckAndRecoverEnvOrDie()
		oa.CheckK8sAvailableOrDie(nil, nil)

		//deploy and clean the one-pd-cluster
		onePDTC := onePDClsuter.Clustrer
		crdUtil.CreateTidbClusterOrDie(onePDTC)
		crdUtil.WaitTidbClusterReadyOrDie(onePDTC, 60*time.Minute)
		crdUtil.DeleteTidbClusterOrDie(onePDTC)

		// deploy
		for _, cluster := range clusters {
			tc := cluster.Clustrer
			crdUtil.CreateTidbClusterOrDie(tc)
			secret := buildSecret(cluster)
			crdUtil.CreateSecretOrDie(secret)
			addDeployedClusterFn(cluster)
		}
		for _, cluster := range clusters {
			tc := cluster.Clustrer
			crdUtil.WaitTidbClusterReadyOrDie(tc, 60*time.Minute)
			crdUtil.CheckDisasterToleranceOrDie(tc)
			oa.BeginInsertDataToOrDie(cluster)
		}
		klog.Infof("clusters deployed and checked")
		slack.NotifyAndCompletedf("clusters deployed and checked, ready to run stability test")

		// upgrade
		namespace := os.Getenv("NAMESPACE")
		oa.RegisterWebHookAndServiceOrDie(ocfg.WebhookConfigName, namespace, ocfg.WebhookServiceName, certCtx)
		for _, cluster := range clusters {
			cluster.Clustrer.Spec.Version = upgradeVersion
			crdUtil.UpdateTidbClusterOrDie(cluster.Clustrer)
			crdUtil.WaitTidbClusterReadyOrDie(cluster.Clustrer, 60*time.Minute)
		}
		klog.Infof("clusters upgraded in checked")

		// configuration change
		for _, cluster := range clusters {
			cluster.Clustrer.Spec.PD.Replicas = int32(cfg.PDMaxReplicas)
			cluster.Clustrer.Spec.TiKV.Config.Set("server.grpc-concurrency", cfg.TiKVGrpcConcurrency)
			cluster.Clustrer.Spec.TiDB.Config.Set("token-limit", cfg.TiDBTokenLimit)

			crdUtil.UpdateTidbClusterOrDie(cluster.Clustrer)
			crdUtil.WaitTidbClusterReadyOrDie(cluster.Clustrer, 60*time.Minute)
		}
		oa.CleanWebHookAndServiceOrDie(ocfg.WebhookConfigName)
		klog.Infof("clusters configurations updated in checked")

		for _, cluster := range clusters {
			crdUtil.CheckDisasterToleranceOrDie(cluster.Clustrer)
		}
		klog.Infof("clusters DisasterTolerance checked")

		//stop node
		physicalNode, node, faultTime := fta.StopNodeOrDie()
		oa.EmitEvent(nil, fmt.Sprintf("StopNode: %s on %s", node, physicalNode))
		oa.CheckFailoverPendingOrDie(deployedClusters, node, &faultTime)
		oa.CheckFailoverOrDie(deployedClusters, node)
		time.Sleep(3 * time.Minute)
		fta.StartNodeOrDie(physicalNode, node)
		oa.EmitEvent(nil, fmt.Sprintf("StartNode: %s on %s", node, physicalNode))
		oa.WaitPodOnNodeReadyOrDie(deployedClusters, node)
		oa.CheckRecoverOrDie(deployedClusters)
		for _, cluster := range deployedClusters {
			crdUtil.WaitTidbClusterReadyOrDie(cluster.Clustrer, 30*time.Minute)
		}
		klog.Infof("clusters node stopped and restarted checked")
		slack.NotifyAndCompletedf("stability test: clusters node stopped and restarted checked")

		// truncate tikv sst file
		oa.TruncateSSTFileThenCheckFailoverOrDie(clusters[0], 5*time.Minute)
		klog.Infof("clusters truncate sst file and checked failover")
		slack.NotifyAndCompletedf("stability test: clusters truncate sst file and checked failover")

		// delete pd data
		oa.DeletePDDataThenCheckFailoverOrDie(clusters[0], 5*time.Minute)
		klog.Infof("cluster[%s/%s] DeletePDDataThenCheckFailoverOrDie success", clusters[0].Namespace, clusters[0].ClusterName)
		slack.NotifyAndCompletedf("stability test: DeletePDDataThenCheckFailoverOrDie success")

		// stop one etcd
		faultEtcd := tests.SelectNode(cfg.ETCDs)
		fta.StopETCDOrDie(faultEtcd)
		defer fta.StartETCDOrDie(faultEtcd)
		time.Sleep(3 * time.Minute)
		oa.CheckEtcdDownOrDie(ocfg, deployedClusters, faultEtcd)
		fta.StartETCDOrDie(faultEtcd)
		klog.Infof("clusters stop on etcd and restart")

		// stop all etcds
		fta.StopETCDOrDie()
		time.Sleep(10 * time.Minute)
		fta.StartETCDOrDie()
		oa.CheckEtcdDownOrDie(ocfg, deployedClusters, "")
		klog.Infof("clusters stop all etcd and restart")

		// stop all kubelets
		fta.StopKubeletOrDie()
		time.Sleep(10 * time.Minute)
		fta.StartKubeletOrDie()
		oa.CheckKubeletDownOrDie(ocfg, deployedClusters, "")
		klog.Infof("clusters stop all kubelets and restart")

		// stop all kube-proxy and k8s/operator/tidbcluster is available
		fta.StopKubeProxyOrDie()
		oa.CheckKubeProxyDownOrDie(ocfg, clusters)
		fta.StartKubeProxyOrDie()
		klog.Infof("clusters stop all kube-proxy and restart")

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
		klog.Infof("clusters stop all kube-scheduler and restart")

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
		klog.Infof("clusters stop all kube-controller and restart")

		// stop one kube-apiserver pod
		faultApiServer := tests.SelectNode(cfg.APIServers)
		klog.Infof("fault ApiServer Node name = %s", faultApiServer)
		fta.StopKubeAPIServerOrDie(faultApiServer)
		defer fta.StartKubeAPIServerOrDie(faultApiServer)
		time.Sleep(3 * time.Minute)
		oa.CheckOneApiserverDownOrDie(ocfg, clusters, faultApiServer)
		fta.StartKubeAPIServerOrDie(faultApiServer)
		klog.Infof("clusters stop one kube-apiserver and restart")

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
		klog.Infof("clusters stop all kube-apiserver and restart")
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
	slack.NotifyAndCompletedf("Succeed stability onetime")
	klog.Infof("################## Stability test finished at: %v\n\n\n\n", time.Now().Format(time.RFC3339))
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
		WebhookEnabled:     true,
		PodWebhookEnabled:  false,
		StsWebhookEnabled:  true,
	}
}

func newTidbClusterConfig(ns, clusterName string) *tests.TidbClusterConfig {
	tidbVersion := cfg.GetTiDBVersionOrDie()
	topologyKey := "rack"
	tc := fixture.GetTidbCluster(ns, clusterName, tidbVersion)
	tc.Spec.PD.StorageClassName = pointer.StringPtr("local-storage")
	tc.Spec.TiKV.StorageClassName = pointer.StringPtr("local-storage")
	tc.Spec.ConfigUpdateStrategy = v1alpha1.ConfigUpdateStrategyRollingUpdate
	return &tests.TidbClusterConfig{
		Namespace:        ns,
		ClusterName:      clusterName,
		OperatorTag:      cfg.OperatorTag,
		PDImage:          fmt.Sprintf("pingcap/pd:%s", tidbVersion),
		TiKVImage:        fmt.Sprintf("pingcap/tikv:%s", tidbVersion),
		TiDBImage:        fmt.Sprintf("pingcap/tidb:%s", tidbVersion),
		PumpImage:        fmt.Sprintf("pingcap/tidb-binlog:%s", tidbVersion),
		StorageClassName: "local-storage",
		UserName:         "root",
		Password:         "",
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
			"pvReclaimPolicy":                "Delete",
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
		Clustrer:               tc,
	}
}

func buildSecret(info *tests.TidbClusterConfig) *corev1.Secret {
	backupSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      info.BackupSecretName,
			Namespace: info.Namespace,
		},
		Data: map[string][]byte{
			"user":     []byte(info.UserName),
			"password": []byte(info.Password),
		},
		Type: corev1.SecretTypeOpaque,
	}
	return &backupSecret
}
