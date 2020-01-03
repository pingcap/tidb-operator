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

package tidbcluster

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/label"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	tcconfig "github.com/pingcap/tidb-operator/pkg/util/config"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/apiserver"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/kubernetes/test/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("[tidb-operator] TiDBCluster", func() {
	f := framework.NewDefaultFramework("tidb-cluster")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var oa tests.OperatorActions
	var cfg *tests.Config
	var config *restclient.Config
	var ocfg *tests.OperatorConfig
	var genericCli client.Client
	var fwCancel context.CancelFunc

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		var err error
		config, err = framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		asCli, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config")
		ctx, cancel := context.WithCancel(context.Background())
		fw, err := portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		oa = tests.NewOperatorActions(cli, c, asCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.Context("Basic: Deploying, Scaling, Update Configuration", func() {
		clusterCfgs := []struct {
			Version string
			Name    string // helm release name, should not conflict with names used in other tests
			Values  map[string]string
		}{
			{
				Version: utilimage.TiDBV3Version,
				Name:    "basic-v3",
			},
			{
				Version: utilimage.TiDBV2Version,
				Name:    "basic-v2",
				Values: map[string]string{
					// verify v2.1.x configuration compatibility
					// https://github.com/pingcap/tidb-operator/pull/950
					"tikv.resources.limits.storage": "1G",
				},
			},
			{
				Version: utilimage.TiDBTLSVersion,
				Name:    "basic-v3-cluster-tls",
				Values: map[string]string{
					"enableTLSCluster": "true",
				},
			},
		}

		for _, clusterCfg := range clusterCfgs {
			localCfg := clusterCfg
			ginkgo.It(fmt.Sprintf("[TiDB Version: %s] %s", localCfg.Version, localCfg.Name), func() {
				cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, localCfg.Name, "", localCfg.Version)
				if len(localCfg.Values) > 0 {
					for k, v := range localCfg.Values {
						cluster.Resources[k] = v
					}
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
		if !ocfg.Enabled(features.AdvancedStatefulSet) {
			serverVersion, err := c.Discovery().ServerVersion()
			framework.ExpectNoError(err, "failed to fetch Kubernetes version")
			sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
			klog.Infof("ServerVersion: %v", serverVersion.String())
			if sv.LessThan(utilversion.MustParseSemantic("v1.13.11")) || // < v1.13.11
				(sv.AtLeast(utilversion.MustParseSemantic("v1.14.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.14.7"))) || // >= v1.14.0 but < v1.14.7
				(sv.AtLeast(utilversion.MustParseSemantic("v1.15.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.15.4"))) { // >= v1.15.0 but < v1.15.4
				// https://github.com/pingcap/tidb-operator/issues/1042#issuecomment-547742565
				framework.Skipf("Skipping HostNetwork test. Kubernetes %v has a bug that StatefulSet may apply revision incorrectly, HostNetwork cannot work well in this cluster", serverVersion)
			}
			ginkgo.By(fmt.Sprintf("Testing HostNetwork feature with Kubernetes %v", serverVersion))
		} else {
			ginkgo.By("Testing HostNetwork feature with Advanced StatefulSet")
		}

		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "host-network", "", "")
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)

		ginkgo.By("switch to host network")
		cluster.RunInHost(true)
		oa.UpgradeTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		ginkgo.By("switch back to pod network")
		cluster.RunInHost(false)
		oa.UpgradeTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)
	})

	ginkgo.It("Upgrading TiDB Cluster", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster", "admin", "")
		cluster.Resources["pd.replicas"] = "3"

		ginkgo.By("Creating webhook certs and self signing it")
		svcName := "webhook"
		certCtx, err := apimachinery.SetupServerCert(ns, svcName)
		framework.ExpectNoError(err, fmt.Sprintf("unable to setup certs for webservice %s", tests.WebhookServiceName))

		ginkgo.By("Starting webhook pod")
		webhookPod, svc := startWebhook(f, cfg.E2EImage, ns, svcName, certCtx.Cert, certCtx.Key)

		ginkgo.By("Register webhook")
		oa.RegisterWebHookAndServiceOrDie(ocfg.WebhookConfigName, ns, svc.Name, certCtx)

		ginkgo.By(fmt.Sprintf("Deploying tidb cluster %s", cluster.ClusterVersion))
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)
		oa.CheckDisasterToleranceOrDie(&cluster)

		upgradeVersions := cfg.GetUpgradeTidbVersionsOrDie()
		ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", cluster.ClusterVersion, upgradeVersions[0]))
		ctx, cancel := context.WithCancel(context.Background())
		assignedNodes := oa.GetTidbMemberAssignedNodesOrDie(&cluster)
		cluster.UpgradeAll(upgradeVersions[0])
		oa.UpgradeTidbClusterOrDie(&cluster)
		oa.CheckUpgradeOrDie(ctx, &cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)
		oa.CheckTidbMemberAssignedNodesOrDie(&cluster, assignedNodes)
		cancel()

		ginkgo.By("Check webhook is still running")
		webhookPod, err = c.CoreV1().Pods(webhookPod.Namespace).Get(webhookPod.Name, metav1.GetOptions{})
		framework.ExpectNoError(err, fmt.Sprintf("unable to get pod %s/%s", webhookPod.Namespace, webhookPod.Name))
		if webhookPod.Status.Phase != v1.PodRunning {
			logs, err := e2epod.GetPodLogs(c, webhookPod.Namespace, webhookPod.Name, "webhook")
			framework.ExpectNoError(err)
			e2elog.Logf("webhook logs: %s", logs)
			e2elog.Fail("webhook pod is not running")
		}

		oa.CleanWebHookAndServiceOrDie(ocfg.WebhookConfigName)
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

		ginkgo.By(fmt.Sprintf("Begin inserting data into cluster %q", clusterA.ClusterName))
		go oa.BeginInsertDataToOrDie(&clusterA)

		// backup and restore
		ginkgo.By(fmt.Sprintf("Backup %q and restore into %q", clusterA.ClusterName, clusterB.ClusterName))
		oa.BackupRestoreOrDie(&clusterA, &clusterB)

		ginkgo.By(fmt.Sprintf("Stop inserting data into cluster %q", clusterA.ClusterName))
		oa.StopInsertDataTo(&clusterA)
	})

	ginkgo.It("Test aggregated apiserver", func() {
		ginkgo.By(fmt.Sprintf("Starting to test apiserver, test apiserver image: %s", cfg.E2EImage))
		framework.Logf("config: %v", config)
		aaCtx := apiserver.NewE2eContext(ns, config, cfg.E2EImage)
		defer aaCtx.Clean()
		aaCtx.Setup()
		aaCtx.Do()
	})

	ginkgo.It("Service: Sync TiDB service", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "service-it", "admin", "")
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		ns := cluster.Namespace
		tcName := cluster.ClusterName

		oldSvc, err := c.CoreV1().Services(ns).Get(controller.TiDBMemberName(tcName), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected TiDB service created by helm chart")
		tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected TiDB cluster created by helm chart")
		if isNil, err := gomega.BeNil().Match(metav1.GetControllerOf(oldSvc)); !isNil {
			e2elog.Failf("Expected TiDB service created by helm chart is orphaned: %v", err)
		}

		ginkgo.By(fmt.Sprintf("Adopt orphaned service created by helm"))
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{}
			return nil
		})
		framework.ExpectNoError(err, "Expected update TiDB cluster")

		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
			svc, err := c.CoreV1().Services(ns).Get(controller.TiDBMemberName(tcName), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return false, err
				}
				e2elog.Logf("error get TiDB service: %v", err)
				return false, nil
			}
			owner := metav1.GetControllerOf(svc)
			if owner == nil {
				e2elog.Logf("tidb service has not been adopted by TidbCluster yet")
				return false, nil
			}
			framework.ExpectEqual(metav1.IsControlledBy(svc, tc), true, "Expected owner is TidbCluster")
			framework.ExpectEqual(svc.Spec.ClusterIP, oldSvc.Spec.ClusterIP, "ClusterIP should be stable across adopting and updating")
			return true, nil
		})
		framework.ExpectNoError(err)

		ginkgo.By(fmt.Sprintf("Sync TiDB service properties"))

		svcType := corev1.ServiceTypeNodePort
		trafficPolicy := corev1.ServiceExternalTrafficPolicyTypeLocal

		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected get TiDB cluster")
			tc.Spec.TiDB.Service.Type = svcType
			tc.Spec.TiDB.Service.ExternalTrafficPolicy = &trafficPolicy
			tc.Spec.TiDB.Service.Annotations = map[string]string{
				"test": "test",
			}
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
			if err != nil && !errors.IsConflict(err) {
				return false, err
			}
			if errors.IsConflict(err) {
				e2elog.Logf("conflicts when updating tidbcluster, retry...")
				return false, nil
			}
			svc, err := c.CoreV1().Services(ns).Get(controller.TiDBMemberName(tcName), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return false, err
				}
				e2elog.Logf("error get TiDB service: %v", err)
				return false, nil
			}
			if isEqual, err := gomega.Equal(svcType).Match(svc.Spec.Type); !isEqual {
				e2elog.Logf("tidb service is not synced, %v", err)
				return false, nil
			}
			if isEqual, err := gomega.Equal(trafficPolicy).Match(svc.Spec.ExternalTrafficPolicy); !isEqual {
				e2elog.Logf("tidb service is not synced, %v", err)
				return false, nil
			}
			if haveKV, err := gomega.HaveKeyWithValue("test", "test").Match(svc.Annotations); !haveKV {
				e2elog.Logf("tidb service is not synced, %v", err)
				return false, nil
			}

			return true, nil
		})

		framework.ExpectNoError(err)
	})

	updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
	// Basic IT for managed in TidbCluster CR
	// TODO: deploy pump through CR in backup and restore IT
	ginkgo.It("Pump: Test managing Pump in TidbCluster CRD", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "pump-it", "admin", "")
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		ginkgo.By("Test adopting pump statefulset created by helm could avoid rolling-update.")
		err := oa.DeployAndCheckPump(&cluster)
		framework.ExpectNoError(err, "Expected pump deployed")

		tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")

		pullPolicy := corev1.PullIfNotPresent
		tc.Spec.Pump = &v1alpha1.PumpSpec{
			BaseImage: "pingcap/tidb-binlog",
			ComponentSpec: v1alpha1.ComponentSpec{
				Version:         &cluster.ClusterVersion,
				ImagePullPolicy: &pullPolicy,
				Affinity: &corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								PodAffinityTerm: corev1.PodAffinityTerm{
									Namespaces:  []string{cluster.Namespace},
									TopologyKey: "rack",
								},
								Weight: 50,
							},
						},
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Effect:   corev1.TaintEffectNoSchedule,
						Key:      "node-role",
						Operator: corev1.TolerationOpEqual,
						Value:    "tidb",
					},
				},
				SchedulerName:        pointer.StringPtr("default-scheduler"),
				ConfigUpdateStrategy: &updateStrategy,
			},
			Replicas:         1,
			StorageClassName: pointer.StringPtr("local-storage"),
			ResourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
			GenericConfig: tcconfig.New(map[string]interface{}{
				"addr":               "0.0.0.0:8250",
				"gc":                 7,
				"data-dir":           "/data",
				"heartbeat-interval": 2,
			}),
		}

		oldPumpSet, err := c.AppsV1().StatefulSets(tc.Namespace).Get(controller.PumpMemberName(tc.Name), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get pump statefulset")

		oldRev := oldPumpSet.Status.CurrentRevision
		framework.ExpectEqual(oldPumpSet.Status.UpdateRevision, oldRev, "Expected pump is not upgrading")

		tcUpdated, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(tc)
		framework.ExpectNoError(err, "Expected update tc")

		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
			pumpSet, err := c.AppsV1().StatefulSets(tc.Namespace).Get(controller.PumpMemberName(tc.Name), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, err
			}
			if err != nil {
				e2elog.Logf("error get pump statefulset: %v", err)
				return false, nil
			}
			if !metav1.IsControlledBy(pumpSet, tcUpdated) {
				e2elog.Logf("expect pump staetfulset adopted by tidbcluster, still waiting...")
				return false, nil
			}
			// The desired state encoded in CRD should be exactly same with the one created by helm chart
			framework.ExpectEqual(pumpSet.Status.CurrentRevision, oldRev, "Expected no rolling-update when adopting pump statefulset")
			framework.ExpectEqual(pumpSet.Status.UpdateRevision, oldRev, "Expected no rolling-update when adopting pump statefulset")

			usingName := member.FindConfigMapVolume(&pumpSet.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.PumpMemberName(tc.Name))
			})
			if usingName == "" {
				e2elog.Fail("cannot find configmap that used by pump statefulset")
			}
			pumpConfigMap, err := c.CoreV1().ConfigMaps(tc.Namespace).Get(usingName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, err
			}
			if err != nil {
				e2elog.Logf("error get pump configmap: %v", err)
				return false, nil
			}
			if !metav1.IsControlledBy(pumpConfigMap, tcUpdated) {
				e2elog.Logf("expect pump configmap adopted by tidbcluster, still waiting...")
				return false, nil
			}

			pumpPeerSvc, err := c.CoreV1().Services(tc.Namespace).Get(controller.PumpPeerMemberName(tc.Name), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, err
			}
			if err != nil {
				e2elog.Logf("error get pump peer service: %v", err)
				return false, nil
			}
			if !metav1.IsControlledBy(pumpPeerSvc, tcUpdated) {
				e2elog.Logf("expect pump peer service adopted by tidbcluster, still waiting...")
				return false, nil
			}
			return true, nil
		})

		framework.ExpectNoError(err)
		// TODO: Add pump configmap rolling-update case
	})

	ginkgo.It("API: Migrate from helm to CRD", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "helm-migration", "admin", "")
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")

		ginkgo.By("Discovery service should be reconciled by tidb-operator")
		discoveryName := controller.DiscoveryMemberName(tc.Name)
		discoveryDep, err := c.AppsV1().Deployments(tc.Namespace).Get(discoveryName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get discovery deployment")
		WaitObjectToBeControlledByOrDie(genericCli, discoveryDep, tc, 5*time.Minute)

		err = utils.WaitForDeploymentComplete(c, discoveryDep, e2elog.Logf, 10*time.Second, 5*time.Minute)
		framework.ExpectNoError(err, "Discovery Deployment should be healthy after managed by tidb-operator")

		err = genericCli.Delete(context.TODO(), discoveryDep)
		framework.ExpectNoError(err, "Expected to delete deployment")

		err = wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
			_, err := c.AppsV1().Deployments(tc.Namespace).Get(discoveryName, metav1.GetOptions{})
			if err != nil {
				e2elog.Logf("wait discovery deployment get created again: %v", err)
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "Discovery Deployment should be recovered by tidb-operator after deletion")

		ginkgo.By("Managing TiDB configmap in TidbCluster CRD in-place should not trigger rolling-udpate")
		// TODO: modify other cases to manage TiDB configmap in CRD by default
		setNameToRevision := map[string]string{
			controller.PDMemberName(tc.Name):   "",
			controller.TiKVMemberName(tc.Name): "",
			controller.TiDBMemberName(tc.Name): "",
		}

		for setName := range setNameToRevision {
			oldSet, err := c.AppsV1().StatefulSets(tc.Namespace).Get(setName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected get statefulset %s", setName)

			oldRev := oldSet.Status.CurrentRevision
			framework.ExpectEqual(oldSet.Status.UpdateRevision, oldRev, "Expected statefulset %s is not upgrading", setName)

			setNameToRevision[setName] = oldRev
		}

		tc, err = cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")
		tc.Spec.TiDB.Config = &v1alpha1.TiDBConfig{}
		tc.Spec.TiDB.ConfigUpdateStrategy = &updateStrategy
		tc.Spec.TiKV.Config = &v1alpha1.TiKVConfig{}
		tc.Spec.TiKV.ConfigUpdateStrategy = &updateStrategy
		tc.Spec.PD.Config = &v1alpha1.PDConfig{}
		tc.Spec.PD.ConfigUpdateStrategy = &updateStrategy
		_, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(tc)
		framework.ExpectNoError(err, "Expected update tidbcluster")

		// check for 2 minutes to ensure the tidb statefulset do not get rolling-update
		err = wait.PollImmediate(10*time.Second, 2*time.Minute, func() (bool, error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected get tidbcluster")
			framework.ExpectEqual(tc.Status.PD.Phase, v1alpha1.NormalPhase, "PD should not be updated")
			framework.ExpectEqual(tc.Status.TiKV.Phase, v1alpha1.NormalPhase, "TiKV should not be updated")
			framework.ExpectEqual(tc.Status.TiDB.Phase, v1alpha1.NormalPhase, "TiDB should not be updated")

			for setName, oldRev := range setNameToRevision {
				newSet, err := c.AppsV1().StatefulSets(tc.Namespace).Get(setName, metav1.GetOptions{})
				framework.ExpectNoError(err, "Expected get tidb statefulset")
				framework.ExpectEqual(newSet.Status.CurrentRevision, oldRev, "Expected no rolling-update of %s when manage config in-place", setName)
				framework.ExpectEqual(newSet.Status.UpdateRevision, oldRev, "Expected no rolling-update of %s when manage config in-place", setName)
			}
			return false, nil
		})

		if err != wait.ErrWaitTimeout {
			e2elog.Failf("Unexpected error when checking tidb statefulset will not get rolling-update: %v", err)
		}

		err = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
			for setName := range setNameToRevision {
				newSet, err := c.AppsV1().StatefulSets(tc.Namespace).Get(setName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				usingName := member.FindConfigMapVolume(&newSet.Spec.Template.Spec, func(name string) bool {
					return strings.HasPrefix(name, setName)
				})
				if usingName == "" {
					e2elog.Failf("cannot find configmap that used by %s", setName)
				}
				usingCm, err := c.CoreV1().ConfigMaps(tc.Namespace).Get(usingName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if !metav1.IsControlledBy(usingCm, tc) {
					e2elog.Logf("expect configmap of %s adopted by tidbcluster, still waiting...", setName)
					return false, nil
				}
			}
			return true, nil
		})

		framework.ExpectNoError(err)
	})

	ginkgo.It("Restarter: Testing restarting by annotations", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "restarter", "admin", "")
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")
		pd_0, err := c.CoreV1().Pods(ns).Get(operatorUtils.GetPodName(tc, v1alpha1.PDMemberType, 0), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get pd-0")
		tikv_0, err := c.CoreV1().Pods(ns).Get(operatorUtils.GetPodName(tc, v1alpha1.TiKVMemberType, 0), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tikv-0")
		tidb_0, err := c.CoreV1().Pods(ns).Get(operatorUtils.GetPodName(tc, v1alpha1.TiDBMemberType, 0), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidb-0")
		pd_0.Annotations[label.AnnPodDeferDeleting] = "true"
		tikv_0.Annotations[label.AnnPodDeferDeleting] = "true"
		tidb_0.Annotations[label.AnnPodDeferDeleting] = "true"
		_, err = c.CoreV1().Pods(ns).Update(pd_0)
		framework.ExpectNoError(err, "Expected update pd-0 restarting ann")
		_, err = c.CoreV1().Pods(ns).Update(tikv_0)
		framework.ExpectNoError(err, "Expected update tikv-0 restarting ann")
		_, err = c.CoreV1().Pods(ns).Update(tidb_0)
		framework.ExpectNoError(err, "Expected update tidb-0 restarting ann")

		f := func(name, namespace string, uid types.UID) (bool, error) {
			pod, err := c.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if _, existed := pod.Annotations[label.AnnPodDeferDeleting]; existed {
				return false, nil
			}
			if uid == pod.UID {
				return false, nil
			}
			return true, nil
		}

		err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
			isPdRestarted, err := f(pd_0.Name, ns, pd_0.UID)
			if !(isPdRestarted && err == nil) {
				return isPdRestarted, err
			}
			isTiKVRestarted, err := f(tikv_0.Name, ns, tikv_0.UID)
			if !(isTiKVRestarted && err == nil) {
				return isTiKVRestarted, err
			}
			isTiDBRestarted, err := f(tidb_0.Name, ns, tidb_0.UID)
			if !(isTiDBRestarted && err == nil) {
				return isTiDBRestarted, err
			}
			return true, nil
		})
		framework.ExpectNoError(err, "Expected tidbcluster pod restarted")
	})

	ginkgo.It("should be operable without helm [API]", func() {
		tc := fixture.GetTidbCluster(ns, "plain-cr", "v2.1.16")
		err := genericCli.Create(context.TODO(), tc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster ready")

		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.PD.Replicas = 5
			tc.Spec.TiKV.Replicas = 5
			tc.Spec.TiDB.Replicas = 4
			return nil
		})
		framework.ExpectNoError(err, "Expected TiDB cluster updated")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster scaled out and ready")

		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Version = utilimage.TiDBV3Version
			return nil
		})
		framework.ExpectNoError(err, "Expected TiDB cluster updated")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster upgraded to new version and ready")

		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.PD.Replicas = 3
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			return nil
		})
		framework.ExpectNoError(err, "Expected TiDB cluster updated")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster scaled in and ready")
	})

	ginkgo.It("TidbMonitor: Deploying and checking monitor", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "monitor-test", "admin", "")
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")

		tm := newTidbMonitor("monitor", tc.Namespace, tc, true, false)
		oa.DeployAndCheckTidbMonitorOrDie(tm)

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
		PumpImage:        fmt.Sprintf("pingcap/tidb-binlog:%s", tidbVersion),
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

