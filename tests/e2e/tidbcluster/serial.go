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
	"encoding/json"
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedpdclient"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"github.com/pingcap/tidb-operator/tests/pkg/mock"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Serial specs describe tests which cannot run in parallel.
var _ = ginkgo.Describe("[tidb-operator][Serial]", func() {
	f := e2eframework.NewDefaultFramework("serial")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var genericCli client.Client
	var aggrCli aggregatorclient.Interface
	var apiExtCli apiextensionsclientset.Interface
	var cfg *tests.Config
	var config *restclient.Config
	var fw portforward.PortForward
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
		mapper, err := apiutil.NewDynamicRESTMapper(config, apiutil.WithLazyDiscovery)
		framework.ExpectNoError(err, "failed to create dynamic RESTMapper")
		genericCli, err = client.New(config, client.Options{
			Scheme: scheme.Scheme,
			Mapper: mapper,
		})
		framework.ExpectNoError(err, "failed to create clientset for controller-runtime")
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for kube-aggregator")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for apiextensions-apiserver")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config for tidb operator")
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	// tidb-operator with pod admission webhook enabled
	ginkgo.Context("[Feature: PodAdmissionWebhook]", func() {

		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:         ns,
				ReleaseName:       "operator",
				Image:             cfg.OperatorImage,
				Tag:               cfg.OperatorTag,
				SchedulerImage:    "k8s.gcr.io/kube-scheduler",
				LogLevel:          "4",
				ImagePullPolicy:   v1.PullIfNotPresent,
				TestMode:          true,
				WebhookEnabled:    true,
				PodWebhookEnabled: true,
				StsWebhookEnabled: true,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie(ocfg)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})

		ginkgo.It("[PodAdmissionWebhook] able to upgrade TiDB Cluster with pod admission webhook", func() {
			log.Logf("start to upgrade tidbcluster with pod admission webhook")
			// deploy new cluster and test upgrade and scale-in/out with pod admission webhook
			tc := fixture.GetTidbCluster(ns, "admission", utilimage.TiDBV3Version)
			tc.Spec.PD.Replicas = 3
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %v", tc)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %v", tc)
			ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", utilimage.TiDBV3Version, utilimage.TiDBV3UpgradeVersion))
			ginkgo.By("Set tikv partition annotation")
			err = setPartitionAnnotation(ns, tc.Name, label.TiKVLabelVal, 1)
			framework.ExpectNoError(err, "set tikv Partition annotation failed")

			ginkgo.By("Upgrade TiDB version")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBV3UpgradeVersion
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster to upgrade tidb version to %v", utilimage.TiDBV3UpgradeVersion)

			err = wait.Poll(5*time.Second, 30*time.Minute, func() (done bool, err error) {
				tikvPod, err := c.CoreV1().Pods(ns).Get(fmt.Sprintf("%s-tikv-1", tc.Name), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tikvPod.Spec.Containers[0].Image != fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV3UpgradeVersion) {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "expect tikv upgrade to tikv-1")
			framework.Logf("tikv have upgraded to tikv-1")
			err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
				tikvSts, err := c.AppsV1().StatefulSets(ns).Get(fmt.Sprintf("%s-tikv", tc.Name), metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				strategy := tikvSts.Spec.UpdateStrategy
				if strategy.RollingUpdate != nil && strategy.RollingUpdate.Partition != nil &&
					*strategy.RollingUpdate.Partition == int32(1) {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "tikv Partition should remain 1 for 3 min")
			framework.Logf("tikv Partition remain 1 for 3 min")
			ginkgo.By("Set annotations to nil")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Annotations = nil
				return nil
			})
			framework.ExpectNoError(err, "failed to set TidbCluster annotation to nil: %v", tc)
			framework.Logf("tidbcluster annotation have been cleaned")
			// TODO: find a more graceful way to check tidbcluster during upgrading
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %v", tc)
		})
	})

	ginkgo.Context("[Feature: Defaulting and Validating]", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:                 ns,
				ReleaseName:               "operator",
				Image:                     cfg.OperatorImage,
				Tag:                       cfg.OperatorTag,
				SchedulerImage:            "k8s.gcr.io/kube-scheduler",
				LogLevel:                  "4",
				ImagePullPolicy:           v1.PullIfNotPresent,
				TestMode:                  true,
				WebhookEnabled:            false,
				ValidatingEnabled:         true,
				DefaultingEnabled:         true,
				SchedulerReplicas:         tests.IntPtr(0),
				ControllerManagerReplicas: tests.IntPtr(0),
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie(ocfg)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})

		ginkgo.It("should perform defaulting and validating properly", func() {

			ginkgo.By("Resources created before webhook enabled could be operated normally")
			legacyTc := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "created-by-helm",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV3Version),
						},
					},
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV3Version),
						},
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: &v1alpha1.PDSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBV3Version),
						},
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
				},
			}
			var err error
			legacyTc, err = cli.PingcapV1alpha1().TidbClusters(ns).Create(legacyTc)
			// err := genericCli.Create(context.TODO(), legacyTc)
			framework.ExpectNoError(err, "Expected create tidbcluster without defaulting and validating")
			ocfg.WebhookEnabled = true
			oa.UpgradeOperatorOrDie(ocfg)
			// now the webhook enabled
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.Image = fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV3Version)
				return nil
			})
			framework.ExpectNoError(err, "Update legacy tidbcluster should not be influenced by validating")
			framework.ExpectEqual(legacyTc.Spec.TiDB.BaseImage, "", "Update legacy tidbcluster should not be influenced by defaulting")

			ginkgo.By("Resources created before webhook will be checked if user migrate it to use new API")
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.BaseImage = "pingcap/tidb"
				legacyTc.Spec.TiKV.BaseImage = "pingcap/tikv"
				legacyTc.Spec.PD.BaseImage = "pingcap/pd"
				legacyTc.Spec.PD.Version = pointer.StringPtr(utilimage.TiDBV3Version)
				return nil
			})
			framework.ExpectNoError(err, "Expected update tidbcluster")
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.BaseImage = ""
				legacyTc.Spec.PD.Version = pointer.StringPtr("")
				return nil
			})
			framework.ExpectError(err,
				"Validating should reject mandatory fields being empty if the resource has already been migrated to use the new API")

			ginkgo.By("Validating should reject legacy fields for newly created cluster")
			newTC := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV3Version),
						},
					},
					TiKV: &v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV3Version),
						},
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBV3Version),
						},
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
				},
			}
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Create(newTC)
			framework.ExpectError(err,
				"Validating should reject legacy fields for newly created cluster")

			ginkgo.By("Defaulting should set proper default for newly created cluster")
			newTC = &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: utilimage.TiDBV3Version,
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 1,
					},
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 1,
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: &v1alpha1.PDSpec{
						Replicas: 1,
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
				},
			}
			newTC, err = cli.PingcapV1alpha1().TidbClusters(ns).Create(newTC)
			framework.ExpectNoError(err, "Though some required fields are omitted, they will be set by defaulting")
			// don't have to check all fields, just take some to test if defaulting set
			if empty, err := gomega.BeEmpty().Match(newTC.Spec.TiDB.BaseImage); empty {
				log.Failf("Expected tidb.baseImage has default value set, %v", err)
			}

			ginkgo.By("Validating should reject illegal update")
			err = controller.GuaranteedUpdate(genericCli, newTC, func() error {
				newTC.Labels = map[string]string{
					label.InstanceLabelKey: "some-insane-label-value",
				}
				return nil
			})
			framework.ExpectError(err, "Could not set instance label with value other than cluster name")

			err = controller.GuaranteedUpdate(genericCli, newTC, func() error {
				c := v1alpha1.NewPDConfig()
				c.Set("replication.max-replicas", 5)
				newTC.Spec.PD.Config = c
				return nil
			})
			framework.ExpectError(err, "PD replication config is immutable through CR")
		})
	})

	ginkgo.Context("[Feature: AutoScaling]", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:         ns,
				ReleaseName:       "operator",
				Image:             cfg.OperatorImage,
				Tag:               cfg.OperatorTag,
				LogLevel:          "4",
				TestMode:          true,
				WebhookEnabled:    true,
				PodWebhookEnabled: true,
				Features: []string{
					"AutoScaling=true",
				},
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie(ocfg)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})

		ginkgo.It("auto-scaling TidbCluster", func() {
			clusterName := "auto-scaling"
			tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBV3Version)
			tc.Spec.PD.Replicas = 1
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			_, err := cli.PingcapV1alpha1().TidbClusters(ns).Create(tc)
			framework.ExpectNoError(err, "Create TidbCluster error")
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err, "Check TidbCluster error")
			monitor := fixture.NewTidbMonitor("monitor", ns, tc, false, false)

			// Replace Prometheus into Mock Prometheus
			a := e2econfig.TestConfig.E2EImage
			colonIdx := strings.LastIndexByte(a, ':')
			image := a[:colonIdx]
			tag := a[colonIdx+1:]
			monitor.Spec.Prometheus.BaseImage = image
			monitor.Spec.Prometheus.Version = tag

			_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(monitor)
			framework.ExpectNoError(err, "Create TidbMonitor error")
			err = tests.CheckTidbMonitor(monitor, cli, c, fw)
			framework.ExpectNoError(err, "Check TidbMonitor error")
			tac := fixture.GetTidbClusterAutoScaler("auto-scaler", ns, tc, monitor)

			duration := "1m"
			mp := &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiKVMemberType.String(),
				Duration:   duration,
				// The cpu requests of tikv is 100m, so the threshold value would be 60*0.1*3*0.8 = 14.4
				// so we would set Value as "5" for each instance so that the sum in each auto-scaling calculating would be 15
				Value:        "5.0",
				QueryType:    "cpu",
				InstancesPod: []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv mock metrics error")

			var defaultMetricSpec = v1alpha1.CustomMetric{
				MetricSpec: autoscalingv2beta2.MetricSpec{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2beta2.MetricTarget{
							AverageUtilization: pointer.Int32Ptr(80),
						},
					},
				},
			}

			// Scale Tikv To 4 replicas and Check, cpu load threshold 80%
			tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{
				BasicAutoScalerSpec: v1alpha1.BasicAutoScalerSpec{
					MaxReplicas:            4,
					MinReplicas:            pointer.Int32Ptr(3),
					MetricsTimeDuration:    &duration,
					ScaleInIntervalSeconds: pointer.Int32Ptr(100),
				},
			}
			tac.Spec.TiKV.Metrics = []v1alpha1.CustomMetric{}
			tac.Spec.TiKV.Metrics = append(tac.Spec.TiKV.Metrics, defaultMetricSpec)

			_, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Create(tac)
			framework.ExpectNoError(err, "Create TidbMonitorClusterAutoScaler error")

			framework.Logf("start to check auto-scaler ref")
			// check TidbCluster Status AutoScaler Ref
			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Status.AutoScaler == nil {
					return false, nil
				}
				if tc.Status.AutoScaler.Name != tac.Name || tc.Status.AutoScaler.Namespace != tac.Namespace {
					return false, fmt.Errorf("wrong tidbcluster auto-scaler ref[%s/%s]", tc.Status.AutoScaler.Namespace, tc.Status.AutoScaler.Name)
				}
				return true, nil
			})
			framework.ExpectNoError(err, "Check Auto-Scaler Ref failed")

			pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(c, fw, ns, clusterName, false)
			framework.ExpectNoError(err, "create pdapi error")
			defer cancel()
			var firstScaleTimestamp int64

			// check tikv scale out to 4 and annotations
			err = wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
				stac, err := cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if stac.Status.TiKV != nil && len(stac.Status.TiKV.MetricsStatusList) > 0 {
					metrics := stac.Status.TiKV.MetricsStatusList[0]
					framework.Logf("tikv threshold value: %v, currentValue: %v, recommended replicas: %v",
						*metrics.ThresholdValue, *metrics.CurrentValue, stac.Status.TiKV.RecommendedReplicas)
				}

				tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				// check replicas
				if tc.Spec.TiKV.Replicas != int32(4) {
					framework.Logf("tikv haven't auto-scale to 4 replicas")
					return false, nil
				}
				if len(tc.Status.TiKV.Stores) != 4 {
					framework.Logf("tikv's stores haven't auto-scale to 4")
					return false, nil
				}
				// check annotations
				if tc.Annotations == nil || len(tc.Annotations) < 1 {
					framework.Logf("tc haven't marked any annotation")
					return false, nil
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					framework.Logf("tac haven't marked any annotation")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
				if !ok {
					framework.Logf("tac has no tikv.tidb.pingcap.com/last-autoscaling-timestamp annotation")
					return false, nil
				}
				firstScaleTimestamp, err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					return false, err
				}
				// check store label
				storeId := ""
				for k, v := range tc.Status.TiKV.Stores {
					if v.PodName == util.GetPodName(tc, v1alpha1.TiKVMemberType, int32(3)) {
						storeId = k
						break
					}
				}
				if storeId == "" {
					return false, nil
				}
				sid, err := strconv.ParseUint(storeId, 10, 64)
				if err != nil {
					return false, err
				}
				info, err := pdClient.GetStore(sid)
				if err != nil {
					return false, nil
				}
				for _, label := range info.Store.Labels {
					if label.Key == "specialUse" && label.Value == "hotRegion" {
						return true, nil
					}
				}
				log.Logf("tikv auto-scale out haven't find the special label")
				return false, nil
			})
			framework.ExpectNoError(err, "check tikv auto-scale to 4 error")
			framework.Logf("success to check tikv auto scale-out to 4 replicas")

			// Scale Tikv To 3 replicas and Check, set cpu load with a lower value
			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiKVMemberType.String(),
				Duration:   duration,
				// The cpu requests of tikv is 100m, so the threshold value would be 60*0.1*4*0.8 = 19.2
				// so we would set Value as "1" for each instance so that the sum in each auto-scaling calculating would be 4
				Value:        "1.0",
				QueryType:    "cpu",
				InstancesPod: []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2", "auto-scaling-tikv-3"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv mock metrics error")

			// check tikv scale-in to 3
			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				stac, err := cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if stac.Status.TiKV != nil && len(stac.Status.TiKV.MetricsStatusList) > 0 {
					metrics := stac.Status.TiKV.MetricsStatusList[0]
					framework.Logf("tikv threshold value: %v, currentValue: %v, recommended replicas: %v",
						*metrics.ThresholdValue, *metrics.CurrentValue, stac.Status.TiKV.RecommendedReplicas)
				}

				tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Spec.TiKV.Replicas != 3 {
					framework.Logf("tikv haven't auto-scale to 3 replicas")
					return false, nil
				}
				if len(tc.Status.TiKV.Stores) != 3 {
					framework.Logf("tikv's store haven't auto-scale to 3")
					return false, nil
				}
				if tc.Annotations != nil && len(tc.Annotations) > 0 {
					v, ok := tc.Annotations[label.AnnTiKVAutoScalingOutOrdinals]
					if ok {
						framework.Logf("tikv auto-scale out annotation still exist, value:%v", v)
						var slice []int32
						err := json.Unmarshal([]byte(v), &slice)
						if err != nil {
							framework.Logf("parse tikv auto-scale out annotation failed, err: %v", err)
							return false, nil
						}
						if len(slice) > 0 {
							framework.Logf("tikv auto-scale out annotation still exists")
							return false, nil
						}
					}
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					framework.Logf("tac haven't marked any annotation")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
				if !ok {
					framework.Logf("tac has no tikv.tidb.pingcap.com/last-autoscaling-timestamp annotation")
					return false, nil
				}
				secondTs, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return false, err
				}
				if secondTs == firstScaleTimestamp {
					framework.Logf("tikv haven't scale yet")
					return false, nil
				}
				if secondTs-firstScaleTimestamp < 100 {
					return false, fmt.Errorf("tikv second scale's interval isn't meeting the interval requirement")
				}
				return true, nil
			})
			framework.ExpectNoError(err, "check tikv auto-scale to 3 error")
			framework.Logf("success to check tikv auto scale-in to 3 replicas")

			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiDBMemberType.String(),
				Duration:   duration,
				// The cpu requests of tidb is 100m, so the threshold value would be 60*0.1*2*0.8 = 9.6
				// so we would set Value as "5" for each instance so that the sum in each auto-scaling calculating would be 10
				Value:        "5.0",
				QueryType:    "cpu",
				InstancesPod: []string{"auto-scaling-tidb-0", "auto-scaling-tidb-1"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tidb mock metrics error")

			// Scale Tidb to 3 replicas and Check
			tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
			framework.ExpectNoError(err, "Get TidbCluster AutoScaler err")
			tac.Spec.TiKV = nil
			tac.Spec.TiDB = &v1alpha1.TidbAutoScalerSpec{
				BasicAutoScalerSpec: v1alpha1.BasicAutoScalerSpec{
					MaxReplicas:            3,
					MinReplicas:            pointer.Int32Ptr(2),
					MetricsTimeDuration:    &duration,
					ScaleInIntervalSeconds: pointer.Int32Ptr(100),
				},
			}
			tac.Spec.TiDB.Metrics = []v1alpha1.CustomMetric{}
			tac.Spec.TiDB.Metrics = append(tac.Spec.TiDB.Metrics, defaultMetricSpec)
			_, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Update(tac)
			framework.ExpectNoError(err, "Update TidbMonitorClusterAutoScaler error")

			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				stac, err := cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if stac.Status.TiDB != nil && len(stac.Status.TiDB.MetricsStatusList) > 0 {
					metrics := stac.Status.TiDB.MetricsStatusList[0]
					framework.Logf("tidb threshold value: %v, currentValue: %v, recommended replicas: %v",
						*metrics.ThresholdValue, *metrics.CurrentValue, stac.Status.TiDB.RecommendedReplicas)
				}

				tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Spec.TiDB.Replicas != 3 {
					log.Logf("tidb haven't auto-scaler to 3 replicas")
					return false, nil
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					log.Logf("tac haven't marked any annotations")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
				if !ok {
					log.Logf("tac haven't marked tidb auto-scaler timstamp annotation")
					return false, nil
				}
				firstScaleTimestamp, err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					return false, err
				}
				return true, nil
			})
			framework.ExpectNoError(err, "check tidb auto-scale to 3 error")
			framework.Logf("success to check tidb auto scale-out to 3 replicas")

			// Scale Tidb to 2 replicas and Check
			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiDBMemberType.String(),
				Duration:   duration,
				// The cpu requests of tidb is 100m, so the threshold value would be 60*0.1*2*0.8 = 9.6
				// so we would set Value as "1" for each instance so that the sum in each auto-scaling calculating would be 3
				Value:        "1.0",
				QueryType:    "cpu",
				InstancesPod: []string{"auto-scaling-tidb-0", "auto-scaling-tidb-1", "auto-scaling-tidb-2"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tidb mock metrics error")

			// Scale Tidb to 2 Replicas and Check
			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				stac, err := cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if stac.Status.TiDB != nil && len(stac.Status.TiDB.MetricsStatusList) > 0 {
					metrics := stac.Status.TiDB.MetricsStatusList[0]
					framework.Logf("tidb threshold value: %v, currentValue: %v, recommended replicas: %v",
						*metrics.ThresholdValue, *metrics.CurrentValue, stac.Status.TiDB.RecommendedReplicas)
				}

				tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Spec.TiDB.Replicas != 2 {
					log.Logf("tidb haven't auto-scaler to 2 replicas")
					return false, nil
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					log.Logf("tac haven't marked any annotations")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
				if !ok {
					log.Logf("tac haven't marked tidb auto-scale timestamp")
					return false, nil
				}
				secondTs, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return false, err
				}
				if secondTs == firstScaleTimestamp {
					log.Logf("tidb haven't scale yet")
					return false, nil
				}
				if secondTs-firstScaleTimestamp < 100 {
					return false, fmt.Errorf("tikv second scale's interval isn't meeting the interval requirement")
				}
				return true, nil
			})
			framework.ExpectNoError(err, "check tidb auto-scale to 2 error")
			framework.Logf("success to check auto scale-in tidb to 2 replicas")

			// Check Storage Auto-Scaling
			framework.Logf("start to check tikv storage auto-scaling")
			mp = &mock.MonitorParams{
				Name: tc.Name,
				// The mock capacity size for the tikv storage
				Value:       fmt.Sprintf("%v", 1024*1024*1024),
				QueryType:   "storage",
				StorageType: "capacity",
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv capacity storage size error")

			mp = &mock.MonitorParams{
				Name: tc.Name,
				// The mock capacity size for the tikv storage
				Value:       fmt.Sprintf("%v", 1024*1024),
				QueryType:   "storage",
				StorageType: "available",
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv available storage size error")

			tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
			framework.ExpectNoError(err, "Get TidbCluster AutoScaler err")
			tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{
				BasicAutoScalerSpec: v1alpha1.BasicAutoScalerSpec{
					MaxReplicas: int32(4),
					Metrics: []v1alpha1.CustomMetric{
						{
							MetricSpec: autoscalingv2beta2.MetricSpec{
								Type: autoscalingv2beta2.ResourceMetricSourceType,
								Resource: &autoscalingv2beta2.ResourceMetricSource{
									Name: corev1.ResourceStorage,
								},
							},
							LeastStoragePressurePeriodSeconds:  pointer.Int64Ptr(30),
							LeastRemainAvailableStoragePercent: pointer.Int64Ptr(90),
						},
					},
				},
			}
			tac.Spec.TiDB = nil
			tac.Status.TiKV = nil
			tac.Status.TiDB = nil
			tacCopy := tac.DeepCopy()
			err = controller.GuaranteedUpdate(genericCli, tac, func() error {
				tac.Spec = tacCopy.Spec
				tac.Status = tacCopy.Status
				return nil
			})
			framework.ExpectNoError(err, "Update TidbMonitorClusterAutoScaler error")

			// check tikv scale-out to 4
			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				stac, err := cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if stac.Status.TiKV != nil && len(stac.Status.TiKV.MetricsStatusList) > 0 {
					metrics := stac.Status.TiKV.MetricsStatusList[0]
					framework.Logf("tikv AvailableStorage:%v, BaselineAvailableStorage:%v, CapacityStorage:%v, StoragePressure:%v",
						*metrics.AvailableStorage, *metrics.BaselineAvailableStorage, *metrics.CapacityStorage, *metrics.StoragePressure)
				}

				tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Spec.TiKV.Replicas != 4 {
					framework.Logf("tikv haven't auto-scale to 4 replicas")
					return false, nil
				}
				if len(tc.Status.TiKV.Stores) != 4 {
					framework.Logf("tikv's store haven't auto-scale to 4")
					return false, nil
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					framework.Logf("tac haven't marked any annotation")
					return false, nil
				}
				_, ok := tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
				if !ok {
					framework.Logf("tac has no tikv.tidb.pingcap.com/last-autoscaling-timestamp annotation")
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "check tikv auto-scale to 4 error")
			framework.Logf("success to check tikv auto scale-in to 4 replicas")

			// Clean scaler
			err = cli.PingcapV1alpha1().TidbClusterAutoScalers(tac.Namespace).Delete(tac.Name, &metav1.DeleteOptions{})
			framework.ExpectNoError(err, "Expect to delete auto-scaler ref")
			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Status.AutoScaler != nil {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "expect auto-scaler ref empty after delete auto-scaler")
			framework.Logf("clean scaler mark success")
		})
	})

