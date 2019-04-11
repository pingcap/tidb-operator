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
	"path/filepath"

	"github.com/golang/glog"

	"github.com/pingcap/tidb-operator/tests/pkg/util"
)

const (
	kubeProxyPath     = "/etc/kubernetes/kube-proxy"
	kubeProxyManifest = "kube-proxy-ds.yml"
	kubeProxyLabel    = "app.kubernetes.io/kube-proxy"
)

// UpdateKubeProxyDaemonset add node affinity for previous kube-proxy daemonset
func (m *Manager) UpdateKubeProxyDaemonset(hostname string) error {
	kubectlPath, err := getKubectlPath()
	if err != nil {
		glog.Infof("skip update kube-proxy daemonset, err: %v", err)
		return nil
	}

	// Get all k8s master nodes
	k8sMasterNodes, err := util.ListK8sNodes(kubectlPath, fmt.Sprintf("%s=", util.LabelNodeRoleMaster))
	if err != nil {
		return err
	}

	// Select the first k8s master node to execute the update operation
	if hostname != k8sMasterNodes[0] {
		glog.Infof("not the first k8s master node, skip update kube-proxy daemonset, node name: %s", hostname)
		return nil
	}

	// Ensure kubeProxyPath exists
	err = util.EnsureDirectoryExist(kubeProxyPath)
	if err != nil {
		return err
	}

	// Render kub-proxy daemonset's template
	kubeProxyManifestFullPath := filepath.Join(kubeProxyPath, kubeProxyManifest)
	f, err := os.Create(kubeProxyManifestFullPath)
	if err != nil {
		return fmt.Errorf("create file %s failed, err: %v", kubeProxyManifestFullPath, err)
	}
	defer f.Close()
	proxyDaemonSetBytes, err := util.ParseTemplate(KubeProxyDaemonSet, struct{ Image, KubeProxyLabel string }{
		Image:          m.kubeProxyImages,
		KubeProxyLabel: kubeProxyLabel,
	})
	if err != nil {
		return fmt.Errorf("parsing kube-proxy daemonset template failed, err: %v", err)
	}
	_, err = f.Write(proxyDaemonSetBytes)
	if err != nil {
		return fmt.Errorf("write kube-proxy daemonset manifest into file %s failed, err: %v", kubeProxyManifestFullPath, err)
	}

	// Apply kube-proxy daemonset manifest
	commandStr := fmt.Sprintf("%s apply -f %s", kubectlPath, kubeProxyManifestFullPath)
	cmd := exec.Command("/bin/sh", "-c", commandStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec: [%s] failed, output: %s, error: %v", commandStr, string(output), err)
	}

	// Add app.kubernetes.io/kube-proxy= label for all k8s nodes
	allK8sNodes, err := util.ListK8sNodes(kubectlPath, "")
	if err != nil {
		return err
	}
	for _, node := range allK8sNodes {
		err := addKubeProxyLabel(kubectlPath, node)
		if err != nil {
			glog.Error(err)
			return err
		}
	}

	glog.Infof("update kube-proxy daemonset success")
	return nil
}

// StartKubeProxy start kube-proxy of the specified node
func (m *Manager) StartKubeProxy(nodeName string) error {
	kubectlPath, err := getKubectlPath()
	if err != nil {
		glog.Error(err)
		return err
	}
	err = addKubeProxyLabel(kubectlPath, nodeName)
	if err != nil {
		glog.Error(err)
		return err
	}

	glog.Infof("node %s kube-proxy is started", nodeName)
	return nil
}

// StopKubeProxy stop kube-proxy of the specified node
func (m *Manager) StopKubeProxy(nodeName string) error {
	kubectlPath, err := getKubectlPath()
	if err != nil {
		glog.Error(err)
		return err
	}
	commandStr := fmt.Sprintf("%s label no %s %s-", kubectlPath, nodeName, kubeProxyLabel)
	cmd := exec.Command("/bin/sh", "-c", commandStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("remove kube-proxy label failed, command: [%s], output: %s, error: %v", commandStr, string(output), err)
		return err
	}

	glog.Infof("node %s kube-proxy is stoped", nodeName)
	return nil
}

func getKubectlPath() (string, error) {
	paths := filepath.SplitList(os.Getenv("PATH"))
	kubectlPath, err := util.FindInPath("kubectl", paths)
	if err != nil {
		return "", fmt.Errorf("can't found kubectl binary, not deployed on the k8s master node, err: %v", err)
	}
	return kubectlPath, nil
}

func addKubeProxyLabel(kubectlPath, nodeName string) error {
	commandStr := fmt.Sprintf("%s label no %s %s= --overwrite", kubectlPath, nodeName, kubeProxyLabel)
	cmd := exec.Command("/bin/sh", "-c", commandStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("add kube-proxy label failed, command: [%s], output: %s, error: %v", commandStr, string(output), err)
	}
	return nil
}
