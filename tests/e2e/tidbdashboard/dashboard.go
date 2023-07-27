// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

var _ = ginkgo.Describe("[TiDBDashboard]", func() {
	f := e2eframework.NewDefaultFramework("td")

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
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, fw, f)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.Context("[Deploy]", func() {
		ginkgo.It("Deploy tidb cluster with cluster component TLS-enabled and tidb dashboard", func() {
			name := "tls-cluster"
			locator := fmt.Sprintf("%s/%s", ns, name)

			tc := getTCForTiDBDashboard(ns, name)
			td := fixture.GetTidbDashboard(ns, name, tc)

			ginkgo.By("Installing initial tidb CA certificate")
			err := tidbcluster.InstallTiDBIssuer(ns, name)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Installing tidb server and client certificate")
			err = tidbcluster.InstallTiDBCertificates(ns, name)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate")

			ginkgo.By("Install tidb cluster certificate")
			framework.ExpectNoError(tidbcluster.InstallTiDBComponentsCertificates(ns, name), "failed to install TiDB server and client certificate")
			<-time.After(30 * time.Second)

			ginkgo.By("Deploy tidb cluster with TLS enabled")
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 10*time.Minute, 10*time.Second)

			ginkgo.By("Deploy tidb dashboard")
			err = genericCli.Create(context.TODO(), td)
			framework.ExpectNoError(err, "failed to create TidbDashboard %s", locator)
			err = oa.WaitForTiDBDashboardReady(td, 5*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbDashboard %s components ready", locator)

			localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, fmt.Sprintf("pod/%s-tidb-dashboard-0", name), 12333)
			framework.ExpectNoError(err, "failed to forward port for TidbDashboard %q", locator)
			defer cancel()

			ginkgo.By("Curl tidb dashboard main page")
			err = checkHttp200(fmt.Sprintf("%s:%d", localHost, localPort))
			framework.ExpectNoError(err, "failed to enable continue profile for TidbDashboard %q", locator)
		})
	})
})

func getTCForTiDBDashboard(ns, name string) *v1alpha1.TidbCluster {
	tc := fixture.GetTidbCluster(ns, name, utilimage.TiDBLatest)

	tc.Spec.PD.Replicas = 3
	tc.Spec.PD.Config.Set("dashboard.internal-proxy", true)
	tc.Spec.TiDB.Replicas = 1
	tc.Spec.TiKV.Replicas = 3

	return tc
}

func checkHttp200(addr string) error {
	url := fmt.Sprintf("http://%s/", addr)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusContinue || resp.StatusCode >= http.StatusBadRequest {
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read body failed: %s", err)
		}

		return fmt.Errorf("code %s msg %s", resp.Status, string(respData))
	}

	return nil
}
