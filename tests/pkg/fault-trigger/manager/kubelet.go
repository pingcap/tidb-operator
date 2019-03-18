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
	"os/exec"

	"github.com/golang/glog"
)

// StartKubelet starts the kubelet service
func (m *Manager) StartKubelet() error {
	shell := "systemctl start kubelet"
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}
	return nil
}

// StopKubelet stops the kubelet service
func (m *Manager) StopKubelet() error {
	shell := "systemctl stop kubelet"
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}
	return nil
}
