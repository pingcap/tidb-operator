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

	"github.com/golang/glog"
	"github.com/juju/errors"
)

// ListVMs lists vms
func (m *Manager) ListVMs() ([]*VM, error) {
	shell := fmt.Sprintf("virsh list --all")
	cmd := exec.Command("/bin/sh", "-c", shell)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	vms := parserVMs(string(stdoutStderr))
	//"virsh domifaddr vm1 --interface eth0 --source agent"
	for _, vm := range vms {
		vmIP, err := getVMIP(vm.Name)
		if err != nil {
			glog.Errorf("can not get vm %s ip", vm.Name)
			continue
		}
		vm.IP = vmIP
	}
	return vms, nil
}

// StopVM stops vm
func (m *Manager) StopVM(v *VM) (string, error) {
	shell := fmt.Sprintf("virsh destroy %s", v.Name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

// StartVM starts vm
func (m *Manager) StartVM(v *VM) (string, error) {
	shell := fmt.Sprintf("virsh start %s", v.Name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

func getVMIP(name string) (string, error) {
	shell := fmt.Sprintf("virsh domifaddr %s --interface eth0 --source agent", name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return parserIP(string(stdoutStderr))
}

func parserIP(data string) (string, error) {
	data = stripEmpty(data)
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "eth0") {
			continue
		}

		fields := strings.Split(line, " ")
		if len(fields) < 4 {
			continue
		}

		ip := strings.Split(fields[3], "/")[0]
		return ip, nil
	}

	return "", errors.NotFoundf("address")
}

func parserVMs(data string) []*VM {
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
			Name: fields[1],
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
