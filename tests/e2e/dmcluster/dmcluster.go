// Copyright 2021 PingCAP, Inc.
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

package dmcluster

import (
	"context"
	"time"

	"github.com/onsi/ginkgo"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
)

var _ = ginkgo.Describe("DMCluster", func() {
	f := e2eframework.NewDefaultFramework("dm-cluster")

	var (
		ns         string
		c          clientset.Interface
		config     *restclient.Config
		cli        versioned.Interface
		asCli      asclientset.Interface
		aggrCli    aggregatorclient.Interface
		apiExtCli  apiextensionsclientset.Interface
		genericCli ctrlCli.Client
		fwCancel   context.CancelFunc
		fw         portforward.PortForward
		cfg        *tests.Config
		ocfg       *tests.OperatorConfig
		oa         *tests.OperatorActions
	)

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
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for kube-aggregator")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for apiextensions-apiserver")
		genericCli, err = ctrlCli.New(config, ctrlCli.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset for controller-runtime")
		ctx, cancel := context.WithCancel(context.Background())
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config for tidb-operator")
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, cfg, nil, fw, f)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.Context("[Feature:DM]", func() {
		ginkgo.It("setup replication for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName := "basic-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 1
			dc.Spec.Worker.Replicas = 1
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a basic migration task")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMSingleTask, ""), "failed to start single source task")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check full data")

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrData(fw, dc.Namespace), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 1), "failed to check incremental data")
		})

		ginkgo.It("scale out with shard task for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName := "scale-out-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 3
			dc.Spec.Worker.Replicas = 1
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a shard migration task but with one source not bound")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMShardTask, "have not bound"), "failed to start shard source task before scale out")

			ginkgo.By("Scale out DM-master and DM-worker")
			err = controller.GuaranteedUpdate(genericCli, dc, func() error {
				dc.Spec.Master.Replicas = 5
				dc.Spec.Worker.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err, "failed to scale out DmCluster: %q", dc.Name)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Start a shard migration task after scale out")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMShardTask, ""), "failed to start shard source task after scale out")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 2), "failed to check full data")

			ginkgo.By("Generate incremental stage data in upstream")
			framework.ExpectNoError(tests.GenDMIncrData(fw, dc.Namespace), "failed to generate incremental stage data in upstream")

			ginkgo.By("Check data for incremental stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 2), "failed to check incremental data")
		})

		ginkgo.It("scale in with shard task for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dcName := "scale-in-dm"
			dc := fixture.GetDMCluster(ns, dcName, utilimage.DMV2)
			dc.Spec.Master.Replicas = 5
			dc.Spec.Worker.Replicas = 2
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)

			ginkgo.By("Create MySQL sources")
			framework.ExpectNoError(tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName)), "failed to create sources for DmCluster %q", dcName)

			ginkgo.By("Generate full stage data in upstream")
			framework.ExpectNoError(tests.GenDMFullData(fw, dc.Namespace), "failed to generate full stage data in upstream")

			ginkgo.By("Start a shard migration task")
			framework.ExpectNoError(tests.StartDMTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName), tests.DMShardTask, ""), "failed to start shard source task before scale in")

			ginkgo.By("Check data for full stage")
			framework.ExpectNoError(tests.CheckDMData(fw, dc.Namespace, 2), "failed to check full data")

			ginkgo.By("Scale in DM-master and DM-worker")
			err = controller.GuaranteedUpdate(genericCli, dc, func() error {
				dc.Spec.Master.Replicas = 3
				dc.Spec.Worker.Replicas = 1
				return nil
			})
			framework.ExpectNoError(err, "failed to scale in DmCluster: %q", dc.Name)
			framework.ExpectNoError(oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second), "failed to wait for DmCluster %q ready", dcName)
		})
	})
})
