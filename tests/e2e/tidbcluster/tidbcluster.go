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

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	tcconfig "github.com/pingcap/tidb-operator/pkg/util/config"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("[tidb-operator] TiDBCluster", func() {
	f := e2eframework.NewDefaultFramework("tidb-cluster")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var aggrCli aggregatorclient.Interface
	var apiExtCli apiextensionsclientset.Interface
	var oa tests.OperatorActions
	var cfg *tests.Config
	var config *restclient.Config
	var ocfg *tests.OperatorConfig
	var genericCli client.Client
	var fwCancel context.CancelFunc
	var fw portforward.PortForward
	/**
	 * StatefulSet or AdvancedStatefulSet interface.
	 */
	var stsGetter func(namespace string) typedappsv1.StatefulSetInterface

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
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config")
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			stsGetter = helper.NewHijackClient(c, asCli).AppsV1().StatefulSets
		} else {
			stsGetter = c.AppsV1().StatefulSets
		}
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
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
				Version: utilimage.TiDBV4Version,
				Name:    "basic-v4",
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

		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "host-network", "", utilimage.TiDBV3Version)
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
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster", "admin", utilimage.TiDBV3Version)
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

		ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", cluster.ClusterVersion, utilimage.TiDBV3UpgradeVersion))
		ctx, cancel := context.WithCancel(context.Background())
		cluster.UpgradeAll(utilimage.TiDBV3UpgradeVersion)
		oa.UpgradeTidbClusterOrDie(&cluster)
		oa.CheckUpgradeOrDie(ctx, &cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)
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
		clusterA := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster3", "admin", utilimage.TiDBV3Version)
		clusterB := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster4", "admin", utilimage.TiDBV3Version)
		oa.DeployTidbClusterOrDie(&clusterA)
		oa.DeployTidbClusterOrDie(&clusterB)
		oa.CheckTidbClusterStatusOrDie(&clusterA)
		oa.CheckTidbClusterStatusOrDie(&clusterB)
		oa.CheckDisasterToleranceOrDie(&clusterA)
		oa.CheckDisasterToleranceOrDie(&clusterB)

		ginkgo.By(fmt.Sprintf("Begin inserting data into cluster %q", clusterA.ClusterName))
		oa.BeginInsertDataToOrDie(&clusterA)

		// backup and restore
		ginkgo.By(fmt.Sprintf("Backup %q and restore into %q", clusterA.ClusterName, clusterB.ClusterName))
		oa.BackupRestoreOrDie(&clusterA, &clusterB)

		ginkgo.By(fmt.Sprintf("Stop inserting data into cluster %q", clusterA.ClusterName))
		oa.StopInsertDataTo(&clusterA)
	})

	ginkgo.It("Service: Sync TiDB service", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "service-it", "admin", utilimage.TiDBV3Version)
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

		ginkgo.By("Sync TiDB service properties")

		ginkgo.By("Updating TiDB service")
		svcType := corev1.ServiceTypeNodePort
		trafficPolicy := corev1.ServiceExternalTrafficPolicyTypeLocal
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.TiDB.Service.Type = svcType
			tc.Spec.TiDB.Service.ExternalTrafficPolicy = &trafficPolicy
			tc.Spec.TiDB.Service.Annotations = map[string]string{
				"test": "test",
			}
			return nil
		})
		framework.ExpectNoError(err)

		ginkgo.By("Waiting for the TiDB service to be synced")
		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
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
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "pump-it", "admin", utilimage.TiDBV3Version)
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

		// If using advanced statefulset, we must upgrade all Kubernetes statefulsets to advanced statefulsets first.
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			stsList, err := c.AppsV1().StatefulSets(ns).List(metav1.ListOptions{})
			framework.ExpectNoError(err)
			for _, sts := range stsList.Items {
				_, err = helper.Upgrade(c, asCli, &sts)
				framework.ExpectNoError(err)
			}
		}

		oldPumpSet, err := stsGetter(tc.Namespace).Get(controller.PumpMemberName(tc.Name), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get pump statefulset")

		oldRev := oldPumpSet.Status.CurrentRevision
		framework.ExpectEqual(oldPumpSet.Status.UpdateRevision, oldRev, "Expected pump is not upgrading")

		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
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
			return nil
		})
		framework.ExpectNoError(err, "Expected update tc")

		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
			pumpSet, err := stsGetter(tc.Namespace).Get(controller.PumpMemberName(tc.Name), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, err
			}
			if err != nil {
				e2elog.Logf("error get pump statefulset: %v", err)
				return false, nil
			}
			if !metav1.IsControlledBy(pumpSet, tc) {
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
			if !metav1.IsControlledBy(pumpConfigMap, tc) {
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
			if !metav1.IsControlledBy(pumpPeerSvc, tc) {
				e2elog.Logf("expect pump peer service adopted by tidbcluster, still waiting...")
				return false, nil
			}
			return true, nil
		})

		framework.ExpectNoError(err)
		// TODO: Add pump configmap rolling-update case
	})

	ginkgo.It("should be operable without helm [API]", func() {
		tc := fixture.GetTidbCluster(ns, "plain-cr", utilimage.TiDBV3Version)
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
			tc.Spec.Version = utilimage.TiDBV3UpgradeVersion
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
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "monitor-test", "admin", utilimage.TiDBV3Version)
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")

		tm := fixture.NewTidbMonitor("e2e-monitor", tc.Namespace, tc, true, true)
		tm.Spec.PVReclaimPolicy = corev1.PersistentVolumeReclaimDelete
		_, err = cli.PingcapV1alpha1().TidbMonitors(tc.Namespace).Create(tm)
		framework.ExpectNoError(err, "Expected tidbmonitor deployed success")
		err = tests.CheckTidbMonitor(tm, cli, c, fw)
		framework.ExpectNoError(err, "Expected tidbmonitor checked success")

		pvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get("e2e-monitor-monitor", metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected fetch tidbmonitor pvc success")
		pvName := pvc.Spec.VolumeName
		pv, err := c.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected fetch tidbmonitor pv success")

		err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
			value, existed := pv.Labels[label.ComponentLabelKey]
			if !existed || value != label.TiDBMonitorVal {
				return false, nil
			}
			value, existed = pv.Labels[label.InstanceLabelKey]
			if !existed || value != "e2e-monitor" {
				return false, nil
			}

			value, existed = pv.Labels[label.NameLabelKey]
			if !existed || value != "tidb-cluster" {
				return false, nil
			}
			value, existed = pv.Labels[label.ManagedByLabelKey]
			if !existed || value != label.TiDBOperator {
				return false, nil
			}
			if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
				return false, fmt.Errorf("pv[%s] 's policy is not Delete", pv.Name)
			}
			return true, nil
		})
		framework.ExpectNoError(err, "monitor pv label error")

		// update TidbMonitor and check whether portName is updated and the nodePort is unchanged
		tm, err = cli.PingcapV1alpha1().TidbMonitors(ns).Get(tm.Name, metav1.GetOptions{})
		framework.ExpectNoError(err, "fetch latest tidbmonitor error")
		tm.Spec.Prometheus.Service.Type = corev1.ServiceTypeNodePort
		tm.Spec.PVReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		tm, err = cli.PingcapV1alpha1().TidbMonitors(ns).Update(tm)
		framework.ExpectNoError(err, "update tidbmonitor service type error")

		var targetPort int32
		err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
			prometheusSvc, err := c.CoreV1().Services(ns).Get(fmt.Sprintf("%s-prometheus", tm.Name), metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if len(prometheusSvc.Spec.Ports) != 1 {
				return false, nil
			}
			if prometheusSvc.Spec.Type != corev1.ServiceTypeNodePort {
				return false, nil
			}
			targetPort = prometheusSvc.Spec.Ports[0].NodePort
			return true, nil
		})
		framework.ExpectNoError(err, "first update tidbmonitor service error")

		tm, err = cli.PingcapV1alpha1().TidbMonitors(ns).Get(tm.Name, metav1.GetOptions{})
		framework.ExpectNoError(err, "fetch latest tidbmonitor again error")
		newPortName := "any-other-word"
		tm.Spec.Prometheus.Service.PortName = &newPortName
		tm, err = cli.PingcapV1alpha1().TidbMonitors(ns).Update(tm)
		framework.ExpectNoError(err, "update tidbmonitor service portName error")

		pvc, err = c.CoreV1().PersistentVolumeClaims(ns).Get("e2e-monitor-monitor", metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected fetch tidbmonitor pvc success")
		pvName = pvc.Spec.VolumeName
		pv, err = c.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected fetch tidbmonitor pv success")

		err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
			prometheusSvc, err := c.CoreV1().Services(ns).Get(fmt.Sprintf("%s-prometheus", tm.Name), metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if len(prometheusSvc.Spec.Ports) != 1 {
				return false, nil
			}
			if prometheusSvc.Spec.Type != corev1.ServiceTypeNodePort {
				return false, nil
			}
			if prometheusSvc.Spec.Ports[0].Name != "any-other-word" {
				return false, nil
			}
			if prometheusSvc.Spec.Ports[0].NodePort != targetPort {
				return false, nil
			}
			if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
				return false, fmt.Errorf("pv[%s] 's policy is not Retain", pv.Name)
			}
			return true, nil
		})
		framework.ExpectNoError(err, "second update tidbmonitor service error")

		err = cli.PingcapV1alpha1().TidbMonitors(tm.Namespace).Delete(tm.Name, &metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete tidbmonitor failed")
		err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if tc.Status.Monitor != nil {
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "tc monitorRef status failed to clean after monitor deleted")
	})

	ginkgo.Context("[Feature: TLS]", func() {
		ginkgo.It("TLS for MySQL Client and TLS between TiDB components", func() {
			ginkgo.By("Installing cert-manager")
			err := installCertManager(f.ClientSet)
			framework.ExpectNoError(err, "failed to install cert-manager")

			tcName := "tls"

			ginkgo.By("Installing tidb issuer")
			err = installTiDBIssuer(ns, tcName)
			framework.ExpectNoError(err, "failed to generate tidb issuer template")

			ginkgo.By("Installing tidb server and client certificate")
			err = installTiDBCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate")

			ginkgo.By("Installing separate tidbInitializer client certificate")
			err = installTiDBInitializerCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install separate tidbInitializer client certificate")

			ginkgo.By("Installing separate dashboard client certificate")
			err = installPDDashboardCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install separate dashboard client certificate")

			ginkgo.By("Installing tidb components certificates")
			err = installTiDBComponentsCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb components certificates")

			ginkgo.By("Creating tidb cluster")
			dashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName)
			tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV4Version)
			tc.Spec.PD.Replicas = 3
			tc.Spec.PD.TLSClientSecretName = &dashTLSName
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			tc.Spec.Pump = &v1alpha1.PumpSpec{
				Replicas:             2,
				BaseImage:            "pingcap/tidb-binlog",
				ResourceRequirements: fixture.WithStorage(fixture.BurstbleSmall, "1Gi"),
				GenericConfig:        tcconfig.New(map[string]interface{}{}),
			}
			err = genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err)

			ginkgo.By("Ensure Dashboard use custom secret")
			foundSecretName := false
			pdSts, err := stsGetter(ns).Get(controller.PDMemberName(tcName), metav1.GetOptions{})
			framework.ExpectNoError(err)
			for _, vol := range pdSts.Spec.Template.Spec.Volumes {
				if vol.Name == "tidb-client-tls" {
					foundSecretName = true
					framework.ExpectEqual(vol.Secret.SecretName, dashTLSName)
				}
			}
			framework.ExpectEqual(foundSecretName, true)

			ginkgo.By("Creating tidb initializer")
			passwd := "admin"
			initName := fmt.Sprintf("%s-initializer", tcName)
			initPassWDName := fmt.Sprintf("%s-initializer-passwd", tcName)
			initTLSName := fmt.Sprintf("%s-initializer-tls", tcName)
			initSecret := fixture.GetInitializerSecret(tc, initPassWDName, passwd)
			_, err = c.CoreV1().Secrets(ns).Create(initSecret)
			framework.ExpectNoError(err)

			ti := fixture.GetTidbInitializer(ns, tcName, initName, initPassWDName, initTLSName)
			err = genericCli.Create(context.TODO(), ti)
			framework.ExpectNoError(err)

			source := &tests.TidbClusterConfig{
				Namespace:      ns,
				ClusterName:    tcName,
				OperatorTag:    cfg.OperatorTag,
				ClusterVersion: utilimage.TiDBV4Version,
			}
			targetTcName := "tls-target"
			targetTc := fixture.GetTidbCluster(ns, targetTcName, utilimage.TiDBV4Version)
			targetTc.Spec.PD.Replicas = 1
			targetTc.Spec.TiKV.Replicas = 1
			targetTc.Spec.TiDB.Replicas = 1
			err = genericCli.Create(context.TODO(), targetTc)
			framework.ExpectNoError(err)
			err = oa.WaitForTidbClusterReady(targetTc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err)

			drainerConfig := &tests.DrainerConfig{
				DrainerName:       "tls-drainer",
				OperatorTag:       cfg.OperatorTag,
				SourceClusterName: tcName,
				Namespace:         ns,
				DbType:            tests.DbTypeTiDB,
				Host:              fmt.Sprintf("%s-tidb.%s.svc.cluster.local", targetTcName, ns),
				Port:              "4000",
				TLSCluster:        true,
				User:              "root",
				Password:          "",
			}

			ginkgo.By("Deploying tidb drainer")
			err = oa.DeployDrainer(drainerConfig, source)
			framework.ExpectNoError(err)
			err = oa.CheckDrainer(drainerConfig, source)
			framework.ExpectNoError(err)

			ginkgo.By("Inserting data into source db")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, tcName, passwd, true))
			framework.ExpectNoError(err)

			ginkgo.By("Checking tidb-binlog works as expected")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, targetTcName, "", false))
			framework.ExpectNoError(err)

			ginkgo.By("Connecting to tidb server to verify the connection is TLS enabled")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns, tcName, passwd))
			framework.ExpectNoError(err)

			ginkgo.By("Scaling out tidb cluster")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.Replicas = 5
				tc.Spec.TiKV.Replicas = 5
				tc.Spec.TiDB.Replicas = 3
				return nil
			})
			framework.ExpectNoError(err)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err)

			ginkgo.By("Scaling in tidb cluster")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.Replicas = 3
				tc.Spec.TiKV.Replicas = 3
				tc.Spec.TiDB.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err)

			ginkgo.By("Upgrading tidb cluster")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBV4UpgradeVersion
				return nil
			})
			framework.ExpectNoError(err)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err)

			ginkgo.By("Deleting cert-manager")
			err = deleteCertManager(f.ClientSet)
			framework.ExpectNoError(err, "failed to delete cert-manager")
		})
	})

	ginkgo.It("[Feature: CDC]", func() {
		ginkgo.By("Creating cdc cluster")
		fromTc := fixture.GetTidbCluster(ns, "cdc-source", utilimage.TiDBNightly)
		fromTc.Spec.PD.Replicas = 3
		fromTc.Spec.TiKV.Replicas = 3
		fromTc.Spec.TiDB.Replicas = 2
		fromTc.Spec.TiCDC = &v1alpha1.TiCDCSpec{
			BaseImage: "pingcap/ticdc",
			Replicas:  3,
		}
		err := genericCli.Create(context.TODO(), fromTc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(fromTc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster ready")

		ginkgo.By("Creating cdc-sink cluster")
		toTc := fixture.GetTidbCluster(ns, "cdc-sink", utilimage.TiDBNightly)
		toTc.Spec.PD.Replicas = 1
		toTc.Spec.TiKV.Replicas = 1
		toTc.Spec.TiDB.Replicas = 1
		err = genericCli.Create(context.TODO(), toTc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(toTc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster ready")

		ginkgo.By("Creating change feed task")
		fromTCName := fromTc.Name
		toTCName := toTc.Name
		args := []string{
			"exec", "-n", ns,
			fmt.Sprintf("%s-0", controller.TiCDCMemberName(fromTCName)),
			"--",
			"/cdc", "cli", "changefeed", "create",
			fmt.Sprintf("--sink-uri=tidb://root:@%s:4000/", controller.TiDBMemberName(toTCName)),
			fmt.Sprintf("--pd=http://%s:2379", controller.PDMemberName(fromTCName)),
		}
		data, err := framework.RunKubectl(args...)
		framework.ExpectNoError(err, fmt.Sprintf("failed to create change feed task: %s, %v", string(data), err))

		ginkgo.By("Inserting data to cdc cluster")
		err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, fromTCName, "", false))
		framework.ExpectNoError(err)

		ginkgo.By("Checking cdc works as expected")
		err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, toTCName, "", false))
		framework.ExpectNoError(err)

		framework.Logf("CDC works as expected")
	})
})

func newTidbClusterConfig(cfg *tests.Config, ns, clusterName, password, tidbVersion string) tests.TidbClusterConfig {
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
			"pd.resources.requests.cpu":      "20m",
			"pd.resources.requests.memory":   "20Mi",
			"tikv.resources.limits.cpu":      "2000m",
			"tikv.resources.limits.memory":   "4Gi",
			"tikv.resources.requests.cpu":    "20m",
			"tikv.resources.requests.memory": "20Mi",
			"tidb.resources.limits.cpu":      "2000m",
			"tidb.resources.limits.memory":   "4Gi",
			"tidb.resources.requests.cpu":    "20m",
			"tidb.resources.requests.memory": "20Mi",
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
		TopologyKey:            "rack",
		EnableConfigMapRollout: true,
		ClusterVersion:         tidbVersion,
	}
}
