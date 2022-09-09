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
	"github.com/pingcap/errors"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/monitor/monitor"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/util"
	utildb "github.com/pingcap/tidb-operator/tests/e2e/util/db"
	"github.com/pingcap/tidb-operator/tests/e2e/util/db/blockwriter"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utiltidb "github.com/pingcap/tidb-operator/tests/e2e/util/tidb"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
)

const (
	OperatorLatestVersion    string = "v1.3.6"
	OperatorPrevMajorVersion string = "v1.2.7"
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
		var oa *tests.OperatorActions

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
				StsWebhookEnabled: true,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.CreateCRDOrDie(ocfg)
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

		ginkgo.It(fmt.Sprintf("should be able to upgrade TiDB Cluster from %s to %s", utilimage.TiDBLatestPrev, utilimage.TiDBLatest), func() {
			log.Logf("start to upgrade tidbcluster with pod admission webhook")
			// deploy new cluster and test upgrade and scale-in/out with pod admission webhook
			ginkgo.By(fmt.Sprintf("start initial TidbCluster %q", utilimage.TiDBLatestPrev))
			tc := fixture.GetTidbCluster(ns, "admission", utilimage.TiDBLatestPrev)
			tc.Spec.PD.Replicas = 3
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 6*time.Minute, 5*time.Second)

			ginkgo.By("Set tikv partition annotation to 1")
			err := setPartitionAnnotation(ns, tc.Name, label.TiKVLabelVal, 1)
			framework.ExpectNoError(err, "set tikv Partition annotation failed")

			ginkgo.By(fmt.Sprintf("Upgrade TidbCluster version to %q", utilimage.TiDBLatest))
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBLatest
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster to upgrade tidb version to %v", utilimage.TiDBLatest)

			ginkgo.By(fmt.Sprintf("wait for tikv-1 pod upgrading to %q", utilimage.TiDBLatest))
			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				tikvPod, err := c.CoreV1().Pods(ns).Get(context.TODO(), fmt.Sprintf("%s-tikv-1", tc.Name), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if tikvPod.Spec.Containers[0].Image != fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBLatest) {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "failed to upgrade tikv-1 to %q", utilimage.TiDBLatest)

			ginkgo.By("Wait to see if tikv sts partition annotation remains 1 for 3 min")
			// TODO: explain the purpose of this testing
			err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
				tikvSts, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), fmt.Sprintf("%s-tikv", tc.Name), metav1.GetOptions{})
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
		var oa *tests.OperatorActions

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
				SchedulerReplicas:         util.IntPtr(0),
				ControllerManagerReplicas: util.IntPtr(0),
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.CreateCRDOrDie(ocfg)
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
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBLatestPrev),
						},
					},
					TiKV: &v1alpha1.TiKVSpec{
						Replicas: 1,
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBLatestPrev),
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
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBLatestPrev),
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
			legacyTc, err = cli.PingcapV1alpha1().TidbClusters(ns).Create(context.TODO(), legacyTc, metav1.CreateOptions{})
			// err := genericCli.Create(context.TODO(),context.TODO(), legacyTc)
			framework.ExpectNoError(err, "Expected create tidbcluster without defaulting and validating")

			ginkgo.By("Enable webhook in operator")
			ocfg.WebhookEnabled = true
			oa.UpgradeOperatorOrDie(ocfg)
			// now the webhook enabled
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.Image = fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBLatest)
				return nil
			})
			framework.ExpectNoError(err, "Update legacy TidbCluster should not be influenced by validating")
			framework.ExpectEqual(legacyTc.Spec.TiDB.BaseImage, "", "Update legacy tidbcluster should not be influenced by defaulting")

			ginkgo.By("Update TidbCluster to use webhook")
			err = controller.GuaranteedUpdate(genericCli, legacyTc, func() error {
				legacyTc.Spec.TiDB.BaseImage = "pingcap/tidb"
				legacyTc.Spec.TiKV.BaseImage = "pingcap/tikv"
				legacyTc.Spec.PD.BaseImage = "pingcap/pd"
				legacyTc.Spec.PD.Version = pointer.StringPtr(utilimage.TiDBLatest)
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
							Image: fmt.Sprintf("pingcap/tidb:%s", utilimage.TiDBLatest),
						},
					},
					TiKV: &v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/tikv:%s", utilimage.TiDBLatest),
						},
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBLatest),
						},
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("10G"),
							},
						},
					},
				},
			}
			_, err = cli.PingcapV1alpha1().TidbClusters(ns).Create(context.TODO(), newTC, metav1.CreateOptions{})
			framework.ExpectError(err, "Validating should reject legacy fields for newly created cluster")

			ginkgo.By("Deploy a new tc")
			newTC = &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: utilimage.TiDBLatest,
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
			newTC, err = cli.PingcapV1alpha1().TidbClusters(ns).Create(context.TODO(), newTC, metav1.CreateOptions{})
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
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.CreateCRDOrDie(ocfg)
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
			tcName := "tidbcluster1"
			tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatestPrev)
			tc.Spec.PD.Replicas = 1
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1

			ginkgo.By(fmt.Sprintf("deploy original tc %q", utilimage.TiDBLatestPrev))
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 10*time.Minute, 5*time.Second)
			pdPods := utilpod.MustListPods(labels.SelectorFromSet(label.New().Instance(tcName).PD().Labels()).String(), ns, c)

			ginkgo.By("Set --selector=version=old for the default TiDB Operator")
			ocfg.Selector = []string{"version=old"}
			oa.UpgradeOperatorOrDie(ocfg)
			log.Logf("Upgrade operator with --set-string \"selector=version=old\"")

			ginkgo.By("Upgrade TidbCluster 1 version, wait for 2 minutes, check that no rolling update occurs")
			err := controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = utilimage.TiDBLatest
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster 1 to upgrade PD version to %v", utilimage.TiDBLatest)

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
			tc2 := fixture.GetTidbCluster(ns, tc2Name, utilimage.TiDBLatestPrev)
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
				pdPods, err = utilpod.ListPods(labels.SelectorFromSet(label.New().Instance(tc2Name).PD().Labels()).String(), ns, c)
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
				tc2.Spec.Version = utilimage.TiDBLatest
				return nil
			})
			framework.ExpectNoError(err, "failed to update TidbCluster 2 to upgrade tidb version to %v", utilimage.TiDBLatest)
			log.Logf("Finished upgrading TidbCluster 2")

			err = oa.WaitForTidbClusterReady(tc2, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc2.Name)

			ginkgo.By(fmt.Sprintf("wait for TidbCluster 2 pd-0 pod upgrading to %q", utilimage.TiDBLatest))
			err = wait.Poll(5*time.Second, 10*time.Minute, func() (done bool, err error) {
				pdPod, err := c.CoreV1().Pods(ns).Get(context.TODO(), fmt.Sprintf("%s-pd-0", tc2.Name), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				if pdPod.Spec.Containers[0].Image != fmt.Sprintf("pingcap/pd:%s", utilimage.TiDBLatest) {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "failed to upgrade TidbCluster 2 pd-0 to %q", utilimage.TiDBLatest)
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
				tidbStatefulSet, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), fmt.Sprintf("%s-tidb", tc.Name), metav1.GetOptions{})
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

	ginkgo.Describe("Upgrade TiDB Operator", func() {
		var (
			operatorVersion   string
			oa                *tests.OperatorActions
			ocfg              *tests.OperatorConfig
			setOperatorConfig func(oa *tests.OperatorConfig)
		)

		ginkgo.JustBeforeEach(func() {
			// deploy older operator and CRDs
			ocfg = &tests.OperatorConfig{
				Namespace:   ns,
				ReleaseName: "operator",
				Tag:         operatorVersion,
				Image:       fmt.Sprintf("pingcap/tidb-operator:%s", operatorVersion),
			}
			oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, fw, f)
			if setOperatorConfig != nil {
				setOperatorConfig(ocfg)
			}

			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.CreateReleasedCRDOrDie(operatorVersion)
			ginkgo.By("Installing TiDB Operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)
		})

		ginkgo.JustAfterEach(func() {
			// clean current operator and CRDs
			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})

		ginkgo.Context("From latest version", func() {
			ginkgo.BeforeEach(func() {
				operatorVersion = OperatorLatestVersion
			})

			ginkgo.It("should not trigger rolling-update", func() {
				tcName := fmt.Sprintf("upgrade-operator-from-%s", strings.ReplaceAll(operatorVersion, ".", "x"))
				tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatest)
				tc = fixture.AddTiFlashForTidbCluster(tc)
				tc = fixture.AddTiCDCForTidbCluster(tc)
				tc = fixture.AddPumpForTidbCluster(tc)
				tc.Spec.PD.Replicas = 1
				tc.Spec.TiKV.Replicas = 1
				tc.Spec.TiDB.Replicas = 1
				tc.Spec.TiFlash.Replicas = 1

				ginkgo.By("Deploy original TiDB cluster")
				utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 6*time.Minute, 5*time.Second)
				selector := MustGetLabelSelectorForComponents(tcName, label.DiscoveryLabelVal) // ignore discovery
				pods := utilpod.MustListPods(selector.String(), ns, c)

				ginkgo.By("Upgrade TiDB Operator and CRDs to current version")
				ocfg.Tag = cfg.OperatorTag
				ocfg.Image = cfg.OperatorImage
				oa.ReplaceCRDOrDie(ocfg)
				oa.UpgradeOperatorOrDie(ocfg)

				ginkgo.By("Wait for pods are not changed in 5 minutes")
				err := utilpod.WaitForPodsAreChanged(c, pods, time.Minute*5)
				framework.ExpectEqual(err, wait.ErrWaitTimeout, "pods should not change in 5 minutes")
			})

			ginkgo.Context("Admission Webhook", func() {
				ginkgo.BeforeEach(func() {
					setOperatorConfig = func(oa *tests.OperatorConfig) {
						oa.TestMode = true
						oa.WebhookEnabled = true
						oa.StsWebhookEnabled = true
					}
				})

				ginkgo.It("should be able to be deployed", func() {
					ginkgo.By("Wait old webhook pods to become ready")
					err := oa.WaitAdmissionWebhookReady(ocfg, time.Minute*3, time.Second*10)
					framework.ExpectNoError(err, "failed to wait old webhook pods to become ready")

					ginkgo.By("Upgrade TiDB Operator and CRDs to current version")
					ocfg.Tag = cfg.OperatorTag
					ocfg.Image = cfg.OperatorImage
					oa.ReplaceCRDOrDie(ocfg)
					oa.UpgradeOperatorOrDie(ocfg)

					ginkgo.By("Wait webhook pods to become ready")
					err = oa.WaitAdmissionWebhookReady(ocfg, time.Minute*3, time.Second*10)
					framework.ExpectNoError(err, "failed to wait webhook pods to become ready")
				})
			})
		})

		ginkgo.Context("From prev major version", func() {

			ginkgo.BeforeEach(func() {
				operatorVersion = OperatorPrevMajorVersion
			})

			type testCase struct {
				name string
				tls  bool
			}
			cases := []testCase{
				{
					name: "should not trigger rolling-update and database work fine",
					tls:  false,
				},
				{
					name: "should not trigger rolling-update and database work fine with TLS-enabled",
					tls:  true,
				},
			}

			for i := range cases {
				testcase := cases[i]
				ginkgo.It(testcase.name, func() {
					dbName := "test"
					tables := []string{"test_0", "test_1", "test_2"}
					recordCount := 30
					batchCount := 100
					expectCount := recordCount * batchCount
					bw := blockwriter.New(blockwriter.WithTableNum(len(tables)),
						blockwriter.WithRecordNum(recordCount),
						blockwriter.WithBatchSize(batchCount),
						blockwriter.WithGenTableName(func(nr int) string {
							return tables[nr]
						}))

					tcName := fmt.Sprintf("upgrade-operator-from-%s", strings.ReplaceAll(operatorVersion, ".", "x"))
					// Use the fixed version v5.3.2 but not `util.TiDBLatestPrev`, because v5.4.1 TiFlash
					// will rolling update if version is large than v5.4.x due to PR #4358.
					// TODO: remove it after prev major version is greater than v1.2.x
					version := "v5.3.2"
					tc := fixture.GetTidbCluster(ns, tcName, version)
					tc = fixture.AddTiFlashForTidbCluster(tc)
					tc = fixture.AddTiCDCForTidbCluster(tc)
					tc = fixture.AddPumpForTidbCluster(tc)
					tc.Spec.PD.Replicas = 1
					tc.Spec.TiKV.Replicas = 1
					tc.Spec.TiDB.Replicas = 1
					tc.Spec.TiFlash.Replicas = 1

					if testcase.tls {
						tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
						tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}

						ginkgo.By("Installing tidb CA certificate")
						err := InstallTiDBIssuer(ns, tcName)
						framework.ExpectNoError(err, "failed to install CA certificate")

						ginkgo.By("Installing tidb server and client certificate")
						err = InstallTiDBCertificates(ns, tcName)
						framework.ExpectNoError(err, "failed to install tidb server and client certificate")

						ginkgo.By("Installing tidbInitializer client certificate")
						err = installTiDBInitializerCertificates(ns, tcName)
						framework.ExpectNoError(err, "failed to install tidbInitializer client certificate")

						ginkgo.By("Installing dashboard client certificate")
						err = installPDDashboardCertificates(ns, tcName)
						framework.ExpectNoError(err, "failed to install dashboard client certificate")

						ginkgo.By("Installing tidb components certificates")
						err = InstallTiDBComponentsCertificates(ns, tcName)
						framework.ExpectNoError(err, "failed to install tidb components certificates")
					}

					ginkgo.By("Deploy original TiDB cluster with prev version")
					utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 6*time.Minute, 5*time.Second)
					selector := MustGetLabelSelectorForComponents(tcName,
						label.DiscoveryLabelVal, // ignore discovery
					)
					pods := utilpod.MustListPods(selector.String(), ns, c)

					dsn, fwcancel, err := utiltidb.PortForwardAndGetTiDBDSN(fw, ns, tcName, "root", "", dbName)
					framework.ExpectNoError(err, "failed to get dsn")
					defer fwcancel()
					db := utildb.NewDatabaseOrDie(dsn)
					defer db.Close()

					ginkgo.By("Prepare data in database")
					err = bw.Write(context.Background(), dsn)
					framework.ExpectNoError(err, "failed to write data")

					ginkgo.By("Create TiFlash replicas for table 0 and ensure it is ready")
					MustCreateTiFlashReplicationForTable(db, dbName, tables[0], expectCount)

					ginkgo.By("Upgrade TiDB Operator and CRDs to current version")
					ocfg.Tag = cfg.OperatorTag
					ocfg.Image = cfg.OperatorImage
					oa.ReplaceCRDOrDie(ocfg)
					oa.UpgradeOperatorOrDie(ocfg)

					ginkgo.By("Wait for pods are not changed in 5 minutes")
					err = utilpod.WaitForPodsAreChanged(c, pods, time.Minute*5)
					framework.ExpectEqual(err, wait.ErrWaitTimeout, "pods should not change in 5 minutes")

					ginkgo.By("Ensure records in table 0 have not changed after upgrading TiDB Operator")
					EnsureRecordsNotChangedForTables(db, "tiflash", dbName, tables[0:1], expectCount)
					EnsureRecordsNotChangedForTables(db, "tikv", dbName, tables[0:1], expectCount)

					ginkgo.By("Create TiFlash replicas for table 1 and ensure it is ready")
					MustCreateTiFlashReplicationForTable(db, dbName, tables[1], expectCount)

					ginkgo.By("Update TiDB cluster to latest version")
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.Version = utilimage.TiDBLatest
						return nil
					})
					framework.ExpectNoError(err, "failed to upgrade TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 15*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "waiting for cluster %q ready", tcName)

					// reopen db after upgrade
					dsn2, fwcancel2, err := utiltidb.PortForwardAndGetTiDBDSN(fw, ns, tcName, "root", "", dbName)
					framework.ExpectNoError(err, "failed to get dsn")
					defer fwcancel2()
					db2 := utildb.NewDatabaseOrDie(dsn2)
					defer db2.Close()

					ginkgo.By("Ensure records in table 0 and table 1 have not changed after upgrading TiDB Operator")
					EnsureRecordsNotChangedForTables(db2, "tiflash", dbName, tables[0:2], expectCount)
					EnsureRecordsNotChangedForTables(db2, "tikv", dbName, tables[0:2], expectCount)

					ginkgo.By("Create TiFlash replicas for table 2 and ensure it is ready")
					MustCreateTiFlashReplicationForTable(db2, dbName, tables[2], expectCount)
				})
			}

		})

		ginkgo.Context("From v1.1.7", func() {
			ginkgo.BeforeEach(func() {
				operatorVersion = "v1.1.7"
			})

			ginkgo.It("should not change old TidbCluster", func() {
				tcName := fmt.Sprintf("upgrade-operator-from-%s", strings.ReplaceAll(operatorVersion, ".", "x"))
				tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV5x3)
				tc = fixture.AddTiFlashForTidbCluster(tc)
				tc = fixture.AddTiCDCForTidbCluster(tc)
				tc = fixture.AddPumpForTidbCluster(tc)
				tc.Spec.PD.Replicas = 3
				tc.Spec.TiKV.Replicas = 1
				tc.Spec.TiDB.Replicas = 1
				tc.Spec.TiCDC.Config = nil

				ginkgo.By("Deploy original TiDB cluster")
				utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 10*time.Minute, 5*time.Second)
				selector := MustGetLabelSelectorForComponents(tcName,
					label.DiscoveryLabelVal,
					label.PumpLabelVal,
					label.TiCDCLabelVal,   // ignore ticdc because of PR #4494
					label.TiFlashLabelVal, // ignore tiflash because of PR #4358
				)
				pods := utilpod.MustListPods(selector.String(), ns, c)

				ginkgo.By("Upgrade TiDB Operator and CRDs to current version")
				ocfg.Tag = cfg.OperatorTag
				ocfg.Image = cfg.OperatorImage
				oa.ReplaceCRDOrDie(ocfg)
				oa.UpgradeOperatorOrDie(ocfg)

				ginkgo.By("Wait for pods are not changed in 5 minutes")
				err := utilpod.WaitForPodsAreChanged(c, pods, time.Minute*5)
				framework.ExpectEqual(err, wait.ErrWaitTimeout, "pods should not change in 5 minutes")
			})

			/*
			  Release: v1.2.0
			  new feature in https://github.com/pingcap/tidb-operator/pull/3440
			  deploy tidbmonitor and upgrade tidb-perator, then tidbmonitor should switch from deployment to statefulset
			*/
			ginkgo.It("should migrate tidbmonitor from deployment to sts", func() {
				tcName := "smooth-tidbcluster"
				tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatest)
				tc.Spec.PD.Replicas = 1
				tc.Spec.TiKV.Replicas = 1
				tc.Spec.TiDB.Replicas = 1

				ginkgo.By("Deploy original TiDB cluster")
				utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 6*time.Minute, 5*time.Second)

				ginkgo.By("Deploy tidb monitor")
				monitorName := "smooth-migrate"
				tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tcName, metav1.GetOptions{})
				framework.ExpectNoError(err, "failed to get tidbcluster")
				tm := fixture.NewTidbMonitor(monitorName, ns, tc, true, true, true)
				_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(context.TODO(), tm, metav1.CreateOptions{})
				framework.ExpectNoError(err, "Expected tidbmonitor deployed success")
				err = tests.CheckTidbMonitor(tm, cli, c, fw)
				framework.ExpectNoError(err, "Expected tidbmonitor checked success")

				deploymentPvcName := fmt.Sprintf("%s-monitor", monitorName)
				deploymentPvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), deploymentPvcName, metav1.GetOptions{})
				framework.ExpectNoError(err, "Expected tidbmonitor deployment pvc success")
				oldVolumeName := deploymentPvc.Spec.VolumeName

				ginkgo.By("Upgrade tidb-operator and CRDs to the latest version")
				ocfg.Tag = cfg.OperatorTag
				ocfg.Image = cfg.OperatorImage
				oa.ReplaceCRDOrDie(ocfg)
				oa.UpgradeOperatorOrDie(ocfg)
				err = tests.CheckTidbMonitor(tm, cli, c, fw)
				framework.ExpectNoError(err, "Expected tidbmonitor checked success under migration")
				err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
					tmSet, err := stsGetter.StatefulSets(ns).Get(context.TODO(), monitor.GetMonitorObjectName(tm), metav1.GetOptions{})
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
					stsPvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), newStsPvcName, metav1.GetOptions{})
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
	})
})