<<<<<<< HEAD
	ginkgo.Context("[Verify: Upgrading Operator from 1.1.0", func() {
		var oa tests.OperatorActions
		var ocfg *tests.OperatorConfig
		var version string

		ginkgo.BeforeEach(func() {
			version = "v1.1.7"
			ocfg = &tests.OperatorConfig{
				Namespace:   ns,
				ReleaseName: "operator",
				Tag:         version,
				Image:       fmt.Sprintf("pingcap/tidb-operator:%s", version),
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.DeployReleasedCRDOrDie(version)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			tests.CleanReleasedCRDOrDie(version)
		})

		ginkgo.It("Deploy TidbCluster and Upgrade Operator", func() {
			tcName := "tidbcluster"
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, tcName, "", utilimage.TiDBV3Version)
			cluster.Resources["pd.replicas"] = "3"
			cluster.Resources["tikv.replicas"] = "3"
			cluster.Resources["tidb.replicas"] = "1"
			cluster.Monitor = false
			cluster.OperatorTag = version
			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)

			getPods := func(ls string) ([]v1.Pod, error) {
				listOptions := metav1.ListOptions{
					LabelSelector: ls,
				}
				podList, err := c.CoreV1().Pods(ns).List(listOptions)
				if err != nil {
					return nil, err
				}
				return podList.Items, nil
			}

			pdPods, err := getPods(labels.SelectorFromSet(label.New().Instance(tcName).PD().Labels()).String())
			framework.ExpectNoError(err, "failed to get pd pods")

			tikvPods, err := getPods(labels.SelectorFromSet(label.New().Instance(tcName).TiKV().Labels()).String())
			framework.ExpectNoError(err, "failed to get tikv pods")

			tidbPods, err := getPods(labels.SelectorFromSet(label.New().Instance(tcName).TiDB().Labels()).String())
=======
			err := genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			pdPods, err := getPods(labels.SelectorFromSet(label.New().Instance(tcName).PD().Labels()).String(), ns, c)
			framework.ExpectNoError(err, "failed to get pd pods")
			tikvPods, err := getPods(labels.SelectorFromSet(label.New().Instance(tcName).TiKV().Labels()).String(), ns, c)
			framework.ExpectNoError(err, "failed to get tikv pods")
			tidbPods, err := getPods(labels.SelectorFromSet(label.New().Instance(tcName).TiDB().Labels()).String(), ns, c)
>>>>>>> b5e3ee5f... e2e: tidb-operator canary deployments (#3764)
			framework.ExpectNoError(err, "failed to get tidb pods")

			ginkgo.By("Upgrade tidb-operator and CRDs to current version")
			ocfg.Tag = cfg.OperatorTag
			ocfg.Image = cfg.OperatorImage
			oa.InstallCRDOrDie(ocfg)
			oa.UpgradeOperatorOrDie(ocfg)

			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				// confirm the pd Pod haven't been changed
				changed, err := utilpod.PodsAreChanged(c, pdPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify pd pods, err:%v", err)
					return false, err
				}
				if changed {
					return true, nil
				}
				log.Logf("confirm pd pods haven't been changed this time")

				// confirm the tikv haven't been changed
				changed, err = utilpod.PodsAreChanged(c, tikvPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify tikv pods, err:%v", err)
					return false, err
				}
				if changed {
					return true, nil
				}
				log.Logf("confirm tikv pods haven't been changed this time")

				// confirm the tidb haven't been changed
				changed, err = utilpod.PodsAreChanged(c, tidbPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify tidb pods, err:%v", err)
					return false, err
				}
				if changed {
					return true, nil
				}
				log.Logf("confirm tidb pods haven't been changed this time")

				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "expect pd/tikv/tidb haven't been changed for 5 minutes")
		})
	})

