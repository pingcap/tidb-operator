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

package tidbcluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	nsutil "github.com/pingcap/tidb-operator/tests/e2e/util/ns"
	pdutil "github.com/pingcap/tidb-operator/tests/e2e/util/pd"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utiltidb "github.com/pingcap/tidb-operator/tests/e2e/util/tidb"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	"k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultClusterDomain = "cluster.local"
	inexistentBaseImage  = "pingcap/inexist"
)

var _ = ginkgo.Describe("[Across Kubernetes]", func() {
	f := e2eframework.NewDefaultFramework("across-kubernetes")

	var (
		namespaces []string
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
		var err error

		// build var
		namespaces = []string{f.Namespace.Name}
		c = f.ClientSet
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

	ginkgo.JustBeforeEach(func() {
		// create all namespaces except framework's namespace
		for _, ns := range namespaces[1:] {
			ginkgo.By(fmt.Sprintf("Building namespace %s", ns))
			_, existed, err := nsutil.CreateNamespaceIfNeeded(ns, c, nil)
			framework.ExpectEqual(existed, false, fmt.Sprintf("namespace %s is existed", ns))
			framework.ExpectNoError(err, fmt.Sprintf("failed to create namespace %s", ns))
		}
	})

	ginkgo.JustAfterEach(func() {
		// delete all namespaces if needed except framework's namespace
		if framework.TestContext.DeleteNamespace && (framework.TestContext.DeleteNamespaceOnFailure || !ginkgo.CurrentGinkgoTestDescription().Failed) {
			for _, ns := range namespaces[1:] {
				ginkgo.By(fmt.Sprintf("Destroying namespace %s", ns))
				_, err := nsutil.DeleteNamespace(ns, c)
				framework.ExpectNoError(err, fmt.Sprintf("failed to create namespace %s", ns))
			}
		}
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.Describe("[Across Kubernetes: Basic]", func() {

		// create three namespace
		ginkgo.BeforeEach(func() {
			ns1 := namespaces[0]
			namespaces = append(namespaces, ns1+"-1", ns1+"-2")
		})

		version := utilimage.TiDBLatest
		clusterDomain := defaultClusterDomain

		ginkgo.It("Deploy cluster across kubernetes", func() {
			ns1 := namespaces[0]
			ns2 := namespaces[1]
			ns3 := namespaces[2]

			tc1 := GetTCForAcrossKubernetes(ns1, "basic-1", version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, "basic-2", version, clusterDomain, tc1)
			tc3 := GetTCForAcrossKubernetes(ns3, "basic-3", version, clusterDomain, tc1)

			ginkgo.By("Deploy all clusters and wait status to be ready")
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, false)

			ginkgo.By("Check deploy status of all clusters")
			err := CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, 5*time.Second, 3*time.Minute)
			framework.ExpectNoError(err, "failed to check status")
		})

		ginkgo.It("Deploy and delete cluster across kubernetes", func() {
			ns1 := namespaces[0]
			ns2 := namespaces[1]

			cluster1Domain := defaultClusterDomain
			cluster2Domain := defaultClusterDomain

			tc1 := GetTCForAcrossKubernetes(ns1, "basic-1", version, cluster1Domain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, "basic-2", version, cluster2Domain, tc1)

			tc1.Spec.PD.Replicas = 3
			tc1.Spec.TiKV.Replicas = 3

			ginkgo.By("Deploy all clusters and wait status to be ready")
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2}, false)

			// connectable test
			_, err := utiltidb.TiDBIsConnectable(fw, tc1.Namespace, tc1.Name, "root", "")()
			framework.ExpectNoError(err, "tc1 is not connectable")
			_, err = utiltidb.TiDBIsConnectable(fw, tc2.Namespace, tc2.Name, "root", "")()
			framework.ExpectNoError(err, "tc2 is not connectable")

			ginkgo.By("Scale in cluster-2, and delete the cluster-2")

			framework.ExpectNoError(controller.GuaranteedUpdate(genericCli, tc2, func() error {
				tc2.Spec.PD.Replicas = 0
				tc2.Spec.TiDB.Replicas = 0
				tc2.Spec.TiKV.Replicas = 0
				tc2.Spec.TiFlash.Replicas = 0
				tc2.Spec.TiCDC.Replicas = 0
				tc2.Spec.Pump.Replicas = 0
				return nil
			}), "failed to scale in cluster 2")
			err = wait.PollImmediate(10*time.Second, 6*time.Minute, func() (bool, error) {
				if err := CheckPeerMembersAndClusterStatus(genericCli, tc1, tc2); err != nil {
					log.Logf("wait for tc2 member deleted, status: %s", err)
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "there are tc2 members remaining in tc1 PeerMembers or some components in tc1 are not healthy.")
			framework.ExpectNoError(genericCli.Delete(context.TODO(), tc2), "failed to delete cluster 2")

			ginkgo.By("Check status of tc1")
			err = oa.WaitForTidbClusterReady(tc1, 2*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "timeout to wait for tc1 to be healthy")

			// connectable test
			ginkgo.By("Check if tc1 is connectable")
			_, err = utiltidb.TiDBIsConnectable(fw, tc1.Namespace, tc1.Name, "root", "")()
			framework.ExpectNoError(err, "tc1 is not connectable")
		})

		ginkgo.It("Join: cluster-2 join cluster-1 and cluster-3 join cluster-2", func() {
			ns1, ns2, ns3 := namespaces[0], namespaces[1], namespaces[2]
			tc1 := GetTCForAcrossKubernetes(ns1, "cluster-1", version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, "cluster-2-join-1", version, clusterDomain, tc1)
			tc3 := GetTCForAcrossKubernetes(ns3, "cluster-3-join-2", version, clusterDomain, tc2)

			ginkgo.By("Deploy all clusters and wait status to be ready")
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, false)

			ginkgo.By("Check status over all clusters")
			err := CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, 5*time.Second, 3*time.Minute)
			framework.ExpectNoError(err, "failed to check status")
		})

		ginkgo.It("Join cluster with existing data", func() {
			ns1, ns2, ns3 := namespaces[0], namespaces[1], namespaces[2]

			tc1 := GetTCForAcrossKubernetes(ns1, "update-1", version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, "update-2", version, clusterDomain, tc1)
			tc3 := GetTCForAcrossKubernetes(ns3, "update-3", version, clusterDomain, tc1)

			ginkgo.By("Deploy the basic cluster-1 with empty cluster domain")
			tc1.Spec.ClusterDomain = ""
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc1, 5*time.Minute, 10*time.Second)

			ginkgo.By("Update cluster domain of cluster-1")
			err := controller.GuaranteedUpdate(genericCli, tc1, func() error {
				tc1.Spec.ClusterDomain = defaultClusterDomain
				return nil
			})
			framework.ExpectNoError(err, "failed to update cluster domain of cluster-1 %s/%s", tc1.Namespace, tc1.Name)
			err = oa.WaitForTidbClusterReady(tc1, 30*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "failed to wait for cluster-1 ready: %s/%s", tc1.Namespace, tc1.Name)

			localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, tc1.Namespace, fmt.Sprintf("svc/%s-pd", tc1.Name), 2379)
			framework.ExpectNoError(err, "failed to port-forward pd server of cluster-1 %s/%s", tc1.Namespace, tc1.Name)
			defer cancel()

			ginkgo.By("Update pd's peerURL of cluster-1")
			pdAddr := fmt.Sprintf("%s:%d", localHost, localPort)
			var resp *pdutil.GetMembersResponse
			err = retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
				resp, err = pdutil.GetMembersV2(pdAddr)
				return err
			})
			framework.ExpectNoError(err, " failed to get pd members of cluster-1 %s/%s", tc1.Namespace, tc1.Name)
			for _, member := range resp.Members {
				peerURLs := []string{}
				for _, url := range member.PeerURLs {
					// url ex: http://cluster-1-pd-0.cluster-1-pd-peer.default.svc:2380
					fields := strings.Split(url, ":")
					fields[1] = fmt.Sprintf("%s.%s", fields[1], clusterDomain)
					peerURLs = append(peerURLs, strings.Join(fields, ":"))
				}
				err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
					return pdutil.UpdateMemberPeerURLs(pdAddr, member.ID, peerURLs)
				})
				framework.ExpectNoError(err, " failed to update peerURLs of pd members of cluster-1 %s/%s", tc1.Namespace, tc1.Name)
			}

			ginkgo.By("Deploy cluster-2, cluster-3 and wait status to be ready")
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc2, tc3}, false)

			ginkgo.By("Deploy status of all clusters")
			err = CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, 5*time.Second, 3*time.Minute)
			framework.ExpectNoError(err, "failed to check status")
		})

		ginkgo.It("Deploy cluster with TLS-enabled across kubernetes", func() {
			ns1, ns2 := namespaces[0], namespaces[1]
			tcName1, tcName2 := "tls-cluster-1", "tls-cluster-2"
			tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)

			ginkgo.By("Prepare TLS resources for clusters")
			MustPrepareXK8sTLSResources(genericCli, tc1, []*v1alpha1.TidbCluster{tc2})

			ginkgo.By("Installing separate dashboard client certificate")
			err := installPDDashboardCertificates(ns1, tcName1)
			framework.ExpectNoError(err, "failed to install separate dashboard client certificate for cluster: %q", tcName1)
			err = installPDDashboardCertificates(ns2, tcName2)
			framework.ExpectNoError(err, "failed to install separate dashboard client certificate for cluster: %q", tcName2)

			ginkgo.By("Creating x-k8s tidb clusters with TLS enabled and wait status to be ready")
			tc1DashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName1)
			tc1.Spec.PD.TLSClientSecretName = &tc1DashTLSName
			tc2DashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName2)
			tc2.Spec.PD.TLSClientSecretName = &tc2DashTLSName
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2}, true)

			ginkgo.By("Ensure Dashboard use custom secret")
			foundSecretName := false
			// only check cluster-2 here, as cluster-1 is deployed the same as a normal tls-enabled tc.
			pdSts, err := c.AppsV1().StatefulSets(ns2).Get(context.TODO(), controller.PDMemberName(tcName2), metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get statefulsets for pd")
			for _, vol := range pdSts.Spec.Template.Spec.Volumes {
				if vol.Name == "tidb-client-tls" {
					foundSecretName = true
					framework.ExpectEqual(vol.Secret.SecretName, tc2DashTLSName)
					break
				}
			}
			framework.ExpectEqual(foundSecretName, true)

			ginkgo.By("Check deploy status over all clusters")
			err = CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2}, 5*time.Second, 3*time.Minute)
			framework.ExpectNoError(err, "failed to check status")

			ginkgo.By("Connecting to tidb server to verify the connection is TLS enabled")
			err = wait.PollImmediate(time.Second*15, time.Minute*10, tidbIsTLSEnabled(fw, c, ns1, tcName1, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName1)
			err = wait.PollImmediate(time.Second*15, time.Minute*10, tidbIsTLSEnabled(fw, c, ns2, tcName2, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName2)
		})

		ginkgo.It("Deploy cluster without setting cluster domain across kubernetes", func() {
			ns1 := namespaces[0]
			ns2 := namespaces[1]
			ns3 := namespaces[2]

			tc1 := GetTCForAcrossKubernetes(ns1, "without-cluster-domain-1", version, "", nil)
			tc2 := GetTCForAcrossKubernetes(ns2, "without-cluster-domain-2", version, "", tc1)
			tc3 := GetTCForAcrossKubernetes(ns3, "without-cluster-domain-3", version, "", tc1)

			ginkgo.By("Deploy all clusters and wait status to be ready")
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, false)

			ginkgo.By("Check deploy status of all clusters")
			err := CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, 5*time.Second, 3*time.Minute)
			framework.ExpectNoError(err, "failed to check status")
		})

		ginkgo.Context("[Without Local PD]", func() {
			type subcase struct {
				name     string
				changeFn func(tc1 *v1alpha1.TidbCluster, tc2 *v1alpha1.TidbCluster)
			}

			cases := []subcase{
				{
					name:     "all components",
					changeFn: nil,
				},
				{
					name: "only tidb",
					changeFn: func(tc1, tc2 *v1alpha1.TidbCluster) {
						tc2.Spec.TiKV = nil
						tc2.Spec.TiFlash = nil
						tc2.Spec.TiCDC = nil
						tc2.Spec.Pump = nil
					},
				},
				{
					name: "only tikv",
					changeFn: func(tc1, tc2 *v1alpha1.TidbCluster) {
						tc2.Spec.TiDB = nil
						tc2.Spec.TiFlash = nil
						tc2.Spec.TiCDC = nil
						tc2.Spec.Pump = nil
					},
				},
				{
					name: "only tiflash",
					changeFn: func(tc1, tc2 *v1alpha1.TidbCluster) {
						tc2.Spec.TiDB = nil
						tc2.Spec.TiKV = nil
						tc2.Spec.TiCDC = nil
						tc2.Spec.Pump = nil
					},
				},
				{
					name: "only ticdc",
					changeFn: func(tc1, tc2 *v1alpha1.TidbCluster) {
						tc2.Spec.TiDB = nil
						tc2.Spec.TiKV = nil
						tc2.Spec.TiFlash = nil
						tc2.Spec.Pump = nil
					},
				},
				{
					name: "only pump",
					changeFn: func(tc1, tc2 *v1alpha1.TidbCluster) {
						tc2.Spec.TiDB = nil
						tc2.Spec.TiKV = nil
						tc2.Spec.TiFlash = nil
						tc2.Spec.TiCDC = nil
					},
				},
			}

			for _, sc := range cases {
				ginkgo.It(fmt.Sprintf("Deploy cluster and cluster-2 deploys %s", sc.name), func() {
					ns1, ns2 := namespaces[0], namespaces[1]
					tc1 := GetTCForAcrossKubernetes(ns1, "basic-1", version, clusterDomain, nil)
					tc2 := GetTCForAcrossKubernetes(ns2, "basic-2-without-pd", version, clusterDomain, tc1)
					tc2.Spec.PD = nil

					if sc.changeFn != nil {
						sc.changeFn(tc1, tc2)
					}

					ginkgo.By("Deploy all clusters and wait status to be ready")
					MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2}, false)

					ginkgo.By("Check deploy status of all clusters")
					err := CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2}, 5*time.Second, 3*time.Minute)
					framework.ExpectNoError(err, "failed to check status")
				})

				ginkgo.It(fmt.Sprintf("Deploy cluster with TLS-enabled and cluster-2 deploys %s", sc.name), func() {
					ns1, ns2 := namespaces[0], namespaces[1]
					tcName1, tcName2 := "tls-cluster-1", "tls-cluster-2-without-pd"
					tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
					tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)
					tc2.Spec.PD = nil

					if sc.changeFn != nil {
						sc.changeFn(tc1, tc2)
					}

					ginkgo.By("Prepare TLS resources for clusters")
					MustPrepareXK8sTLSResources(genericCli, tc1, []*v1alpha1.TidbCluster{tc2})

					ginkgo.By("Deploy all clusters and wait status to be ready")
					MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2}, true)

					ginkgo.By("Check deploy status of all clusters")
					err := CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2}, 5*time.Second, 3*time.Minute)
					framework.ExpectNoError(err, "failed to check status")
				})
			}

		})

	})

	ginkgo.Describe("[Failover]", func() {
		// create three namespace
		ginkgo.BeforeEach(func() {
			ns1 := namespaces[0]
			namespaces = append(namespaces, ns1+"-1", ns1+"-2")
		})

		version := utilimage.TiDBLatest
		clusterDomain := defaultClusterDomain

		ginkgo.It("Components work normally after pod restart though PD/TiKVs in the same TC failed", func() {
			ns1, ns2, ns3 := namespaces[0], namespaces[1], namespaces[2]
			tcName1, tcName2, tcName3 := "cluster-1", "cluster-2", "cluster-3"
			tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)
			tc3 := GetTCForAcrossKubernetes(ns3, tcName3, version, clusterDomain, tc1)

			ginkgo.By("Prepare TLS resources for clusters")
			MustPrepareXK8sTLSResources(genericCli, tc1, []*v1alpha1.TidbCluster{tc2, tc3})

			ginkgo.By("Creating x-k8s tidb clusters with TLS enabled and wait status to be ready")
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, true)

			ginkgo.By("Fail TiKV in cluster-1 by setting a wrong image")
			tikvBaseImage := tc1.Spec.TiKV.BaseImage
			err := controller.GuaranteedUpdate(genericCli, tc1, func() error {
				tc1.Spec.TiKV.BaseImage = inexistentBaseImage
				return nil
			})
			framework.ExpectNoError(err, "updating tikv with an inexistent image %q for %q", tc1.Spec.TiKV.BaseImage, tcName1)
			// force operator to trigger an upgrade.
			err = c.AppsV1().StatefulSets(ns1).Delete(context.TODO(), fmt.Sprintf("%s-tikv", tcName1), metav1.DeleteOptions{})
			framework.ExpectNoError(err, "deleting sts of tikv for %q", tcName1)

			ginkgo.By("Waiting for tikv pod to be unavailable")
			err = utiltc.WaitForTCCondition(cli, tc1.Namespace, tc1.Name, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					down := true
					for _, store := range tc.Status.TiKV.Stores {
						down = down && (store.State == "Disconnected" || store.State == v1alpha1.TiKVStateDown)
					}
					return down, nil
				})
			framework.ExpectNoError(err, "waiting for tikv pod to be unavailable")

			ginkgo.By("Restart other component pods")
			podList, err := c.CoreV1().Pods(ns1).List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err, "list pods under namespace %q", ns1)
			for _, pod := range podList.Items {
				if !strings.Contains(pod.Name, "tikv") {
					err := c.CoreV1().Pods(ns1).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
					framework.ExpectNoError(err, "failed to delete pod %q", pod.Name)
				}
			}

			ginkgo.By("Check cluster components status")
			componentsFilter := make(map[v1alpha1.MemberType]struct{}, 1)
			componentsFilter[v1alpha1.TiKVMemberType] = struct{}{}
			err = oa.WaitForTidbComponentsReady(tc1, componentsFilter, 15*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "waiting for other components to be ready")
			err = wait.PollImmediate(time.Second*15, time.Minute*10, tidbIsTLSEnabled(fw, c, ns1, tcName1, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName1)

			ginkgo.By("Rollback TiKV baseImage and Fail PD in cluster-1 by setting a wrong image")
			err = controller.GuaranteedUpdate(genericCli, tc1, func() error {
				tc1.Spec.TiKV.BaseImage = tikvBaseImage
				tc1.Spec.PD.BaseImage = inexistentBaseImage
				return nil
			})
			framework.ExpectNoError(err, "updating tikv with image %q and pd with an image %q for %q", tc1.Spec.TiKV.BaseImage, tc1.Spec.PD.BaseImage, tcName1)
			// force operator to trigger an upgrade.
			err = c.AppsV1().StatefulSets(ns1).Delete(context.TODO(), fmt.Sprintf("%s-pd", tcName1), metav1.DeleteOptions{})
			framework.ExpectNoError(err, "deleting sts of pd for %q", tcName1)
			err = c.AppsV1().StatefulSets(ns1).Delete(context.TODO(), fmt.Sprintf("%s-tikv", tcName1), metav1.DeleteOptions{})
			framework.ExpectNoError(err, "deleting sts of tikv for %q", tcName1)

			ginkgo.By("Waiting for pd pods to be in unhealthy state")
			err = utiltc.WaitForTCCondition(cli, ns1, tcName1, time.Minute*10, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					healthy := false
					for _, member := range tc.Status.PD.Members {
						healthy = healthy || member.Health
					}
					// we consider it healthy only when member is healthy and synced is true
					return !(healthy && tc.Status.PD.Synced), nil
				})
			framework.ExpectNoError(err, "waiting for the pd to be in unhealthy state")

			ginkgo.By("Restart other components and check cluster status")
			podList, err = c.CoreV1().Pods(ns1).List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err, "list pods under namespace %q", ns1)
			for _, pod := range podList.Items {
				if !strings.Contains(pod.Name, "pd") {
					err := c.CoreV1().Pods(ns1).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
					framework.ExpectNoError(err, "failed to delete pod %q", pod.Name)
				}
			}

			ginkgo.By("Check cluster components status")
			delete(componentsFilter, v1alpha1.TiKVMemberType)
			componentsFilter[v1alpha1.PDMemberType] = struct{}{}
			err = oa.WaitForTidbComponentsReady(tc1, componentsFilter, 15*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "waiting for other components to be ready")
			err = wait.PollImmediate(time.Second*15, time.Minute*10, tidbIsTLSEnabled(fw, c, ns1, tcName1, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName1)
		})

		ginkgo.It("TiDBCluter Should work when one of the TidbCluster or the k8s fails", func() {
			ns1, ns2, ns3 := namespaces[0], namespaces[1], namespaces[2]
			tcName1, tcName2, tcName3 := "cluster-1", "cluster-2", "cluster-3"
			tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)
			tc3 := GetTCForAcrossKubernetes(ns3, tcName3, version, clusterDomain, tc1)

			ginkgo.By("Prepare TLS resources for clusters")
			MustPrepareXK8sTLSResources(genericCli, tc1, []*v1alpha1.TidbCluster{tc2, tc3})

			ginkgo.By("Creating x-k8s tidb clusters with TLS enabled and wait status to be ready")
			MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, true)

			ginkgo.By("Fail all components in cluster-2 by deleting ns")
			err := c.CoreV1().Namespaces().Delete(context.TODO(), ns2, metav1.DeleteOptions{})
			framework.ExpectNoError(err, "deleting namespsace %q", ns2)
			err = wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
				_, err = c.CoreV1().Namespaces().Get(context.TODO(), ns2, metav1.GetOptions{})
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			framework.ExpectNoError(err, "waiting namespsace %q to be deleted", ns2)

			ginkgo.By("Check status of other clusters")
			err = oa.WaitForTidbClusterReady(tc1, 10*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "%q cluster not healthy after cluster %q fail", tcName1, tcName2)
			err = oa.WaitForTidbClusterReady(tc3, 10*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "%q cluster not healthy after cluster %q fail", tcName3, tcName2)
			err = CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc3}, 5*time.Second, 3*time.Minute)
			framework.ExpectNoError(err, "failed to check status after cluster %q fail", tcName2)

			ginkgo.By("Check functionality of other clusters by querying tidb")
			err = wait.PollImmediate(time.Second*15, time.Minute*10, tidbIsTLSEnabled(fw, c, ns1, tcName1, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName1)
			err = wait.PollImmediate(time.Second*15, time.Minute*10, tidbIsTLSEnabled(fw, c, ns3, tcName3, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName3)
		})

		ginkgo.It("Failed to join in cluster-1 when PD crash and succeed after pd restart", func() {
			ns1, ns2 := namespaces[0], namespaces[1]
			tcName1, tcName2 := "cluster-1", "cluster-2"
			tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)

			ginkgo.By("Prepare TLS resources for clusters")
			MustPrepareXK8sTLSResources(genericCli, tc1, []*v1alpha1.TidbCluster{tc2})

			ginkgo.By("Creating tidb cluster-1 with TLS enabled")
			tc1.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc1.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc1, 10*time.Minute, 30*time.Second)

			ginkgo.By("Fail PD in cluster-1 by setting a wrong image")
			baseImage := tc1.Spec.PD.BaseImage
			err := controller.GuaranteedUpdate(genericCli, tc1, func() error {
				tc1.Spec.PD.BaseImage = inexistentBaseImage
				return nil
			})
			framework.ExpectNoError(err, "updating pd with an inexistent image %q for %q", tc1.Spec.PD.BaseImage, tcName1)

			ginkgo.By("Waiting for pd pods to be in unhealthy state")
			err = utiltc.WaitForTCCondition(cli, ns1, tcName1, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					healthy := false
					for _, member := range tc.Status.PD.Members {
						healthy = healthy || member.Health
					}
					// we consider it healthy only when member is healthy and synced is true
					return !(healthy && tc.Status.PD.Synced), nil
				})
			framework.ExpectNoError(err, "waiting for the pd to be in unhealthy state")

			ginkgo.By("Join cluster-2 into cluster-1 when pd failed")
			tc2.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc2.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			err = genericCli.Create(context.TODO(), tc2)
			framework.ExpectNoError(err, "create TidbCluster %q", tc2.Name)
			err = oa.WaitForTidbClusterReady(tc2, 5*time.Minute, 10*time.Second)
			framework.ExpectError(err, "%q should not be able to join %q as pd fails", tcName2, tcName1)

			ginkgo.By("Recover PD in cluster-1")
			err = controller.GuaranteedUpdate(genericCli, tc1, func() error {
				tc1.Spec.PD.BaseImage = baseImage
				return nil
			})
			framework.ExpectNoError(err, "updating pd with previous image %q for %q", tc1.Spec.PD.BaseImage, tcName1)
			// force operator to trigger a pd upgrade when pd is down.
			err = c.AppsV1().StatefulSets(ns1).Delete(context.TODO(), fmt.Sprintf("%s-pd", tcName1), metav1.DeleteOptions{})
			framework.ExpectNoError(err, "deleting sts of pd for %q", tcName1)

			ginkgo.By("Join cluster-2 into cluster-1 when pd running normally")
			err = oa.WaitForTidbClusterReady(tc2, 10*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "waiting for %q ready", tcName2)
			err = CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2}, 5*time.Second, 3*time.Minute)
			framework.ExpectNoError(err, "%q failed to join into %q", tcName2, tcName1)
		})
	})

	ginkgo.Describe("Start Script Version", func() {
		type testcase struct {
			nameSuffix string
			tlsEnable  bool
		}

		cases := []testcase{
			{
				nameSuffix: "",
				tlsEnable:  false,
			},
			{
				nameSuffix: "and enable TLS",
				tlsEnable:  true,
			},
		}

		ginkgo.BeforeEach(func() {
			ns1 := namespaces[0]
			namespaces = append(namespaces, ns1+"-1")
		})

		version := utilimage.TiDBLatest
		clusterDomain := defaultClusterDomain

		for _, testcase := range cases {
			ginkgo.It("deploy cluster with start script v2 "+testcase.nameSuffix, func() {
				ns1 := namespaces[0]
				ns2 := namespaces[1]

				tc1 := GetTCForAcrossKubernetes(ns1, "start-script-v2-tc1", version, clusterDomain, nil)
				tc2 := GetTCForAcrossKubernetes(ns2, "start-script-v2-tc2", version, clusterDomain, tc1)
				tc1.Spec.StartScriptVersion = v1alpha1.StartScriptV2
				tc2.Spec.StartScriptVersion = v1alpha1.StartScriptV2

				if testcase.tlsEnable {
					ginkgo.By("Prepare TLS resources for clusters")
					MustPrepareXK8sTLSResources(genericCli, tc1, []*v1alpha1.TidbCluster{tc2})
				}

				ginkgo.By("Deploy all clusters and wait status to be ready")
				MustCreateXK8sTCWithComponentsReady(genericCli, oa, []*v1alpha1.TidbCluster{tc1, tc2}, false)

				ginkgo.By("Check deploy status of all clusters")
				err := CheckStatusWhenAcrossK8sWithTimeout(cli, []*v1alpha1.TidbCluster{tc1, tc2}, 5*time.Second, 3*time.Minute)
				framework.ExpectNoError(err, "failed to check status")
			})

			ginkgo.It("migrate start script from v1 to v2 "+testcase.nameSuffix, func() {
				ns := namespaces[0]
				tcName := "migrate-start-script-xk8s"

				tc := GetTCForAcrossKubernetes(ns, tcName, version, clusterDomain, nil)

				if testcase.tlsEnable {
					ginkgo.By("Prepare TLS resources for clusters")
					MustPrepareXK8sTLSResources(genericCli, tc, []*v1alpha1.TidbCluster{})
				}

				ginkgo.By("Deploy tidb cluster")
				utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)
				oldTC, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tcName, metav1.GetOptions{})
				framework.ExpectNoError(err, "failed to get tc %s/%s", ns, tcName)

				ginkgo.By("Update tc to use start script v2")
				err = controller.GuaranteedUpdate(genericCli, tc, func() error {
					tc.Spec.StartScriptVersion = v1alpha1.StartScriptV2
					return nil
				})
				framework.ExpectNoError(err, "failed to start script version to v2")

				ginkgo.By(fmt.Sprintf("Wait for phase is %q", v1alpha1.UpgradePhase))
				utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.PDMemberType, v1alpha1.UpgradePhase, 3*time.Minute, time.Second*10)

				ginkgo.By("Wait for cluster is ready")
				err = oa.WaitForTidbClusterReady(tc, 15*time.Minute, 10*time.Second)
				framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc.Name)

				ginkgo.By("Check status of components not changed")
				err = utiltc.CheckComponentStatusNotChanged(cli, oldTC)
				framework.ExpectNoError(err, "failed to check component status of tc %s/%s not changed", ns, tcName)
			})
		}

	})
})

