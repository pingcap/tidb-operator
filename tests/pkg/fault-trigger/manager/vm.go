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
	"sync"

	"github.com/golang/glog"
)

type VMManager interface {
	Name() string
	ListVMs() ([]*VM, error)
	StopVM(*VM) error
	StartVM(*VM) error
}

type VirshVMManager struct {
	*sync.RWMutex
	vmCache map[string]string
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
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
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
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	glog.Infof("virtual machine %s is stopped", v.Name)

	return nil
}

// StartVM starts vm
func (m *VirshVMManager) StartVM(v *VM) error {
	shell := fmt.Sprintf("virsh start %s", v.Name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	glog.Infof("virtual machine %s is started", v.Name)

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

func stripEmpty(data string) string {
	stripLines := []string{}
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		stripFields := []string{}
		fields := strings.Split(line, " ")
		for _, field := range fields {
			if len(field) > 0 {
				stripFields = append(stripFields, field)
			}
		}
		stripLine := strings.Join(stripFields, " ")
		stripLines = append(stripLines, stripLine)
	}
	return strings.Join(stripLines, "\n")
}

type QMVMManager struct {
	*sync.RWMutex
	subNet  string
	vmCache map[string]string
}

func (qm *QMVMManager) Name() string {
	return "qm"
}

func (qm *QMVMManager) ListVMs() ([]*VM, error) {
	shell := fmt.Sprintf("qm list")
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return nil, err
	}
	vms := qm.parserVMs(string(output))
	return vms, nil
}

func (qm *QMVMManager) StartVM(vm *VM) error {
	shell := fmt.Sprintf("qm start %s", vm.Name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	glog.Infof("virtual machine %s is started", vm.Name)

	return nil
}

func (qm *QMVMManager) StopVM(vm *VM) error {
	shell := fmt.Sprintf("qm stop %s", vm.Name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return err
	}

	glog.Infof("virtual machine %s is stopped", vm.Name)
	return nil
}

// example input:
// VMID NAME                 STATUS     MEM(MB)    BOOTDISK(GB) PID
// 101 CentOS7600           stopped    1024              32.00 0
// 104 to20190915-tongmu    running    8192             500.00 34863``
func (qm *QMVMManager) parserVMs(data string) []*VM {
	vms := []*VM{}
	data = stripEmpty(data)
	lines := strings.Split(data, "\n")
	var startIndex int
	for i, line := range lines {
		if strings.Contains(line, "VMID") {
			startIndex = i + 1
			break
		}
	}
	vmLines := lines[startIndex:]
	for _, line := range vmLines {
		fields := strings.Split(line, " ")
		if len(fields) >= 6 {
			vm := &VM{
				Name:   fields[0],
				Status: fields[2],
			}
			vms = append(vms, vm)
		}
	}
	return vms
}
