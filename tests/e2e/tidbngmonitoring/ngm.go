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

package tidbngmonitoring

import (
	"context"
	"fmt"
	"time"

	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/tidbcluster"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilngm "github.com/pingcap/tidb-operator/tests/e2e/util/ngm"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"

	"github.com/onsi/ginkgo"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("[TiDBNGMonitoring]", func() {
	f := e2eframework.NewDefaultFramework("tidb-ng-monitoring")

	var (
		ns         string
		c          clientset.Interface
		cli        versioned.Interface
		asCli      asclientset.Interface
		aggrCli    aggregatorclient.Interface
		apiExtCli  apiextensionsclientset.Interface
		oa         *tests.OperatorActions
		cfg        *tests.Config
		config     *restclient.Config
		ocfg       *tests.OperatorConfig
		genericCli ctrlCli.Client
		fwCancel   context.CancelFunc
		fw         portforward.PortForward
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
		genericCli, err = ctrlCli.New(config, ctrlCli.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset for controller-runtime")
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset kube-aggregator")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset apiextensions-apiserver")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config for tidb-operator")
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.Context("[Deploy]", func() {

		ginkgo.It("Deploy tidb cluster and ngm", func() {
			name := "normal-cluster"
			locator := fmt.Sprintf(ns, name)

			tc := GetTCForTiDBNGMonitoring(ns, name)
			tngm := fixture.GetTidbNGMonitoring(ns, name, tc)

			ginkgo.By(fmt.Sprintf("Deploy tidb cluster %s", locator))
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By(fmt.Sprintf("Deploy tidb ng monitoring %s", locator))
			err := genericCli.Create(context.TODO(), tngm)
			framework.ExpectNoError(err, "failed to create TidbNGMonitoring %s", locator)
			err = oa.WaitForTiDBNGMonitoringReady(tngm, 1*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbNGMonitoring %s components ready", locator)

			ginkgo.By(fmt.Sprintf("Enable continue profile for tidb ng monitoring %s", locator))
			err = utilngm.SetConfig(ns, name, &utilngm.NGMConfig{Conprof: &utilngm.NGMConprofConfig{Enable: true}})
			framework.ExpectNoError(err, "failed to enable continue profile for TidbNGMonitoring %s", locator)

			ginkgo.By(fmt.Sprintf("Ensure continue profile is enabled for tidb ng monitoring %s", locator))
			cfg, err := utilngm.GetConfig(ns, name)
			framework.ExpectEqual(cfg.Conprof.Enable, true, "continue profile isn't enabled for TidbNGMonitoring %s", locator)
		})

		ginkgo.It("Deploy tidb cluster with TLS-enabled and ngm", func() {
			name := "tls-cluster"
			locator := fmt.Sprintf(ns, name)

			tc := GetTCForTiDBNGMonitoring(ns, name)
			tngm := fixture.GetTidbNGMonitoring(ns, name, tc)

			ginkgo.By("Installing initial tidb CA certificate")
			err := tidbcluster.InstallTiDBIssuer(ns, name)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Install TiDB server and client certificate")
			framework.ExpectNoError(tidbcluster.InstallTiDBComponentsCertificates(ns, name), "failed to install TiDB server and client certificate")
			<-time.After(30 * time.Second)

			ginkgo.By(fmt.Sprintf("Deploy tidb cluster %s with TLS enabled", locator))
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By(fmt.Sprintf("Deploy tidb ng monitoring %s", locator))
			err = genericCli.Create(context.TODO(), tngm)
			framework.ExpectNoError(err, "failed to create TidbNGMonitoring %s", locator)
			err = oa.WaitForTiDBNGMonitoringReady(tngm, 1*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbNGMonitoring %s components ready", locator)

			ginkgo.By(fmt.Sprintf("Enable continue profile for tidb ng monitoring %s", locator))
			err = utilngm.SetConfig(ns, name, &utilngm.NGMConfig{Conprof: &utilngm.NGMConprofConfig{Enable: true}})
			framework.ExpectNoError(err, "failed to enable continue profile for TidbNGMonitoring %s", locator)

			ginkgo.By(fmt.Sprintf("Ensure continue profile is enabled for tidb ng monitoring %s", locator))
			cfg, err := utilngm.GetConfig(ns, name)
			framework.ExpectEqual(cfg.Conprof.Enable, true, "continue profile isn't enabled for TidbNGMonitoring %s", locator)
		})
	})

})

func GetTCForTiDBNGMonitoring(ns, name string) *v1alpha1.TidbCluster {
	tc := fixture.GetTidbCluster(ns, name, utilimage.TiDBLatest)

	tc.Spec.PD.Replicas = 3
	tc.Spec.PD.Config.Set("dashboard.internal-proxy", true)
	tc.Spec.TiDB.Replicas = 1
	tc.Spec.TiKV.Replicas = 3

	return tc
}