func CheckPeerMembersAndClusterStatus(clusterCli ctrlCli.Client, tc *v1alpha1.TidbCluster, expectNonExistedTc *v1alpha1.TidbCluster) error {
	checkTidbCluster := v1alpha1.TidbCluster{}
	clusterCli.Get(context.TODO(), types.NamespacedName{Namespace: tc.Namespace, Name: tc.Name}, &checkTidbCluster)

	if checkTidbCluster.Status.PD.Phase != v1alpha1.NormalPhase || checkTidbCluster.Status.TiKV.Phase != v1alpha1.NormalPhase || checkTidbCluster.Status.TiDB.Phase != v1alpha1.NormalPhase || checkTidbCluster.Status.TiFlash.Phase != v1alpha1.NormalPhase || checkTidbCluster.Status.Pump.Phase != v1alpha1.NormalPhase || checkTidbCluster.Status.TiCDC.Phase != v1alpha1.NormalPhase {
		return fmt.Errorf("the tc %s is not healthy", tc.Name)
	}

	for _, peerMember := range checkTidbCluster.Status.PD.PeerMembers {
		if strings.Contains(peerMember.Name, expectNonExistedTc.Name) {
			return fmt.Errorf("the PD peer members contain the member belonging to the deleted tc")
		}
	}

	for _, peerStore := range checkTidbCluster.Status.TiKV.PeerStores {
		if strings.Contains(peerStore.PodName, expectNonExistedTc.Name) {
			return fmt.Errorf("the TiKV peer stores contain the store belonging to the deleted tc")
		}
	}

	for _, peerStore := range checkTidbCluster.Status.TiFlash.PeerStores {
		if strings.Contains(peerStore.PodName, expectNonExistedTc.Name) {
			return fmt.Errorf("the TiFlash peer stores contain the store belonging to the deleted tc")
		}
	}
	return nil
}

