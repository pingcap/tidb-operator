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
// limitations under the License.

package predicates

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	replicas int32 = 3
)

type ha struct {
	kubeCli   kubernetes.Interface
	podListFn func(ns, clusterName, component string) (*apiv1.PodList, error)
}

// NewHA returns a Predicate
func NewHA(kubeCli kubernetes.Interface) Predicate {
	h := &ha{
		kubeCli: kubeCli,
	}
	h.podListFn = h.realPodListFn
	return h
}

func (h *ha) Name() string {
	return "HighAvailability"
}

// 1. First, we sort all the nodes we get from kube-scheduler by how many same kind of pod it contains,
//    find the nodes that have least pods.
// 2. When scheduling the first replicas pods, we must ensure no previous pods on the nodes.
// 3. For later pods, we choose the nodes that have least pods.
func (h *ha) Filter(clusterName string, pod *apiv1.Pod, nodes []apiv1.Node) ([]apiv1.Node, error) {
	ns := pod.GetNamespace()
	podName := pod.GetName()

	var component string
	var exist bool
	if component, exist = pod.Labels[label.ComponentLabelKey]; !exist {
		return nodes, fmt.Errorf("can't find component in pod labels: %s/%s", ns, podName)
	}
	podList, err := h.podListFn(ns, clusterName, component)
	if err != nil {
		return nil, err
	}

	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[string][]string)
	for _, node := range nodes {
		nodeMap[node.GetName()] = make([]string, 0)
	}
	for _, pod := range podList.Items {
		podName1 := pod.GetName()
		nodeName := pod.Spec.NodeName
		ordinal1, err := util.GetOrdinalFromPodName(podName1)
		if err != nil {
			return nil, err
		}

		if ordinal1 < ordinal && nodeName == "" {
			return nil, fmt.Errorf("waiting for pod: %s/%s to be scheduled", ns, podName1)
		}
		if nodeName == "" {
			continue
		}
		if nodeMap[nodeName] == nil {
			continue
		}

		nodeMap[nodeName] = append(nodeMap[nodeName], podName1)
	}

	var min int
	var minInitialized bool
	for _, podNameArr := range nodeMap {
		count := len(podNameArr)
		if !minInitialized {
			minInitialized = true
			min = count
		}
		if count < min {
			min = count
		}
	}
	if ordinal < replicas && min != 0 {
		return nil, fmt.Errorf("the first %d pods can't be scheduled to the same node", replicas)
	}

	minNodeNames := make([]string, 0)
	for nodeName, podNameArr := range nodeMap {
		if len(podNameArr) == min {
			minNodeNames = append(minNodeNames, nodeName)
		}
	}

	return getNodeFromNames(nodes, minNodeNames), nil
}

func (h *ha) realPodListFn(ns, clusterName, component string) (*apiv1.PodList, error) {
	selector := label.New().Cluster(clusterName).Component(component).Labels()
	return h.kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector).String(),
	})
}
