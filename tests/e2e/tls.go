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

package e2e

import (
	"fmt"
	"os/exec"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

func installCertManager(cli clientset.Interface) error {
	cmd := "kubectl apply -f /cert-manager.yaml --validate=false"
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install cert-manager %s %v", string(data), err)
	}

	err := e2epod.WaitForPodsRunningReady(cli, "cert-manager", 3, 0, 10*time.Minute, nil)
	if err != nil {
		return err
	}

	// It may take a minute or so for the TLS assets required for the webhook to function to be provisioned.
	time.Sleep(time.Minute)
	return nil
}

func deleteCertManager(cli clientset.Interface) error {
	cmd := "kubectl delete -f /cert-manager.yaml --ignore-not-found"
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete cert-manager %s %v", string(data), err)
	}

	return wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		podList, err := cli.CoreV1().Pods("cert-manager").List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		for _, pod := range podList.Items {
			err := e2epod.WaitForPodNotFoundInNamespace(cli, pod.Name, "cert-manager", 5*time.Minute)
			if err != nil {
				framework.Logf("failed to wait for pod cert-manager/%s disappear", pod.Name)
				return false, nil
			}
		}

		return true, nil
	})
}
