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
	"strings"
	"time"

	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	utilconfig "github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/tidbngmonitoring"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/tidbcluster"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	utiltngm "github.com/pingcap/tidb-operator/tests/e2e/util/tngm"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("[TiDBNGMonitoring]", func() {
	f := e2eframework.NewDefaultFramework("tngm")

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
			locator := fmt.Sprintf("%s/%s", ns, name)

			tc := GetTCForTiDBNGMonitoring(ns, name)
			tngm := fixture.GetTidbNGMonitoring(ns, name, tc)

			ginkgo.By("Deploy tidb cluster")
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Deploy tidb ng monitoring")
			err := genericCli.Create(context.TODO(), tngm)
			framework.ExpectNoError(err, "failed to create TidbNGMonitoring %s", locator)
			err = oa.WaitForTiDBNGMonitoringReady(tngm, 1*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbNGMonitoring %s components ready", locator)

			localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, fmt.Sprintf("pod/%s-ng-monitoring-0", name), 12020)
			framework.ExpectNoError(err, "failed to forward port for TidbNGMonitoring %s", locator)
			defer cancel()

			ginkgo.By("Enable continue profile for tidb ng monitoring")
			err = utiltngm.SetConfig(fmt.Sprintf("%s:%d", localHost, localPort), &utiltngm.ConfigureNGMReq{Conprof: &utiltngm.NGMConprofConfig{Enable: true}})
			framework.ExpectNoError(err, "failed to enable continue profile for TidbNGMonitoring %s", locator)

			ginkgo.By("Ensure continue profile is enabled for tidb ng monitoring")
			cfg, err := utiltngm.GetConfig(fmt.Sprintf("%s:%d", localHost, localPort))
			framework.ExpectEqual(cfg.Conprof.Enable, true, "continue profile isn't enabled for TidbNGMonitoring %s", locator)
		})

		ginkgo.It("Deploy tidb cluster with TLS-enabled and ngm", func() {
			name := "tls-cluster"
			locator := fmt.Sprintf("%s/%s", ns, name)

			tc := GetTCForTiDBNGMonitoring(ns, name)
			tngm := fixture.GetTidbNGMonitoring(ns, name, tc)

			ginkgo.By("Installing initial tidb CA certificate")
			err := tidbcluster.InstallTiDBIssuer(ns, name)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Install tidb cluster certificate")
			framework.ExpectNoError(tidbcluster.InstallTiDBComponentsCertificates(ns, name), "failed to install TiDB server and client certificate")
			<-time.After(30 * time.Second)

			ginkgo.By("Deploy tidb cluster with TLS enabled")
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Deploy tidb ng monitoring")
			err = genericCli.Create(context.TODO(), tngm)
			framework.ExpectNoError(err, "failed to create TidbNGMonitoring %s", locator)
			err = oa.WaitForTiDBNGMonitoringReady(tngm, 1*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbNGMonitoring %s components ready", locator)

			localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, fmt.Sprintf("pod/%s-ng-monitoring-0", name), 12020)
			framework.ExpectNoError(err, "failed to forward port for TidbNGMonitoring %q", locator)
			defer cancel()

			ginkgo.By("Enable continue profile for tidb ng monitoring")
			err = utiltngm.SetConfig(fmt.Sprintf("%s:%d", localHost, localPort), &utiltngm.ConfigureNGMReq{Conprof: &utiltngm.NGMConprofConfig{Enable: true}})
			framework.ExpectNoError(err, "failed to enable continue profile for TidbNGMonitoring %q", locator)

			ginkgo.By("Ensure continue profile is enabled for tidb ng monitoring")
			cfg, err := utiltngm.GetConfig(fmt.Sprintf("%s:%d", localHost, localPort))
			framework.ExpectEqual(cfg.Conprof.Enable, true, "continue profile isn't enabled for TidbNGMonitoring %q", locator)
		})

	})

	ginkgo.Context("[Update]", func() {
		ginkgo.It("Update config of ngm", func() {
			name := "update-cfg"
			locator := fmt.Sprintf("%s/%s", ns, name)

			tc := GetTCForTiDBNGMonitoring(ns, name)
			tngm := fixture.GetTidbNGMonitoring(ns, name, tc)

			ginkgo.By("Deploy tidb cluster")
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Deploy tidb ng monitoring")
			err := genericCli.Create(context.TODO(), tngm)
			framework.ExpectNoError(err, "failed to create TidbNGMonitoring %s", locator)
			err = oa.WaitForTiDBNGMonitoringReady(tngm, 1*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbNGMonitoring %q components ready", locator)

			ginkgo.By("Update config of ngm")
			err = controller.GuaranteedUpdate(genericCli, tngm, func() error {
				cfg := utilconfig.New(make(map[string]interface{}))
				cfg.Set("log.level", "DEBUG")
				tngm.Spec.NGMonitoring.Config = cfg
				return nil
			})
			framework.ExpectNoError(err, "failed to update config")
			utiltngm.MustWaitForNGMPhase(cli, tngm, v1alpha1.UpgradePhase, 5*time.Minute, 2*time.Second)
			err = oa.WaitForTiDBNGMonitoringReady(tngm, 1*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbNGMonitoring %q components ready", locator)

			ginkgo.By("Check config of ngm")
			stsName := tidbngmonitoring.NGMonitoringName(name)
			sts, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), stsName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get statefulset %s/%s", ns, stsName)
			cmName := mngerutils.FindConfigMapVolume(&sts.Spec.Template.Spec, func(cmName string) bool {
				return strings.HasPrefix(cmName, tidbngmonitoring.NGMonitoringName(name))
			})
			cm, err := c.CoreV1().ConfigMaps(ns).Get(context.TODO(), cmName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get config map %s/%s", ns, cmName)
			gomega.Expect(cm.Data["config-file"]).To(gomega.ContainSubstring("level = \"DEBUG\""), fmt.Sprintf("config map %s/%s isn't updated", ns, cmName))
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
