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

package manager

import (
	"fmt"
	"os"
	"os/exec"

	"k8s.io/klog"
)

const (
	staticPodPath                 = "/etc/kubernetes/manifests"
	staticPodTmpPath              = "/etc/kubernetes/tmp"
	kubeAPIServerManifes          = "kube-apiserver.yaml"
	kubeControllerManagerManifest = "kube-controller-manager.yaml"
	kubeSchedulerManifest         = "kube-scheduler.yaml"

	KubeAPIServerService         = "kube-apiserver"
	KubeSchedulerService         = "kube-scheduler"
	KubeControllerManagerService = "kube-controller-manager"
)

// StartKubeScheduler starts the kube-scheduler service
func (m *Manager) StartKubeScheduler() error {
	return m.startStaticPodService(KubeSchedulerService, kubeSchedulerManifest)
}

// StopKubeScheduler stops the kube-scheduler service
func (m *Manager) StopKubeScheduler() error {
	return m.stopStaticPodService(KubeSchedulerService, kubeSchedulerManifest)
}

// StartKubeAPIServer starts the apiserver
func (m *Manager) StartKubeAPIServer() error {
	return m.startStaticPodService(KubeAPIServerService, kubeAPIServerManifes)
}

// StopKubeAPIServer stops the apiserver
func (m *Manager) StopKubeAPIServer() error {
	return m.stopStaticPodService(KubeAPIServerService, kubeAPIServerManifes)
}

// StartKubeControllerManager starts the kube-controller-manager service
func (m *Manager) StartKubeControllerManager() error {
	return m.startStaticPodService(KubeControllerManagerService, kubeControllerManagerManifest)
}

// StopKubeControllerManager stops the kube-controller-manager service
func (m *Manager) StopKubeControllerManager() error {
	return m.stopStaticPodService(KubeControllerManagerService, kubeControllerManagerManifest)
}

func (m *Manager) stopStaticPodService(serviceName string, fileName string) error {
	manifest := fmt.Sprintf("%s/%s", staticPodPath, fileName)
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		klog.Infof("%s had been stopped before", serviceName)
		return nil
	}
	shell := fmt.Sprintf("mkdir -p %s && mv %s %s", staticPodTmpPath, manifest, staticPodTmpPath)

	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	klog.Infof("%s is stopped", serviceName)

	return nil
}

func (m *Manager) startStaticPodService(serviceName string, fileName string) error {
	manifest := fmt.Sprintf("%s/%s", staticPodTmpPath, fileName)
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		klog.Infof("%s had been started before", serviceName)
		return nil
	}
	shell := fmt.Sprintf("mv %s %s", manifest, staticPodPath)

	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	klog.Infof("%s is started", serviceName)

	return nil
}
