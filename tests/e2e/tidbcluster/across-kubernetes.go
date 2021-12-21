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
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
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
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
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

	ginkgo.Describe("[Basic]", func() {

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

			ginkgo.By("Deploy the basic cluster-1")
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc1, 5*time.Minute, 10*time.Second)

			ginkgo.By("Deploy the basic cluster-2")
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc2, 5*time.Minute, 10*time.Second)

			ginkgo.By("Deploy the basic cluster-3")
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc3, 5*time.Minute, 10*time.Second)

			ginkgo.By("Deploy status of all clusters")
			err := CheckClusterDomainEffect(cli, []*v1alpha1.TidbCluster{tc1, tc2, tc3})
			framework.ExpectNoError(err, "failed to check status")
		})

		ginkgo.It("Deploy cluster with TLS-enabled across kubernetes", func() {
			ns1, ns2 := namespaces[0], namespaces[1]
			tcName1, tcName2 := "tls-cluster-1", "tls-cluster-2"
			tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)

			ginkgo.By("Installing initial tidb CA certificate")
			err := InstallTiDBIssuer(ns1, tcName1)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Export initial CA secret and install into other tidb clusters")
			var caSecret *v1.Secret
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				caSecret, err = c.CoreV1().Secrets(ns1).Get(context.TODO(), fmt.Sprintf("%s-ca-secret", tcName1), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "error export initial CA secert")
			caSecret.Namespace = ns2
			caSecret.ObjectMeta.ResourceVersion = ""
			err = genericCli.Create(context.TODO(), caSecret)
			framework.ExpectNoError(err, "error installing CA secert into cluster %q", tcName2)

			ginkgo.By("Installing tidb cluster issuer with initial ca")
			err = InstallXK8sTiDBIssuer(ns2, tcName2, tcName1)
			framework.ExpectNoError(err, "failed to install tidb issuer for cluster %q", tcName2)

			ginkgo.By("Installing tidb server and client certificate")
			err = InstallXK8sTiDBCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName1)
			err = InstallXK8sTiDBCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName2)

			ginkgo.By("Installing tidb components certificates")
			err = InstallXK8sTiDBComponentsCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName1)
			err = InstallXK8sTiDBComponentsCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName2)

			ginkgo.By("Installing separate dashboard client certificate")
			err = installPDDashboardCertificates(ns1, tcName1)
			framework.ExpectNoError(err, "failed to install separate dashboard client certificate for cluster: %q", tcName1)
			err = installPDDashboardCertificates(ns2, tcName2)
			framework.ExpectNoError(err, "failed to install separate dashboard client certificate for cluster: %q", tcName2)

			ginkgo.By("Creating tidb cluster-1 with TLS enabled")
			tc1DashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName1)
			tc1.Spec.PD.TLSClientSecretName = &tc1DashTLSName
			tc1.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc1.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc1, 10*time.Minute, 30*time.Second)

			ginkgo.By("Creating tidb cluster-2 with TLS enabled")
			tc2DashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName2)
			tc2.Spec.PD.TLSClientSecretName = &tc2DashTLSName
			tc2.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc2.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc2, 10*time.Minute, 30*time.Second)

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

			ginkgo.By("Connecting to tidb server to verify the connection is TLS enabled")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns1, tcName1, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName1)
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns2, tcName2, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName2)
		})
	})

	ginkgo.Describe("4300[Failover]", func() {
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

			ginkgo.By("Installing initial tidb CA certificate")
			err := InstallTiDBIssuer(ns1, tcName1)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Export initial CA secret and install into other tidb clusters")
			var caSecret *v1.Secret
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				caSecret, err = c.CoreV1().Secrets(ns1).Get(context.TODO(), fmt.Sprintf("%s-ca-secret", tcName1), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "error export initial CA secert")
			caSecret.ObjectMeta.ResourceVersion = ""
			caSecret.Namespace = ns2
			err = genericCli.Create(context.TODO(), caSecret)
			framework.ExpectNoError(err, "error installing CA secert into cluster %q", tcName2)
			caSecret.ObjectMeta.ResourceVersion = ""
			caSecret.Namespace = ns3
			err = genericCli.Create(context.TODO(), caSecret)
			framework.ExpectNoError(err, "error installing CA secert into cluster %q", tcName3)

			ginkgo.By("Installing tidb cluster issuer with initial ca")
			err = InstallXK8sTiDBIssuer(ns2, tcName2, tcName1)
			framework.ExpectNoError(err, "failed to install tidb issuer for cluster %q", tcName2)
			err = InstallXK8sTiDBIssuer(ns3, tcName3, tcName1)
			framework.ExpectNoError(err, "failed to install tidb issuer for cluster %q", tcName3)

			ginkgo.By("Installing tidb server and client certificate")
			err = InstallXK8sTiDBCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName1)
			err = InstallXK8sTiDBCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName2)
			err = InstallXK8sTiDBCertificates(ns3, tcName3, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName3)

			ginkgo.By("Installing tidb components certificates")
			err = InstallXK8sTiDBComponentsCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName1)
			err = InstallXK8sTiDBComponentsCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName2)
			err = InstallXK8sTiDBComponentsCertificates(ns3, tcName3, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName3)

			ginkgo.By("Creating x-k8s tidb clusters with TLS enabled")
			tc1.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc1.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc1, 10*time.Minute, 30*time.Second)

			tc2.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc2.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc2, 10*time.Minute, 30*time.Second)

			tc3.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc3.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc3, 10*time.Minute, 30*time.Second)

			ginkgo.By("Fail PD in cluster-1 by setting a wrong image")
			local, err := cli.PingcapV1alpha1().TidbClusters(tc1.Namespace).Get(context.TODO(), tc1.Name, metav1.GetOptions{})
			framework.ExpectNoError(err, "getting tidbcluster %s/%s", tc1.Namespace, tc1.Name)
			local.Spec.PD.BaseImage = inexistentBaseImage
			err = genericCli.Update(context.TODO(), local)
			framework.ExpectNoError(err, "updating pd with an inexistent image %q for %q", tc1.Spec.PD.BaseImage, tcName1)

			ginkgo.By("Waiting for pd pods to be in unhealthy state")
			err = utiltc.WaitForTidbClusterCondition(cli, ns1, tcName1, time.Minute*5, func(tc *v1alpha1.TidbCluster) (bool, error) {
				return !tc.Status.PD.Synced, nil
			})
			framework.ExpectNoError(err, "waiting for the pd to be in unhealthy state")

			ginkgo.By("Restart all component pods except pd and check cluster status")
			podList, err := c.CoreV1().Pods(ns1).List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err, "list pods under namespace %q", ns1)
			for _, pod := range podList.Items {
				if !strings.Contains(pod.Name, "pd") {
					err := c.CoreV1().Pods(ns1).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
					framework.ExpectNoError(err, "failed to delete pod %q", pod.Name)
				}
			}

			componentsFilter := make(map[v1alpha1.MemberType]struct{}, 2)
			componentsFilter[v1alpha1.PDMemberType] = struct{}{}
			err = oa.WaitForTidbComponentsReady(tc1, componentsFilter, 5*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "restarting components when pd failed")

			ginkgo.By("Fail TiKV in cluster-1 by setting a wrong image")
			local, err = cli.PingcapV1alpha1().TidbClusters(tc1.Namespace).Get(context.TODO(), tc1.Name, metav1.GetOptions{})
			framework.ExpectNoError(err, "getting tidbcluster %s/%s", tc1.Namespace, tc1.Name)
			local.Spec.TiKV.BaseImage = inexistentBaseImage
			err = genericCli.Update(context.TODO(), local)
			framework.ExpectNoError(err, "updating tikv with an inexistent image %q for %q", tc1.Spec.TiKV.BaseImage, tcName1)
			// force operator to trigger a pd upgrade when pd is down.
			err = c.AppsV1().StatefulSets(ns1).Delete(context.TODO(), fmt.Sprintf("%s-tikv", tcName1), metav1.DeleteOptions{})
			framework.ExpectNoError(err, "deleting sts of tikv for %q", tcName1)

			ginkgo.By("Waiting for tikv store to be in Down state")
			err = utiltc.WaitForTidbClusterCondition(cli, tc1.Namespace, tc1.Name, time.Minute*5, func(tc *v1alpha1.TidbCluster) (bool, error) {
				for _, store := range tc.Status.TiKV.Stores {
					if store.State == v1alpha1.TiKVStateDown {
						return true, nil
					}
				}
				return false, nil
			})
			framework.ExpectNoError(err, "waiting for the tikv store to be in down state")

			ginkgo.By("Restart other component pods and check cluster status")
			podList, err = c.CoreV1().Pods(ns1).List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err, "list pods under namespace %q", ns1)
			for _, pod := range podList.Items {
				if !strings.Contains(pod.Name, "pd") && !strings.Contains(pod.Name, "tikv") {
					err := c.CoreV1().Pods(ns1).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
					framework.ExpectNoError(err, "failed to delete pod %q", pod.Name)
				}
			}

			componentsFilter[v1alpha1.TiKVMemberType] = struct{}{}
			err = oa.WaitForTidbComponentsReady(tc1, componentsFilter, 5*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "restarting components when tikv fails")
		})

		ginkgo.It("TiDBCluter Should work when one of the TidbCluster or the k8s fails", func() {
			ns1, ns2, ns3 := namespaces[0], namespaces[1], namespaces[2]
			tcName1, tcName2, tcName3 := "cluster-1", "cluster-2", "cluster-3"
			tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)
			tc3 := GetTCForAcrossKubernetes(ns3, tcName3, version, clusterDomain, tc1)

			ginkgo.By("Installing initial tidb CA certificate")
			err := InstallTiDBIssuer(ns1, tcName1)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Export initial CA secret and install into other tidb clusters")
			var caSecret *v1.Secret
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				caSecret, err = c.CoreV1().Secrets(ns1).Get(context.TODO(), fmt.Sprintf("%s-ca-secret", tcName1), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "error export initial CA secert")
			caSecret.ObjectMeta.ResourceVersion = ""
			caSecret.Namespace = ns2
			err = genericCli.Create(context.TODO(), caSecret)
			framework.ExpectNoError(err, "error installing CA secert into cluster %q", tcName2)
			caSecret.ObjectMeta.ResourceVersion = ""
			caSecret.Namespace = ns3
			err = genericCli.Create(context.TODO(), caSecret)
			framework.ExpectNoError(err, "error installing CA secert into cluster %q", tcName3)

			ginkgo.By("Installing tidb cluster issuer with initial ca")
			err = InstallXK8sTiDBIssuer(ns2, tcName2, tcName1)
			framework.ExpectNoError(err, "failed to install tidb issuer for cluster %q", tcName2)
			err = InstallXK8sTiDBIssuer(ns3, tcName3, tcName1)
			framework.ExpectNoError(err, "failed to install tidb issuer for cluster %q", tcName3)

			ginkgo.By("Installing tidb server and client certificate")
			err = InstallXK8sTiDBCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName1)
			err = InstallXK8sTiDBCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName2)
			err = InstallXK8sTiDBCertificates(ns3, tcName3, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName3)

			ginkgo.By("Installing tidb components certificates")
			err = InstallXK8sTiDBComponentsCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName1)
			err = InstallXK8sTiDBComponentsCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName2)
			err = InstallXK8sTiDBComponentsCertificates(ns3, tcName3, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName3)

			ginkgo.By("Creating x-k8s tidb clusters with TLS enabled")
			tc1.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc1.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc1, 10*time.Minute, 30*time.Second)

			tc2.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc2.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc2, 10*time.Minute, 30*time.Second)

			tc3.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc3.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc3, 10*time.Minute, 30*time.Second)

			ginkgo.By("Fail all components in cluster-2 by deleting ns")
			propForeground := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				PropagationPolicy: &propForeground,
			}
			err = c.CoreV1().Namespaces().Delete(context.TODO(), ns2, opts)
			framework.ExpectNoError(err, "deleting namespsace %q", ns2)

			ginkgo.By("Waiting for namespace to be deleted")
			wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
				_, err = c.CoreV1().Namespaces().Get(context.TODO(), ns2, metav1.GetOptions{})
				if err != nil {
					return true, nil
				}
				return false, nil
			})

			ginkgo.By("Check status of other clusters")
			err = oa.WaitForTidbClusterReady(tc1, 10*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "%q cluster not healthy after cluster %q fail", tcName1, tcName2)
			err = oa.WaitForTidbClusterReady(tc3, 10*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "%q cluster not healthy after cluster %q fail", tcName3, tcName2)

			ginkgo.By("Check functionality of other clusters by querying tidb")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns1, tcName1, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName1)
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns3, tcName3, ""))
			framework.ExpectNoError(err, "connect to TLS tidb %s timeout", tcName3)
		})

		ginkgo.It("Failed to join in cluster-1 when PD crash and succeed after pd restart", func() {
			ns1, ns2 := namespaces[0], namespaces[1]
			tcName1, tcName2 := "cluster-1", "cluster-2"
			tc1 := GetTCForAcrossKubernetes(ns1, tcName1, version, clusterDomain, nil)
			tc2 := GetTCForAcrossKubernetes(ns2, tcName2, version, clusterDomain, tc1)

			ginkgo.By("Installing initial tidb CA certificate")
			err := InstallTiDBIssuer(ns1, tcName1)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Export initial CA secret and install into other tidb clusters")
			var caSecret *v1.Secret
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				caSecret, err = c.CoreV1().Secrets(ns1).Get(context.TODO(), fmt.Sprintf("%s-ca-secret", tcName1), metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			framework.ExpectNoError(err, "error export initial CA secert")
			caSecret.Namespace = ns2
			caSecret.ObjectMeta.ResourceVersion = ""
			err = genericCli.Create(context.TODO(), caSecret)
			framework.ExpectNoError(err, "error installing CA secert into cluster %q", tcName2)

			ginkgo.By("Installing tidb cluster issuer with initial ca")
			err = InstallXK8sTiDBIssuer(ns2, tcName2, tcName1)
			framework.ExpectNoError(err, "failed to install tidb issuer for cluster %q", tcName2)

			ginkgo.By("Installing tidb server and client certificate")
			err = InstallXK8sTiDBCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName1)
			err = InstallXK8sTiDBCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate for cluster: %q", tcName2)

			ginkgo.By("Installing tidb components certificates")
			err = InstallXK8sTiDBComponentsCertificates(ns1, tcName1, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName1)
			err = InstallXK8sTiDBComponentsCertificates(ns2, tcName2, clusterDomain)
			framework.ExpectNoError(err, "failed to install tidb components certificates for cluster: %q", tcName2)

			ginkgo.By("Creating tidb cluster-1 with TLS enabled")
			tc1.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc1.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc1, 10*time.Minute, 30*time.Second)

			ginkgo.By("Fail PD in cluster-1 by setting a wrong image")
			local, err := cli.PingcapV1alpha1().TidbClusters(tc1.Namespace).Get(context.TODO(), tc1.Name, metav1.GetOptions{})
			framework.ExpectNoError(err, "getting tidbcluster %s/%s", tc1.Namespace, tc1.Name)
			baseImage := local.Spec.PD.BaseImage
			local.Spec.PD.BaseImage = inexistentBaseImage
			err = genericCli.Update(context.TODO(), local)
			framework.ExpectNoError(err, "updating pd with an inexistent image %q for %q", tc1.Spec.PD.BaseImage, tcName1)

			ginkgo.By("Waiting for pd pods to be in unhealthy state")
			err = utiltc.WaitForTidbClusterCondition(cli, ns1, tcName1, time.Minute*5, func(tc *v1alpha1.TidbCluster) (bool, error) {
				return !tc.Status.PD.Synced, nil
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
			tc1.Spec.PD.BaseImage = baseImage
			err = genericCli.Update(context.TODO(), tc1)
			framework.ExpectNoError(err, "updating pd with previous image %q for %q", tc1.Spec.PD.BaseImage, tcName1)
			// force operator to trigger a pd upgrade when pd is down.
			err = c.AppsV1().StatefulSets(ns1).Delete(context.TODO(), fmt.Sprintf("%s-pd", tcName1), metav1.DeleteOptions{})
			framework.ExpectNoError(err, "deleting sts of pd for %q", tcName1)

			ginkgo.By("Join cluster-2 into cluster-1 when pd running normally")
			err = oa.WaitForTidbClusterReady(tc2, 10*time.Minute, 30*time.Second)
			framework.ExpectNoError(err, "%q failed to join into %q", tcName2, tcName1)
		})
	})
})

