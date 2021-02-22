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
	astsHelper "github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/monitor/monitor"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	tcconfig "github.com/pingcap/tidb-operator/pkg/util/config"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedpdclient"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/kubernetes/test/utils"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("TiDBCluster", func() {
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
	var genericCli ctrlCli.Client
	var fwCancel context.CancelFunc
	var fw portforward.PortForward
	/**
	 * StatefulSet or AdvancedStatefulSet getter interface.
	 */
	var stsGetter typedappsv1.StatefulSetsGetter
	var crdUtil *tests.CrdTestUtil

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		var err error
		config, err = framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for pingcap")
		asCli, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for advanced-statefulset")
		genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset for controller-runtime")
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset kube-aggregator")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset apiextensions-apiserver")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config for tidb-operator")
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		// OperatorFeatures := map[string]bool{"AutoScaling": true}
		// cfg.OperatorFeatures = OperatorFeatures
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			stsGetter = astsHelper.NewHijackClient(c, asCli).AppsV1()
		} else {
			stsGetter = c.AppsV1()
		}
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
		crdUtil = tests.NewCrdTestUtil(cli, c, asCli, stsGetter)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	// basic deploy, scale out, scale in, change configuration tests
	ginkgo.Describe("when using version", func() {
		versions := []string{utilimage.TiDBV3, utilimage.TiDBV4}
		for _, version := range versions {
			version := version
			versionDashed := strings.ReplaceAll(version, ".", "-")
			ginkgo.Context(version, func() {
				ginkgo.It("should scale out tc successfully", func() {
					ginkgo.By("Deploy a basic tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("basic-%s", versionDashed), version)
					tc.Spec.TiDB.Replicas = 1
					tc.Spec.TiKV.SeparateRocksDBLog = pointer.BoolPtr(true)
					tc.Spec.TiKV.SeparateRaftLog = pointer.BoolPtr(true)
					tc.Spec.TiKV.LogTailer = &v1alpha1.LogTailerSpec{
						ResourceRequirements: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
					}
					_, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
					framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)

					ginkgo.By("scale out tidb, tikv, pd")
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.TiDB.Replicas = 2
						tc.Spec.TiKV.Replicas = 4
						// this must be 5, or one pd pod will not be scheduled, reference: https://docs.pingcap.com/tidb-in-kubernetes/stable/tidb-scheduler#pd-component
						tc.Spec.PD.Replicas = 5
						return nil
					})
					framework.ExpectNoError(err, "failed to scale out TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)
				})

				ginkgo.It("should scale in tc successfully", func() {
					ginkgo.By("Deploy a basic tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("basic-%s", versionDashed), version)
					tc.Spec.TiKV.Replicas = 4
					tc.Spec.PD.Replicas = 5
					_, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
					framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)

					ginkgo.By("Scale in tidb, tikv, pd")
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.TiDB.Replicas = 1
						tc.Spec.TiKV.Replicas = 3
						tc.Spec.PD.Replicas = 3
						return nil
					})
					framework.ExpectNoError(err, "failed to scale in TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)
				})

				ginkgo.It("should change configurations successfully", func() {
					ginkgo.By("Deploy a basic tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("basic-%s", versionDashed), version)
					tc.Spec.TiDB.Replicas = 1
					_, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
					framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)

					ginkgo.By("Change configuration")
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.ConfigUpdateStrategy = v1alpha1.ConfigUpdateStrategyRollingUpdate
						tc.Spec.PD.MaxFailoverCount = pointer.Int32Ptr(4)
						tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(4)
						tc.Spec.TiDB.MaxFailoverCount = pointer.Int32Ptr(4)
						return nil
					})
					framework.ExpectNoError(err, "failed to change configuration of TidbCluster: %q", tc.Name)
					err = crdUtil.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
				})
			})
		}
	})

	/**
	 * This test case switches back and forth between pod network and host network of a single cluster.
	 * Note that only one cluster can run in host network mode at the same time.
	 */
	ginkgo.It("should switch between pod network and host network", func() {
		if !ocfg.Enabled(features.AdvancedStatefulSet) {
			serverVersion, err := c.Discovery().ServerVersion()
			framework.ExpectNoError(err, "failed to fetch Kubernetes version")
			sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
			log.Logf("ServerVersion: %v", serverVersion.String())
			if sv.LessThan(utilversion.MustParseSemantic("v1.13.11")) || // < v1.13.11
				(sv.AtLeast(utilversion.MustParseSemantic("v1.14.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.14.7"))) || // >= v1.14.0 and < v1.14.7
				(sv.AtLeast(utilversion.MustParseSemantic("v1.15.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.15.4"))) { // >= v1.15.0 and < v1.15.4
				// https://github.com/pingcap/tidb-operator/issues/1042#issuecomment-547742565
				framework.Skipf("Skipping HostNetwork test. Kubernetes %v has a bug that StatefulSet may apply revision incorrectly, HostNetwork cannot work well in this cluster", serverVersion)
			}
			log.Logf("Testing HostNetwork feature with Kubernetes %v", serverVersion)
		} else {
			log.Logf("Testing HostNetwork feature with Advanced StatefulSet")
		}

		ginkgo.By("Deploy initial tc")
		tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, "host-network", "", utilimage.TiDBV4)
		tcCfg.Resources["pd.replicas"] = "1"
		tcCfg.Resources["tidb.replicas"] = "1"
		tcCfg.Resources["tikv.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&tcCfg)

		ginkgo.By("Switch to host network")
		tcCfg.SetHostNetwork(true)
		oa.UpgradeTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)

		ginkgo.By("Switch back to pod network")
		tcCfg.SetHostNetwork(false)
		oa.UpgradeTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)
	})

	ginkgo.It("should upgrade TidbCluster with webhook enabled", func() {
		tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, "cluster", "admin", utilimage.TiDBV4Prev)
		tcCfg.Resources["pd.replicas"] = "3"

		ginkgo.By("Creating webhook certs and self signing it")
		svcName := "webhook"
		certCtx, err := apimachinery.SetupServerCert(ns, svcName)
		framework.ExpectNoError(err, "failed to setup certs for apimachinery webservice %s", tests.WebhookServiceName)

		ginkgo.By("Starting webhook pod")
		webhookPod, svc := startWebhook(c, cfg.E2EImage, ns, svcName, certCtx.Cert, certCtx.Key)

		ginkgo.By("Register webhook")
		oa.RegisterWebHookAndServiceOrDie(ocfg.WebhookConfigName, ns, svc.Name, certCtx)

		ginkgo.By(fmt.Sprintf("Deploying tidb cluster %s", tcCfg.ClusterVersion))
		oa.DeployTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)
		oa.CheckDisasterToleranceOrDie(&tcCfg)

		ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", tcCfg.ClusterVersion, utilimage.TiDBV4))
		ctx, cancel := context.WithCancel(context.Background())
		tcCfg.UpgradeAll(utilimage.TiDBV4)
		oa.UpgradeTidbClusterOrDie(&tcCfg)
		oa.CheckUpgradeOrDie(ctx, &tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)
		cancel()

		ginkgo.By("Check webhook is still running")
		webhookPod, err = c.CoreV1().Pods(webhookPod.Namespace).Get(webhookPod.Name, metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get webhook pod %s/%s", webhookPod.Namespace, webhookPod.Name)
		if webhookPod.Status.Phase != corev1.PodRunning {
			logs, err := pod.GetPodLogs(c, webhookPod.Namespace, webhookPod.Name, "webhook")
			framework.ExpectNoError(err, "failed to get pod log %s/%s", webhookPod.Namespace, webhookPod.Name)
			log.Logf("webhook logs: %s", logs)
			log.Fail("webhook pod is not running")
		}

		ginkgo.By("Clean up webhook")
		oa.CleanWebHookAndServiceOrDie(ocfg.WebhookConfigName)
	})

	ginkgo.It("should keep tidb service in sync", func() {
		ginkgo.By("Deploy initial tc")
		tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, "service", "admin", utilimage.TiDBV4)
		tcCfg.Resources["pd.replicas"] = "1"
		tcCfg.Resources["tidb.replicas"] = "1"
		tcCfg.Resources["tikv.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)

		ns := tcCfg.Namespace
		tcName := tcCfg.ClusterName

		oldSvc, err := c.CoreV1().Services(ns).Get(controller.TiDBMemberName(tcName), metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get service for TidbCluster: %v", tcCfg)
		tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get TidbCluster: %v", tcCfg)
		if isNil, err := gomega.BeNil().Match(metav1.GetControllerOf(oldSvc)); !isNil {
			log.Failf("Expected TiDB service created by helm chart is orphaned: %v", err)
		}

		ginkgo.By("Adopt orphaned service created by helm")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{}
			return nil
		})
		framework.ExpectNoError(err, "failed to update tidb service of TidbCluster: %q", tc.Name)
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			svc, err := c.CoreV1().Services(ns).Get(controller.TiDBMemberName(tcName), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return false, err
				}
				log.Logf("failed to get TiDB service: %v", err)
				return false, nil
			}
			owner := metav1.GetControllerOf(svc)
			if owner == nil {
				log.Logf("tidb service has not been adopted by TidbCluster yet")
				return false, nil
			}
			framework.ExpectEqual(metav1.IsControlledBy(svc, tc), true, "Expected tidb service owner is TidbCluster")
			framework.ExpectEqual(svc.Spec.ClusterIP, oldSvc.Spec.ClusterIP, "tidb service ClusterIP should be stable across adopting and updating")
			return true, nil
		})
		framework.ExpectNoError(err, "failed to wait for TidbCluster managed svc to be ready: %q", tc.Name)

		ginkgo.By("Updating TiDB service")
		trafficPolicy := corev1.ServiceExternalTrafficPolicyTypeLocal
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.TiDB.Service.Type = corev1.ServiceTypeNodePort
			tc.Spec.TiDB.Service.ExternalTrafficPolicy = &trafficPolicy
			tc.Spec.TiDB.Service.Annotations = map[string]string{
				"test": "test",
			}
			return nil
		})
		framework.ExpectNoError(err, "failed to update tidb service of TidbCluster: %q", tc.Name)

		ginkgo.By("Waiting for the TiDB service to be synced")
		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			svc, err := c.CoreV1().Services(ns).Get(controller.TiDBMemberName(tcName), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return false, err
				}
				log.Logf("error get TiDB service: %v", err)
				return false, nil
			}
			if isEqual, err := gomega.Equal(corev1.ServiceTypeNodePort).Match(svc.Spec.Type); !isEqual {
				log.Logf("tidb service type is not %s, %v", corev1.ServiceTypeNodePort, err)
				return false, nil
			}
			if isEqual, err := gomega.Equal(trafficPolicy).Match(svc.Spec.ExternalTrafficPolicy); !isEqual {
				log.Logf("tidb service traffic policy is not %s, %v", svc.Spec.ExternalTrafficPolicy, err)
				return false, nil
			}
			if haveKV, err := gomega.HaveKeyWithValue("test", "test").Match(svc.Annotations); !haveKV {
				log.Logf("tidb service has no annotation test=test, %v", err)
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "failed to wait for TidbCluster managed svc to be ready: %q", tc.Name)
	})

	// Basic IT for managed in TidbCluster CR
	// TODO: deploy pump through CR in backup and restore
	// TODO: Add pump configmap rolling-update case
	ginkgo.It("should adopt helm created pump with TidbCluster CR", func() {
		ginkgo.By("Deploy initial tc")
		tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, "pump", "admin", utilimage.TiDBV4)
		tcCfg.Resources["pd.replicas"] = "1"
		tcCfg.Resources["tikv.replicas"] = "1"
		tcCfg.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)

		ginkgo.By("Deploy pump using helm")
		err := oa.DeployAndCheckPump(&tcCfg)
		framework.ExpectNoError(err, "failed to deploy pump for TidbCluster: %v", tcCfg)

		tc, err := cli.PingcapV1alpha1().TidbClusters(tcCfg.Namespace).Get(tcCfg.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get TidbCluster: %v", tcCfg)

		// If using advanced statefulset, we must upgrade all Kubernetes statefulsets to advanced statefulsets first.
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			log.Logf("advanced statefulset enabled, upgrading all sts to asts")
			stsList, err := c.AppsV1().StatefulSets(ns).List(metav1.ListOptions{})
			framework.ExpectNoError(err, "failed to list statefulsets in ns %s", ns)
			for _, sts := range stsList.Items {
				_, err = astsHelper.Upgrade(c, asCli, &sts)
				framework.ExpectNoError(err, "failed to upgrade statefulset %s/%s", sts.Namespace, sts.Name)
			}
		}

		oldPumpSet, err := stsGetter.StatefulSets(tc.Namespace).Get(controller.PumpMemberName(tc.Name), metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get statefulset for pump: %q", tc.Name)

		oldRev := oldPumpSet.Status.CurrentRevision
		framework.ExpectEqual(oldPumpSet.Status.UpdateRevision, oldRev, "Expected pump is not upgrading")

		ginkgo.By("Update tc.spec.pump")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			pullPolicy := corev1.PullIfNotPresent
			updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
			tc.Spec.Pump = &v1alpha1.PumpSpec{
				BaseImage: "pingcap/tidb-binlog",
				ComponentSpec: v1alpha1.ComponentSpec{
					Version:         &tcCfg.ClusterVersion,
					ImagePullPolicy: &pullPolicy,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									PodAffinityTerm: corev1.PodAffinityTerm{
										Namespaces:  []string{tcCfg.Namespace},
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
				Config: tcconfig.New(map[string]interface{}{
					"addr":               "0.0.0.0:8250",
					"gc":                 7,
					"data-dir":           "/data",
					"heartbeat-interval": 2,
				}),
			}
			return nil
		})
		framework.ExpectNoError(err, "failed to update TidbCluster: %q", tc.Name)

		ginkgo.By("Wait for pump sts to be controlled by tc")
		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
			pumpSet, err := stsGetter.StatefulSets(tc.Namespace).Get(controller.PumpMemberName(tc.Name), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, err
			}
			if err != nil {
				log.Logf("error get pump statefulset: %v", err)
				return false, nil
			}
			if !metav1.IsControlledBy(pumpSet, tc) {
				log.Logf("expect pump statefulset adopted by tidbcluster, still waiting...")
				return false, nil
			}
			// The desired state encoded in CRD should be exactly same with the one created by helm chart
			framework.ExpectEqual(pumpSet.Status.CurrentRevision, oldRev, "Expected no rolling-update when adopting pump statefulset")
			framework.ExpectEqual(pumpSet.Status.UpdateRevision, oldRev, "Expected no rolling-update when adopting pump statefulset")

			cmName := member.FindConfigMapVolume(&pumpSet.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.PumpMemberName(tc.Name))
			})
			if cmName == "" {
				log.Fail("cannot find configmap used by pump statefulset")
			}
			pumpConfigMap, err := c.CoreV1().ConfigMaps(tc.Namespace).Get(cmName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, err
			}
			if err != nil {
				log.Logf("error get pump configmap: %v", err)
				return false, nil
			}
			if !metav1.IsControlledBy(pumpConfigMap, tc) {
				log.Logf("expect pump configmap adopted by tidbcluster, still waiting...")
				return false, nil
			}

			pumpPeerSvc, err := c.CoreV1().Services(tc.Namespace).Get(controller.PumpPeerMemberName(tc.Name), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, err
			}
			if err != nil {
				log.Logf("error get pump peer service: %v", err)
				return false, nil
			}
			if !metav1.IsControlledBy(pumpPeerSvc, tc) {
				log.Logf("expect pump peer service adopted by tidbcluster, still waiting...")
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "failed to wait for pump to be controlled by TidbCluster: %q", tc.Name)
	})

	ginkgo.It("should migrate from helm to CR", func() {
		ginkgo.By("Deploy initial tc")
		tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, "helm-migration", "admin", utilimage.TiDBV4)
		tcCfg.Resources["pd.replicas"] = "1"
		tcCfg.Resources["tikv.replicas"] = "1"
		tcCfg.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)

		tc, err := cli.PingcapV1alpha1().TidbClusters(tcCfg.Namespace).Get(tcCfg.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get TidbCluster: %v", tcCfg)

		ginkgo.By("Discovery service should be reconciled by tidb-operator")
		discoveryName := controller.DiscoveryMemberName(tc.Name)
		discoveryDeploy, err := c.AppsV1().Deployments(tc.Namespace).Get(discoveryName, metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get discovery deployment for TidbCluster: %q", tc.Name)
		WaitObjectToBeControlledByOrDie(genericCli, discoveryDeploy, tc, 5*time.Minute)

		err = utils.WaitForDeploymentComplete(c, discoveryDeploy, log.Logf, 10*time.Second, 5*time.Minute)
		framework.ExpectNoError(err, "waiting for discovery deployment timeout, should be healthy after managed by tidb-operator: %v", discoveryDeploy)

		ginkgo.By("Delete the discovery deployment and wait for recreation by tc")
		err = genericCli.Delete(context.TODO(), discoveryDeploy)
		framework.ExpectNoError(err, "failed to delete discovery deployment: %v", discoveryDeploy)

		err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
			_, err := c.AppsV1().Deployments(tc.Namespace).Get(discoveryName, metav1.GetOptions{})
			if err != nil {
				log.Logf("wait discovery deployment get created again: %v", err)
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "Discovery Deployment should be recovered by tidb-operator after deletion")

		ginkgo.By("Change configmaps in TidbCluster CR in-place")
		// TODO: modify other cases to manage TiDB configmap in CRD by default
		setNameToRevision := map[string]string{
			controller.PDMemberName(tc.Name):   "",
			controller.TiKVMemberName(tc.Name): "",
			controller.TiDBMemberName(tc.Name): "",
		}

		for setName := range setNameToRevision {
			oldSet, err := stsGetter.StatefulSets(tc.Namespace).Get(setName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected to get statefulset %s", setName)

			oldRev := oldSet.Status.CurrentRevision
			framework.ExpectEqual(oldSet.Status.UpdateRevision, oldRev, "Expected statefulset %s is not upgrading", setName)

			setNameToRevision[setName] = oldRev
		}

		tc, err = cli.PingcapV1alpha1().TidbClusters(tcCfg.Namespace).Get(tcCfg.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected to get tidbcluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
			tc.Spec.TiDB.Config = v1alpha1.NewTiDBConfig()
			tc.Spec.TiDB.ConfigUpdateStrategy = &updateStrategy
			tc.Spec.TiKV.Config = v1alpha1.NewTiKVConfig()
			tc.Spec.TiKV.ConfigUpdateStrategy = &updateStrategy
			tc.Spec.PD.Config = v1alpha1.NewPDConfig()
			tc.Spec.PD.ConfigUpdateStrategy = &updateStrategy
			return nil
		})
		framework.ExpectNoError(err, "failed to update configs of TidbCluster: %q", tc.Name)

		ginkgo.By("Wait for a while to ensure no rolling-update happens to pd, tikv, tidb sts")
		err = wait.PollImmediate(10*time.Second, 2*time.Minute, func() (bool, error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(tcCfg.Namespace).Get(tcCfg.ClusterName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected get tidbcluster")
			framework.ExpectEqual(tc.Status.PD.Phase, v1alpha1.NormalPhase, "PD should not be updated")
			framework.ExpectEqual(tc.Status.TiKV.Phase, v1alpha1.NormalPhase, "TiKV should not be updated")
			framework.ExpectEqual(tc.Status.TiDB.Phase, v1alpha1.NormalPhase, "TiDB should not be updated")

			for setName, oldRev := range setNameToRevision {
				newSet, err := stsGetter.StatefulSets(tc.Namespace).Get(setName, metav1.GetOptions{})
				framework.ExpectNoError(err, "Expected get tidb statefulset")
				framework.ExpectEqual(newSet.Status.CurrentRevision, oldRev, "Expected no rolling-update of %s when manage config in-place", setName)
				framework.ExpectEqual(newSet.Status.UpdateRevision, oldRev, "Expected no rolling-update of %s when manage config in-place", setName)
			}
			return false, nil
		})
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "Unexpected error when checking tidb statefulset will not get rolling-update: %v", err)

		ginkgo.By("Check configmaps controlled by tc")
		err = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
			for setName := range setNameToRevision {
				newSet, err := stsGetter.StatefulSets(tc.Namespace).Get(setName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				cmName := member.FindConfigMapVolume(&newSet.Spec.Template.Spec, func(name string) bool {
					return strings.HasPrefix(name, setName)
				})
				if cmName == "" {
					log.Failf("cannot find configmap used by sts %s", setName)
				}
				usingCm, err := c.CoreV1().ConfigMaps(tc.Namespace).Get(cmName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if !metav1.IsControlledBy(usingCm, tc) {
					log.Logf("configmap %q of sts %q not controlled by tc %q, still waiting...", usingCm, setName, tc.Name)
					return false, nil
				}
			}
			return true, nil
		})
		framework.ExpectNoError(err, "failed to wait for configmaps to be controlled by TidbCluster")
	})

	ginkgo.It("should manage tidb monitor normally", func() {
		ginkgo.By("Deploy initial tc")
		tc := fixture.GetTidbCluster(ns, "monitor-test", utilimage.TiDBV4)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
		framework.ExpectNoError(err, "Expected create tidbcluster")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected get tidbcluster")

		ginkgo.By("Deploy tidbmonitor")
		tm := fixture.NewTidbMonitor("monitor-test", ns, tc, true, true, false)
		pvpDelete := corev1.PersistentVolumeReclaimDelete
		tm.Spec.PVReclaimPolicy = &pvpDelete
		_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(tm)
		framework.ExpectNoError(err, "Expected tidbmonitor deployed success")
		err = tests.CheckTidbMonitor(tm, cli, c, fw)
		framework.ExpectNoError(err, "Expected tidbmonitor checked success")

		ginkgo.By("Check tidbmonitor pv label")
		err = wait.Poll(5*time.Second, 2*time.Minute, func() (done bool, err error) {
			pvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(monitor.GetMonitorFirstPVCName(tm.Name), metav1.GetOptions{})
			if err != nil {
				log.Logf("failed to get monitor pvc")
				return false, nil
			}
			pvName := pvc.Spec.VolumeName
			pv, err := c.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
			if err != nil {
				log.Logf("failed to get monitor pv")
				return false, nil
			}

			value, existed := pv.Labels[label.ComponentLabelKey]
			if !existed || value != label.TiDBMonitorVal {
				log.Logf("label %q not synced", label.ComponentLabelKey)
				return false, nil
			}
			value, existed = pv.Labels[label.InstanceLabelKey]
			if !existed || value != "monitor-test" {
				log.Logf("label %q not synced", label.InstanceLabelKey)
				return false, nil
			}
			value, existed = pv.Labels[label.NameLabelKey]
			if !existed || value != "tidb-cluster" {
				log.Logf("label %q not synced", label.NameLabelKey)
				return false, nil
			}
			value, existed = pv.Labels[label.ManagedByLabelKey]
			if !existed || value != label.TiDBOperator {
				log.Logf("label %q not synced", label.ManagedByLabelKey)
				return false, nil
			}
			if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
				return false, fmt.Errorf("pv[%s]'s policy is not \"Delete\"", pv.Name)
			}
			return true, nil
		})
		framework.ExpectNoError(err, "tidbmonitor pv label error")

		// update TidbMonitor and check whether portName is updated and the nodePort is unchanged
		ginkgo.By("Update tidbmonitor service")
		err = controller.GuaranteedUpdate(genericCli, tm, func() error {
			tm.Spec.Prometheus.Service.Type = corev1.ServiceTypeNodePort
			retainPVP := corev1.PersistentVolumeReclaimRetain
			tm.Spec.PVReclaimPolicy = &retainPVP
			return nil
		})
		framework.ExpectNoError(err, "update tidbmonitor service type error")

		var targetPort int32
		err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
			prometheusSvc, err := c.CoreV1().Services(ns).Get(fmt.Sprintf("%s-prometheus", tm.Name), metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if len(prometheusSvc.Spec.Ports) != 3 {
				return false, nil
			}
			if prometheusSvc.Spec.Type != corev1.ServiceTypeNodePort {
				return false, nil
			}
			targetPort = prometheusSvc.Spec.Ports[0].NodePort
			return true, nil
		})
		framework.ExpectNoError(err, "failed to wait for tidbmonitor service to be changed")

		err = controller.GuaranteedUpdate(genericCli, tm, func() error {
			newPortName := "any-other-word"
			tm.Spec.Prometheus.Service.PortName = &newPortName
			return nil
		})
		framework.ExpectNoError(err, "update tidbmonitor service portName error")

		pvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(monitor.GetMonitorFirstPVCName(tm.Name), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected fetch tidbmonitor pvc success")
		pvName := pvc.Spec.VolumeName
		err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
			prometheusSvc, err := c.CoreV1().Services(ns).Get(fmt.Sprintf("%s-prometheus", tm.Name), metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if len(prometheusSvc.Spec.Ports) != 3 {
				return false, nil
			}
			if prometheusSvc.Spec.Type != corev1.ServiceTypeNodePort {
				log.Logf("prometheus service type haven't be changed")
				return false, nil
			}
			if prometheusSvc.Spec.Ports[0].Name != "any-other-word" {
				log.Logf("prometheus port name haven't be changed")
				return false, nil
			}
			if prometheusSvc.Spec.Ports[0].NodePort != targetPort {
				return false, nil
			}
			pv, err := c.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
				log.Logf("prometheus PersistentVolumeReclaimPolicy haven't be changed")
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "second update tidbmonitor service error")

		ginkgo.By("Verify tidbmonitor and tidbcluster PVReclaimPolicy won't affect each other")
		err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			tm, err = cli.PingcapV1alpha1().TidbMonitors(ns).Get(tm.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if *tc.Spec.PVReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
				log.Logf("tidbcluster PVReclaimPolicy changed into %v", *tc.Spec.PVReclaimPolicy)
				return true, nil
			}
			if *tm.Spec.PVReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
				log.Logf("tidbmonitor PVReclaimPolicy changed into %v", *tm.Spec.PVReclaimPolicy)
				return true, nil
			}
			return false, nil
		})
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "tidbmonitor and tidbcluster PVReclaimPolicy should not affect each other")

		ginkgo.By("Delete tidbmonitor")
		err = cli.PingcapV1alpha1().TidbMonitors(tm.Namespace).Delete(tm.Name, &metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete tidbmonitor failed")
		err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
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

	ginkgo.It("can be paused and resumed", func() {
		ginkgo.By("Deploy initial tc")
		tcName := "paused"
		tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV4Prev)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		err := genericCli.Create(context.TODO(), tc)
		framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

		podListBeforePaused, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{})
		framework.ExpectNoError(err, "failed to list pods in ns %s", ns)

		ginkgo.By("Pause the tidb cluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Paused = true
			return nil
		})
		framework.ExpectNoError(err, "failed to pause TidbCluster: %q", tc.Name)

		ginkgo.By(fmt.Sprintf("upgrade tc version to %q", utilimage.TiDBV4))
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Version = utilimage.TiDBV4
			return nil
		})
		framework.ExpectNoError(err, "failed to upgrade TidbCluster version: %q", tc.Name)

		ginkgo.By("Check pods are not changed when the tidb cluster is paused for 3 min")
		err = utilpod.WaitForPodsAreChanged(c, podListBeforePaused.Items, 3*time.Minute)
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "Pods are changed when the tidb cluster is paused")

		ginkgo.By("Resume the tidb cluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Paused = false
			return nil
		})
		framework.ExpectNoError(err, "failed to resume TidbCluster: %q", tc.Name)

		ginkgo.By("Check the tidb cluster will be upgraded")
		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
			listOptions := metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(label.New().Instance(tcName).Component(label.TiKVLabelVal).Labels()).String(),
			}
			podList, err := c.CoreV1().Pods(ns).List(listOptions)
			if err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
			for _, pod := range podList.Items {
				for _, c := range pod.Spec.Containers {
					if c.Name == v1alpha1.TiKVMemberType.String() {
						if c.Image == tc.TiKVImage() {
							return true, nil
						}
					}
				}
			}
			return false, nil
		})
		framework.ExpectNoError(err, "failed to wait for tikv upgraded: %q", tc.Name)
	})

	ginkgo.Context("[Feature: AutoFailover]", func() {
		// TODO: explain purpose of this case
		ginkgo.It("should clear TiDB failureMembers when scale TiDB to zero", func() {
			ginkgo.By("Deploy initial tc with bad tidb pre-start script")
			tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, "tidb-scale", "admin", utilimage.TiDBV4)
			tcCfg.Resources["pd.replicas"] = "3"
			tcCfg.Resources["tikv.replicas"] = "1"
			tcCfg.Resources["tidb.replicas"] = "1"

			tcCfg.TiDBPreStartScript = strconv.Quote("exit 1")
			oa.DeployTidbClusterOrDie(&tcCfg)

			ginkgo.By("check tidb failure member count")
			ns := tcCfg.Namespace
			tcName := tcCfg.ClusterName
			err := wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
				var tc *v1alpha1.TidbCluster
				var err error
				if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
					return false, nil
				}
				if len(tc.Status.TiDB.FailureMembers) == 1 {
					return true, nil
				}
				log.Logf("the number of failed tidb member is not 1 (current: %d)", len(tc.Status.TiDB.FailureMembers))
				return false, nil
			})
			framework.ExpectNoError(err, "expect tidb failure member count = 1")

			ginkgo.By("Scale tidb to 0")
			tcCfg.ScaleTiDB(0)
			oa.ScaleTidbClusterOrDie(&tcCfg)
			err = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
				var tc *v1alpha1.TidbCluster
				var err error
				if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
					return false, nil
				}
				if tc.Status.TiDB.StatefulSet.Replicas != 0 {
					log.Logf("failed to scale tidb member to zero (current: %d)", tc.Status.TiDB.StatefulSet.Replicas)
					return false, nil
				}
				if len(tc.Status.TiDB.FailureMembers) != 0 {
					log.Logf("failed to clear fail member (current: %d)", len(tc.Status.TiDB.FailureMembers))
					return false, nil
				}
				log.Logf("scale tidb member to zero successfully")
				return true, nil
			})
			framework.ExpectNoError(err, "not clear TiDB failureMembers when scale TiDB to zero")
		})
	})

	ginkgo.Context("[Feature: TLS]", func() {
		ginkgo.It("should enable TLS for MySQL Client and between TiDB components", func() {
			tcName := "tls"

			ginkgo.By("Installing tidb CA certificate")
			err := installTiDBIssuer(ns, tcName)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Installing tidb server and client certificate")
			err = installTiDBCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate")

			ginkgo.By("Installing tidbInitializer client certificate")
			err = installTiDBInitializerCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidbInitializer client certificate")

			ginkgo.By("Installing dashboard client certificate")
			err = installPDDashboardCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install dashboard client certificate")

			ginkgo.By("Installing tidb components certificates")
			err = installTiDBComponentsCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb components certificates")

			ginkgo.By("Creating tidb cluster with TLS enabled")
			dashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName)
			tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV4Prev)
			tc.Spec.PD.Replicas = 3
			tc.Spec.PD.TLSClientSecretName = &dashTLSName
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			tc.Spec.Pump = &v1alpha1.PumpSpec{
				Replicas:             1,
				BaseImage:            "pingcap/tidb-binlog",
				ResourceRequirements: fixture.WithStorage(fixture.BurstableSmall, "1Gi"),
				Config: tcconfig.New(map[string]interface{}{
					"addr": "0.0.0.0:8250",
				}),
			}
			err = genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			ginkgo.By("Ensure Dashboard use custom secret")
			foundSecretName := false
			pdSts, err := stsGetter.StatefulSets(ns).Get(controller.PDMemberName(tcName), metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get statefulsets for pd: %q", tc.Name)
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
			framework.ExpectNoError(err, "failed to create secret for TidbInitializer: %v", initSecret)

			ti := fixture.GetTidbInitializer(ns, tcName, initName, initPassWDName, initTLSName)
			err = genericCli.Create(context.TODO(), ti)
			framework.ExpectNoError(err, "failed to create TidbInitializer: %v", ti)

			source := &tests.TidbClusterConfig{
				Namespace:      ns,
				ClusterName:    tcName,
				OperatorTag:    cfg.OperatorTag,
				ClusterVersion: utilimage.TiDBV4,
			}
			targetTcName := "tls-target"
			targetTc := fixture.GetTidbCluster(ns, targetTcName, utilimage.TiDBV4)
			targetTc.Spec.PD.Replicas = 1
			targetTc.Spec.TiKV.Replicas = 1
			targetTc.Spec.TiDB.Replicas = 1
			err = genericCli.Create(context.TODO(), targetTc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %v", targetTc)
			err = oa.WaitForTidbClusterReady(targetTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster timeout: %v", targetTc)

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
			framework.ExpectNoError(err, "failed to deploy drainer: %v", drainerConfig)
			err = oa.CheckDrainer(drainerConfig, source)
			framework.ExpectNoError(err, "failed to check drainer: %v", drainerConfig)

			ginkgo.By("Inserting data into source db")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, tcName, passwd, true))
			framework.ExpectNoError(err, "insert data into source db timeout")

			ginkgo.By("Checking tidb-binlog works as expected")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, targetTcName, "", false))
			framework.ExpectNoError(err, "check data correct timeout")

			ginkgo.By("Connecting to tidb server to verify the connection is TLS enabled")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns, tcName, passwd))
			framework.ExpectNoError(err, "connect to TLS tidb timeout")

			ginkgo.By("Scaling out tidb cluster")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.Replicas = 5
				tc.Spec.TiKV.Replicas = 5
				tc.Spec.TiDB.Replicas = 3
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster: %q", tc.Name)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			ginkgo.By("Scaling in tidb cluster")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.Replicas = 3
				tc.Spec.TiKV.Replicas = 3
				tc.Spec.TiDB.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster: %q", tc.Name)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			ginkgo.By("Upgrading tidb cluster")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBV4
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster: %q", tc.Name)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)
		})

		ginkgo.It("should enable TLS for MySQL Client and between Heterogeneous TiDB components", func() {
			tcName := "origintls"
			heterogeneousTcName := "heterogeneoustls"

			ginkgo.By("Installing tidb CA certificate")
			err := installTiDBIssuer(ns, tcName)
			framework.ExpectNoError(err, "failed to generate tidb issuer template")

			ginkgo.By("Installing tidb server and client certificate")
			err = installTiDBCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate")

			ginkgo.By("Installing heterogeneous tidb server and client certificate")
			err = installHeterogeneousTiDBCertificates(ns, heterogeneousTcName, tcName)
			framework.ExpectNoError(err, "failed to install heterogeneous tidb server and client certificate")

			ginkgo.By("Installing separate tidbInitializer client certificate")
			err = installTiDBInitializerCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install separate tidbInitializer client certificate")

			ginkgo.By("Installing separate dashboard client certificate")
			err = installPDDashboardCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install separate dashboard client certificate")

			ginkgo.By("Installing tidb components certificates")
			err = installTiDBComponentsCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb components certificates")
			err = installHeterogeneousTiDBComponentsCertificates(ns, heterogeneousTcName, tcName)
			framework.ExpectNoError(err, "failed to install heterogeneous tidb components certificates")

			ginkgo.By("Creating tidb cluster")
			dashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName)
			tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV4)
			tc.Spec.PD.Replicas = 1
			tc.Spec.PD.TLSClientSecretName = &dashTLSName
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			tc.Spec.Pump = &v1alpha1.PumpSpec{
				Replicas:             1,
				BaseImage:            "pingcap/tidb-binlog",
				ResourceRequirements: fixture.WithStorage(fixture.BurstableSmall, "1Gi"),
				Config: tcconfig.New(map[string]interface{}{
					"addr": "0.0.0.0:8250",
				}),
			}
			err = genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			ginkgo.By("Creating heterogeneous tidb cluster")
			heterogeneousTc := fixture.GetTidbCluster(ns, heterogeneousTcName, utilimage.TiDBV4)
			heterogeneousTc.Spec.PD = nil
			heterogeneousTc.Spec.TiKV.Replicas = 1
			heterogeneousTc.Spec.TiDB.Replicas = 1
			heterogeneousTc.Spec.TiFlash = &v1alpha1.TiFlashSpec{Replicas: 1,
				BaseImage: "pingcap/tiflash", StorageClaims: []v1alpha1.StorageClaim{
					{Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10G"),
						},
					}},
				}}
			heterogeneousTc.Spec.Cluster = &v1alpha1.TidbClusterRef{
				Name: tcName,
			}

			heterogeneousTc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			heterogeneousTc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			err = genericCli.Create(context.TODO(), heterogeneousTc)
			framework.ExpectNoError(err, "failed to create heterogeneous TidbCluster: %v", heterogeneousTc)

			ginkgo.By("Waiting heterogeneous tls tidb cluster ready")
			err = oa.WaitForTidbClusterReady(heterogeneousTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			ginkgo.By("Checking heterogeneous tls tidb cluster status")
			err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
				var err error
				if _, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(heterogeneousTc.Name, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, heterogeneousTc.Name, err)
					return false, nil
				}
				log.Logf("start check heterogeneous cluster storeInfo: %s/%s", ns, heterogeneousTc.Name)
				pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(c, fw, ns, tcName, true)
				framework.ExpectNoError(err, "create pdClient error")
				defer cancel()
				storeInfo, err := pdClient.GetStores()
				if err != nil {
					log.Logf("failed to get stores, %v", err)
				}
				if storeInfo.Count != 3 {
					log.Logf("failed to check stores (current: %d)", storeInfo.Count)
					return false, nil
				}
				log.Logf("check heterogeneous tc successfully")
				return true, nil
			})
			framework.ExpectNoError(err, "check heterogeneous TidbCluster timeout: %v", heterogeneousTc)

			ginkgo.By("Ensure Dashboard use custom secret")
			foundSecretName := false
			pdSts, err := stsGetter.StatefulSets(ns).Get(controller.PDMemberName(tcName), metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get statefulsets for pd")
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
			framework.ExpectNoError(err, "failed to create secret for TidbInitializer: %v", initSecret)

			ti := fixture.GetTidbInitializer(ns, tcName, initName, initPassWDName, initTLSName)
			err = genericCli.Create(context.TODO(), ti)
			framework.ExpectNoError(err, "failed to create TidbInitializer: %v", ti)

			source := &tests.TidbClusterConfig{
				Namespace:      ns,
				ClusterName:    tcName,
				OperatorTag:    cfg.OperatorTag,
				ClusterVersion: utilimage.TiDBV4,
			}
			targetTcName := "tls-target"
			targetTc := fixture.GetTidbCluster(ns, targetTcName, utilimage.TiDBV4)
			targetTc.Spec.PD.Replicas = 1
			targetTc.Spec.TiKV.Replicas = 1
			targetTc.Spec.TiDB.Replicas = 1
			err = genericCli.Create(context.TODO(), targetTc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %v", targetTc)
			err = oa.WaitForTidbClusterReady(targetTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			drainerConfig := &tests.DrainerConfig{
				DrainerName:       "origintls-drainer",
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
			framework.ExpectNoError(err, "failed to deploy drainer: %v", drainerConfig)
			err = oa.CheckDrainer(drainerConfig, source)
			framework.ExpectNoError(err, "failed to check drainer: %v", drainerConfig)

			ginkgo.By("Inserting data into source db")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, tcName, passwd, true))
			framework.ExpectNoError(err, "insert data into source db timeout")

			ginkgo.By("Checking tidb-binlog works as expected")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, targetTcName, "", false))
			framework.ExpectNoError(err, "check data correct timeout")

			ginkgo.By("Connecting to tidb server to verify the connection is TLS enabled")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns, tcName, passwd))
			framework.ExpectNoError(err, "connect to TLS tidb timeout")
		})
	})

	ginkgo.It("should ensure changing TiDB service annotations won't change TiDB service type NodePort", func() {
		ginkgo.By("Deploy initial tc")
		// Create TidbCluster with NodePort to check whether node port would change
		nodeTc := fixture.GetTidbCluster(ns, "nodeport", utilimage.TiDBV4)
		nodeTc.Spec.PD.Replicas = 1
		nodeTc.Spec.TiKV.Replicas = 1
		nodeTc.Spec.TiDB.Replicas = 1
		nodeTc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
			ServiceSpec: v1alpha1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
			},
		}
		err := genericCli.Create(context.TODO(), nodeTc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(nodeTc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster ready")

		// expect tidb service type is Nodeport
		// NOTE: should we poll here or just check for once? because tc is ready
		var s *corev1.Service
		err = wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
			s, err = c.CoreV1().Services(ns).Get("nodeport-tidb", metav1.GetOptions{})
			if err != nil {
				log.Logf(err.Error())
				return false, nil
			}
			if s.Spec.Type != corev1.ServiceTypeNodePort {
				return false, fmt.Errorf("nodePort tidbcluster tidb service type isn't NodePort")
			}
			return true, nil
		})
		framework.ExpectNoError(err, "wait for tidb service sync timeout")
		oldPorts := s.Spec.Ports

		ensureSvcNodePortUnchangedFor1Min := func() {
			err := wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
				s, err := c.CoreV1().Services(ns).Get("nodeport-tidb", metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if s.Spec.Type != corev1.ServiceTypeNodePort {
					return false, err
				}
				for _, dport := range s.Spec.Ports {
					for _, eport := range oldPorts {
						if dport.Port == eport.Port && dport.NodePort != eport.NodePort {
							return false, fmt.Errorf("nodePort tidbcluster tidb service NodePort changed")
						}
					}
				}
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "service NodePort should not change in 1 minute")
		}

		ginkgo.By("Make sure tidb service NodePort doesn't changed")
		ensureSvcNodePortUnchangedFor1Min()

		ginkgo.By("Change tidb service annotation")
		nodeTc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get("nodeport", metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get TidbCluster")
		err = controller.GuaranteedUpdate(genericCli, nodeTc, func() error {
			nodeTc.Spec.TiDB.Service.Annotations = map[string]string{
				"foo": "bar",
			}
			return nil
		})
		framework.ExpectNoError(err, "failed to change tidb service annotation")

		// check whether the tidb svc have updated
		err = wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
			s, err := c.CoreV1().Services(ns).Get("nodeport-tidb", metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if s.Annotations == nil {
				return false, nil
			}
			v, ok := s.Annotations["foo"]
			if !ok {
				return false, nil
			}
			if v != "bar" {
				return false, fmt.Errorf("tidb svc annotation foo not equal bar")
			}
			return true, nil
		})
		framework.ExpectNoError(err, "failed wait for tidb service sync")

		ginkgo.By("Make sure tidb service NodePort doesn't changed")
		// check whether NodePort have changed for 1 min
		ensureSvcNodePortUnchangedFor1Min()
		log.Logf("tidbcluster tidb service NodePort haven't changed after update")
	})

	ginkgo.Context("[Feature: Heterogeneous Cluster]", func() {
		ginkgo.It("should join heterogeneous cluster into an existing cluster", func() {
			// Create TidbCluster with NodePort to check whether node port would change
			ginkgo.By("Deploy origin tc")
			originTc := fixture.GetTidbCluster(ns, "origin", utilimage.TiDBV4)
			originTc.Spec.PD.Replicas = 1
			originTc.Spec.TiKV.Replicas = 1
			originTc.Spec.TiDB.Replicas = 1
			err := genericCli.Create(context.TODO(), originTc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(originTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			ginkgo.By("Deploy heterogeneous tc")
			heterogeneousTc := fixture.GetTidbCluster(ns, "heterogeneous", utilimage.TiDBV4)
			heterogeneousTc.Spec.PD = nil
			heterogeneousTc.Spec.TiKV.Replicas = 1
			heterogeneousTc.Spec.TiDB.Replicas = 1
			heterogeneousTc.Spec.TiFlash = &v1alpha1.TiFlashSpec{Replicas: 1,
				BaseImage: "pingcap/tiflash", StorageClaims: []v1alpha1.StorageClaim{
					{Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10G"),
						},
					}},
				}}
			heterogeneousTc.Spec.Cluster = &v1alpha1.TidbClusterRef{
				Name: originTc.Name,
			}
			err = genericCli.Create(context.TODO(), heterogeneousTc)
			framework.ExpectNoError(err, "Expected Heterogeneous TiDB cluster created")
			err = oa.WaitForTidbClusterReady(heterogeneousTc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err, "Expected Heterogeneous TiDB cluster ready")

			ginkgo.By("Wait for heterogeneous tc to join origin tc")
			err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
				var err error
				if _, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(heterogeneousTc.Name, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, heterogeneousTc.Name, err)
					return false, nil
				}
				log.Logf("start check heterogeneous cluster storeInfo: %s/%s", ns, heterogeneousTc.Name)
				pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(c, fw, ns, originTc.Name, false)
				framework.ExpectNoError(err, "create pdClient error")
				defer cancel()
				storeInfo, err := pdClient.GetStores()
				if err != nil {
					log.Logf("failed to get stores, %v", err)
				}
				if storeInfo.Count != 3 {
					log.Logf("failed to check stores (current: %d)", storeInfo.Count)
					return false, nil
				}
				log.Logf("check heterogeneous tc successfully")
				return true, nil
			})
			framework.ExpectNoError(err, "failed to wait heterogeneous tc to join origin tc")
		})
	})

	ginkgo.It("[Feature: CDC]", func() {
		ginkgo.By("Creating cdc cluster")
		fromTc := fixture.GetTidbCluster(ns, "cdc-source", utilimage.TiDBV4)
		fromTc.Spec.PD.Replicas = 3
		fromTc.Spec.TiKV.Replicas = 3
		fromTc.Spec.TiDB.Replicas = 2
		fromTc.Spec.TiCDC = &v1alpha1.TiCDCSpec{
			BaseImage: "pingcap/ticdc",
			Replicas:  3,
		}
		err := genericCli.Create(context.TODO(), fromTc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(fromTc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster ready")

		ginkgo.By("Creating cdc-sink cluster")
		toTc := fixture.GetTidbCluster(ns, "cdc-sink", utilimage.TiDBV4)
		toTc.Spec.PD.Replicas = 1
		toTc.Spec.TiKV.Replicas = 1
		toTc.Spec.TiDB.Replicas = 1
		err = genericCli.Create(context.TODO(), toTc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(toTc, 30*time.Minute, 5*time.Second)
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
		framework.ExpectNoError(err, "failed to create change feed task: %s, %v", string(data), err)

		ginkgo.By("Inserting data to cdc cluster")
		err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, fromTCName, "", false))
		framework.ExpectNoError(err, "insert data to cdc cluster timeout")

		ginkgo.By("Checking cdc works as expected")
		err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, toTCName, "", false))
		framework.ExpectNoError(err, "check cdc timeout")

		log.Logf("CDC works as expected")
	})

	ginkgo.Context("when stores number is equal to 3", func() {
		ginkgo.It("forbid to scale in TiKV and the state of all stores are up", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "scale-in-tikv", utilimage.TiDBV4)
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
			framework.ExpectNoError(err, "Expected create tidbcluster")
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected get tidbcluster")

			ginkgo.By("Scale in tikv to 2 replicas")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.TiKV.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err, "failed to scale in tikv")

			ginkgo.By("Expect up stores number stays 3")
			pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(c, fw, ns, tc.Name, false)
			framework.ExpectNoError(err, "create pdClient error")
			defer cancel()
			storesInfo, err := pdClient.GetStores()
			framework.ExpectNoError(err, "get stores info error")

			_ = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
				framework.ExpectEqual(storesInfo.Count, 3, "Expect number of stores is 3")
				for _, store := range storesInfo.Stores {
					framework.ExpectEqual(store.Store.StateName, "Up", "Expect state of stores are Up")
				}
				return false, nil
			})
		})
	})

	ginkgo.It("TiKV should mount multiple pvc", func() {
		ginkgo.By("Deploy initial tc with addition")
		clusterName := "tidb-multiple-pvc-scale"
		tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBV4)
		tc.Spec.TiKV.StorageVolumes = []v1alpha1.StorageVolume{
			{
				Name:        "wal",
				StorageSize: "2Gi",
				MountPath:   "/var/lib/wal",
			},
			{
				Name:        "titan",
				StorageSize: "2Gi",
				MountPath:   "/var/lib/titan",
			},
		}
		tc.Spec.TiDB.StorageVolumes = []v1alpha1.StorageVolume{
			{
				Name:        "log",
				StorageSize: "2Gi",
				MountPath:   "/var/log",
			},
		}
		tc.Spec.PD.StorageVolumes = []v1alpha1.StorageVolume{
			{
				Name:        "log",
				StorageSize: "2Gi",
				MountPath:   "/var/log",
			},
		}

		tc.Spec.PD.Config.Set("log.file.filename", "/var/log/tidb/tidb.log")
		tc.Spec.PD.Config.Set("log.level", "warn")
		tc.Spec.TiDB.Config.Set("log.file.max-size", "300")
		tc.Spec.TiDB.Config.Set("log.file.max-days", "1")
		tc.Spec.TiDB.Config.Set("log.file.filename", "/var/log/tidb/tidb.log")
		tc.Spec.TiDB.Config.Set("log.level", "warn")
		tc.Spec.TiDB.Config.Set("log.file.max-size", "300")
		tc.Spec.TiDB.Config.Set("log.file.max-days", "1")
		tc.Spec.TiKV.Config.Set("rocksdb.wal-dir", "/var/lib/wal")
		tc.Spec.TiKV.Config.Set("titan.dirname", "/var/lib/titan")
		tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, clusterName, "admin", utilimage.TiDBV4)
		tcCfg.Resources["pd.replicas"] = "1"
		tcCfg.Resources["tikv.replicas"] = "4"
		tcCfg.Resources["tidb.replicas"] = "1"
		tcCfg.Clustrer = tc

		log.Logf("deploying tidb cluster [%s/%s]", tcCfg.Namespace, tcCfg.ClusterName)
		oa.DeployTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)

		ginkgo.By("Scale multiple pvc tidb cluster")
		tcCfg.ScaleTiKV(3)
		oa.UpgradeTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)

		ginkgo.By("Scale out multiple pvc tidb cluster")
		tcCfg.ScaleTiKV(4)
		oa.UpgradeTidbClusterOrDie(&tcCfg)
		oa.CheckTidbClusterStatusOrDie(&tcCfg)
	})

	// test cases for tc upgrade
	ginkgo.Context("upgrade works correctly", func() {
		ginkgo.It("for tc and components version", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "upgrade-version", utilimage.TiDBV4Prev)
			pvRetain := corev1.PersistentVolumeReclaimRetain
			tc.Spec.PVReclaimPolicy = &pvRetain
			tc.Spec.PD.StorageClassName = pointer.StringPtr("local-storage")
			tc.Spec.TiKV.StorageClassName = pointer.StringPtr("local-storage")
			tc.Spec.TiDB.StorageClassName = pointer.StringPtr("local-storage")
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Update tc version")
			err := controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBV4
				return nil
			})
			framework.ExpectNoError(err, "failed to update tc version to %q", utilimage.TiDBV4)
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", tc.Namespace, tc.Name)

			ginkgo.By("Update components version")
			componentVersion := utilimage.TiDBV4Prev
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.Version = pointer.StringPtr(componentVersion)
				tc.Spec.TiKV.Version = pointer.StringPtr(componentVersion)
				tc.Spec.TiDB.Version = pointer.StringPtr(componentVersion)
				return nil
			})
			framework.ExpectNoError(err, "failed to update components version to %q", componentVersion)
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", tc.Namespace, tc.Name)

			ginkgo.By("Check components version")
			pdSts := mustGetSts(genericCli, ns, controller.PDMemberName(tc.Name))
			pdImage := fmt.Sprintf("pingcap/pd:%s", componentVersion)
			framework.ExpectEqual(pdSts.Spec.Template.Spec.Containers[0].Image, pdImage, "pd sts image should be %q", pdImage)
			tikvSts := mustGetSts(genericCli, ns, controller.TiKVMemberName(tc.Name))
			tikvImage := fmt.Sprintf("pingcap/tikv:%s", componentVersion)
			framework.ExpectEqual(tikvSts.Spec.Template.Spec.Containers[0].Image, tikvImage, "tikv sts image should be %q", tikvImage)
			tidbSts := mustGetSts(genericCli, ns, controller.TiDBMemberName(tc.Name))
			tidbImage := fmt.Sprintf("pingcap/tidb:%s", componentVersion)
			// the 0th container for tidb pod is slowlog, which runs busybox
			framework.ExpectEqual(tidbSts.Spec.Template.Spec.Containers[1].Image, tidbImage, "tidb sts image should be %q", tidbImage)
		})

		ginkgo.It("for configuration update", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "update-config", utilimage.TiDBV4)
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Update components configuration")
			err := controller.GuaranteedUpdate(genericCli, tc, func() error {
				pdCfg := v1alpha1.NewPDConfig()
				tikvCfg := v1alpha1.NewTiKVConfig()
				tidbCfg := v1alpha1.NewTiDBConfig()
				pdCfg.Set("level", "info")
				tikvCfg.Set("level", "info")
				tidbCfg.Set("level", "info")
				tc.Spec.PD.Config = pdCfg
				tc.Spec.TiKV.Config = tikvCfg
				tc.Spec.TiDB.Config = tidbCfg
				return nil
			})
			framework.ExpectNoError(err, "failed to update components configuration")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", tc.Namespace, tc.Name)

			ginkgo.By("Check components configuration")
			pdSts := mustGetSts(genericCli, ns, controller.PDMemberName(tc.Name))
			pdCmName := member.FindConfigMapVolume(&pdSts.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.PDMemberName(tc.Name))
			})
			pdCm := mustGetCm(genericCli, ns, pdCmName)
			gomega.Expect(pdCm.Data["config-file"]).To(gomega.ContainSubstring("level = \"info\""))

			tikvSts := mustGetSts(genericCli, ns, controller.TiKVMemberName(tc.Name))
			tikvCmName := member.FindConfigMapVolume(&tikvSts.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.TiKVMemberName(tc.Name))
			})
			tikvCm := mustGetCm(genericCli, ns, tikvCmName)
			gomega.Expect(tikvCm.Data["config-file"]).To(gomega.ContainSubstring("level = \"info\""))

			tidbSts := mustGetSts(genericCli, ns, controller.TiDBMemberName(tc.Name))
			tidbCmName := member.FindConfigMapVolume(&tidbSts.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.TiDBMemberName(tc.Name))
			})
			tidbCm := mustGetCm(genericCli, ns, tidbCmName)
			gomega.Expect(tidbCm.Data["config-file"]).To(gomega.ContainSubstring("level = \"info\""))
		})

		// this case merge scale-in/scale-out into one case, may seems a little bit dense
		// when scale-in, replica is first set to 5 and changed to 3
		// when scale-out, replica is first set to 3 and changed to 5
		ginkgo.Context("while concurrently scale PD", func() {
			operation := []string{"in", "out"}
			for _, op := range operation {
				ginkgo.It(op, func() {
					ginkgo.By("Deploy initial tc")
					tcName := fmt.Sprintf("scale-%s-pd-concurrently", op)
					tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV4Prev)
					if op == "in" {
						tc.Spec.PD.Replicas = 5
					} else {
						tc.Spec.PD.Replicas = 3
					}
					utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

					ginkgo.By("Upgrade PD version")
					err := controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.PD.Version = pointer.StringPtr(utilimage.TiDBV4)
						return nil
					})
					framework.ExpectNoError(err, "failed to update PD version to %q", utilimage.TiDBV4)

					ginkgo.By(fmt.Sprintf("Wait for PD phase is %q", v1alpha1.UpgradePhase))
					err = wait.PollImmediate(10*time.Second, 3*time.Minute, func() (bool, error) {
						tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tcName, metav1.GetOptions{})
						if err != nil {
							log.Logf("failed to get TidbCluster %s/%s: %v", tc.Namespace, tcName, err)
							return false, nil
						}
						if tc.Status.PD.Phase != v1alpha1.UpgradePhase {
							log.Logf("tc.Status.PD.Phase = %q, not %q yet", tc.Status.PD.Phase, v1alpha1.UpgradePhase)
							return false, nil
						}
						return true, nil
					})
					framework.ExpectNoError(err, "wait for PD phase failed")

					ginkgo.By(fmt.Sprintf("Scale %s PD while in %q phase", op, v1alpha1.UpgradePhase))
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						if op == "in" {
							tc.Spec.PD.Replicas = 3
						} else {
							tc.Spec.PD.Replicas = 5
						}
						return nil
					})
					framework.ExpectNoError(err, "failed to scale %s PD", op)
					err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %s/%s", tc.Namespace, tc.Name)

					ginkgo.By("Check PD replicas")
					tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tcName, metav1.GetOptions{})
					framework.ExpectNoError(err, "failed to get TidbCluster %s/%s: %v", tc.Namespace, tcName, err)
					if op == "in" {
						framework.ExpectEqual(int(tc.Spec.PD.Replicas), 3)
					} else {
						framework.ExpectEqual(int(tc.Spec.PD.Replicas), 5)
					}
				})
			}
		})

		// similar to PD scale-in/scale-out case above, need to check no evict leader scheduler left
		ginkgo.Context("while concurrently scale TiKV", func() {
			operation := []string{"in", "out"}
			for _, op := range operation {
				ginkgo.It(op, func() {
					ginkgo.By("Deploy initial tc")
					tcName := fmt.Sprintf("scale-%s-tikv-concurrently", op)
					tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV4Prev)
					if op == "in" {
						tc.Spec.TiKV.Replicas = 4
					} else {
						tc.Spec.TiKV.Replicas = 3
					}
					utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

					ginkgo.By("Upgrade TiKV version")
					err := controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.TiKV.Version = pointer.StringPtr(utilimage.TiDBV4)
						return nil
					})
					framework.ExpectNoError(err, "failed to update TiKV version to %q", utilimage.TiDBV4)

					ginkgo.By(fmt.Sprintf("Wait for TiKV phase is %q", v1alpha1.UpgradePhase))
					err = wait.PollImmediate(10*time.Second, 3*time.Minute, func() (bool, error) {
						tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tcName, metav1.GetOptions{})
						if err != nil {
							log.Logf("failed to get TidbCluster %s/%s: %v", tc.Namespace, tcName, err)
							return false, nil
						}
						if tc.Status.TiKV.Phase != v1alpha1.UpgradePhase {
							log.Logf("tc.Status.TiKV.Phase = %q, not %q yet", tc.Status.TiKV.Phase, v1alpha1.UpgradePhase)
							return false, nil
						}
						return true, nil
					})
					framework.ExpectNoError(err, "wait for TiKV phase failed")

					ginkgo.By(fmt.Sprintf("Scale %s TiKV while in %q phase", op, v1alpha1.UpgradePhase))
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						if op == "in" {
							tc.Spec.TiKV.Replicas = 3
						} else {
							tc.Spec.TiKV.Replicas = 4
						}
						return nil
					})
					framework.ExpectNoError(err, "failed to scale %s TiKV", op)
					err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %s/%s", tc.Namespace, tc.Name)

					ginkgo.By("Check TiKV replicas")
					tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tcName, metav1.GetOptions{})
					framework.ExpectNoError(err, "failed to get TidbCluster %s/%s: %v", tc.Namespace, tcName, err)
					if op == "in" {
						framework.ExpectEqual(int(tc.Spec.TiKV.Replicas), 3)
					} else {
						framework.ExpectEqual(int(tc.Spec.TiKV.Replicas), 4)
					}
				})
			}
		})

		ginkgo.It("with bad PD config, then recover after force upgrading PD", func() {
			ginkgo.By("Deploy initial tc with incorrect PD image")
			tc := fixture.GetTidbCluster(ns, "force-upgrade-pd", utilimage.TiDBV4Prev)
			tc.Spec.PD.BaseImage = "wrong-pd-image"
			err := genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "failed to create TidbCluster %s/%s", tc.Namespace, tc.Name)

			ginkgo.By("Wait for PD Pod exist")
			err = wait.PollImmediate(10*time.Second, 3*time.Minute, func() (bool, error) {
				_, err := getPod(genericCli, tc.Namespace, tc.Name)
				if err != nil {
					log.Logf("failed to get Pod %s/%s: %v", tc.Namespace, tc.Name, err)
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "no Pod found for PD")

			ginkgo.By("Wait for 3 min and no PD Pod should be ready")
			err = wait.PollImmediate(10*time.Second, 3*time.Minute, func() (bool, error) {
				pdPod, err := getPod(genericCli, tc.Namespace, tc.Name)
				framework.ExpectNoError(err, "no Pod found for PD")
				if pdPod.Status.Phase != corev1.PodRunning {
					log.Logf("pdPod.Status.Phase = %q, not %q yet", pdPod.Status.Phase, corev1.PodRunning)
					return false, nil
				}
				return true, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "found PD Pod ready with wrong image")

			ginkgo.By("Update PD Pod to correct image")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.BaseImage = "pingcap/pd"
				return nil
			})
			framework.ExpectNoError(err, "failed to change PD Pod image")

			ginkgo.By("Wait for 3 min and no PD Pod should be ready")
			err = wait.PollImmediate(10*time.Second, 3*time.Minute, func() (bool, error) {
				pdPod, err := getPod(genericCli, tc.Namespace, tc.Name)
				framework.ExpectNoError(err, "no Pod found for PD")
				if pdPod.Status.Phase != corev1.PodRunning {
					log.Logf("pdPod.Status.Phase = %q, not %q yet", pdPod.Status.Phase, corev1.PodRunning)
					return false, nil
				}
				return true, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "found PD Pod ready with wrong image")

			ginkgo.By("Annotate TidbCluster for force upgrade")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Annotations["tidb.pingcap.com/force-upgrade"] = "true"
				tc.Spec.PD.BaseImage = "wrong" // we need to make this right later
				return nil
			})
			framework.ExpectNoError(err, "failed to annotate tc for force upgrade")
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				tc := mustGetTidbCluster(genericCli, tc.Namespace, tc.Name)
				val, exist := tc.Annotations["tidb.pingcap.com/force-upgrade"]
				if !exist {
					log.Logf("annotation tidb.pingcap.com/force-upgrade not exist in tc")
					return false, nil
				}
				framework.ExpectEqual(val, "true", "tc annotation tidb.pingcap.com/force-upgrade is not \"true\", but %q", val)
				return true, nil
			})
			framework.ExpectNoError(err, "failed to wait for annotation tidb.pingcap.com/force-upgrade")

			ginkgo.By("Update PD Pod to correct image")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.BaseImage = "pingcap/pd"
				return nil
			})
			framework.ExpectNoError(err, "failed to change PD Pod image")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
		})
	})
})

