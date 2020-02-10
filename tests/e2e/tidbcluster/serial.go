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
	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
)

var _ = ginkgo.Describe("[tidb-operator][Serial]", func() {
	f := framework.NewDefaultFramework("serial")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var aggrCli aggregatorclient.Interface
	var apiExtCli apiextensionsclientset.Interface
	var hc clientset.Interface
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
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config")
		hc = helper.NewHijackClient(c, asCli)
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

	// tidb-operator with AdvancedStatefulSet feature enabled
	ginkgo.Context("[Feature: AdvancedStatefulSet]", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions
		var genericCli client.Client

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:      "pingcap",
				ReleaseName:    "operator",
				Image:          cfg.OperatorImage,
				Tag:            cfg.OperatorTag,
				SchedulerImage: "k8s.gcr.io/kube-scheduler",
				Features: []string{
					"StableScheduling=true",
					"AdvancedStatefulSet=true",
				},
				LogLevel:        "4",
				ImagePullPolicy: v1.PullIfNotPresent,
				TestMode:        true,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie(ocfg)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
			var err error
			genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
			framework.ExpectNoError(err, "failed to create clientset")
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})

		ginkgo.It("Scaling tidb cluster with advanced statefulset", func() {
			clusterName := "deploy"
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, clusterName, "", "")
			cluster.Resources["pd.replicas"] = "3"
			cluster.Resources["tikv.replicas"] = "5"
			cluster.Resources["tidb.replicas"] = "3"
			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)

			scalingTests := tests.GetAstsCases()

			for _, st := range scalingTests {
				tests.AstsScalingTest(st, clusterName, ns, c, hc, cli, genericCli)
			}

			oa.CheckTidbClusterStatusOrDie(&cluster)
		})
	})

	ginkgo.It("[Feature: AdvancedStatefulSet] Upgrade to advanced statefulset", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions
		var genericCli client.Client

		ocfg = &tests.OperatorConfig{
			Namespace:      "pingcap",
			ReleaseName:    "operator",
			Image:          cfg.OperatorImage,
			Tag:            cfg.OperatorTag,
			SchedulerImage: "k8s.gcr.io/kube-scheduler",
			Features: []string{
				"StableScheduling=true",
				"AdvancedStatefulSet=false",
			},
			LogLevel:        "4",
			ImagePullPolicy: v1.PullIfNotPresent,
			TestMode:        true,
		}
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
		ginkgo.By("Installing CRDs")
		oa.CleanCRDOrDie()
		oa.InstallCRDOrDie(ocfg)
		ginkgo.By("Installing tidb-operator without AdvancedStatefulSet feature")
		oa.CleanOperatorOrDie(ocfg)
		oa.DeployOperatorOrDie(ocfg)
		var err error
		genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset")

		defer func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		}()

		tc := fixture.GetTidbCluster(ns, "sts", utilimage.TiDBV3Version)
		err = genericCli.Create(context.TODO(), tc)
		framework.ExpectNoError(err)
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err)

		listOption := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				label.InstanceLabelKey: tc.Name,
			}).String(),
		}
		stsList, err := c.AppsV1().StatefulSets(tc.Namespace).List(listOption)
		framework.ExpectNoError(err)
		if len(stsList.Items) < 3 {
			e2elog.Failf("at least 3 statefulsets must be created, got %d", len(stsList.Items))
		}

		podListBeforeUpgrade, err := c.CoreV1().Pods(tc.Namespace).List(metav1.ListOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Upgrading tidb-operator with AdvancedStatefulSet feature")
		ocfg.Features = []string{
			"StableScheduling=true",
			"AdvancedStatefulSet=true",
		}
		oa.InstallCRDOrDie(ocfg)
		oa.UpgradeOperatorOrDie(ocfg)

		ginkgo.By("Wait for the advanced statefulsets are created and Kubernetes statfulsets are deleted")
		err = wait.PollImmediate(time.Second*5, time.Minute*5, func() (bool, error) {
			advancedStsList, err := asCli.AppsV1().StatefulSets(tc.Namespace).List(listOption)
			if err != nil {
				return false, nil
			}
			if len(advancedStsList.Items) != len(stsList.Items) {
				klog.Infof("advanced statefulsets got %d, expect %d", len(advancedStsList.Items), len(stsList.Items))
				return false, nil
			}
			stsListAfterUpgrade, err := c.AppsV1().StatefulSets(tc.Namespace).List(listOption)
			if err != nil {
				return false, nil
			}
			if len(stsListAfterUpgrade.Items) != 0 {
				klog.Infof("Kubernetes statefulsets got %d, expect %d", len(stsListAfterUpgrade.Items), 0)
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err)

		ginkgo.By("Make sure pods are not affected")
		err = utilpod.WaitForPodsAreNotAffected(c, podListBeforeUpgrade.Items, time.Minute*3)
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "Pods was not affeteced after the operator is upgraded")
	})

	// tidb-operator with pod admission webhook enabled
	ginkgo.Context("[Feature: PodAdmissionWebhook]", func() {

		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:         "pingcap",
				ReleaseName:       "operator",
				Image:             cfg.OperatorImage,
				Tag:               cfg.OperatorTag,
				SchedulerImage:    "k8s.gcr.io/kube-scheduler",
				LogLevel:          "4",
				ImagePullPolicy:   v1.PullIfNotPresent,
				TestMode:          true,
				WebhookEnabled:    true,
				PodWebhookEnabled: true,
				StsWebhookEnabled: false,
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
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "admission", "", "")
			cluster.Resources["pd.replicas"] = "3"
			cluster.Resources["tikv.replicas"] = "3"
			cluster.Resources["tidb.replicas"] = "2"

			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)
			upgradeVersions := cfg.GetUpgradeTidbVersionsOrDie()
			ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", cluster.ClusterVersion, upgradeVersions[0]))
			cluster.UpgradeAll(upgradeVersions[0])
			oa.UpgradeTidbClusterOrDie(&cluster)
			// TODO: find a more graceful way to check tidbcluster during upgrading
			oa.CheckTidbClusterStatusOrDie(&cluster)
			oa.CleanTidbClusterOrDie(&cluster)
		})
	})

	ginkgo.Context("[Feature: Defaulting and Validating]", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:                 "pingcap",
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
					TiDB: v1alpha1.TiDBSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV3Version),
						},
					},
					TiKV: v1alpha1.TiKVSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV3Version),
						},
					},
					PD: v1alpha1.PDSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBV3Version),
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
			legacyTc.Spec.TiDB.Image = fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV3Version)
			legacyTc, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(legacyTc)
			framework.ExpectNoError(err, "Update legacy tidbcluster should not be influenced by validating")
			framework.ExpectEqual(legacyTc.Spec.TiDB.BaseImage, "", "Update legacy tidbcluster should not be influenced by defaulting")

			ginkgo.By("Resources created before webhook will be checked if user migrate it to use new API")
			legacyTc.Spec.TiDB.BaseImage = "pingcap/tidb"
			legacyTc.Spec.TiKV.BaseImage = "pingcap/tikv"
			legacyTc.Spec.PD.BaseImage = "pingcap/pd"
			legacyTc.Spec.PD.Version = pointer.StringPtr(utilimage.TiDBV3Version)
			legacyTc, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(legacyTc)
			framework.ExpectNoError(err, "Expected update tidbcluster")
			legacyTc.Spec.TiDB.BaseImage = ""
			legacyTc.Spec.PD.Version = pointer.StringPtr("")
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(legacyTc)
			framework.ExpectError(err,
				"Validating should reject mandatory fields being empty if the resource has already been migrated to use the new API")

			ginkgo.By("Validating should reject legacy fields for newly created cluster")
			newTC := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBV3Version),
						},
					},
					TiKV: v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBV3Version),
						},
					},
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBV3Version),
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
					TiDB: v1alpha1.TiDBSpec{
						Replicas: 1,
					},
					TiKV: v1alpha1.TiKVSpec{
						Replicas: 1,
					},
					PD: v1alpha1.PDSpec{
						Replicas: 1,
					},
				},
			}
			newTC, err = cli.PingcapV1alpha1().TidbClusters(ns).Create(newTC)
			framework.ExpectNoError(err, "Though some required fields are omitted, they will be set by defaulting")
			// don't have to check all fields, just take some to test if defaulting set
			if empty, err := gomega.BeEmpty().Match(newTC.Spec.TiDB.BaseImage); empty {
				e2elog.Failf("Expected tidb.baseImage has default value set, %v", err)
			}
			if isNil, err := gomega.BeNil().Match(newTC.Spec.TiDB.Config); isNil {
				e2elog.Failf("Expected tidb.config has default value set, %v", err)
			}

			ginkgo.By("Validating should reject illegal update")
			newTC.Labels = map[string]string{
				label.InstanceLabelKey: "some-insane-label-value",
			}
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(newTC)
			framework.ExpectError(err, "Could not set instance label with value other than cluster name")

			newTC.Spec.PD.Config.Replication = &v1alpha1.PDReplicationConfig{
				MaxReplicas: func() *uint64 { i := uint64(5); return &i }(),
			}
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(newTC)
			framework.ExpectError(err, "PD replication config is immutable through CR")
		})
	})

	// tidb-operator with AdvancedStatefulSet feature enabled and pod admission webhook enabled
	ginkgo.Context("[Feature: AdvancedStatefulsetPodAdmissionWebhook]", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions
		var genericCli client.Client

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:      "pingcap",
				ReleaseName:    "operator",
				Image:          cfg.OperatorImage,
				Tag:            cfg.OperatorTag,
				SchedulerImage: "k8s.gcr.io/kube-scheduler",
				Features: []string{
					"StableScheduling=true",
					"AdvancedStatefulSet=true",
				},
				LogLevel:          "4",
				ImagePullPolicy:   v1.PullIfNotPresent,
				TestMode:          true,
				WebhookEnabled:    true,
				PodWebhookEnabled: true,
				StsWebhookEnabled: false,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie(ocfg)
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
			var err error
			genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
			framework.ExpectNoError(err, "failed to create clientset")
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})

		ginkgo.It("able to deploy TiDB Cluster with advanced statefulset", func() {
			clusterName := "deploy"
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, clusterName, "", "")
			cluster.Monitor = false
			cluster.EnablePVReclaim = true
			cluster.Resources["pd.replicas"] = "3"
			cluster.Resources["tikv.replicas"] = "5"
			cluster.Resources["tidb.replicas"] = "3"
			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)

			scalingTests := tests.GetAstsCases()

			for _, st := range scalingTests {
				tests.AstsScalingTest(st, clusterName, ns, c, hc, cli, genericCli)
			}

			oa.CheckTidbClusterStatusOrDie(&cluster)
		})
	})

})