func newTidbMonitor(name, namespace string, tc *v1alpha1.TidbCluster, grafanaEnabled, persist bool) *v1alpha1.TidbMonitor {
	monitor := &v1alpha1.TidbMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.TidbMonitorSpec{
			Clusters: []v1alpha1.TidbClusterRef{
				{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			},
			Prometheus: v1alpha1.PrometheusSpec{
				LogLevel: "info",
				Service: v1alpha1.ServiceSpec{
					Type: "ClusterIP",
				},
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage: "prom/prometheus",
					Version:   "v2.11.1",
				},
			},
			Reloader: v1alpha1.ReloaderSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage: "pingcap/tidb-monitor-reloader",
					Version:   "v1.0.1",
				},
				Service: v1alpha1.ServiceSpec{
					Type: "ClusterIP",
				},
			},
			Initializer: v1alpha1.InitializerSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage: "pingcap/tidb-monitor-initializer",
					Version:   "v3.0.5",
				},
			},
			Persistent: persist,
		},
	}
	if grafanaEnabled {
		monitor.Spec.Grafana = &v1alpha1.GrafanaSpec{
			MonitorContainer: v1alpha1.MonitorContainer{
				BaseImage: "grafana/grafana",
				Version:   "6.0.1",
			},
			Username: "admin",
			Password: "admin",
			Service: v1alpha1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
			},
			LogLevel: "info",
		}
	}

	return monitor
}
