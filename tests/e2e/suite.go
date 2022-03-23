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

package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2emetrics "k8s.io/kubernetes/test/e2e/framework/metrics"
)

// CleanupSuite is the boilerplate that can be used after tests on ginkgo were run, on the SynchronizedAfterSuite step.
// Similar to SynchronizedBeforeSuite, we want to run some operations only once (such as collecting cluster logs).
// Here, the order of functions is reversed; first, the function which runs everywhere,
// and then the function that only runs on the first Ginkgo node.
func CleanupSuite() {
	// Run on all Ginkgo nodes
	framework.Logf("Running AfterSuite actions on all nodes")
	framework.RunCleanupActions()
}

// AfterSuiteActions are actions that are run on ginkgo's SynchronizedAfterSuite
func AfterSuiteActions() {
	// Run only Ginkgo on node 1
	framework.Logf("Running AfterSuite actions on node 1")
	if framework.TestContext.ReportDir != "" {
		framework.CoreDump(framework.TestContext.ReportDir)
	}
	if framework.TestContext.GatherSuiteMetricsAfterTest {
		if err := gatherTestSuiteMetrics(); err != nil {
			framework.Logf("Error gathering metrics: %v", err)
		}
	}
	if framework.TestContext.NodeKiller.Enabled {
		close(framework.TestContext.NodeKiller.NodeKillerStopCh)
	}
}

func gatherTestSuiteMetrics() error {
	framework.Logf("Gathering metrics")
	c, err := framework.LoadClientset()
	if err != nil {
		return fmt.Errorf("error loading client: %v", err)
	}

	// Grab metrics for apiserver, scheduler, controller-manager, kubelet (for non-kubemark case) and cluster autoscaler (optionally).
	grabber, err := e2emetrics.NewMetricsGrabber(c, nil, !framework.ProviderIs("kubemark"), true, true, true, framework.TestContext.IncludeClusterAutoscalerMetrics)
	if err != nil {
		return fmt.Errorf("failed to create MetricsGrabber: %v", err)
	}

	received, err := grabber.Grab()
	if err != nil {
		return fmt.Errorf("failed to grab metrics: %v", err)
	}

	metricsForE2E := (*e2emetrics.ComponentCollection)(&received)
	metricsJSON := metricsForE2E.PrintJSON()
	if framework.TestContext.ReportDir != "" {
		filePath := path.Join(framework.TestContext.ReportDir, "MetricsForE2ESuite_"+time.Now().Format(time.RFC3339)+".json")
		if err := ioutil.WriteFile(filePath, []byte(metricsJSON), 0644); err != nil {
			return fmt.Errorf("error writing to %q: %v", filePath, err)
		}
	} else {
		framework.Logf("\n\nTest Suite Metrics:\n%s\n", metricsJSON)
	}

	return nil
}

func dumpPodsForOperator(kubecli kubernetes.Interface, reportDir string, ns string) error {
	dumpDir := filepath.Join(framework.TestContext.ReportDir, "logs", "tidb-operator")
	// full permission (0777) for the log directory to avoid "permission denied" for later kubetest2 log dump.
	err := os.MkdirAll(dumpDir, 0777)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dumpDir, err)
	}

	podList, err := kubecli.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list pods in ns %s failed: %v", ns, err)
	}

	for _, pod := range podList.Items {
		err = tests.DumpPod(dumpDir, &pod)
		if err != nil {
			return fmt.Errorf("failed to dump log for pod %s/%s: %s", pod.Namespace, pod.Name, err)
		}
	}

	return nil
}

func dumpResoures(cli versioned.Interface, kubecli kubernetes.Interface, reportDir string) error {
	dumpDir := filepath.Join(framework.TestContext.ReportDir, "resources", "tidb-operator")
	err := os.MkdirAll(dumpDir, 0777)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dumpDir, err)
	}

	// list resources
	objs := make(map[string]metav1.Object)
	list, err := kubecli.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(list.Items) == 0 {
		return nil
	}
	for _, namespace := range list.Items {
		ns := namespace.Name

		tclist, err := cli.PingcapV1alpha1().TidbClusters(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("list tidbclusters failed: %v", err)
		}
		for i := range tclist.Items {
			objs["tidbcluster"] = &tclist.Items[i]
		}

		backupList, err := cli.PingcapV1alpha1().Backups(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("list backups failed: %v", err)
		}
		for i := range backupList.Items {
			objs["backup"] = &backupList.Items[i]
		}

		restoreList, err := cli.PingcapV1alpha1().Restores(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("list restores failed: %v", err)
		}
		for i := range restoreList.Items {
			objs["restore"] = &restoreList.Items[i]
		}
	}

	// dump resources
	for typ, obj := range objs {
		if err := tests.DumpResource(dumpDir, typ, obj); err != nil {
			return fmt.Errorf("failed to dump %s %s/%s: %v", typ, obj.GetNamespace(), obj.GetName(), err)
		}
	}

	return nil
}