<<<<<<< HEAD
	ginkgo.Context("upgrading tidb-operator in the same minor series should not trigger rolling-update", func() {
		var oa tests.OperatorActions
=======
	ginkgo.Describe("Canary Deploy TiDB Operator", func() {
		var oa *tests.OperatorActions
		var ocfg *tests.OperatorConfig

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:       ns,
				ReleaseName:     "operator",
				Image:           cfg.OperatorImage,
				Tag:             cfg.OperatorTag,
				ImagePullPolicy: v1.PullIfNotPresent,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.InstallCRDOrDie(ocfg)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})

		ginkgo.It("Deploy TidbCluster and check the result", func() {
			ginkgo.By(fmt.Sprintf("deploy original tc %q", utilimage.TiDBV4Prev))
			tcName := "tidbcluster1"
			tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV4Prev)
			tc.Spec.PD.Replicas = 1
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1

			err := genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			pdPods, err := getPods(labels.SelectorFromSet(label.New().Instance(tcName).PD().Labels()).String(), ns, c)
			framework.ExpectNoError(err, "failed to get PD pods")

			ginkgo.By("Set --selector=version=old for the default TiDB Operator")
			ocfg.Selector = []string{"version=old"}
			oa.UpgradeOperatorOrDie(ocfg)
			log.Logf("Upgrade operator with --set-string \"selector=version=old\"")

			ginkgo.By("Upgrade TidbCluster 1 version, wait for 2 minutes, check that no rolling update occurs")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBV4
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster 1 to upgrade PD version to %v", utilimage.TiDBV4)

			err = wait.Poll(5*time.Second, 2*time.Minute, func() (done bool, err error) {
				// confirm the TidbCluster 1 PD haven't been changed
				changed, err := utilpod.PodsAreChanged(c, pdPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify TidbCluster 1 PD pods, err:%v", err)
					return false, err
				}
				if changed {
					return true, nil
				}
				log.Logf("confirm TidbCluster 1 PD pods haven't been changed this time")
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "expect TidbCluster 1 PD haven't been changed for 2 minutes")
			log.Logf("Upgrade TidbCluster 1 to new version, but no rolling update occurs")

			ginkgo.By("Set label version=old to TidbCluster 1, check that PD of TidbCluster 1 has been rolling updated")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Labels = map[string]string{"version": "old"}
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster 1 to set label version=old")

			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				// confirm the PD Pods have been changed
				changed, err := utilpod.PodsAreChanged(c, pdPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify PD pods, err:%v", err)
					return false, err
				}
				if changed {
					return true, nil
				}
				log.Logf("PD pods haven't been changed yet")
				return false, nil
			})
			framework.ExpectNoError(err, "expect PD pods have been changed in 5 minutes")
			log.Logf("Upgrade TidbCluster 1 with label version=old and PD pods have been rolling updated.")

			ginkgo.By("Deploy TidbCluster 2 with label version=new")
			tc2Name := "tidbcluster2"
			tc2 := fixture.GetTidbCluster(ns, tc2Name, utilimage.TiDBV4Prev)
			tc2.Spec.PD.Replicas = 1
			tc2.Spec.TiKV.Replicas = 1
			tc2.Spec.TiDB.Replicas = 1
			tc2.Labels = map[string]string{"version": "new"}

			err = genericCli.Create(context.TODO(), tc2)
			framework.ExpectNoError(err, "Expected Tidbcluster 2 be created")
			log.Logf("Finished deploying TidbCluster 2 with label version=new")

			ginkgo.By("Wait for 2 minutes and check that no PD Pod is created")
			err = wait.Poll(5*time.Second, 2*time.Minute, func() (created bool, err error) {
				// confirm the Tidbcluster 2 PD Pods don't be created
				pdPods, err = getPods(labels.SelectorFromSet(label.New().Instance(tc2Name).PD().Labels()).String(), ns, c)
				if err != nil {
					log.Logf("ERROR: meet error during get PD pods, err:%v", err)
					return false, nil
				}
				if len(pdPods) != 0 {
					return true, nil
				}
				log.Logf("Tidbcluster 2 PD pods have not been created yet")
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "No PD Pods created for Tidbcluster 2")
			log.Logf("Confirm that no PD Pods created for Tidbcluster 2")

			ginkgo.By("Deploy TiDB Operator 2 with --selector=version=new")
			ocfg2 := &tests.OperatorConfig{
				Namespace:       ns + "-canary-deploy",
				ReleaseName:     "operator2",
				Image:           cfg.OperatorImage,
				Tag:             cfg.OperatorTag,
				ImagePullPolicy: v1.PullIfNotPresent,
				Selector:        []string{"version=new"},
				//FIXME: AppendReleaseSuffix: true,
			}
			oa.DeployOperatorOrDie(ocfg2)
			log.Logf("Finished deploying TiDB Operator 2 with --selector=version=new")

			ginkgo.By("Check TidbCluster 2 is ready")
			err = oa.WaitForTidbClusterReady(tc2, 10*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster2 ready")
			log.Logf("confirm TidbCluster 2 is ready")

			ginkgo.By("Delete the default TiDB Operator")
			oa.CleanOperatorOrDie(ocfg)
			log.Logf("Finished deleting the default Operator")

			ginkgo.By("Upgrade TiDB version of TidbCluster 2")
			err = controller.GuaranteedUpdate(genericCli, tc2, func() error {
				tc2.Spec.Version = utilimage.TiDBV4
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster 2 to upgrade tidb version to %v", utilimage.TiDBV4)
			log.Logf("Finished upgrading TidbCluster 2")

			err = oa.WaitForTidbClusterReady(tc2, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc2.Name)

			ginkgo.By(fmt.Sprintf("wait for TidbCluster 2 pd-0 pod upgrading to %q", utilimage.TiDBV4))
			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				pdPod, err := c.CoreV1().Pods(ns).Get(fmt.Sprintf("%s-pd-0", tc2.Name), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if pdPod.Spec.Containers[0].Image != fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBV4) {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "failed to upgrade TidbCluster 2 pd-0 to %q", utilimage.TiDBV4)
			log.Logf("Finished upgrading TidbCluster 2")

			ginkgo.By("Deploy the default TiDB Operator with --selector=version=old")
			ocfg = &tests.OperatorConfig{
				Namespace:       ns,
				ReleaseName:     "operator",
				Image:           cfg.OperatorImage,
				Tag:             cfg.OperatorTag,
				ImagePullPolicy: v1.PullIfNotPresent,
				Selector:        []string{"version=old"},
			}
			oa.DeployOperatorOrDie(ocfg)
			log.Logf("Finished deploying TiDB Operator 1 with --selector=version=old")

			ginkgo.By("Delete the TiDB Operator 2")
			oa.CleanOperatorOrDie(ocfg2)
			log.Logf("Finished deleting the Operator 2")

			ginkgo.By("Scale-out TiDB of TidbCluster 1, wait for 5 minuts, check that scale up occurs")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.TiDB.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err, "failed to scale out TidbCluster 1")

			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				tidbStatefulSet, err := c.AppsV1().StatefulSets(ns).Get(fmt.Sprintf("%s-tidb", tc.Name), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if *tidbStatefulSet.Spec.Replicas != 2 {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "expect TiDB in tidbcluster 1 to scale out to 2")
			log.Logf("Succeed to scale out TiDB of TidbCluster 1")
		})
	})
	ginkgo.Describe("upgrading tidb-operator in the same minor series", func() {
		var oa *tests.OperatorActions
>>>>>>> b5e3ee5f... e2e: tidb-operator canary deployments (#3764)
		var ocfg *tests.OperatorConfig
		var version string

		ginkgo.BeforeEach(func() {
			version = "v1.1.0-rc.2"
			ocfg = &tests.OperatorConfig{
				Namespace:   ns,
				ReleaseName: "operator",
				Tag:         version,
				Image:       fmt.Sprintf("pingcap/tidb-operator:%s", version),
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.DeployReleasedCRDOrDie(version)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
		})

		ginkgo.It("basic deployment", func() {
			framework.Skipf("duplicated test")
			tcName := "basic"
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, tcName, "", utilimage.TiDBV3Version)
			cluster.Resources["pd.replicas"] = "1"
			cluster.Resources["tikv.replicas"] = "1"
			cluster.Resources["tidb.replicas"] = "1"
			cluster.Monitor = false
			cluster.OperatorTag = version
			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)

			listOptions := metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(label.New().Instance(tcName).Labels()).String(),
			}
			podList, err := c.CoreV1().Pods(ns).List(listOptions)
			framework.ExpectNoError(err, "failed to list pods in ns %s: %v", ns, listOptions)

			ginkgo.By("Upgrade tidb-operator and CRDs to current version")
			ocfg.Tag = cfg.OperatorTag
			ocfg.Image = cfg.OperatorImage
			oa.InstallCRDOrDie(ocfg)
			oa.UpgradeOperatorOrDie(ocfg)

			ginkgo.By("Wait for pods are not changed in 5 minutes")
			err = utilpod.WaitForPodsAreChanged(c, podList.Items, time.Minute*5)
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "pods should not change in 5 minutes")
		})
	})
})

func setPartitionAnnotation(namespace, tcName, component string, ordinal int) error {
	// add annotation to pause statefulset upgrade process
	output, err := framework.RunKubectl("annotate", "tc", tcName, "-n", namespace, fmt.Sprintf("tidb.pingcap.com/%s-partition=%d", component, ordinal), "--overwrite")
	if err != nil {
		return fmt.Errorf("fail to set annotation for [%s/%s], component: %s, partition: %d, err: %v, output: %s", namespace, tcName, component, ordinal, err, output)
	}
	return nil
}

func getPods(ls string, ns string, c clientset.Interface) ([]v1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: ls,
	}
	podList, err := c.CoreV1().Pods(ns).List(listOptions)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}
