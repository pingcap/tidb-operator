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
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"github.com/pingcap/tidb-operator/tests/pkg/mock"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
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
	/**
	 * StatefulSet or AdvancedStatefulSet getter interface.
	 */

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
		framework.ExpectNoError(err)
		genericCli, err = client.New(config, client.Options{
			Scheme: scheme.Scheme,
			Mapper: mapper,
		})
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
			klog.Info("start to upgrade tidbcluster with pod admission webhook")
			// deploy new cluster and test upgrade and scale-in/out with pod admission webhook
			tc := fixture.GetTidbCluster(ns, "admission", utilimage.TiDBV3Version)
			tc.Spec.PD.Replicas = 3
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
			framework.ExpectNoError(err)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err)
			ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", utilimage.TiDBV3Version, utilimage.TiDBV3UpgradeVersion))
			ginkgo.By("Set tikv partition annotation")
			err = setPartitionAnnotation(ns, tc.Name, label.TiKVLabelVal, 1)
			framework.ExpectNoError(err, "set tikv Partition annotation failed")

			ginkgo.By("Upgrade TiDB version")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBV3UpgradeVersion
				return nil
			})
			framework.ExpectNoError(err)

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
			framework.ExpectNoError(err)
			framework.Logf("tidbcluster annotation have been cleaned")
			// TODO: find a more graceful way to check tidbcluster during upgrading
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err)
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
				e2elog.Failf("Expected tidb.baseImage has default value set, %v", err)
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
				newTC.Spec.PD.Config = &v1alpha1.PDConfig{
					Replication: &v1alpha1.PDReplicationConfig{
						MaxReplicas: func() *uint64 { i := uint64(5); return &i }(),
					},
				}
				return nil
			})
			framework.ExpectError(err, "PD replication config is immutable through CR")
		})
	})

	ginkgo.Context("[Verify: Upgrading Operator from 1.1.0", func() {
		var oa tests.OperatorActions
		var ocfg *tests.OperatorConfig
		var version string

		ginkgo.BeforeEach(func() {
			version = "v1.1.0"
			ocfg = &tests.OperatorConfig{
				Namespace:   ns,
				ReleaseName: "operator",
				Tag:         version,
				Image:       fmt.Sprintf("pingcap/tidb-operator:%s", version),
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			tests.DeployReleasedCRDOrDie(version)
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
					klog.Errorf("meet error during verify pd pods, err:%v", err)
					return true, nil
				}
				if changed {
					return true, nil
				}
				klog.Infof("confirm pd pods haven't been changed this time")

				// confirm the tikv haven't been changed
				changed, err = utilpod.PodsAreChanged(c, tikvPods)()
				if err != nil {
					klog.Errorf("meet error during verify tikv pods, err:%v", err)
					return true, nil
				}
				if changed {
					return true, nil
				}
				klog.Infof("confirm tikv pods haven't been changed this time")

				// confirm the tidb haven't been changed
				changed, err = utilpod.PodsAreChanged(c, tidbPods)()
				if err != nil {
					klog.Errorf("meet error during verify tidb pods, err:%v", err)
					return true, nil
				}
				if changed {
					return true, nil
				}
				klog.Infof("confirm tidb pods haven't been changed this time")

				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "expect pd/tikv/tidb haven't been changed for 5 minutes")
		})
	})

	ginkgo.Context("upgrading tidb-operator in the same minor series should not trigger rolling-update", func() {
		var oa tests.OperatorActions
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
			tests.DeployReleasedCRDOrDie(version)
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
			framework.ExpectNoError(err)

			ginkgo.By("Upgrade tidb-operator and CRDs to current version")
			ocfg.Tag = cfg.OperatorTag
			ocfg.Image = cfg.OperatorImage
			oa.InstallCRDOrDie(ocfg)
			oa.UpgradeOperatorOrDie(ocfg)

			ginkgo.By("Wait for pods are not changed in 5 minutes")
			err = utilpod.WaitForPodsAreChanged(c, podList.Items, time.Minute*5)
			framework.ExpectEqual(err, wait.ErrWaitTimeout)
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
			tc := fixture.GetTidbCluster(ns, clusterName, "nightly")
			tc.Spec.PD.Replicas = 1
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			tc.Spec.PD.Config.PDServerCfg = &v1alpha1.PDServerConfig{
				MetricStorage: pointer.StringPtr("http://monitor-prometheus:9090"),
			}

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

			duration := "60s"
			mp := &mock.MonitorParams{
				Name:                tc.Name,
				KubernetesNamespace: tc.Namespace,
				MemberType:          v1alpha1.TiKVMemberType.String(),
				Duration:            duration,
				// The CPU of TiKV is guaranteed 1000m
				// To reach 50% utilization, the sum of cpu usage time should be at least 60 * 3 * 0.5 = 90
				Value:        "35.0",
				QueryType:    "cpu_usage",
				InstancesPod: []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv cpu usage mock metrics error")

			mp = &mock.MonitorParams{
				Name:                tc.Name,
				KubernetesNamespace: tc.Namespace,
				MemberType:          v1alpha1.TiKVMemberType.String(),
				Duration:            duration,
				Value:               "1.0",
				QueryType:           "cpu_quota",
				InstancesPod:        []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv cpu quota mock metrics error")

			tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{}
			tac.Spec.TiKV.Rules = map[v1.ResourceName]v1alpha1.AutoRule{
				v1.ResourceCPU: {
					MaxThreshold: 0.5,
					MinThreshold: func() *float64 {
						v := 0.2
						return &v
					}(),
				},
			}

			_, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Create(tac)
			framework.ExpectNoError(err, "Create TidbClusterAutoScaler error")

			var autoTc v1alpha1.TidbCluster
			// A new cluster should be created and there is one TiKV replica
			err = wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
				tcList, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).List(metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.AutoInstanceLabelKey, "auto-scaler", label.BaseTCLabelKey, tc.Name),
				})

				if err != nil {
					return false, err
				}

				if len(tcList.Items) < 1 {
					framework.Logf("autoscaling tikv cluster is not created")
					return false, nil
				}

				autoTc = tcList.Items[0]
				if autoTc.Spec.TiKV.Replicas >= 1 {
					framework.Logf("autoscaling tikv cluster tc[%s/%s] created", autoTc.Namespace, autoTc.Name)
					return true, nil
				}

				return false, nil
			})
			framework.ExpectNoError(err, "check create autoscaling tikv cluster error")
			framework.Logf("success to check create autoscaling tikv cluster")

			autoTiKV := fmt.Sprintf("%s-tikv-0", tc.Name)

			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiKVMemberType.String(),
				Duration:   duration,
				// The CPU of TiKV is guaranteed 1000m
				// To reach 50% utilization, the sum of cpu usage time should be at least 60 * 3 * 0.5 = 90
				Value:        "35.0",
				QueryType:    "cpu_usage",
				InstancesPod: []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2", autoTiKV},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv cpu usage mock metrics error")

			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiKVMemberType.String(),
				Duration:   duration,
				// The CPU of TiKV is guaranteed 1000m
				Value:        "1.0",
				QueryType:    "cpu_quota",
				InstancesPod: []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2", autoTiKV},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv cpu quota mock metrics error")

			err = wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
				tc, err = cli.PingcapV1alpha1().TidbClusters(autoTc.Namespace).Get(autoTc.Name, metav1.GetOptions{})

				if err != nil {
					return false, err
				}

				autoTc = *tc

				if tc.Spec.TiKV.Replicas < 2 {
					framework.Logf("autoscaling tikv cluster is not scaled out")
					return false, nil
				}

				if tc.Spec.TiKV.Replicas >= 2 {
					framework.Logf("autoscaling tikv cluster tc[%s/%s] scaled out", autoTc.Namespace, autoTc.Name)
					return true, nil
				}

				return false, nil
			})
			framework.ExpectNoError(err, "check scale out existing autoscaling tikv cluster error")
			framework.Logf("success to check scale out existing autoscaling tikv cluster")

			pods := []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2"}
			for i := int32(0); i < autoTc.Spec.TiKV.Replicas; i++ {
				pods = append(pods, fmt.Sprintf("%s-tikv-%d", autoTc.Name, i))
			}

			mp = &mock.MonitorParams{
				Name:         tc.Name,
				MemberType:   v1alpha1.TiKVMemberType.String(),
				Duration:     duration,
				Value:        "0.0",
				QueryType:    "cpu_usage",
				InstancesPod: pods,
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv cpu usage mock metrics error")

			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiKVMemberType.String(),
				Duration:   duration,
				// The CPU of TiKV is guaranteed 1000m
				Value:        "1.0",
				QueryType:    "cpu_quota",
				InstancesPod: pods,
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv cpu quota mock metrics error")

			err = wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
				tc, err = cli.PingcapV1alpha1().TidbClusters(autoTc.Namespace).Get(autoTc.Name, metav1.GetOptions{})

				if err != nil {
					if errors.IsNotFound(err) {
						return true, nil
					}
					return false, err
				}

				autoTc = *tc

				if tc.Spec.TiKV.Replicas > 0 {
					framework.Logf("autoscaling tikv cluster is not gracefully shutting down")
					return false, nil
				}

				if tc.Spec.TiKV.Replicas <= 0 {
					framework.Logf("autoscaling tikv cluster tc[%s/%s] is shutting down", autoTc.Namespace, autoTc.Name)
					return true, nil
				}

				return false, nil
			})

			framework.ExpectNoError(err, "failed to check shut down autoscaling tikv cluster")
			framework.Logf("success to check shut down autoscaling tikv cluster")

			err = wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
				tcList, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).List(metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.AutoInstanceLabelKey, "auto-scaler", label.BaseTCLabelKey, tc.Name),
				})

				if err != nil {
					return false, err
				}

				if len(tcList.Items) > 0 {
					framework.Logf("autoscaling tikv cluster is not deleted")
					return false, nil
				}

				framework.Logf("autoscaling tikv cluster deleted")
				return true, nil
			})
			framework.ExpectNoError(err, "check delete autoscaling tikv cluster error")
			framework.Logf("success to check delete autoscaling tikv cluster")

			// Clean autoscaler
			err = cli.PingcapV1alpha1().TidbClusterAutoScalers(tac.Namespace).Delete(tac.Name, &metav1.DeleteOptions{})
			framework.ExpectNoError(err, "failed to delete auto-scaler")
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
