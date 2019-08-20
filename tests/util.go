// Copyright 2018 PingCAP, Inc.
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
// limitations under the License.package spec

package tests

import (
	"bytes"
	"fmt"
	"math/rand"
	"text/template"
	"time"

	"github.com/pingcap/tidb-operator/tests/slack"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Keep will keep the fun running in the period, otherwise the fun return error
func KeepOrDie(interval time.Duration, period time.Duration, fun func() error) {
	timeline := time.Now().Add(period)
	for {
		if time.Now().After(timeline) {
			break
		}
		err := fun()
		if err != nil {
			slack.NotifyAndPanic(err)
		}
		time.Sleep(interval)
	}
}

func SelectNode(nodes []Nodes) string {
	rand.Seed(time.Now().Unix())
	index := rand.Intn(len(nodes))
	vmNodes := nodes[index].Nodes
	index2 := rand.Intn(len(vmNodes))
	return vmNodes[index2]
}

func GetKubeApiserverPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetPodsByLabels(kubeCli, node, map[string]string{"component": "kube-apiserver"})
}

func GetKubeSchedulerPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetPodsByLabels(kubeCli, node, map[string]string{"component": "kube-scheduler"})
}

func GetKubeControllerManagerPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetPodsByLabels(kubeCli, node, map[string]string{"component": "kube-controller-manager"})
}

func GetKubeDNSPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetPodsByLabels(kubeCli, node, map[string]string{"k8s-app": "kube-dns"})
}

func GetKubeProxyPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetPodsByLabels(kubeCli, node, map[string]string{"k8s-app": "kube-proxy"})
}

func GetPodsByLabels(kubeCli kubernetes.Interface, node string, lables map[string]string) (*corev1.Pod, error) {
	selector := labels.Set(lables).AsSelector()
	options := metav1.ListOptions{LabelSelector: selector.String()}
	componentPods, err := kubeCli.CoreV1().Pods("kube-system").List(options)
	if err != nil {
		return nil, err
	}
	for _, componentPod := range componentPods.Items {
		if componentPod.Spec.NodeName == node {
			return &componentPod, nil
		}
	}
	return nil, nil
}

var affinityTemp string = `{{.Kind}}:
  config: |
{{range .Config}}    {{.}}
{{end}}
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: {{.Weight}}
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/instance: {{.ClusterName}}
              app.kubernetes.io/component: {{.Kind}}
          topologyKey: {{.TopologyKey}}
          namespaces:
          - {{.Namespace}}
`

var binlogTemp string = `binlog:
  pump:
    tolerations:
    - key: node-role
      operator: Equal
      value: tidb
      effect: "NoSchedule"
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 50
          podAffinityTerm:
            topologyKey: {{.TopologyKey}}
            namespaces:
            - {{.Namespace}}
{{if .PumpConfig}}
  	config: |
{{range .PumpConfig}}      {{.}}
{{end}}{{end}}
  drainer:
    tolerations:
    - key: node-role
      operator: Equal
      value: tidb
      effect: "NoSchedule"
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 50
          podAffinityTerm:
            topologyKey: {{.TopologyKey}}
            namespaces:
            - {{.Namespace}}
{{if .DrainerConfig}}
    config: |
{{range .DrainerConfig}}      {{.}}
{{end}}{{end}}
`

type AffinityInfo struct {
	ClusterName string
	Kind        string
	Weight      int
	Namespace   string
	TopologyKey string
	Config      []string
}

type BinLogInfo struct {
	PumpConfig    []string
	DrainerConfig []string
	Namespace     string
	TopologyKey   string
}

func GetSubValuesOrDie(clusterName, namespace, topologyKey string, pdConfig []string, tikvConfig []string, tidbConfig []string, pumpConfig []string, drainerConfig []string) string {
	temp, err := template.New("dt-affinity").Parse(affinityTemp)
	if err != nil {
		slack.NotifyAndPanic(err)
	}

	pdbuff := new(bytes.Buffer)
	err = temp.Execute(pdbuff, &AffinityInfo{ClusterName: clusterName, Kind: "pd", Weight: 50, Namespace: namespace, TopologyKey: topologyKey, Config: pdConfig})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	tikvbuff := new(bytes.Buffer)
	err = temp.Execute(tikvbuff, &AffinityInfo{ClusterName: clusterName, Kind: "tikv", Weight: 50, Namespace: namespace, TopologyKey: topologyKey, Config: tikvConfig})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	tidbbuff := new(bytes.Buffer)
	err = temp.Execute(tidbbuff, &AffinityInfo{ClusterName: clusterName, Kind: "tidb", Weight: 50, Namespace: namespace, TopologyKey: topologyKey, Config: tidbConfig})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	subValues := fmt.Sprintf("%s%s%s", pdbuff.String(), tikvbuff.String(), tidbbuff.String())

	if pumpConfig == nil && drainerConfig == nil {
		return subValues
	}

	btemp, err := template.New("binlog").Parse(binlogTemp)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	binlogbuff := new(bytes.Buffer)
	err = btemp.Execute(binlogbuff, &BinLogInfo{PumpConfig: pumpConfig, DrainerConfig: drainerConfig, Namespace: namespace, TopologyKey: topologyKey})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	subValues = fmt.Sprintf("%s%s", subValues, binlogbuff.String())
	return subValues
}

const (
	PodPollInterval = 2 * time.Second
	// PodTimeout is how long to wait for the pod to be started or
	// terminated.
	PodTimeout = 5 * time.Minute
)

func waitForPodNotFoundInNamespace(c kubernetes.Interface, podName, ns string, timeout time.Duration) error {
	return wait.PollImmediate(PodPollInterval, timeout, func() (bool, error) {
		_, err := c.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil // done
		}
		if err != nil {
			return true, err // stop wait with error
		}
		return false, nil
	})
}

func waitForComponentStatus(c kubernetes.Interface, component string, statusType corev1.ComponentConditionType, status corev1.ConditionStatus) error {
	return wait.PollImmediate(time.Second, 5*time.Minute, func() (bool, error) {
		componentStatus, err := c.CoreV1().ComponentStatuses().Get(component, metav1.GetOptions{})
		if err != nil {
			return true, err // stop wait with error
		}
		found := false
		for _, condition := range componentStatus.Conditions {
			if condition.Type == statusType && condition.Status == status {
				found = true
				break
			}
		}
		return found, nil
	})
}
