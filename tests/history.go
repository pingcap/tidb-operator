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

package tests

import (
	"fmt"
	"os"
	"os/exec"

	"k8s.io/klog"
)

func DeployReleasedCRDOrDie(version string) {
	cmd := fmt.Sprintf(`kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/%s/manifests/crd.yaml`, version)
	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		klog.Fatalf(fmt.Sprintf("failed to deploy crd: %v, %s", err, string(res)))
	}
}

func CleanReleasedCRDOrDie(version string) {
	cmd := fmt.Sprintf(`kubectl delete -f https://raw.githubusercontent.com/pingcap/tidb-operator/%s/manifests/crd.yaml`, version)
	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		klog.Fatalf(fmt.Sprintf("failed to clean crd: %v, %s", err, string(res)))
	}
}

func DownloadReleasedOperatorChartOrDie(version string) {
	err := mkReleasedChartDir(version)
	if err != nil {
		klog.Fatalf("failed to create folder /chart/%s, err:%v", version, err)
	}
	cmd := fmt.Sprintf(`helm fetch pingcap/tidb-operator --version=%s --untar=true --untardir=/chart/%s`, version, version)
	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to download operator chart: %v, %s", err, string(res))
	}
}

func DownloadReleasedTidbClusterChartOrDie(version string) {
	err := mkReleasedChartDir(version)
	if err != nil {
		klog.Fatalf("failed to create folder /chart/%s, err:%v", version, err)
	}
	cmd := fmt.Sprintf(`helm fetch pingcap/tidb-cluster --version=%s --untar=true --untardir=/chart/%s`, version, version)
	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to download tidbcluster chart: %v, %s", err, string(res))
	}
}

func mkReleasedChartDir(version string) error {
	folderPath := fmt.Sprintf("/chart/%s", version)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		err = os.Mkdir(folderPath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
