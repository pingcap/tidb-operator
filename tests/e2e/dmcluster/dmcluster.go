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

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
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
		ns        string
		c         clientset.Interface
		config    *restclient.Config
		cli       versioned.Interface
		asCli     asclientset.Interface
		aggrCli   aggregatorclient.Interface
		apiExtCli apiextensionsclientset.Interface
		fwCancel  context.CancelFunc
		fw        portforward.PortForward
		cfg       *tests.Config
		ocfg      *tests.OperatorConfig
		oa        *tests.OperatorActions
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
			dc := fixture.GetDMCluster(ns, "basic-dm", utilimage.DMV2)
			dc.Spec.Master.Replicas = 1
			dc.Spec.Worker.Replicas = 1
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dcName)
			err = oa.WaitForDmClusterReady(dc, 30*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "failed to wait for DmCluster ready: %q", dcName)

			ginkgo.By("Create MySQL sources")
			err = tests.CreateDMSources(fw, dc.Namespace, controller.DMMasterMemberName(dcName))
			framework.ExpectNoError(err, "failed to create sources for DmCluster: %q", dcName)

			ginkgo.By("Generate full stage date in upstream")
			err = tests.GenDMFullData(fw, dc.Namespace)
			framework.ExpectNoError(err, "failed to generate full stage data in upstream")

			ginkgo.By("Start a basic migration task")
			err = tests.StartDMSingleSourceTask(fw, dc.Namespace, controller.DMMasterMemberName(dcName))
			framework.ExpectNoError(err, "failed to start single source task")

			ginkgo.By("Check data for full stage")
			err = tests.CheckDMFullData(fw, dc.Namespace, 1)
			framework.ExpectNoError(err, "failed to check full data")
		})
	})
})
