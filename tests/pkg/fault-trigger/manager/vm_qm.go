package manager

import (
	"fmt"
	"os/exec"
	"strings"

	glog "k8s.io/klog"
)

type QMVMManager struct {
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
