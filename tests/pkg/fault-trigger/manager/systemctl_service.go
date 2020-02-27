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
	"os/exec"

	"k8s.io/klog"
)

const (
	ETCDService    = "etcd"
	KubeletService = "kubelet"
)

// StartETCD starts the etcd service
func (m *Manager) StartETCD() error {
	return m.systemctlStartService(ETCDService)
}

// StopETCD stops the etcd service
func (m *Manager) StopETCD() error {
	return m.systemctlStopService(ETCDService)
}

// StartKubelet starts the kubelet service
func (m *Manager) StartKubelet() error {
	return m.systemctlStartService(KubeletService)
}

// StopKubelet stops the kubelet service
func (m *Manager) StopKubelet() error {
	return m.systemctlStopService(KubeletService)
}

func (m *Manager) systemctlStartService(serviceName string) error {
	shell := fmt.Sprintf("systemctl start %s", serviceName)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	klog.Infof("%s is started", serviceName)

	return nil
}

func (m *Manager) systemctlStopService(serviceName string) error {
	shell := fmt.Sprintf("systemctl stop %s", serviceName)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	klog.Infof("%s is stopped", serviceName)

	return nil
}