func GetTCForAcrossKubernetes(ns, name, version, clusterDomain string, joinTC *v1alpha1.TidbCluster) *v1alpha1.TidbCluster {
	tc := fixture.GetTidbCluster(ns, name, version)
	tc = fixture.AddTiFlashForTidbCluster(tc)
	tc = fixture.AddTiCDCForTidbCluster(tc)
	tc = fixture.AddPumpForTidbCluster(tc)

	tc.Spec.PD.Replicas = 1
	tc.Spec.TiDB.Replicas = 1
	tc.Spec.TiKV.Replicas = 1
	tc.Spec.TiFlash.Replicas = 1
	tc.Spec.TiCDC.Replicas = 1
	tc.Spec.Pump.Replicas = 1
	tc.Spec.AcrossK8s = true

	tc.Spec.ClusterDomain = clusterDomain
	if joinTC != nil {
		tc.Spec.Cluster = &v1alpha1.TidbClusterRef{
			Namespace:     joinTC.Namespace,
			Name:          joinTC.Name,
			ClusterDomain: joinTC.Spec.ClusterDomain,
		}
	}

	return tc
}

func MustPrepareXK8sTLSResources(cli ctrlCli.Client, initialTC *v1alpha1.TidbCluster, restTCs []*v1alpha1.TidbCluster) {
	err := InstallTiDBIssuer(initialTC.Namespace, initialTC.Name)
	framework.ExpectNoError(err, "failed to install CA certificate")

	caSecret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: initialTC.Namespace, Name: fmt.Sprintf("%s-ca-secret", initialTC.Name)}}
	err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
		err = cli.Get(context.TODO(), ctrlCli.ObjectKeyFromObject(caSecret), caSecret)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	framework.ExpectNoError(err, "error export initial CA secert")

	for _, tc := range restTCs {
		caSecret.Namespace = tc.Namespace
		caSecret.ObjectMeta.ResourceVersion = ""
		cli.Create(context.TODO(), caSecret)

		err = InstallXK8sTiDBIssuer(tc.Namespace, tc.Name, initialTC.Name)
		framework.ExpectNoError(err, "failed to install tidb issuer for cluster %q", tc.Name)
	}

	tidbclusters := append(restTCs, initialTC)
	for _, tc := range tidbclusters {
		err := InstallXK8sTiDBCertificates(tc.Namespace, tc.Name, tc.Spec.ClusterDomain)
		framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tc.Name)

		err = InstallXK8sTiDBComponentsCertificates(tc.Namespace, tc.Name, tc.Spec.ClusterDomain, tc.Spec.PD == nil)
		framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tc.Name)
	}
}