func GetTCForAcrossKubernetes(ns, name, version, clusterDomain string, joinTC *v1alpha1.TidbCluster) *v1alpha1.TidbCluster {
	tc := fixture.GetTidbCluster(ns, name, version)
	tc = fixture.AddTiFlashForTidbCluster(tc)
	// tc = fixture.AddTiCDCForTidbCluster(tc)
	// tc = fixture.AddPumpForTidbCluster(tc)

	tc.Spec.PD.Replicas = 1
	tc.Spec.TiDB.Replicas = 1
	tc.Spec.TiKV.Replicas = 1
	tc.Spec.TiFlash.Replicas = 1
	// tc.Spec.TiCDC.Replicas = 1
	// tc.Spec.Pump.Replicas = 1

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

func CheckClusterDomainEffect(cli versioned.Interface, tidbclusters []*v1alpha1.TidbCluster) error {
	// we use deploy cluster in multi namespace to mock deploying across kubernetes,
	// so we need to check status of tc to ensure `clusterDomain` is effect.

	for i := range tidbclusters {
		tc, err := cli.PingcapV1alpha1().TidbClusters(tidbclusters[i].Namespace).
			Get(context.TODO(), tidbclusters[i].Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		tidbclusters[i] = tc
	}

	for _, tc := range tidbclusters {
		pdNameSuffix := fmt.Sprintf("%s-pd-peer.%s.svc.%s", tc.Name, tc.Namespace, tc.Spec.ClusterDomain)
		tikvIPSuffix := fmt.Sprintf("%s-tikv-peer.%s.svc.%s", tc.Name, tc.Namespace, tc.Spec.ClusterDomain)
		tiflashIPSuffix := fmt.Sprintf("%s-tiflash-peer.%s.svc.%s", tc.Name, tc.Namespace, tc.Spec.ClusterDomain)
		pumpHostSuffix := fmt.Sprintf("%s-pump.%s.svc.%s", tc.Name, tc.Namespace, tc.Spec.ClusterDomain)

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
					if othertc == tc {
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
					if othertc == tc {
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
					if othertc == tc {
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
					if othertc == tc {
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
