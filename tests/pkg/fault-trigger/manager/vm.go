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
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

// ListVMs lists vms
func (m *Manager) ListVMs() ([]*VM, error) {
	shell := fmt.Sprintf("virsh list --all")
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		return nil, err
	}
	vms := parserVMs(string(output))
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
func (m *Manager) StopVM(v *VM) error {
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
func (m *Manager) StartVM(v *VM) error {
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

func getVMIP(name string) (string, error) {
	shell := fmt.Sprintf("virsh domifaddr %s --interface eth0 --source agent", name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Warningf("exec: [%s] failed, output: %s, error: %v", shell, string(output), err)
		mac, err := getVMMac(name)
		if err != nil {
			return "", err
		}

		ipNeighShell := fmt.Sprintf("ip neigh | grep -i %s", mac)
		cmd = exec.Command("/bin/sh", "-c", ipNeighShell)
		ipNeighOutput, err := cmd.CombinedOutput()
		if err != nil {
			glog.Errorf("exec: [%s] failed, output: %s, error: %v", ipNeighShell, string(ipNeighOutput), err)
			return "", err
		}

		return parserIPFromIPNeign(string(ipNeighOutput))
	}

	return parserIP(string(output))
}

func getVMMac(name string) (string, error) {
	shell := fmt.Sprintf("virsh domiflist %s", name)
	cmd := exec.Command("/bin/sh", "-c", shell)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return parserMac(string(output))
}

// example input :
// Interface  Type       Source     Model       MAC
// -------------------------------------------------------
// vnet1      bridge     br0        virtio      52:54:00:d4:9e:bb
// output: 52:54:00:d4:9e:bb, nil
func parserMac(data string) (string, error) {
	data = stripEmpty(data)
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		if !strings.Contains(line, "bridge") {
			continue
		}

		fields := strings.Split(line, " ")
		if len(fields) < 5 {
			continue
		}

		return fields[4], nil
	}

	return "", errors.New("mac not found")
}

// example input:
// Name       MAC address          Protocol     Address
// -------------------------------------------------------------------------------
// eth0       52:54:00:4c:5b:c0    ipv4         172.16.30.216/24
// -          -                    ipv6         fe80::5054:ff:fe4c:5bc0/64
// output: 172.16.30.216, nil
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

	return "", errors.New("ip not found")
}

// example input:
// 172.16.30.216 dev br0 lladdr 52:54:00:4c:5b:c0 STALE
// output: 172.16.30.216, nil
func parserIPFromIPNeign(data string) (string, error) {
	fields := strings.Split(strings.Trim(data, "\n"), " ")
	if len(fields) != 6 {
		return "", errors.New("ip not found")
	}

	return fields[0], nil
}

// example input:
//  Id    Name                           State
// ----------------------------------------------------
// 6     vm2                            running
// 11    vm3                            running
// 12    vm1                            running
// -     vm-template                    shut off
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
