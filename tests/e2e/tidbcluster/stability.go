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
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utiltidb "github.com/pingcap/tidb-operator/tests/e2e/util/tidb"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Stability specs describe tests which involve disruptive operations, e.g.
// stop kubelet, kill nodes, empty pd/tikv data.
// Like serial tests, they cannot run in parallel too.
var _ = ginkgo.Describe("[tidb-operator][Stability]", func() {
	f := framework.NewDefaultFramework("stability")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var aggrCli aggregatorclient.Interface
	var apiExtCli apiextensionsclientset.Interface
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

	// TODO generate more contexts for different operator values
	ginkgo.Context("operator with default values", func() {
		var ocfg *tests.OperatorConfig
		var oa tests.OperatorActions
		var genericCli client.Client

		ginkgo.BeforeEach(func() {
			ocfg = &tests.OperatorConfig{
				Namespace:   ns,
				ReleaseName: "operator",
				Image:       cfg.OperatorImage,
				Tag:         cfg.OperatorTag,
				LogLevel:    "4",
				TestMode:    true,
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

		testCases := []struct {
			name string
			fn   func()
		}{
			{
				name: "tidb-operator does not exist",
				fn: func() {
					ginkgo.By("Uninstall tidb-operator")
					oa.CleanOperatorOrDie(ocfg)
				},
			},
		}

		for _, test := range testCases {
			ginkgo.It("tidb cluster should not be affected while "+test.name, func() {
				clusterName := "test"
				tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBV3Version)
				err := genericCli.Create(context.TODO(), tc)
				framework.ExpectNoError(err)
				err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
				framework.ExpectNoError(err)

				test.fn()

				ginkgo.By("Check tidb cluster is not affected")
				listOptions := metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(label.New().Instance(clusterName).Labels()).String(),
				}
				podList, err := c.CoreV1().Pods(ns).List(listOptions)
				framework.ExpectNoError(err)
				err = wait.PollImmediate(time.Second*30, time.Minute*5, func() (bool, error) {
					var ok bool
					var err error
					framework.Logf("check whether pods of cluster %q are changed", clusterName)
					ok, err = utilpod.PodsAreChanged(c, podList.Items)()
					if ok || err != nil {
						// pod changed or some error happened
						return true, err
					}
					framework.Logf("check whether pods of cluster %q are running", clusterName)
					newPodList, err := c.CoreV1().Pods(ns).List(listOptions)
					if err != nil {
						return false, err
					}
					for _, pod := range newPodList.Items {
						if pod.Status.Phase != v1.PodRunning {
							return false, fmt.Errorf("pod %s/%s is not running", pod.Namespace, pod.Name)
						}
					}
					framework.Logf("check whehter tidb cluster %q is connectable", clusterName)
					ok, err = utiltidb.TiDBIsConnectable(fw, ns, clusterName, "root", "")()
					if !ok || err != nil {
						// not connectable or some error happened
						return true, err
					}
					return false, nil
				})
				framework.ExpectEqual(err, wait.ErrWaitTimeout, "TiDB cluster is not affeteced")
			})
		}

	})

})
