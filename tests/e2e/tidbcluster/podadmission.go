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

var _ = ginkgo.Describe("[tidb-operator][PodAdmission]", func() {
	f := framework.NewDefaultFramework("pod-admission")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var cfg *tests.Config
	var config *restclient.Config
	var fw portforward.PortForward
	var fwCancel context.CancelFunc
	var ocfg *tests.OperatorConfig
	var oa tests.OperatorActions

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
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
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
			StsWebhookEnabled: false,
			PodWebhookEnabled: true,
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
		ginkgo.By("Uninstalling CRDs")
		oa.CleanCRDOrDie()
		ginkgo.By("Uninstall tidb-operator")
		oa.CleanOperatorOrDie(ocfg)
		if fwCancel != nil {
			fwCancel()
		}
	})

	// tidb-operator with AdvancedStatefulSet feature enabled
	ginkgo.Context("[Feature: PodAdmissionWebhook]", func() {

		ginkgo.It("Upgrade TidbCluster with pod admission webhook", func() {
			cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "deploy", "", "")
			cluster.Resources["pd.replicas"] = "3"
			cluster.Resources["tikv.replicas"] = "3"
			cluster.Resources["tidb.replicas"] = "2"
			oa.DeployTidbClusterOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)
			upgradeVersions := cfg.GetUpgradeTidbVersionsOrDie()
			ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", cluster.ClusterVersion, upgradeVersions[0]))
			cluster.UpgradeAll(upgradeVersions[0])
			oa.UpgradeTidbClusterOrDie(&cluster)
			oa.CheckUpgradeWithPodWebhookOrDie(&cluster)
			oa.CheckTidbClusterStatusOrDie(&cluster)
			oa.CleanTidbClusterOrDie(&cluster)
		})
	})

})
