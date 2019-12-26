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
// limitations under the License.package spec

package tidbcluster

import (
	"context"
	"fmt"
	_ "net/http/pprof"

	"github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onsi/ginkgo"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = ginkgo.Describe("[tidb-operator][Serial]", func() {
	f := framework.NewDefaultFramework("serial")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var cfg *tests.Config
	var config *restclient.Config
	var fw portforward.PortForward
	var fwCancel context.CancelFunc
	var genericCli client.Client

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
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config")
		genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset")
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
			oa = tests.NewOperatorActions(cli, c, asCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie()
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

		ginkgo.It("able to deploy TiDB Cluster with advanced statefulset", func() {
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "deploy", "", "")
			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)
		})
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
			oa = tests.NewOperatorActions(cli, c, asCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie()
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

		ginkgo.It("able to upgrade TiDB Cluster with pod admission webhook", func() {
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

		ginkgo.It("should perform defaulting and validating properly", func() {
			ocfg = &tests.OperatorConfig{
				Namespace:       "pingcap",
				ReleaseName:     "operator",
				Image:           cfg.OperatorImage,
				Tag:             cfg.OperatorTag,
				SchedulerImage:  "k8s.gcr.io/kube-scheduler",
				LogLevel:        "4",
				ImagePullPolicy: v1.PullIfNotPresent,
				TestMode:        true,
				WebhookEnabled:  false,
				// Reduce resource footprint and disable controller sync
				ControllerManagerReplicas: 0,
				SchedulerReplicas:         0,
			}
			oa = tests.NewOperatorActions(cli, c, asCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
			ginkgo.By("Installing CRDs")
			oa.CleanCRDOrDie()
			oa.InstallCRDOrDie()
			ginkgo.By("Installing tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			oa.DeployOperatorOrDie(ocfg)

			ginkgo.By("Resources created before webhook enabled could be operated normally")
			legacyTc := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "created-by-helm",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/tidb:v2.1.18",
						},
					},
					TiKV: v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/tikv:v2.1.18",
						},
					},
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v2.1.18",
						},
					},
				},
			}
			err := genericCli.Create(context.TODO(), legacyTc)
			framework.ExpectNoError(err, "Expected create tidbcluster without defaulting and validating")
			ocfg.WebhookEnabled = true
			oa.UpgradeOperatorOrDie(ocfg)
			// now the webhook enabled
			legacyTc.Spec.TiDB.Image = "pingcap/tidb:v3.0.7"
			err = genericCli.Update(context.TODO(), legacyTc)
			framework.ExpectNoError(err, "Update legacy tidbcluster should not be influenced by validating")
			framework.ExpectEqual(legacyTc.Spec.TiDB.BaseImage, "", "Update legacy tidbcluster should not be influenced by defaulting")

			ginkgo.By("Resources created before webhook will be checked if user migrate it to use new API")
			legacyTc.Spec.TiDB.BaseImage = "pingcap/tidb"
			legacyTc.Spec.TiKV.BaseImage = "pingcap/tikv"
			legacyTc.Spec.PD.BaseImage = "pingcap/pd"
			legacyTc.Spec.PD.Version = "v3.0.7"
			err = genericCli.Update(context.TODO(), legacyTc)
			framework.ExpectNoError(err, "Expected update tidbcluster")
			legacyTc.Spec.TiDB.BaseImage = ""
			legacyTc.Spec.PD.Version = ""
			err = genericCli.Update(context.TODO(), legacyTc)
			framework.ExpectError(err,
				"Validating should reject mandatory fields being empty if the resource has already been migrated to use the new API")

			ginkgo.By("Validating should reject legacy fields for newly created cluster")
			newTC := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: v1alpha1.TiDBSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/tidb:v2.1.18",
						},
					},
					TiKV: v1alpha1.TiKVSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/tikv:v2.1.18",
						},
					},
					PD: v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v2.1.18",
						},
					},
				},
			}
			err = genericCli.Create(context.TODO(), newTC)
			framework.ExpectError(err,
				"Validating should reject legacy fields for newly created cluster")

			ginkgo.By("Defaulting should set proper default for newly created cluster")
			newTC = &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "newly-created",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.7",
					TiDB: v1alpha1.TiDBSpec{
						Replicas: 3,
					},
					TiKV: v1alpha1.TiKVSpec{
						Replicas: 3,
					},
					PD: v1alpha1.PDSpec{
						Replicas: 3,
					},
				},
			}
			err = genericCli.Create(context.TODO(), newTC)
			framework.ExpectNoError(err, "Though some required fields are omitted, they will be set by defaulting")
			// don't have to check all fields, just take some to test if defaulting set
			if empty, err := gomega.BeEmpty().Match(newTC.Spec.TiDB.BaseImage); empty {
				e2elog.Failf("Expected tidb.baseImage has default value set, %v", err)
			}
			if isNil, err := gomega.BeNil().Match(newTC.Spec.TiDB.Config); isNil {
				e2elog.Failf("Expected tidb.config has default value set, %v", err)
			}

			ginkgo.By("Validating should reject illegal update")
			newTC.Labels[label.InstanceLabelKey] = "some-insane-label-value"
			err = genericCli.Update(context.TODO(), newTC)
			framework.ExpectError(err, "Could not set instance label with value other than cluster name")

			newTC.Spec.PD.Config.Replication = &v1alpha1.PDReplicationConfig{
				MaxReplicas: func() *uint64 { i := uint64(5); return &i }(),
			}
			err = genericCli.Update(context.TODO(), newTC)
			framework.ExpectError(err, "PD replication config is immutable through CR")

			ginkgo.By("Uninstall tidb-operator")
			oa.CleanOperatorOrDie(ocfg)
			ginkgo.By("Uninstalling CRDs")
			oa.CleanCRDOrDie()
		})
	})

})
