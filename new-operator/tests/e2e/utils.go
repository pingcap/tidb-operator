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

	"time"

	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/pingcap/tidb-operator/new-operator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	ns               = "e2e"
	helmName         = "tidb-cluster"
	operatorNs       = "pingcap"
	operatorHelmName = "tidb-operator"
	clusterName      = "demo-cluster"
)

var (
	cli     versioned.Interface
	kubeCli kubernetes.Interface
)

func clearOperator() error {
	_, err := execCmd(`kubectl get pv --output=name | xargs -I {} \
		kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'`)
	if err != nil {
		return err
	}

	_, err = execCmd(fmt.Sprintf("helm del --purge %s", helmName))
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
		result, err = execCmd("kubectl get pv --output=name")
		if err != nil || result != "" {
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

	_, err = execCmd(fmt.Sprintf(
		"helm install /charts/tidb-cluster -f /tidb-cluster-values.yaml -n %s --namespace=%s",
		helmName,
		ns))
	if err != nil {
		return err
	}

	return nil
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
