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
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
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
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"github.com/pingcap/tidb-operator/tests/pkg/mock"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func mustToString(set sets.Int32) string {
	b, err := json.Marshal(set.List())
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Serial specs describe tests which cannot run in parallel.
var _ = ginkgo.Describe("[tidb-operator][Serial]", func() {
	f := e2eframework.NewDefaultFramework("serial")

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
	ginkgo.Context("[Feature: AdvancedStatefulSet][Feature: Webhook]", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions
		var genericCli client.Client

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:      ns,
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

		ginkgo.It("Scaling tidb cluster with advanced statefulset", func() {
			clusterName := "scaling-with-asts"
			tc := fixture.GetTidbClusterWithTiFlash(ns, clusterName, utilimage.TiDBV4Version)
			tc.Spec.PD.Replicas = 3
			tc.Spec.TiKV.Replicas = 5
			tc.Spec.TiDB.Replicas = 5
			tc.Spec.TiFlash.Replicas = 5
			err := genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err)

			scalingTests := []struct {
				name        string
				component   string // tikv,pd,tidb,tiflash
				replicas    int32
				deleteSlots sets.Int32
			}{
				{
					name:        "Scaling in tikv from 5 to 3 by deleting pods 1 and 3",
					component:   "tikv",
					replicas:    3,
					deleteSlots: sets.NewInt32(1, 3),
				},
				{
					name:        "Scaling out tikv from 3 to 4 by adding pod 3",
					component:   "tikv",
					replicas:    4,
					deleteSlots: sets.NewInt32(1),
				},
				{
					name:        "Scaling tikv by adding pod 1 and deleting pod 2",
					component:   "tikv",
					replicas:    4,
					deleteSlots: sets.NewInt32(2),
				},
				{
					name:        "Scaling in tiflash from 5 to 3 by deleting pods 1 and 3",
					component:   "tiflash",
					replicas:    3,
					deleteSlots: sets.NewInt32(1, 3),
				},
				{
					name:        "Scaling out tiflash from 3 to 4 by adding pod 3",
					component:   "tiflash",
					replicas:    4,
					deleteSlots: sets.NewInt32(1),
				},
				{
					name:        "Scaling tiflash by adding pod 1 and deleting pod 2",
					component:   "tiflash",
					replicas:    4,
					deleteSlots: sets.NewInt32(2),
				},
				{
					name:        "Scaling in tidb from 3 to 2 by deleting pod 1",
					component:   "tidb",
					replicas:    2,
					deleteSlots: sets.NewInt32(1),
				},
				{
					name:        "Scaling in tidb from 2 to 0",
					component:   "tidb",
					replicas:    0,
					deleteSlots: sets.NewInt32(),
				},
				{
					name:        "Scaling out tidb from 0 to 2 by adding pods 0 and 2",
					component:   "tidb",
					replicas:    2,
					deleteSlots: sets.NewInt32(1),
				},
				{
					name:        "Scaling tidb from 2 to 4 by deleting pods 2 and adding pods 3, 4 and 5",
					component:   "tidb",
					replicas:    4,
					deleteSlots: sets.NewInt32(1, 2),
				},
				{
					name:        "Scaling out pd from 3 to 5 by adding pods 3, 4",
					component:   "pd",
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				},
				{
					name:        "Scaling in pd from 5 to 3 by deleting pods 0 and 3",
					component:   "pd",
					replicas:    3,
					deleteSlots: sets.NewInt32(0, 3),
				},
				{
					name:        "Scaling out pd from 3 to 5 by adding pods 5 and 6",
					component:   "pd",
					replicas:    5,
					deleteSlots: sets.NewInt32(0, 3),
				},
			}

			for _, st := range scalingTests {
				ginkgo.By(st.name)
				replicas := st.replicas
				stsName := fmt.Sprintf("%s-%s", clusterName, st.component)

				sts, err := hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
				framework.ExpectNoError(err)

				oldPodList := e2esset.GetPodList(c, sts)

				ginkgo.By(fmt.Sprintf("Scaling sts %s/%s to replicas %d and setting deleting pods to %v (old replicas: %d, old delete slots: %v)", ns, stsName, replicas, st.deleteSlots.List(), *sts.Spec.Replicas, helper.GetDeleteSlots(sts).List()))
				tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
				framework.ExpectNoError(err)
				err = controller.GuaranteedUpdate(genericCli, tc, func() error {
					if tc.Annotations == nil {
						tc.Annotations = map[string]string{}
					}
					if st.component == "tikv" {
						tc.Annotations[label.AnnTiKVDeleteSlots] = mustToString(st.deleteSlots)
						tc.Spec.TiKV.Replicas = replicas
					} else if st.component == "tiflash" {
						tc.Annotations[label.AnnTiFlashDeleteSlots] = mustToString(st.deleteSlots)
						tc.Spec.TiFlash.Replicas = replicas
					} else if st.component == "pd" {
						tc.Annotations[label.AnnPDDeleteSlots] = mustToString(st.deleteSlots)
						tc.Spec.PD.Replicas = replicas
					} else if st.component == "tidb" {
						tc.Annotations[label.AnnTiDBDeleteSlots] = mustToString(st.deleteSlots)
						tc.Spec.TiDB.Replicas = replicas
					} else {
						return fmt.Errorf("unsupported component: %v", st.component)
					}
					return nil
				})
				framework.ExpectNoError(err)

				ginkgo.By(fmt.Sprintf("Waiting for all pods of tidb cluster component %s (sts: %s/%s) are in desired state (replicas: %d, delete slots: %v)", st.component, ns, stsName, st.replicas, st.deleteSlots.List()))
				err = wait.PollImmediate(time.Second*5, time.Minute*15, func() (bool, error) {
					// check replicas and delete slots are synced
					sts, err = hc.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
					if err != nil {
						return false, nil
					}
					if *sts.Spec.Replicas != st.replicas {
						klog.Infof("replicas of sts %s/%s is %d, expects %d", ns, stsName, *sts.Spec.Replicas, st.replicas)
						return false, nil
					}
					if !helper.GetDeleteSlots(sts).Equal(st.deleteSlots) {
						klog.Infof("delete slots of sts %s/%s is %v, expects %v", ns, stsName, helper.GetDeleteSlots(sts).List(), st.deleteSlots.List())
						return false, nil
					}
					// check all desired pods are running and ready
					return utilstatefulset.IsAllDesiredPodsRunningAndReady(hc, sts), nil
				})
				framework.ExpectNoError(err)

				ginkgo.By(fmt.Sprintf("Verify other pods of sts %s/%s should not be affected", ns, stsName))
				newPodList := e2esset.GetPodList(c, sts)
				framework.ExpectEqual(len(newPodList.Items), int(*sts.Spec.Replicas))
				for _, newPod := range newPodList.Items {
					for _, oldPod := range oldPodList.Items {
						// if the pod is not new or deleted in scaling, it should not be affected
						if oldPod.Name == newPod.Name && oldPod.UID != newPod.UID {
							framework.Failf("pod %s/%s should not be affected (UID: %s, OLD UID: %s)", newPod.Namespace, newPod.Name, newPod.UID, oldPod.UID)
						}
					}
				}
			}

			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err)
		})
	})

	ginkgo.It("[Feature: AdvancedStatefulSet] Upgrade to advanced statefulset", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions
		var genericCli client.Client

		ocfg = &tests.OperatorConfig{
			Namespace:      ns,
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

		podListBeforeUpgrade, err := c.CoreV1().Pods(tc.Namespace).List(listOption)
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

		ginkgo.By("Make sure pods are not changed")
		err = utilpod.WaitForPodsAreChanged(c, podListBeforeUpgrade.Items, time.Minute*3)
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "Pods are changed after the operator is upgraded")
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
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "admission", "", utilimage.TiDBV3Version)
			cluster.Resources["pd.replicas"] = "3"
			cluster.Resources["tikv.replicas"] = "3"
			cluster.Resources["tidb.replicas"] = "2"

			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)
			ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", cluster.ClusterVersion, utilimage.TiDBV3UpgradeVersion))
			cluster.UpgradeAll(utilimage.TiDBV3UpgradeVersion)
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
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: v1alpha1.PDSpec{
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
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: v1alpha1.PDSpec{
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
					TiDB: v1alpha1.TiDBSpec{
						Replicas: 1,
					},
					TiKV: v1alpha1.TiKVSpec{
						Replicas: 1,
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: v1alpha1.PDSpec{
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
			newTC.Labels = map[string]string{
				label.InstanceLabelKey: "some-insane-label-value",
			}
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(newTC)
			framework.ExpectError(err, "Could not set instance label with value other than cluster name")

			newTC.Spec.PD.Config = &v1alpha1.PDConfig{
				Replication: &v1alpha1.PDReplicationConfig{
					MaxReplicas: func() *uint64 { i := uint64(5); return &i }(),
				},
			}
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(newTC)
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
				InstancesPod: []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv mock metrics error")

			var defaultMetricSpec = autoscalingv2beta2.MetricSpec{
				Type: autoscalingv2beta2.ResourceMetricSourceType,
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2beta2.MetricTarget{
						AverageUtilization: pointer.Int32Ptr(80),
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
			tac.Spec.TiKV.Metrics = []autoscalingv2beta2.MetricSpec{}
			tac.Spec.TiKV.Metrics = append(tac.Spec.TiKV.Metrics, defaultMetricSpec)

			_, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Create(tac)
			framework.ExpectNoError(err, "Create TidbMonitorClusterAutoScaler error")
			pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(c, fw, ns, clusterName, false, nil)
			framework.ExpectNoError(err, "create pdapi error")
			defer cancel()
			var firstScaleTimestamp int64
			err = wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
				tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				// check replicas
				if tc.Spec.TiKV.Replicas != int32(4) {
					klog.Infof("tikv haven't auto-scale to 4 replicas")
					return false, nil
				}
				if len(tc.Status.TiKV.Stores) != 4 {
					klog.Infof("tikv's stores haven't auto-scale to 4")
					return false, nil
				}
				// check annotations
				if tc.Annotations == nil || len(tc.Annotations) < 1 {
					klog.Infof("tc haven't marked any annotation")
					return false, nil
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					klog.Infof("tac haven't marked any annotation")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
				if !ok {
					klog.Infof("tac haven't marked any annotation")
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
				klog.Infof("tikv auto-scale out haven't find the special label")
				return false, nil
			})
			framework.ExpectNoError(err, "check tikv auto-scale to 4 error")
			klog.Info("success to check tikv auto scale-out to 4 replicas")

			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiKVMemberType.String(),
				Duration:   duration,
				// The cpu requests of tikv is 100m, so the threshold value would be 60*0.1*4*0.8 = 19.2
				// so we would set Value as "1" for each instance so that the sum in each auto-scaling calculating would be 4
				Value:        "1.0",
				InstancesPod: []string{"auto-scaling-tikv-0", "auto-scaling-tikv-1", "auto-scaling-tikv-2", "auto-scaling-tikv-3"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tikv mock metrics error")

			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Spec.TiKV.Replicas != 3 {
					klog.Info("tikv haven't auto-scale to 3 replicas")
					return false, nil
				}
				if len(tc.Status.TiKV.Stores) != 3 {
					klog.Info("tikv's store haven't auto-scale to 3")
					return false, nil
				}
				if tc.Annotations != nil && len(tc.Annotations) > 0 {
					_, ok := tc.Annotations[label.AnnTiKVAutoScalingOutOrdinals]
					if ok {
						klog.Infof("tikv auto-scale out annotation still exists")
						return false, nil
					}
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					klog.Infof("tc haven't marked any annotation")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
				if !ok {
					klog.Infof("tac haven't marked any annotation")
					return false, nil
				}
				secondTs, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return false, err
				}
				if secondTs == firstScaleTimestamp {
					klog.Info("tikv haven't scale yet")
					return false, nil
				}
				if secondTs-firstScaleTimestamp < 100 {
					return false, fmt.Errorf("tikv second scale's interval isn't meeting the interval requirement")
				}
				return true, nil
			})
			framework.ExpectNoError(err, "check tikv auto-scale to 3 error")
			klog.Info("success to check tikv auto scale-in to 3 replicas")

			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiDBMemberType.String(),
				Duration:   duration,
				// The cpu requests of tidb is 100m, so the threshold value would be 60*0.1*2*0.8 = 9.6
				// so we would set Value as "5" for each instance so that the sum in each auto-scaling calculating would be 10
				Value:        "5.0",
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
			tac.Spec.TiDB.Metrics = []autoscalingv2beta2.MetricSpec{}
			tac.Spec.TiDB.Metrics = append(tac.Spec.TiDB.Metrics, defaultMetricSpec)
			_, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Update(tac)
			framework.ExpectNoError(err, "Update TidbMonitorClusterAutoScaler error")

			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Spec.TiDB.Replicas != 3 {
					klog.Info("tidb haven't auto-scaler to 3 replicas")
					return false, nil
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					klog.Info("tac haven't marked any annotations")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
				if !ok {
					klog.Info("tac haven't marked tidb auto-scaler timstamp annotation")
					return false, nil
				}
				firstScaleTimestamp, err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					return false, err
				}
				return true, nil
			})
			framework.ExpectNoError(err, "check tidb auto-scale to 3 error")
			klog.Infof("success to check tidb auto scale-out to 3 replicas")

			mp = &mock.MonitorParams{
				Name:       tc.Name,
				MemberType: v1alpha1.TiDBMemberType.String(),
				Duration:   duration,
				// The cpu requests of tidb is 100m, so the threshold value would be 60*0.1*2*0.8 = 9.6
				// so we would set Value as "1" for each instance so that the sum in each auto-scaling calculating would be 3
				Value:        "1.0",
				InstancesPod: []string{"auto-scaling-tidb-0", "auto-scaling-tidb-1", "auto-scaling-tidb-2"},
			}
			err = mock.SetPrometheusResponse(monitor.Name, monitor.Namespace, mp, fw)
			framework.ExpectNoError(err, "set tidb mock metrics error")

			// Scale Tidb to 2 Replicas and Check
			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				tc, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tc.Spec.TiDB.Replicas != 2 {
					klog.Info("tidb haven't auto-scaler to 2 replicas")
					return false, nil
				}
				tac, err = cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Get(tac.Name, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tac.Annotations == nil || len(tac.Annotations) < 1 {
					klog.Info("tac haven't marked any annotations")
					return false, nil
				}
				v, ok := tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
				if !ok {
					klog.Info("tac haven't marked tidb auto-scale timestamp")
					return false, nil
				}
				secondTs, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return false, err
				}
				if secondTs == firstScaleTimestamp {
					klog.Info("tidb haven't scale yet")
					return false, nil
				}
				if secondTs-firstScaleTimestamp < 100 {
					return false, fmt.Errorf("tikv second scale's interval isn't meeting the interval requirement")
				}
				return true, nil
			})
			framework.ExpectNoError(err, "check tidb auto-scale to 2 error")
			klog.Info("success to check auto scale-in tidb to 2 replicas")
		})
	})

	ginkgo.Context("[Verify: Upgrading Operator from 1.0.6", func() {
		var oa tests.OperatorActions
		var ocfg *tests.OperatorConfig
		var version string

		ginkgo.BeforeEach(func() {
			version = "v1.0.6"
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
			cluster.Resources["tidb.replicas"] = "2"
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

			tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get tc")

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

			// confirm the tidb has been changed
			err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
				newTc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				// wait tidb to be updated
				if tc.Status.TiDB.StatefulSet.CurrentRevision == newTc.Status.TiDB.StatefulSet.CurrentRevision {
					klog.Info("wait tidb to be updated")
					return false, nil
				}
				// wait tidb finish updating
				if newTc.Status.TiDB.StatefulSet.CurrentRevision != newTc.Status.TiDB.StatefulSet.UpdateRevision {
					klog.Info("wait tidb finish updating")
					return false, nil
				}

				// confirm the tidb pod have been changed
				changed, err := utilpod.PodsAreChanged(c, tidbPods)()
				if changed {
					klog.Infof("confirm tidb pods have been changed")
				} else {
					if err != nil {
						klog.Errorf("meet error during verify tidb pods, err:%v", err)
						return false, nil
					}
					if !changed {
						return false, fmt.Errorf("tidb should be updated after operator upgrading")
					}
				}
				return true, nil
			})
			framework.ExpectNoError(err, "Failed to check Tidb Status After Upgrading Operator")

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
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "expect tikv and pd haven't been changed for 5 minutes")
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
})
