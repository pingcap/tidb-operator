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

// Modified from https://github.com/kubernetes/kubernetes/blob/v1.16.0/test/e2e/e2e.go

package e2e

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	"github.com/pingcap/tidb-operator/tests/e2e/tidbcluster"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilnode "github.com/pingcap/tidb-operator/tests/e2e/util/node"
	utiloperator "github.com/pingcap/tidb-operator/tests/e2e/util/operator"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	runtimeutils "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/component-base/logs"
	aggregatorclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	storageutil "k8s.io/kubernetes/pkg/apis/storage/v1/util"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/kubernetes/test/e2e/framework/pod"

	// ensure auth plugins are loaded
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	// ensure that cloud providers are loaded
	_ "k8s.io/kubernetes/test/e2e/framework/providers/aws"
	_ "k8s.io/kubernetes/test/e2e/framework/providers/gce"
)

var (
	operatorKillerStopCh chan struct{}
)

// This is modified from framework.SetupSuite().
// setupSuite is the boilerplate that can be used to setup ginkgo test suites, on the SynchronizedBeforeSuite step.
// There are certain operations we only want to run once per overall test invocation
// (such as deleting old namespaces, or verifying that all system pods are running.
// Because of the way Ginkgo runs tests in parallel, we must use SynchronizedBeforeSuite
// to ensure that these operations only run on the first parallel Ginkgo node.
func setupSuite(c kubernetes.Interface, extClient versioned.Interface, apiExtClient apiextensionsclientset.Interface) {
	// Run only on Ginkgo node 1

	// Delete any namespaces except those created by the system. This ensures no
	// lingering resources are left over from a previous test run.
	if framework.TestContext.CleanStart {
		reservedNamespaces := []string{
			metav1.NamespaceSystem,
			metav1.NamespaceDefault,
			metav1.NamespacePublic,
			v1.NamespaceNodeLease,
		}
		if framework.TestContext.Provider == "kind" {
			// kind local path provisioner namespace since 0.7.0
			// https://github.com/kubernetes-sigs/kind/blob/v0.7.0/pkg/build/node/storage.go#L35
			reservedNamespaces = append(reservedNamespaces, "local-path-storage")
		} else if framework.TestContext.Provider == "openshift" {
			reservedNamespaces = append(reservedNamespaces, "openshift")
		}
		deleted, err := framework.DeleteNamespaces(c, nil, reservedNamespaces)
		if err != nil {
			log.Failf("Error deleting orphaned namespaces: %v", err)
		}

		// try to clean up backups
		if err := ForceCleanBackups(c, extClient, apiExtClient); err != nil {
			log.Failf("Error clean backups: %v", err)
		}

		log.Logf("Waiting for deletion of the following namespaces: %v", deleted)
		if err := framework.WaitForNamespacesDeleted(c, deleted, framework.NamespaceCleanupTimeout); err != nil {
			log.Failf("Failed to delete orphaned namespaces %v: %v", deleted, err)
		}
	}

	// In large clusters we may get to this point but still have a bunch
	// of nodes without Routes created. Since this would make a node
	// unschedulable, we need to wait until all of them are schedulable.
	framework.ExpectNoError(framework.WaitForAllNodesSchedulable(c, framework.TestContext.NodeSchedulableTimeout), "some nodes are not schedulable")

	// If NumNodes is not specified then auto-detect how many are scheduleable and not tainted
	if framework.TestContext.CloudConfig.NumNodes == framework.DefaultNumNodes {
		framework.TestContext.CloudConfig.NumNodes = len(framework.GetReadySchedulableNodesOrDie(c).Items)
	}

	// Ensure all pods are running and ready before starting tests (otherwise,
	// cluster infrastructure pods that are being pulled or started can block
	// test pods from running, and tests that ensure all pods are running and
	// ready will fail).
	podStartupTimeout := framework.TestContext.SystemPodsStartupTimeout
	// TODO: In large clusters, we often observe a non-starting pods due to
	// #41007. To avoid those pods preventing the whole test runs (and just
	// wasting the whole run), we allow for some not-ready pods (with the
	// number equal to the number of allowed not-ready nodes).
	if err := pod.WaitForPodsRunningReady(c, metav1.NamespaceSystem, int32(framework.TestContext.MinStartupPods), int32(framework.TestContext.AllowedNotReadyNodes), podStartupTimeout, map[string]string{}); err != nil {
		framework.DumpAllNamespaceInfo(c, metav1.NamespaceSystem)
		framework.LogFailedContainers(c, metav1.NamespaceSystem, log.Logf)
		log.Failf("Error waiting for all pods to be running and ready: %v", err)
	}

	if err := framework.WaitForDaemonSets(c, metav1.NamespaceSystem, int32(framework.TestContext.AllowedNotReadyNodes), framework.TestContext.SystemDaemonsetStartupTimeout); err != nil {
		log.Logf("WARNING: Waiting for all daemonsets to be ready failed: %v", err)
	}

	ginkgo.By("Initializing all nodes")
	nodeList, err := c.CoreV1().Nodes().List(metav1.ListOptions{})
	framework.ExpectNoError(err, "failed to list nodes")
	for _, node := range nodeList.Items {
		framework.Logf("Initializing node %q", node.Name)
		framework.ExpectNoError(utilnode.InitNode(&node), fmt.Sprintf("initializing node %s failed", node.Name))
	}

	// By using default storage class in GKE/EKS (aws), network attached storage
	// which be used and we must clean them later.
	// We set local-storage class as default for simplicity.
	// The default storage class of kind is local-path-provisioner which
	// consumes local storage like local-volume-provisioner. However, it's not
	// stable in our e2e testing.
	if framework.TestContext.Provider == "gke" || framework.TestContext.Provider == "aws" {
		defaultSCName := "local-storage"
		list, err := c.StorageV1().StorageClasses().List(metav1.ListOptions{})
		framework.ExpectNoError(err, "list storage class failed")
		// only one storage class can be marked default
		// https://kubernetes.io/docs/tasks/administer-cluster/change-default-storage-class/#changing-the-default-storageclass
		var localStorageSC *storagev1.StorageClass
		for i, sc := range list.Items {
			if sc.Name == defaultSCName {
				localStorageSC = &list.Items[i]
			} else if storageutil.IsDefaultAnnotation(sc.ObjectMeta) {
				delete(sc.ObjectMeta.Annotations, storageutil.IsDefaultStorageClassAnnotation)
				_, err = c.StorageV1().StorageClasses().Update(&sc)
				framework.ExpectNoError(err, "update storage class failed, %v", sc)
			}
		}
		if localStorageSC == nil {
			log.Fail("local-storage storage class not found")
		}
		if localStorageSC.Annotations == nil {
			localStorageSC.Annotations = map[string]string{}
		}
		localStorageSC.Annotations[storageutil.IsDefaultStorageClassAnnotation] = "true"
		log.Logf("Setting %q as the default storage class", localStorageSC.Name)
		_, err = c.StorageV1().StorageClasses().Update(localStorageSC)
		framework.ExpectNoError(err, "update storage class failed, %v", localStorageSC)
	}

	// Log the version of the server and this client.
	log.Logf("e2e test version: %s", version.Get().GitVersion)

	dc := c.Discovery()

	serverVersion, serverErr := dc.ServerVersion()
	if serverErr != nil {
		log.Logf("Unexpected server error retrieving version: %v", serverErr)
	}
	if serverVersion != nil {
		log.Logf("kube-apiserver version: %s", serverVersion.GitVersion)
	}
}

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	cleaners := []struct {
		text string
		cmd  string
	}{
		{
			text: "Clear all helm releases",
			cmd:  "helm ls --all --short | xargs -n 1 -r helm uninstall",
		},
		{
			text: "Clear tidb-operator apiservices",
			cmd:  "kubectl delete apiservices -l app.kubernetes.io/name=tidb-operator",
		},
		{
			text: "Clear tidb-operator validatingwebhookconfigurations",
			cmd:  "kubectl delete validatingwebhookconfiguration -l app.kubernetes.io/name=tidb-operator",
		},
		{
			text: "Clear tidb-operator mutatingwebhookconfigurations",
			cmd:  "kubectl delete mutatingwebhookconfiguration -l app.kubernetes.io/name=tidb-operator",
		},
	}
	for _, p := range cleaners {
		ginkgo.By(p.text)
		cmd := exec.Command("sh", "-c", p.cmd)
		output, err := cmd.CombinedOutput()
		if err != nil {
			framework.Failf("failed to %s (cmd: %q, error: %v, output: %s", p.text, p.cmd, err, string(output))
		}
	}

	// Get clients
	config, err := framework.LoadConfig()
	framework.ExpectNoError(err, "failed to load config")
	config.QPS = 20
	config.Burst = 50
	cli, err := versioned.NewForConfig(config)
	framework.ExpectNoError(err, "failed to create clientset for pingcap")
	kubeCli, err := kubernetes.NewForConfig(config)
	framework.ExpectNoError(err, "failed to create clientset for Kubernetes")
	aggrCli, err := aggregatorclientset.NewForConfig(config)
	framework.ExpectNoError(err, "failed to create clientset for kube-aggregator")
	apiExtCli, err := apiextensionsclientset.NewForConfig(config)
	framework.ExpectNoError(err, "failed to create clientset for apiextensions-apiserver")
	asCli, err := asclientset.NewForConfig(config)
	framework.ExpectNoError(err, "failed to create clientset for advanced-statefulset")
	clientRawConfig, err := e2econfig.LoadClientRawConfig()
	framework.ExpectNoError(err, "failed to load raw config for tidb operator")
	fw, err := portforward.NewPortForwarder(context.Background(), e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
	framework.ExpectNoError(err, "failed to create port forwarder")

	setupSuite(kubeCli, cli, apiExtCli)
	// override with hard-coded value
	e2econfig.TestConfig.ManifestDir = "/manifests"
	framework.Logf("====== e2e configuration ======")
	framework.Logf("%s", e2econfig.TestConfig.MustPrettyPrintJSON())
	// preload images
	if e2econfig.TestConfig.PreloadImages {
		ginkgo.By("Preloading images")
		if err := utilimage.PreloadImages(); err != nil {
			framework.Failf("failed to pre-load images: %v", err)
		}
	}

	ginkgo.By("Recycle all local PVs")
	pvList, err := kubeCli.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	framework.ExpectNoError(err, "failed to list persistent volumes")
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName != "local-storage" {
			continue
		}
		if pv.Spec.PersistentVolumeReclaimPolicy == v1.PersistentVolumeReclaimDelete {
			continue
		}
		log.Logf("Update reclaim policy of PV %s to %s", pv.Name, v1.PersistentVolumeReclaimDelete)
		pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimDelete
		_, err = kubeCli.CoreV1().PersistentVolumes().Update(&pv)
		framework.ExpectNoError(err, "failed to update pv %s", pv.Name)
	}

	ginkgo.By("Wait for all local PVs to be available")
	err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
		pvList, err := kubeCli.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, pv := range pvList.Items {
			if pv.Spec.StorageClassName != "local-storage" {
				continue
			}
			if pv.Status.Phase != v1.VolumeAvailable {
				return false, nil
			}
		}
		return true, nil
	})
	framework.ExpectNoError(err, "failed to wait for all PVs to be available")

	ginkgo.By("Labeling nodes")
	oa := tests.NewOperatorActions(cli, kubeCli, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, nil, e2econfig.TestConfig, nil, fw, nil)
	oa.LabelNodesOrDie()
	if e2econfig.TestConfig.InstallOperator {
		OperatorFeatures := map[string]bool{"AutoScaling": true}
		e2econfig.TestConfig.OperatorFeatures = OperatorFeatures
		ocfg := e2econfig.NewDefaultOperatorConfig(e2econfig.TestConfig)
		ginkgo.By("Installing CRDs")
		oa.CleanCRDOrDie()
		oa.InstallCRDOrDie(ocfg)
		ginkgo.By("Installing tidb-operator")
		oa.CleanOperatorOrDie(ocfg)
		oa.DeployOperatorOrDie(ocfg)
		if e2econfig.TestConfig.OperatorKiller.Enabled {
			operatorKiller := utiloperator.NewOperatorKiller(e2econfig.TestConfig.OperatorKiller, kubeCli, func() ([]v1.Pod, error) {
				podList, err := kubeCli.CoreV1().Pods(ocfg.Namespace).List(metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{
						"app.kubernetes.io/name": "tidb-operator",
					}).String(),
				})
				if err != nil {
					return nil, err
				}
				return podList.Items, nil
			})
			operatorKillerStopCh := make(chan struct{})
			go operatorKiller.Run(operatorKillerStopCh)
		}

		// only deploy MySQL and TiDB for DM if CRDs and TiDB Operator installed.
		// setup upstream MySQL instances and the downstream TiDB cluster for DM testing.
		// if we can only setup these resource for DM tests with something like `--focus` or `--skip`, that should be better.
		oa.DeployDMMySQLOrDie(tests.DMMySQLNamespace)
		oa.DeployDMTiDBOrDie()
	} else {
		ginkgo.By("Skip installing tidb-operator")
		ginkgo.By("Skip installing MySQL and TiDB for DM tests")
	}

	ginkgo.By("Installing cert-manager")
	err = tidbcluster.InstallCertManager(kubeCli)
	framework.ExpectNoError(err, "failed to install cert-manager")
	return nil
}, func(data []byte) {
	// Run on all Ginkgo nodes
	framework.SetupSuitePerGinkgoNode()
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	framework.CleanupSuite()

}, func() {
	framework.AfterSuiteActions()
	if operatorKillerStopCh != nil {
		close(operatorKillerStopCh)
	}
	config, _ := framework.LoadConfig()
	config.QPS = 20
	config.Burst = 50
	cli, _ := versioned.NewForConfig(config)
	kubeCli, _ := kubernetes.NewForConfig(config)
	if !ginkgo.CurrentGinkgoTestDescription().Failed {
		ginkgo.By("Clean labels")
		err := tests.CleanNodeLabels(kubeCli)
		framework.ExpectNoError(err, "failed to clean labels")
	}

	ginkgo.By("Deleting cert-manager")
	err := tidbcluster.DeleteCertManager(kubeCli)
	framework.ExpectNoError(err, "failed to delete cert-manager")

	err = tests.CleanDMMySQL(kubeCli, tests.DMMySQLNamespace)
	framework.ExpectNoError(err, "failed to clean DM MySQL")
	err = tests.CleanDMTiDB(cli, kubeCli)
	framework.ExpectNoError(err, "failed to clean DM TiDB")

	ginkgo.By("Uninstalling tidb-operator")
	ocfg := e2econfig.NewDefaultOperatorConfig(e2econfig.TestConfig)

	// kubetest2 can only dump running pods' log (copy from container log directory),
	// but if we want to get test coverage reports for tidb-operator, we need to shutdown the processes/pods),
	// so we choose to copy logs before uninstall tidb-operator.
	// NOTE: if we can get the whole test result from all parallel Ginkgo nodes with Ginkgo v2 later, we can also choose to:
	// - dump logs if the test failed.
	// - (uninstall tidb-operator and) generate test coverage reports if the test passed.
	// ref: https://github.com/onsi/ginkgo/issues/361#issuecomment-814203240

	if framework.TestContext.ReportDir != "" {
		ginkgo.By("Dumping logs for tidb-operator")
		logPath := filepath.Join(framework.TestContext.ReportDir, "logs", "tidb-operator")
		// full permission (0777) for the log directory to avoid "permission denied" for later kubetest2 log dump.
		framework.ExpectNoError(os.MkdirAll(logPath, 0777), "failed to create log directory for tidb-operator components")

		podList, err2 := kubeCli.CoreV1().Pods(ocfg.Namespace).List(metav1.ListOptions{})
		framework.ExpectNoError(err2, "failed to list pods for tidb-operator")
		for _, pod := range podList.Items {
			log.Logf("dumping logs for pod %s/%s", pod.Namespace, pod.Name)
			err2 = tests.DumpPod(logPath, &pod)
			framework.ExpectNoError(err2, "failed to dump log for pod %s/%s", pod.Namespace, pod.Name)
		}
	}

	ginkgo.By("Uninstalling tidb-operator")
	err = tests.CleanOperator(ocfg)
	framework.ExpectNoError(err, "failed to uninstall operator")

	ginkgo.By("Wait for tidb-operator to be uninstalled")
	err = wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		podList, err2 := kubeCli.CoreV1().Pods(ocfg.Namespace).List(metav1.ListOptions{})
		framework.ExpectNoError(err2, "failed to list pods for tidb-operator")
		return len(podList.Items) == 0, nil
	})
	framework.ExpectNoError(err, "failed to wait for tidb-operator to be uninstalled")
})

