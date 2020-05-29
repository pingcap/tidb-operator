// Copyright 2020 PingCAP, Inc.
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
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/tests"
	"github.com/pingcap/tidb-operator/tests/apiserver"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/kubernetes/test/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("[tidb-operator][Stability]", func() {
	f := e2eframework.NewDefaultFramework("br")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var aggrCli aggregatorclient.Interface
	var apiExtCli apiextensionsclientset.Interface
	var oa tests.OperatorActions
	var cfg *tests.Config
	var config *restclient.Config
	var ocfg *tests.OperatorConfig
	var genericCli client.Client
	var fwCancel context.CancelFunc
	var fw portforward.PortForward
	/**
	 * StatefulSet or AdvancedStatefulSet interface.
	 */
	var stsGetter func(namespace string) typedappsv1.StatefulSetInterface

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
		genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
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
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			stsGetter = helper.NewHijackClient(c, asCli).AppsV1().StatefulSets
		} else {
			stsGetter = c.AppsV1().StatefulSets
		}
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.It("[Feature: AutoFailover] clear TiDB failureMembers when scale TiDB to zero", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "tidb-scale", "admin", utilimage.TiDBV3Version)
		cluster.Resources["pd.replicas"] = "3"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"

		cluster.TiDBPreStartScript = strconv.Quote("exit 1")
		oa.DeployTidbClusterOrDie(&cluster)

		e2elog.Logf("checking tidb cluster [%s/%s] failed member", cluster.Namespace, cluster.ClusterName)
		ns := cluster.Namespace
		tcName := cluster.ClusterName
		err := wait.PollImmediate(15*time.Second, 15*time.Minute, func() (bool, error) {
			var tc *v1alpha1.TidbCluster
			var err error
			if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
				e2elog.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			if len(tc.Status.TiDB.FailureMembers) == 0 {
				e2elog.Logf("the number of failed member is zero")
				return false, nil
			}
			e2elog.Logf("the number of failed member is not zero (current: %d)", len(tc.Status.TiDB.FailureMembers))
			return true, nil
		})
		framework.ExpectNoError(err, "tidb failover not work")

		cluster.ScaleTiDB(0)
		oa.ScaleTidbClusterOrDie(&cluster)

		e2elog.Logf("checking tidb cluster [%s/%s] scale to zero", cluster.Namespace, cluster.ClusterName)
		err = wait.PollImmediate(15*time.Second, 10*time.Minute, func() (bool, error) {
			var tc *v1alpha1.TidbCluster
			var err error
			if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
				e2elog.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			if tc.Status.TiDB.StatefulSet.Replicas != 0 {
				e2elog.Logf("failed to scale tidb member to zero (current: %d)", tc.Status.TiDB.StatefulSet.Replicas)
				return false, nil
			}
			if len(tc.Status.TiDB.FailureMembers) != 0 {
				e2elog.Logf("failed to clear fail member (current: %d)", len(tc.Status.TiDB.FailureMembers))
				return false, nil
			}
			e2elog.Logf("scale tidb member to zero successfully")
			return true, nil
		})
		framework.ExpectNoError(err, "not clear TiDB failureMembers when scale TiDB to zero")
	})

	ginkgo.It("Test aggregated apiserver", func() {
		ginkgo.By(fmt.Sprintf("Starting to test apiserver, test apiserver image: %s", cfg.E2EImage))
		framework.Logf("config: %v", config)
		aaCtx := apiserver.NewE2eContext(ns, config, cfg.E2EImage)
		defer aaCtx.Clean()
		aaCtx.Setup()
		aaCtx.Do()
	})

	ginkgo.It("Restarter: Testing restarting by annotations", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "restarter", "admin", utilimage.TiDBV3Version)
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")
		pd_0, err := c.CoreV1().Pods(ns).Get(operatorUtils.GetPodName(tc, v1alpha1.PDMemberType, 0), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get pd-0")
		tikv_0, err := c.CoreV1().Pods(ns).Get(operatorUtils.GetPodName(tc, v1alpha1.TiKVMemberType, 0), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tikv-0")
		tidb_0, err := c.CoreV1().Pods(ns).Get(operatorUtils.GetPodName(tc, v1alpha1.TiDBMemberType, 0), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidb-0")
		pd_0.Annotations[label.AnnPodDeferDeleting] = "true"
		tikv_0.Annotations[label.AnnPodDeferDeleting] = "true"
		tidb_0.Annotations[label.AnnPodDeferDeleting] = "true"
		_, err = c.CoreV1().Pods(ns).Update(pd_0)
		framework.ExpectNoError(err, "Expected update pd-0 restarting ann")
		_, err = c.CoreV1().Pods(ns).Update(tikv_0)
		framework.ExpectNoError(err, "Expected update tikv-0 restarting ann")
		_, err = c.CoreV1().Pods(ns).Update(tidb_0)
		framework.ExpectNoError(err, "Expected update tidb-0 restarting ann")

		f := func(name, namespace string, uid types.UID) (bool, error) {
			pod, err := c.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					// ignore not found error (pod is deleted and recreated again in restarting)
					return false, nil
				}
				return false, err
			}
			if _, existed := pod.Annotations[label.AnnPodDeferDeleting]; existed {
				return false, nil
			}
			if uid == pod.UID {
				return false, nil
			}
			return true, nil
		}

		err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
			isPdRestarted, err := f(pd_0.Name, ns, pd_0.UID)
			if !(isPdRestarted && err == nil) {
				return isPdRestarted, err
			}
			isTiKVRestarted, err := f(tikv_0.Name, ns, tikv_0.UID)
			if !(isTiKVRestarted && err == nil) {
				return isTiKVRestarted, err
			}
			isTiDBRestarted, err := f(tidb_0.Name, ns, tidb_0.UID)
			if !(isTiDBRestarted && err == nil) {
				return isTiDBRestarted, err
			}
			return true, nil
		})
		framework.ExpectNoError(err, "Expected tidbcluster pod restarted")
	})

	ginkgo.It("[Feature: AutoFailover] clear TiDB failureMembers when scale TiDB to zero", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "tidb-scale", "admin", utilimage.TiDBV3Version)
		cluster.Resources["pd.replicas"] = "3"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"

		cluster.TiDBPreStartScript = strconv.Quote("exit 1")
		oa.DeployTidbClusterOrDie(&cluster)

		e2elog.Logf("checking tidb cluster [%s/%s] failed member", cluster.Namespace, cluster.ClusterName)
		ns := cluster.Namespace
		tcName := cluster.ClusterName
		err := wait.PollImmediate(15*time.Second, 15*time.Minute, func() (bool, error) {
			var tc *v1alpha1.TidbCluster
			var err error
			if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
				e2elog.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			if len(tc.Status.TiDB.FailureMembers) == 0 {
				e2elog.Logf("the number of failed member is zero")
				return false, nil
			}
			e2elog.Logf("the number of failed member is not zero (current: %d)", len(tc.Status.TiDB.FailureMembers))
			return true, nil
		})
		framework.ExpectNoError(err, "tidb failover not work")

		cluster.ScaleTiDB(0)
		oa.ScaleTidbClusterOrDie(&cluster)

		e2elog.Logf("checking tidb cluster [%s/%s] scale to zero", cluster.Namespace, cluster.ClusterName)
		err = wait.PollImmediate(15*time.Second, 10*time.Minute, func() (bool, error) {
			var tc *v1alpha1.TidbCluster
			var err error
			if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
				e2elog.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			if tc.Status.TiDB.StatefulSet.Replicas != 0 {
				e2elog.Logf("failed to scale tidb member to zero (current: %d)", tc.Status.TiDB.StatefulSet.Replicas)
				return false, nil
			}
			if len(tc.Status.TiDB.FailureMembers) != 0 {
				e2elog.Logf("failed to clear fail member (current: %d)", len(tc.Status.TiDB.FailureMembers))
				return false, nil
			}
			e2elog.Logf("scale tidb member to zero successfully")
			return true, nil
		})
		framework.ExpectNoError(err, "not clear TiDB failureMembers when scale TiDB to zero")
	})

	ginkgo.It("Ensure Service NodePort Not Change", func() {
		// Create TidbCluster with NodePort to check whether node port would change
		nodeTc := fixture.GetTidbCluster(ns, "nodeport", utilimage.TiDBV3Version)
		nodeTc.Spec.PD.Replicas = 1
		nodeTc.Spec.TiKV.Replicas = 1
		nodeTc.Spec.TiDB.Replicas = 1
		nodeTc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
			ServiceSpec: v1alpha1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
			},
		}
		err := genericCli.Create(context.TODO(), nodeTc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(nodeTc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster ready")

		// expect tidb service type is Nodeport
		var s *corev1.Service
		err = wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
			s, err = c.CoreV1().Services(ns).Get("nodeport-tidb", metav1.GetOptions{})
			if err != nil {
				framework.Logf(err.Error())
				return false, nil
			}
			if s.Spec.Type != corev1.ServiceTypeNodePort {
				return false, fmt.Errorf("nodePort tidbcluster tidb service type isn't NodePort")
			}
			return true, nil
		})
		framework.ExpectNoError(err)
		ports := s.Spec.Ports

		// f is the function to check whether service nodeport have changed for 1 min
		f := func() error {
			return wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
				s, err := c.CoreV1().Services(ns).Get("nodeport-tidb", metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if s.Spec.Type != corev1.ServiceTypeNodePort {
					return false, err
				}
				for _, dport := range s.Spec.Ports {
					for _, eport := range ports {
						if dport.Port == eport.Port && dport.NodePort != eport.NodePort {
							return false, fmt.Errorf("nodePort tidbcluster tidb service NodePort changed")
						}
					}
				}
				return false, nil
			})
		}
		// check whether nodeport have changed for 1 min
		err = f()
		framework.ExpectEqual(err, wait.ErrWaitTimeout)
		framework.Logf("tidbcluster tidb service NodePort haven't changed")

		nodeTc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get("nodeport", metav1.GetOptions{})
		framework.ExpectNoError(err)
		err = controller.GuaranteedUpdate(genericCli, nodeTc, func() error {
			nodeTc.Spec.TiDB.Service.Annotations = map[string]string{
				"foo": "bar",
			}
			return nil
		})
		framework.ExpectNoError(err)

		// check whether the tidb svc have updated
		err = wait.Poll(5*time.Second, 2*time.Minute, func() (done bool, err error) {
			s, err := c.CoreV1().Services(ns).Get("nodeport-tidb", metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if s.Annotations == nil {
				return false, nil
			}
			v, ok := s.Annotations["foo"]
			if !ok {
				return false, nil
			}
			if v != "bar" {
				return false, fmt.Errorf("tidb svc annotation foo not equal bar")
			}
			return true, nil
		})
		framework.ExpectNoError(err)
		framework.Logf("tidb nodeport svc updated")

		// check whether nodeport have changed for 1 min
		err = f()
		framework.ExpectEqual(err, wait.ErrWaitTimeout)
		framework.Logf("nodePort tidbcluster tidb service NodePort haven't changed after update")
	})

	updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
	ginkgo.It("API: Migrate from helm to CRD", func() {
		cluster := newTidbClusterConfig(e2econfig.TestConfig, ns, "helm-migration", "admin", utilimage.TiDBV3Version)
		cluster.Resources["pd.replicas"] = "1"
		cluster.Resources["tikv.replicas"] = "1"
		cluster.Resources["tidb.replicas"] = "1"
		oa.DeployTidbClusterOrDie(&cluster)
		oa.CheckTidbClusterStatusOrDie(&cluster)

		tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")

		ginkgo.By("Discovery service should be reconciled by tidb-operator")
		discoveryName := controller.DiscoveryMemberName(tc.Name)
		discoveryDep, err := c.AppsV1().Deployments(tc.Namespace).Get(discoveryName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get discovery deployment")
		WaitObjectToBeControlledByOrDie(genericCli, discoveryDep, tc, 5*time.Minute)

		err = utils.WaitForDeploymentComplete(c, discoveryDep, e2elog.Logf, 10*time.Second, 5*time.Minute)
		framework.ExpectNoError(err, "Discovery Deployment should be healthy after managed by tidb-operator")

		err = genericCli.Delete(context.TODO(), discoveryDep)
		framework.ExpectNoError(err, "Expected to delete deployment")

		err = wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
			_, err := c.AppsV1().Deployments(tc.Namespace).Get(discoveryName, metav1.GetOptions{})
			if err != nil {
				e2elog.Logf("wait discovery deployment get created again: %v", err)
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "Discovery Deployment should be recovered by tidb-operator after deletion")

		ginkgo.By("Managing TiDB configmap in TidbCluster CRD in-place should not trigger rolling-udpate")
		// TODO: modify other cases to manage TiDB configmap in CRD by default
		setNameToRevision := map[string]string{
			controller.PDMemberName(tc.Name):   "",
			controller.TiKVMemberName(tc.Name): "",
			controller.TiDBMemberName(tc.Name): "",
		}

		for setName := range setNameToRevision {
			oldSet, err := stsGetter(tc.Namespace).Get(setName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected get statefulset %s", setName)

			oldRev := oldSet.Status.CurrentRevision
			framework.ExpectEqual(oldSet.Status.UpdateRevision, oldRev, "Expected statefulset %s is not upgrading", setName)

			setNameToRevision[setName] = oldRev
		}

		tc, err = cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected get tidbcluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.TiDB.Config = &v1alpha1.TiDBConfig{}
			tc.Spec.TiDB.ConfigUpdateStrategy = &updateStrategy
			tc.Spec.TiKV.Config = &v1alpha1.TiKVConfig{}
			tc.Spec.TiKV.ConfigUpdateStrategy = &updateStrategy
			tc.Spec.PD.Config = &v1alpha1.PDConfig{}
			tc.Spec.PD.ConfigUpdateStrategy = &updateStrategy
			return nil
		})
		framework.ExpectNoError(err, "Expected update tidbcluster")

		// check for 2 minutes to ensure the tidb statefulset do not get rolling-update
		err = wait.PollImmediate(10*time.Second, 2*time.Minute, func() (bool, error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Expected get tidbcluster")
			framework.ExpectEqual(tc.Status.PD.Phase, v1alpha1.NormalPhase, "PD should not be updated")
			framework.ExpectEqual(tc.Status.TiKV.Phase, v1alpha1.NormalPhase, "TiKV should not be updated")
			framework.ExpectEqual(tc.Status.TiDB.Phase, v1alpha1.NormalPhase, "TiDB should not be updated")

			for setName, oldRev := range setNameToRevision {
				newSet, err := stsGetter(tc.Namespace).Get(setName, metav1.GetOptions{})
				framework.ExpectNoError(err, "Expected get tidb statefulset")
				framework.ExpectEqual(newSet.Status.CurrentRevision, oldRev, "Expected no rolling-update of %s when manage config in-place", setName)
				framework.ExpectEqual(newSet.Status.UpdateRevision, oldRev, "Expected no rolling-update of %s when manage config in-place", setName)
			}
			return false, nil
		})

		if err != wait.ErrWaitTimeout {
			e2elog.Failf("Unexpected error when checking tidb statefulset will not get rolling-update: %v", err)
		}

		err = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
			for setName := range setNameToRevision {
				newSet, err := stsGetter(tc.Namespace).Get(setName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				usingName := member.FindConfigMapVolume(&newSet.Spec.Template.Spec, func(name string) bool {
					return strings.HasPrefix(name, setName)
				})
				if usingName == "" {
					e2elog.Failf("cannot find configmap that used by %s", setName)
				}
				usingCm, err := c.CoreV1().ConfigMaps(tc.Namespace).Get(usingName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if !metav1.IsControlledBy(usingCm, tc) {
					e2elog.Logf("expect configmap of %s adopted by tidbcluster, still waiting...", setName)
					return false, nil
				}
			}
			return true, nil
		})

		framework.ExpectNoError(err)
	})

	ginkgo.It("TiDB cluster can be paused and unpaused", func() {
		tcName := "paused"
		tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBV3Version)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		err := genericCli.Create(context.TODO(), tc)
		framework.ExpectNoError(err)
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 15*time.Second)
		framework.ExpectNoError(err)

		podListBeforePaused, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Pause the tidb cluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Paused = true
			return nil
		})
		framework.ExpectNoError(err)
		ginkgo.By("Make a change")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Version = utilimage.TiDBV3UpgradeVersion
			return nil
		})
		framework.ExpectNoError(err)

		ginkgo.By("Check pods are not changed when the tidb cluster is paused")
		err = utilpod.WaitForPodsAreChanged(c, podListBeforePaused.Items, time.Minute*5)
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "Pods are changed when the tidb cluster is paused")

		ginkgo.By("Unpause the tidb cluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Paused = false
			return nil
		})
		framework.ExpectNoError(err)

		ginkgo.By("Check the tidb cluster will be upgraded now")
		listOptions := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(label.New().Instance(tcName).Component(label.TiKVLabelVal).Labels()).String(),
		}
		err = wait.PollImmediate(5*time.Second, 15*time.Minute, func() (bool, error) {
			podList, err := c.CoreV1().Pods(ns).List(listOptions)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			}
			for _, pod := range podList.Items {
				for _, c := range pod.Spec.Containers {
					if c.Name == v1alpha1.TiKVMemberType.String() {
						if c.Image == tc.TiKVImage() {
							return true, nil
						}
					}
				}
			}
			return false, nil
		})
		framework.ExpectNoError(err)
	})

})
