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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DumpPod dumps logs for a pod.
func DumpPod(logPath string, pod *corev1.Pod) error {
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

func DumpResource(dir string, typ string, obj metav1.Object) error {
	obj.SetManagedFields(nil)

	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}

	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s_%s_%s.yaml", typ, obj.GetName(), obj.GetNamespace())
	filepath := filepath.Join(dir, filename)
	err = ioutil.WriteFile(filepath, yamlBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}