// TODO: refactor it to combine with code in /tests/e2e/br/framework/framework.go
// If namespace is deleted directly, all resource in this namespace will be deleted.
// However, backup finalizer is depend on some resource such as Secret in the namespace, so finalizer will always fail and block namespace deletion.
func ForceCleanBackups(kubeClient kubernetes.Interface, extClient versioned.Interface, apiExtClient apiextensionsclientset.Interface) error {
	if _, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().Get("backups.pingcap.com", metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return nil
	}
	nsList, err := kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, item := range nsList.Items {
		ns := item.Name
		bl, err := extClient.PingcapV1alpha1().Backups(ns).List(metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list backups in namespace %s: %v", ns, err)
		}
		for i := range bl.Items {
			name := bl.Items[i].Name
			if err := extClient.PingcapV1alpha1().Backups(ns).Delete(name, nil); err != nil {
				return fmt.Errorf("failed to delete backup(%s) in namespace %s: %v", name, ns, err)
			}
			// use patch to avoid update conflicts
			patch := []byte(`[{"op":"remove","path":"/metadata/finalizers"}]`)
			if _, err := extClient.PingcapV1alpha1().Backups(ns).Patch(name, types.JSONPatchType, patch); err != nil {
				return fmt.Errorf("failed to clean backup(%s) finalizers in namespace %s: %v", name, ns, err)
			}
		}
	}
	return err
}

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func RunE2ETests(t *testing.T) {
	runtimeutils.ReallyCrash = true
	logs.InitLogs()
	defer logs.FlushLogs()

	gomega.RegisterFailHandler(log.Fail)

	// Disable serial and stability tests by default unless they are explicitly requested.
	if config.GinkgoConfig.FocusString == "" && config.GinkgoConfig.SkipString == "" {
		config.GinkgoConfig.SkipString = `\[Stability\]|\[Serial\]`
	}

	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []ginkgo.Reporter
	if framework.TestContext.ReportDir != "" {
		// TODO: we should probably only be trying to create this directory once
		// rather than once-per-Ginkgo-node.
		if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
			log.Logf("ERROR: Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v%02d.xml", framework.TestContext.ReportPrefix, config.GinkgoConfig.ParallelNode))))
		}
	}
	log.Logf("Starting e2e run %q on Ginkgo node %d", framework.RunID, config.GinkgoConfig.ParallelNode)

	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "tidb-operator e2e suite", r)
}