func setPartitionAnnotation(namespace, tcName, component string, ordinal int) error {
	// add annotation to pause statefulset upgrade process
	output, err := framework.RunKubectl(namespace, "annotate", "tc", tcName, fmt.Sprintf("tidb.pingcap.com/%s-partition=%d", component, ordinal), "--overwrite")
	if err != nil {
		return fmt.Errorf("fail to set annotation for [%s/%s], component: %s, partition: %d, err: %v, output: %s", namespace, tcName, component, ordinal, err, output)
	}
	return nil
}

func MustGetLabelSelectorForComponents(tcName string, filterComponents ...string) labels.Selector {
	selector := labels.SelectorFromSet(label.New().Instance(tcName).Labels())

	r, err := labels.NewRequirement(label.ComponentLabelKey, selection.NotIn, filterComponents)
	framework.ExpectNoError(err, "failed to create label requirement")

	return selector.Add(*r)
}

// MustCreateTiFlashReplicationForTable create TiFLash replication and ensure it is ready
func MustCreateTiFlashReplicationForTable(db *utildb.Database, dbName string, table string, expectCount int) {
	err := utildb.CreateTiFlashReplicationAndWaitToComplete(db.TiFlashAction(), dbName, table, 1, time.Minute)
	framework.ExpectNoError(err, "failed to create TiFlash replication for %s", table)
	count, err := utildb.Count(db, "tiflash", dbName, table)
	framework.ExpectNoError(err, "failed to count records in %s by using %s", table, "tiflash")
	framework.ExpectEqual(count, expectCount, "count of records in %s changed by using %s", table, "tiflash")
}

// EnsureRecordsNotChangedForTables ensure records not changed for tables
func EnsureRecordsNotChangedForTables(db *utildb.Database, engine string, dbName string, tables []string, expectCount int) {
	for _, table := range tables {
		count, err := utildb.Count(db, engine, dbName, table)
		framework.ExpectNoError(err, "failed to count records in %s by using %s", table, engine)
		framework.ExpectEqual(count, expectCount, "count of records in %s changed by using %s", table, engine)
	}
}
