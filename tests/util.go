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
	"fmt"
	"math/rand"
	"time"

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
			panic(err)
		}
		time.Sleep(interval)
	}
}

func SelectNode(nodes []Nodes) string {
	rand.Seed(time.Now().Unix())
	index := rand.Intn(len(nodes))
	return nodes[index].Nodes[0]
}

func GetApiserverPodOrDie(kubeCli kubernetes.Interface, node string) *corev1.Pod {
	selector := labels.Set(map[string]string{"component": "kube-apiserver"}).AsSelector()
	options := metav1.ListOptions{LabelSelector: selector.String()}
	apiserverPods, err := kubeCli.CoreV1().Pods("kube-system").List(options)
	if err != nil {
		panic(err)
	}
	for _, apiserverPod := range apiserverPods.Items {
		if apiserverPod.Spec.NodeName == node {
			return &apiserverPod
		}
	}
	panic(fmt.Errorf("can't find apiserver in node:%s", node))
}

func GetControllerManagerPodOrDie(kubeCli kubernetes.Interface, node string) *corev1.Pod {
	selector := labels.Set(map[string]string{"component": "kube-controller-manager"}).AsSelector()
	options := metav1.ListOptions{LabelSelector: selector.String()}
	apiserverPods, err := kubeCli.CoreV1().Pods("kube-system").List(options)
	if err != nil {
		panic(err)
	}
	for _, apiserverPod := range apiserverPods.Items {
		if apiserverPod.Spec.NodeName == node {
			return &apiserverPod
		}
	}
	panic(fmt.Errorf("can't find controller-manager in node:%s", node))
}
