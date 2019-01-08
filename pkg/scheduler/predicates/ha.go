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
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ha struct {
	kubeCli   kubernetes.Interface
	cli       versioned.Interface
	podListFn func(ns, instanceName, component string) (*apiv1.PodList, error)
	pvcGetFn  func(ns, pvcName string) (*apiv1.PersistentVolumeClaim, error)
	tcGetFn   func(ns, tcName string) (*v1alpha1.TidbCluster, error)
}

// NewHA returns a Predicate
func NewHA(kubeCli kubernetes.Interface, cli versioned.Interface) Predicate {
	h := &ha{
		kubeCli: kubeCli,
		cli:     cli,
	}
	h.podListFn = h.realPodListFn
	h.pvcGetFn = h.realPVCGetFn
	h.tcGetFn = h.realTCGetFn
	return h
}

func (h *ha) Name() string {
	return "HighAvailability"
}

// 1. return the node to kube-scheduler if there is only one node and the pod's pvc is bound
// 2. return these nodes that have least pods and its pods count is less than replicas/2 to kube-scheduler
// 3. let kube-scheduler to make the final decision
func (h *ha) Filter(instanceName string, pod *apiv1.Pod, nodes []apiv1.Node) ([]apiv1.Node, error) {
	ns := pod.GetNamespace()
	podName := pod.GetName()
	component := pod.Labels[label.ComponentLabelKey]
	tcName := getTCNameFromPod(pod, component)

	if len(nodes) == 0 {
		return nil, fmt.Errorf("kube nodes is empty")
	}

	if len(nodes) == 1 {
		pvcName := fmt.Sprintf("%s-%s", component, podName)
		pvc, err := h.pvcGetFn(ns, pvcName)
		if err != nil {
			return nil, err
		}
		if pvc.Status.Phase == apiv1.ClaimBound {
			return nodes, nil
		}
	}

	podList, err := h.podListFn(ns, instanceName, component)
	if err != nil {
		return nil, err
	}
	tc, err := h.tcGetFn(ns, tcName)
	if err != nil {
		return nil, err
	}
	replicas := getReplicasFrom(tc, component)

	nodeMap := make(map[string][]string)
	for _, node := range nodes {
		nodeMap[node.GetName()] = make([]string, 0)
	}
	for _, pod := range podList.Items {
		pName := pod.GetName()
		nodeName := pod.Spec.NodeName
		if nodeName == "" || nodeMap[nodeName] == nil {
			continue
		}

		nodeMap[nodeName] = append(nodeMap[nodeName], pName)
	}
	glog.V(4).Infof("nodeMap: %+v", nodeMap)

	min := -1
	minNodeNames := make([]string, 0)
	for nodeName, podNames := range nodeMap {
		podsCount := len(podNames)
		if podsCount+1 >= int(replicas+1)/2 {
			continue
		}
		if min == -1 {
			min = podsCount
		}

		if podsCount > min {
			continue
		}
		if podsCount < min {
			min = podsCount
			minNodeNames = make([]string, 0)
		}
		minNodeNames = append(minNodeNames, nodeName)
	}

	if len(minNodeNames) == 0 {
		return nil, fmt.Errorf("can't find a node from: %v, nodeMap: %v", nodes, nodeMap)
	}
	return getNodeFromNames(nodes, minNodeNames), nil
}

func (h *ha) realPodListFn(ns, instanceName, component string) (*apiv1.PodList, error) {
	selector := label.New().Instance(instanceName).Component(component).Labels()
	return h.kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector).String(),
	})
}

func (h *ha) realPVCGetFn(ns, pvcName string) (*apiv1.PersistentVolumeClaim, error) {
	return h.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
}

func (h *ha) realTCGetFn(ns, tcName string) (*v1alpha1.TidbCluster, error) {
	return h.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
}

func getTCNameFromPod(pod *apiv1.Pod, component string) string {
	return strings.TrimSuffix(pod.GenerateName, fmt.Sprintf("-%s-", component))
}

func getReplicasFrom(tc *v1alpha1.TidbCluster, component string) int32 {
	if component == v1alpha1.PDMemberType.String() {
		return tc.Spec.PD.Replicas
	}

	return tc.Spec.TiKV.Replicas
}
