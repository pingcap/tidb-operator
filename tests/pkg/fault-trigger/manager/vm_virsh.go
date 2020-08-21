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
	"strings"

	"k8s.io/klog"
)

type VirshVMManager struct {
}

func (m *VirshVMManager) Name() string {
	return "virsh"
}

// ListVMs lists vms
func (m *VirshVMManager) ListVMs() ([]*VM, error) {
	shell := fmt.Sprintf("virsh list --all")
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return nil, err
	}
	vms := m.parserVMs(string(output))
	return vms, nil
}

// StopVM stops vm
func (m *VirshVMManager) StopVM(v *VM) error {
	shell := fmt.Sprintf("virsh destroy %s", v.Name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	klog.Infof("virtual machine %s is stopped", v.Name)

	return nil
}

// StartVM starts vm
func (m *VirshVMManager) StartVM(v *VM) error {
	shell := fmt.Sprintf("virsh start %s", v.Name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	klog.Infof("virtual machine %s is started", v.Name)

	return nil
}

// example input:
//  Id    Name                           State
// ----------------------------------------------------
// 6     vm2                            running
// 11    vm3                            running
// 12    vm1                            running
// -     vm-template                    shut off
func (m *VirshVMManager) parserVMs(data string) []*VM {
	data = stripEmpty(data)
	lines := strings.Split(data, "\n")
	var vms []*VM
	for _, line := range lines {
		fields := strings.Split(line, " ")
		if len(fields) < 3 {
			continue
		}
		if !strings.HasPrefix(fields[1], "vm") {
			continue
		}

		if strings.HasPrefix(fields[1], "vm-template") {
			continue
		}
		vm := &VM{
			Name:   fields[1],
			Status: fields[2],
		}
		vms = append(vms, vm)
	}
	return vms
}
