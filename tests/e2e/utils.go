// Copyright 2018 PingCAP, Inc.
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
// limitations under the License.package spec

package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo" // revive:disable:dot-imports
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	ns               = "tidb-cluster-e2e"
	helmName         = "tidb-cluster-e2e"
	operatorNs       = "tidb-operator-e2e"
	operatorHelmName = "tidb-operator-e2e"
	clusterName      = "demo-cluster"
	testTableName    = "demo_table"
	testTableVal     = "demo_val"
)

var (
	cli     versioned.Interface
	kubeCli kubernetes.Interface
)

func clearOperator() error {
	_, err := execCmd(fmt.Sprintf("helm del --purge %s", helmName))
	if err != nil && isNotFound(err) {
		return err
	}

	_, err = execCmd(fmt.Sprintf("kubectl delete pvc -n %s --all", ns))
	if err != nil {
		return err
	}

	_, err = execCmd(fmt.Sprintf("helm del --purge %s", operatorHelmName))
	if err != nil && isNotFound(err) {
		return err
	}

	err = wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		result, err := execCmd(fmt.Sprintf("kubectl get po --output=name -n %s", ns))
		if err != nil || result != "" {
			return false, nil
		}
		_, err = execCmd(fmt.Sprintf(`kubectl get pv -l %s=%s,%s=%s --output=name | xargs -I {} \
		kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'`,
			label.NamespaceLabelKey, ns, label.InstanceLabelKey, clusterName))
		if err != nil {
			logf(err.Error())
		}
		result, _ = execCmd(fmt.Sprintf("kubectl get pv -l %s=%s,%s=%s 2>/dev/null|grep Released", label.NamespaceLabelKey, ns, label.InstanceLabelKey, clusterName))
		if result != "" {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func installOperator() error {
	_, err := execCmd(fmt.Sprintf(
		"helm install /charts/tidb-operator -f /tidb-operator-values.yaml -n %s --namespace=%s",
		operatorHelmName,
		operatorNs))
	if err != nil {
		return err
	}

	By("When create a TiDB cluster")
	_, err = execCmd(fmt.Sprintf(
		"helm install /charts/tidb-cluster -f /tidb-cluster-values.yaml -n %s --namespace=%s",
		helmName,
		ns))
	if err != nil {
		return err
	}

	monitorRestartCount()

	return nil
}

func monitorRestartCount() {
	maxRestartCount := int32(3)

	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-time.After(5 * time.Second):
				labelSelector := label.New().Cluster(clusterName)
				podList, err := kubeCli.CoreV1().Pods(ns).List(
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(
							labelSelector.Labels(),
						).String(),
					},
				)
				if err != nil {
					continue
				}

				for _, pod := range podList.Items {
					for _, cs := range pod.Status.ContainerStatuses {
						if cs.RestartCount > maxRestartCount {
							Fail(fmt.Sprintf("POD: %s/%s's container: %s's restartCount is greater than: %d",
								pod.GetNamespace(), pod.GetName(), cs.Name, maxRestartCount))
							return
						}
					}
				}
			}
		}
	}()
}

func execCmd(cmdStr string) (string, error) {
	logf(fmt.Sprintf("$ %s\n", cmdStr))
	result, err := exec.Command("/bin/sh", "-c", cmdStr).CombinedOutput()
	resultStr := string(result)
	logf(resultStr)
	if err != nil {
		logf(err.Error())
		return resultStr, err
	}

	return resultStr, nil
}

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func log(level string, format string, args ...interface{}) {
	fmt.Fprintf(GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

// logf log a message in INFO format
func logf(format string, args ...interface{}) {
	log("INFO", format, args...)
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}
