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
	"math/rand"
	"time"

	"github.com/pingcap/tidb-operator/tests/slack"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

func GetApiserverPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetKubeComponent(kubeCli, node, "kube-apiserver")
}

func GetSchedulerPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetKubeComponent(kubeCli, node, "kube-scheduler")
}

func GetDNSPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetKubeComponent(kubeCli, node, "kube-dns")
}

func GetControllerManagerPod(kubeCli kubernetes.Interface, node string) (*corev1.Pod, error) {
	return GetKubeComponent(kubeCli, node, "kube-controller-manager")
}

func GetKubeComponent(kubeCli kubernetes.Interface, node string, componentName string) (*corev1.Pod, error) {
	selector := labels.Set(map[string]string{"component": componentName}).AsSelector()
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
