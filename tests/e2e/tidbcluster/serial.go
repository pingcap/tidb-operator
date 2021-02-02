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
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/monitor/monitor"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Serial specs describe tests which cannot run in parallel.
var _ = ginkgo.Describe("[Serial]", func() {
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
	var stsGetter typedappsv1.StatefulSetsGetter
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
		stsGetter = c.AppsV1()
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	// tidb-operator with pod admission webhook enabled
	ginkgo.Describe("[Feature: PodAdmissionWebhook]", func() {

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

		ginkgo.It(fmt.Sprintf("should be able to upgrade TiDB Cluster from %s to %s", utilimage.TiDBV4Version, utilimage.TiDBV4UpgradeVersion), func() {
			log.Logf("start to upgrade tidbcluster with pod admission webhook")
			// deploy new cluster and test upgrade and scale-in/out with pod admission webhook
			ginkgo.By(fmt.Sprintf("start initial TidbCluster %q", utilimage.TiDBV4Version))
			tc := fixture.GetTidbCluster(ns, "admission", utilimage.TiDBV4Version)
			tc.Spec.PD.Replicas = 3
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %v", tc)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %v", tc)

			ginkgo.By("Set tikv partition annotation to 1")
			err = setPartitionAnnotation(ns, tc.Name, label.TiKVLabelVal, 1)
			framework.ExpectNoError(err, "set tikv Partition annotation failed")

			ginkgo.By(fmt.Sprintf("Upgrade TidbCluster version to %q", utilimage.TiDBV4UpgradeVersion))
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBV4UpgradeVersion
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster to upgrade tidb version to %v", utilimage.TiDBV4UpgradeVersion)

			ginkgo.By(fmt.Sprintf("wait for tikv-1 pod upgrading to %q", utilimage.TiDBV4UpgradeVersion))
			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				tikvPod, err := c.CoreV1().Pods(ns).Get(fmt.Sprintf("%s-tikv-1", tc.Name), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tikvPod.Spec.Containers[0].Image != fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV4UpgradeVersion) {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "failed to upgrade tikv-1 to %q", utilimage.TiDBV4UpgradeVersion)

			ginkgo.By("Wait to see if tikv sts partition annotation remains 1 for 3 min")
			// TODO: explain the purpose of this testing
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
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "tikv sts partition annotation should remain 1 for 3 min")

			ginkgo.By("Set tc annotations to nil")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Annotations = nil
				return nil
			})
			framework.ExpectNoError(err, "failed to set TidbCluster annotation to nil: %v", tc)

			// TODO: find a more graceful way to check tidbcluster during upgrading
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %v", tc)
		})
	})

	ginkgo.Describe("[Feature: Defaulting and Validating]", func() {
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
			ginkgo.By("Deploy a legacy tc")
			legacyTc := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "created-by-helm",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV4UpgradeVersion),
						},
					},
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV4UpgradeVersion),
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
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBV4UpgradeVersion),
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

			ginkgo.By("Enable webhook in operator")
			ocfg.WebhookEnabled = true
			oa.UpgradeOperatorOrDie(ocfg)
			// now the webhook enabled
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.Image = fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV4UpgradeVersion)
				return nil
			})
			framework.ExpectNoError(err, "Update legacy TidbCluster should not be influenced by validating")
			framework.ExpectEqual(legacyTc.Spec.TiDB.BaseImage, "", "Update legacy tidbcluster should not be influenced by defaulting")

			ginkgo.By("Update TidbCluster to use webhook")
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.BaseImage = "pingcap/tidb"
				legacyTc.Spec.TiKV.BaseImage = "pingcap/tikv"
				legacyTc.Spec.PD.BaseImage = "pingcap/pd"
				legacyTc.Spec.PD.Version = pointer.StringPtr(utilimage.TiDBV4UpgradeVersion)
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster")

			ginkgo.By("Set empty values to test validating")
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.BaseImage = ""
				legacyTc.Spec.PD.Version = pointer.StringPtr("")
				return nil
			})
			framework.ExpectError(err, "Validating should reject empty mandatory fields")

			ginkgo.By("Deploy a new tc with legacy fields")
			// TODO: explain the purpose of this testing
			newTC := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV4UpgradeVersion),
						},
					},
					TiKV: &v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV4UpgradeVersion),
						},
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBV4UpgradeVersion),
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
			framework.ExpectError(err, "Validating should reject legacy fields for newly created cluster")

			ginkgo.By("Deploy a new tc")
			newTC = &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: utilimage.TiDBV4UpgradeVersion,
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
			framework.ExpectNoError(err, "required fields should be set by defaulting")
			// don't have to check all fields, just take some to test if defaulting set
			if empty, err := gomega.BeEmpty().Match(newTC.Spec.TiDB.BaseImage); empty {
				log.Failf("Expected tidb.baseImage has default value set, %v", err)
			}

			ginkgo.By("Update tc with invalid instance label value")
			err = controller.GuaranteedUpdate(genericCli, newTC, func() error {
				newTC.Labels = map[string]string{
					label.InstanceLabelKey: "some-insane-label-value",
				}
				return nil
			})
			framework.ExpectError(err, "Could not set instance label with value other than cluster name")

			ginkgo.By("Update pd replication config in tc")
			err = controller.GuaranteedUpdate(genericCli, newTC, func() error {
				c := v1alpha1.NewPDConfig()
				c.Set("replication.max-replicas", 5)
				newTC.Spec.PD.Config = c
				return nil
			})
			framework.ExpectError(err, "PD replication config is immutable through CR")
		})
	})

	ginkgo.Describe("Upgrading Operator from 1.1.7 to latest", func() {
		var oa tests.OperatorActions
		var ocfg *tests.OperatorConfig
		var operatorVersion string

		ginkgo.BeforeEach(func() {
			operatorVersion = "v1.1.7"
			ocfg = &tests.OperatorConfig{
				Namespace:       ns,
				ReleaseName:     "operator",
				Tag:             operatorVersion,
				Image:           fmt.Sprintf("pingcap/tidb-operator:%s", operatorVersion),
				ImagePullPolicy: v1.PullIfNotPresent,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.DeployReleasedCRDOrDie(operatorVersion)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			tests.CleanReleasedCRDOrDie(operatorVersion)
		})

		ginkgo.It("should not change old TidbCluster", func() {
			ginkgo.By(fmt.Sprintf("deploy original tc %q", utilimage.TiDBV4UpgradeVersion))
			tcName := "tidbcluster"
			tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, tcName, "", utilimage.TiDBV4UpgradeVersion)
			tcCfg.Resources["pd.replicas"] = "3"
			tcCfg.Resources["tikv.replicas"] = "3"
			tcCfg.Resources["tidb.replicas"] = "1"
			tcCfg.Monitor = false
			tcCfg.OperatorTag = operatorVersion
			oa.DeployTidbClusterOrDie(&tcCfg)
			oa.CheckTidbClusterStatusOrDie(&tcCfg)

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

			ginkgo.By("Upgrade tidb-operator and CRDs to the latest version")
			ocfg.Tag = cfg.OperatorTag
			ocfg.Image = cfg.OperatorImage
			oa.InstallCRDOrDie(ocfg)
			oa.UpgradeOperatorOrDie(ocfg)
			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				// confirm the pd Pod haven't been changed
				changed, err := utilpod.PodsAreChanged(c, pdPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify pd pods, err:%v", err)
					return true, nil
				}
				if changed {
					return true, nil
				}
				log.Logf("confirm pd pods haven't been changed this time")

				// confirm the tikv haven't been changed
				changed, err = utilpod.PodsAreChanged(c, tikvPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify tikv pods, err:%v", err)
					return true, nil
				}
				if changed {
					return true, nil
				}
				log.Logf("confirm tikv pods haven't been changed this time")

				// confirm the tidb haven't been changed
				changed, err = utilpod.PodsAreChanged(c, tidbPods)()
				if err != nil {
					log.Logf("ERROR: meet error during verify tidb pods, err:%v", err)
					return true, nil
				}
				if changed {
					return true, nil
				}
				log.Logf("confirm tidb pods haven't been changed this time")

				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "expect pd/tikv/tidb haven't been changed for 5 minutes")
		})

		/*
		  Release: v1.2.0
		  new feature in https://github.com/pingcap/tidb-operator/pull/3440
		  deploy tidbmonitor and upgrade tidb-perator, then tidbmonitor should switch from deployment to statefulset
		*/
		ginkgo.It("should migrate tidbmonitor from deployment to sts", func() {
			ginkgo.By("deploy initial tc")
			tcName := "smooth-tidbcluster"
			tcCfg := newTidbClusterConfig(e2econfig.TestConfig, ns, tcName, "admin", utilimage.TiDBV4UpgradeVersion)
			tcCfg.Resources["pd.replicas"] = "3"
			tcCfg.Resources["tikv.replicas"] = "3"
			tcCfg.Resources["tidb.replicas"] = "1"
			oa.DeployTidbClusterOrDie(&tcCfg)
			oa.CheckTidbClusterStatusOrDie(&tcCfg)

			ginkgo.By("deploy tidb monitor")
			monitorName := "smooth-migrate"
			tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get tidbcluster")
			tm := fixture.NewTidbMonitor(monitorName, ns, tc, true, true, true)
			_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(tm)
			framework.ExpectNoError(err, "Expected tidbmonitor deployed success")
			err = tests.CheckTidbMonitor(tm, cli, c, fw)
			framework.ExpectNoError(err, "Expected tidbmonitor checked success")

			deploymentPvcName := fmt.Sprintf("%s-monitor", monitorName)
			deploymentPvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(deploymentPvcName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected tidbmonitor deployment pvc success")
			oldVolumeName := deploymentPvc.Spec.VolumeName

			ginkgo.By("Upgrade tidb-operator and CRDs to the latest version")
			ocfg.Tag = cfg.OperatorTag
			ocfg.Image = cfg.OperatorImage
			oa.InstallCRDOrDie(ocfg)
			oa.UpgradeOperatorOrDie(ocfg)
			err = tests.CheckTidbMonitor(tm, cli, c, fw)
			framework.ExpectNoError(err, "Expected tidbmonitor checked success under migration")
			err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
				tmSet, err := stsGetter.StatefulSets(ns).Get(monitor.GetMonitorObjectName(tm), metav1.GetOptions{})
				if err != nil {
					log.Logf("ERROR: failed to get statefulset: %s/%s, %v", ns, tmSet, err)
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "Expected tidbmonitor sts success")
			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				newStsPvcName := monitor.GetMonitorFirstPVCName(tm.Name)
				log.Logf("tidbmonitor newStsPvcName:%s", newStsPvcName)
				stsPvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(newStsPvcName, metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						log.Logf("tm[%s/%s]'s first sts pvc not found,tag:%s,image:%s", ns, tm.Name, cfg.OperatorTag, cfg.OperatorImage)
						return false, nil
					}
					log.Logf("ERROR: get tidbmonitor sts pvc err:%v", err)
					return false, nil
				}
				if stsPvc.Spec.VolumeName == oldVolumeName {
					return true, nil
				}
				log.Logf("tidbmonitor sts pv unequal to old deployment pv")
				return false, nil
			})
			framework.ExpectNoError(err, "Expected tidbmonitor sts use pv of old deployment")
			err = tests.CheckTidbMonitor(tm, cli, c, fw)
			framework.ExpectNoError(err, "Expected tidbmonitor checked success")
		})
	})

	ginkgo.Describe("upgrading tidb-operator in the same minor series", func() {
		var oa tests.OperatorActions
		var ocfg *tests.OperatorConfig
		var operatorVersion string

		ginkgo.BeforeEach(func() {
			operatorVersion = "v1.1.0"
			ocfg = &tests.OperatorConfig{
				Namespace:   ns,
				ReleaseName: "operator",
				Tag:         operatorVersion,
				Image:       fmt.Sprintf("pingcap/tidb-operator:%s", operatorVersion),
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.DeployReleasedCRDOrDie(operatorVersion)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
		})

		ginkgo.It("should not trigger rolling-update", func() {
			// TODO: resolve the duplication
			framework.Skipf("duplicated test")
			tcName := "basic"
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, tcName, "", utilimage.TiDBV4UpgradeVersion)
			cluster.Resources["pd.replicas"] = "1"
			cluster.Resources["tikv.replicas"] = "1"
			cluster.Resources["tidb.replicas"] = "1"
			cluster.Monitor = false
			cluster.OperatorTag = operatorVersion
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
