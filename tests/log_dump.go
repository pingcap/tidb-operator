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

package tests

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (oa *operatorActions) DumpAllLogs(operatorInfo *OperatorConfig, testClusters []*TidbClusterConfig) error {
	logPath := fmt.Sprintf("/%s/%s", oa.cfg.LogDir, "operator-stability")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err = os.MkdirAll(logPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// dump all resources info
	resourceLogFile, err := os.Create(filepath.Join(logPath, "resources"))
	if err != nil {
		return err
	}
	defer resourceLogFile.Close()
	resourceWriter := bufio.NewWriter(resourceLogFile)
	dumpLog("kubectl get po -owide -n kube-system", resourceWriter)
	dumpLog(fmt.Sprintf("kubectl get po -owide -n %s", operatorInfo.Namespace), resourceWriter)
	dumpLog("kubectl get pv", resourceWriter)
	dumpLog("kubectl get pv -oyaml", resourceWriter)
	dumpedNamespace := map[string]bool{}
	for _, testCluster := range testClusters {
		if _, exist := dumpedNamespace[testCluster.Namespace]; !exist {
			dumpLog(fmt.Sprintf("kubectl get po,pvc,svc,cm,cronjobs,jobs,statefulsets,tidbclusters -owide -n %s", testCluster.Namespace), resourceWriter)
			dumpLog(fmt.Sprintf("kubectl get po,pvc,svc,cm,cronjobs,jobs,statefulsets,tidbclusters -n %s -oyaml", testCluster.Namespace), resourceWriter)
			dumpedNamespace[testCluster.Namespace] = true
		}
	}

	// dump operator components's log
	operatorPods, err := oa.kubeCli.CoreV1().Pods(operatorInfo.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range operatorPods.Items {
		err := dumpPod(logPath, &pod)
		if err != nil {
			return err
		}
	}

	// dump all test clusters's logs
	dumpedNamespace = map[string]bool{}
	for _, testCluster := range testClusters {
		if _, exist := dumpedNamespace[testCluster.Namespace]; !exist {
			clusterPodList, err := oa.kubeCli.CoreV1().Pods(testCluster.Namespace).List(metav1.ListOptions{})
			if err != nil {
				return err
			}
			for _, pod := range clusterPodList.Items {
				err := dumpPod(logPath, &pod)
				if err != nil {
					return err
				}
			}
			dumpedNamespace[testCluster.Namespace] = true
		}
	}

	return nil
}

func dumpPod(logPath string, pod *corev1.Pod) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s.log", pod.Name, pod.Namespace)))
	if err != nil {
		return err
	}
	defer logFile.Close()
	plogFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-p.log", pod.Name, pod.Namespace)))
	if err != nil {
		return err
	}
	defer plogFile.Close()

	logWriter := bufio.NewWriter(logFile)
	plogWriter := bufio.NewWriter(plogFile)
	defer logWriter.Flush()
	defer plogWriter.Flush()

	for _, c := range pod.Spec.Containers {
		dumpLog(fmt.Sprintf("kubectl logs -n %s %s -c %s", pod.Namespace, pod.GetName(), c.Name), logWriter)
		dumpLog(fmt.Sprintf("kubectl logs -n %s %s -c %s -p", pod.Namespace, pod.GetName(), c.Name), plogWriter)
	}

	return nil
}

func dumpLog(cmdStr string, writer *bufio.Writer) {
	writer.WriteString(fmt.Sprintf("$ %s\n", cmdStr))
	cmd := exec.Command("/bin/sh", "-c", "/usr/local/bin/"+cmdStr)
	cmd.Stderr = writer
	cmd.Stdout = writer
	err := cmd.Run()
	if err != nil {
		writer.WriteString(err.Error())
	}
}