func MustCreateXK8sTCWithComponentsReady(cli ctrlCli.Client, oa *tests.OperatorActions, tidbclusters []*v1alpha1.TidbCluster, tlsEnabled bool) {
	for _, tc := range tidbclusters {
		if tc.Spec.TiDB != nil {
			tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: tlsEnabled}
		}
		tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: tlsEnabled}
		err := cli.Create(context.TODO(), tc)
		framework.ExpectNoError(err, "failed to create TidbCluster %s/%s", tc.Namespace, tc.Name)
	}

	// for faster cluster creation, checking status after creating all clusters.
	for _, tc := range tidbclusters {
		err := oa.WaitForTidbClusterReady(tc, 15*time.Minute, 10*time.Second)
		framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", tc.Namespace, tc.Name)
	}
}

func CheckStatusWhenAcrossK8sWithTimeout(cli versioned.Interface, tidbclusters []*v1alpha1.TidbCluster, interval time.Duration, timeout time.Duration) error {
	var lastErr error
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		if err := CheckStatusWhenAcrossK8s(cli, tidbclusters); err != nil {
			lastErr = err
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("%s, last error: %v", err, lastErr)
	}
	return nil
}

func CheckStatusWhenAcrossK8s(cli versioned.Interface, tidbclusters []*v1alpha1.TidbCluster) error {
	// we use deploy cluster in multi namespace to mock deploying across kubernetes,
	// so we need to check status of tc to ensure `acrossK8s` is effect.

	for i := range tidbclusters {
		tc, err := cli.PingcapV1alpha1().TidbClusters(tidbclusters[i].Namespace).
			Get(context.TODO(), tidbclusters[i].Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		tidbclusters[i] = tc
	}

	for _, tc := range tidbclusters {
		clusterDomainSuffix := ""
		if tc.Spec.ClusterDomain != "" {
			clusterDomainSuffix = fmt.Sprintf(".%s", tc.Spec.ClusterDomain)
		}
		pdNameSuffix := fmt.Sprintf("%s-pd-peer.%s.svc%s", tc.Name, tc.Namespace, clusterDomainSuffix)
		tikvIPSuffix := fmt.Sprintf("%s-tikv-peer.%s.svc%s", tc.Name, tc.Namespace, clusterDomainSuffix)
		tiflashIPSuffix := fmt.Sprintf("%s-tiflash-peer.%s.svc%s", tc.Name, tc.Namespace, clusterDomainSuffix)
		pumpHostSuffix := fmt.Sprintf("%s-pump.%s.svc%s", tc.Name, tc.Namespace, clusterDomainSuffix)

		// pd
		if tc.Spec.PD != nil && tc.Spec.PD.Replicas != 0 {
			for name, info := range tc.Status.PD.Members {
				// check member format
				ok := strings.HasSuffix(name, pdNameSuffix)
				if !ok {
					return fmt.Errorf("pd name %s don't have suffix %s", name, pdNameSuffix)
				}
				ok = strings.HasSuffix(info.Name, pdNameSuffix)
				if !ok {
					return fmt.Errorf("pd name %s don't have suffix %s", name, pdNameSuffix)
				}
				ok = strings.Contains(info.ClientURL, pdNameSuffix)
				if !ok {
					return fmt.Errorf("pd clientURL %s don't contain %s", name, pdNameSuffix)
				}

				// check if member exist in other tc
				for _, othertc := range tidbclusters {
					if othertc == tc || othertc.Spec.PD == nil {
						continue
					}
					exist := false
					for peerName := range othertc.Status.PD.PeerMembers {
						if peerName == name {
							exist = true
							break
						}
					}
					if !exist {
						return fmt.Errorf("pd name %s don't exist in other tc %s/%s", name, othertc.Namespace, othertc.Name)
					}
				}
			}
		}
		// tikv
		if tc.Spec.TiKV != nil && tc.Spec.TiKV.Replicas != 0 {
			for _, ownStore := range tc.Status.TiKV.Stores {
				// check store format
				ok := strings.HasSuffix(ownStore.IP, tikvIPSuffix)
				if !ok {
					return fmt.Errorf("tikv store ip %s don't have suffix %s", ownStore.IP, tiflashIPSuffix)
				}

				// check if store exist in other tc
				for _, othertc := range tidbclusters {
					if othertc == tc || othertc.Spec.TiKV == nil {
						continue
					}
					exist := false
					for _, peerStore := range othertc.Status.TiKV.PeerStores {
						if peerStore.IP == ownStore.IP {
							exist = true
							break
						}
					}
					if !exist {
						return fmt.Errorf("tikv store ip %s don't exist in other tc %s/%s",
							ownStore.IP, othertc.Namespace, othertc.Name)
					}
				}
			}

		}
		// tiflash
		if tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas != 0 {
			for _, ownStore := range tc.Status.TiFlash.Stores {
				// check store format
				ok := strings.HasSuffix(ownStore.IP, tiflashIPSuffix)
				if !ok {
					return fmt.Errorf("tiflash store ip %s don't have suffix %s", ownStore.IP, tiflashIPSuffix)
				}

				// check if store exist in other tc
				for _, othertc := range tidbclusters {
					if othertc == tc || othertc.Spec.TiFlash == nil {
						continue
					}
					exist := false
					for _, peerStore := range othertc.Status.TiFlash.PeerStores {
						if peerStore.IP == ownStore.IP {
							exist = true
							break
						}
					}
					if !exist {
						return fmt.Errorf("tiflash store ip %s don't exist in other tc %s/%s",
							ownStore.IP, othertc.Namespace, othertc.Name)
					}
				}
			}

		}
		// pump
		if tc.Spec.Pump != nil && tc.Spec.Pump.Replicas != 0 {
			// check member format
			ownMembers := []*v1alpha1.PumpNodeStatus{}
			for _, member := range tc.Status.Pump.Members {
				if strings.Contains(member.Host, pumpHostSuffix) {
					ownMembers = append(ownMembers, member)
					break
				}
			}
			if len(ownMembers) == 0 {
				return fmt.Errorf("all pump member hosts don't contain %s", pumpHostSuffix)
			}

			// check if member exist in other tc
			for _, ownMember := range ownMembers {
				for _, othertc := range tidbclusters {
					if othertc == tc || othertc.Spec.Pump == nil {
						continue
					}
					exist := false
					for _, peerMember := range othertc.Status.Pump.Members {
						if peerMember.Host == ownMember.Host {
							exist = true
							break
						}
					}
					if !exist {
						return fmt.Errorf("pump member host %s don't exist in other tc %s/%s",
							ownMember.Host, othertc.Namespace, othertc.Name)
					}
				}
			}

		}
	}

	return nil
}