// TODO: deprecate this
func newTidbClusterConfig(cfg *tests.Config, ns, clusterName, password, tcVersion string) tests.TidbClusterConfig {
	return tests.TidbClusterConfig{
		Namespace:        ns,
		ClusterName:      clusterName,
		EnablePVReclaim:  false,
		OperatorTag:      cfg.OperatorTag,
		PDImage:          fmt.Sprintf("pingcap/pd:%s", tcVersion),
		TiKVImage:        fmt.Sprintf("pingcap/tikv:%s", tcVersion),
		TiDBImage:        fmt.Sprintf("pingcap/tidb:%s", tcVersion),
		PumpImage:        fmt.Sprintf("pingcap/tidb-binlog:%s", tcVersion),
		StorageClassName: "local-storage",
		Password:         password,
		UserName:         "root",
		InitSecretName:   fmt.Sprintf("%s-set-secret", clusterName),
		BackupSecretName: fmt.Sprintf("%s-backup-secret", clusterName),
		BackupName:       "backup",
		Resources: map[string]string{
			"discovery.resources.limits.cpu":      "1000m",
			"discovery.resources.limits.memory":   "2Gi",
			"discovery.resources.requests.cpu":    "20m",
			"discovery.resources.requests.memory": "20Mi",
			"pd.resources.limits.cpu":             "1000m",
			"pd.resources.limits.memory":          "2Gi",
			"pd.resources.requests.cpu":           "20m",
			"pd.resources.requests.memory":        "20Mi",
			"tikv.resources.limits.cpu":           "2000m",
			"tikv.resources.limits.memory":        "4Gi",
			"tikv.resources.requests.cpu":         "20m",
			"tikv.resources.requests.memory":      "20Mi",
			"tidb.resources.limits.cpu":           "2000m",
			"tidb.resources.limits.memory":        "4Gi",
			"tidb.resources.requests.cpu":         "20m",
			"tidb.resources.requests.memory":      "20Mi",
			"pvReclaimPolicy":                     "Delete",
			"tidb.initSql":                        strconv.Quote("create database e2e;"),
			"discovery.image":                     cfg.OperatorImage,
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
		ClusterVersion:         tcVersion,
	}
}
