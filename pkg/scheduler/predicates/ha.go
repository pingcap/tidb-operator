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
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

type ha struct {
	lock          sync.Mutex
	kubeCli       kubernetes.Interface
	cli           versioned.Interface
	podListFn     func(ns, instanceName, component string) (*apiv1.PodList, error)
	podGetFn      func(ns, podName string) (*apiv1.Pod, error)
	pvcGetFn      func(ns, pvcName string) (*apiv1.PersistentVolumeClaim, error)
	tcGetFn       func(ns, tcName string) (*v1alpha1.TidbCluster, error)
	pvcListFn     func(ns, instanceName, component string) (*apiv1.PersistentVolumeClaimList, error)
	updatePVCFn   func(*apiv1.PersistentVolumeClaim) error
	acquireLockFn func(*apiv1.Pod) (*apiv1.PersistentVolumeClaim, *apiv1.PersistentVolumeClaim, error)
	recorder      record.EventRecorder
}

// NewHA returns a Predicate
func NewHA(kubeCli kubernetes.Interface, cli versioned.Interface, recorder record.EventRecorder) Predicate {
	h := &ha{
		kubeCli:  kubeCli,
		cli:      cli,
		recorder: recorder,
	}
	h.podListFn = h.realPodListFn
	h.podGetFn = h.realPodGetFn
	h.pvcGetFn = h.realPVCGetFn
	h.tcGetFn = h.realTCGetFn
	h.pvcListFn = h.realPVCListFn
	h.updatePVCFn = h.realUpdatePVCFn
	h.acquireLockFn = h.realAcquireLock
	return h
}

func (h *ha) Name() string {
	return "HighAvailability"
}

// 1. return the node to kube-scheduler if there is only one node and the pod's pvc is bound
// 2. return these nodes that have least pods and its pods count is less than (replicas+1)/2 to kube-scheduler
// 3. let kube-scheduler to make the final decision
func (h *ha) Filter(instanceName string, pod *apiv1.Pod, nodes []apiv1.Node) ([]apiv1.Node, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	ns := pod.GetNamespace()
	podName := pod.GetName()
	component := pod.Labels[label.ComponentLabelKey]
	tcName := getTCNameFromPod(pod, component)

	if len(nodes) == 0 {
		return nil, fmt.Errorf("kube nodes is empty")
	}
	if _, _, err := h.acquireLockFn(pod); err != nil {
		return nil, err
	}

	if len(nodes) == 1 {
		pvcName := pvcName(component, podName)
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
		// replicas less than 3 cannot achieve high availability
		if replicas < 3 {
			minNodeNames = append(minNodeNames, nodeName)
			continue
		}

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
		msg := fmt.Sprintf("can't schedule to nodes: %v, because these pods had been scheduled to nodes: %v", GetNodeNames(nodes), nodeMap)
		h.recorder.Event(pod, apiv1.EventTypeWarning, "FailedScheduling", msg)
		return nil, errors.New(msg)
	}
	return getNodeFromNames(nodes, minNodeNames), nil
}

// kubernetes scheduling is parallel, to achieve HA, we must ensure the scheduling is serial,
// so when a pod is scheduling, we set an annotation to its PVC, other pods can't be scheduled at this time,
// delete the PVC's annotation when the pod is scheduled(PVC is bound and the pod's nodeName is set)
func (h *ha) realAcquireLock(pod *apiv1.Pod) (*apiv1.PersistentVolumeClaim, *apiv1.PersistentVolumeClaim, error) {
	ns := pod.GetNamespace()
	component := pod.Labels[label.ComponentLabelKey]
	instanceName := pod.Labels[label.InstanceLabelKey]
	podName := pod.GetName()
	pvcList, err := h.pvcListFn(ns, instanceName, component)
	if err != nil {
		return nil, nil, err
	}

	currentPVCName := pvcName(component, podName)
	var currentPVC *apiv1.PersistentVolumeClaim
	var schedulingPVC *apiv1.PersistentVolumeClaim
	items := pvcList.Items
	for i := range items {
		if items[i].GetName() == currentPVCName {
			currentPVC = &items[i]
		}
		if items[i].Annotations[label.AnnPVCPodScheduling] != "" && schedulingPVC == nil {
			schedulingPVC = &items[i]
		}
	}

	if currentPVC == nil {
		return schedulingPVC, currentPVC, fmt.Errorf("can't find current Pod %s/%s's PVC", ns, podName)
	}
	if schedulingPVC == nil {
		return schedulingPVC, currentPVC, h.setCurrentPodScheduling(currentPVC)
	}
	if schedulingPVC == currentPVC {
		return schedulingPVC, currentPVC, nil
	}

	// if pvc is not defer deleting(has AnnPVCDeferDeleting annotation means defer deleting), we must wait for its scheduling
	// else clear its AnnPVCPodScheduling annotation and acquire the lock
	if schedulingPVC.Annotations[label.AnnPVCDeferDeleting] == "" {
		schedulingPodName := getPodNameFromPVC(schedulingPVC)
		schedulingPod, err := h.podGetFn(ns, schedulingPodName)
		if err != nil {
			return schedulingPVC, currentPVC, err
		}
		if schedulingPVC.Status.Phase != apiv1.ClaimBound || schedulingPod.Spec.NodeName == "" {
			return schedulingPVC, currentPVC, fmt.Errorf("waiting for Pod %s/%s scheduling", ns, strings.TrimPrefix(schedulingPVC.GetName(), component+"-"))
		}
	}

	delete(schedulingPVC.Annotations, label.AnnPVCPodScheduling)
	err = h.updatePVCFn(schedulingPVC)
	if err != nil {
		return schedulingPVC, currentPVC, err
	}
	return schedulingPVC, currentPVC, h.setCurrentPodScheduling(currentPVC)
}

func (h *ha) realPodListFn(ns, instanceName, component string) (*apiv1.PodList, error) {
	selector := label.New().Instance(instanceName).Component(component).Labels()
	return h.kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector).String(),
	})
}

func (h *ha) realPodGetFn(ns, podName string) (*apiv1.Pod, error) {
	return h.kubeCli.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
}

func (h *ha) realPVCListFn(ns, instanceName, component string) (*apiv1.PersistentVolumeClaimList, error) {
	selector := label.New().Instance(instanceName).Component(component).Labels()
	return h.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector).String(),
	})
}

func (h *ha) realUpdatePVCFn(pvc *apiv1.PersistentVolumeClaim) error {
	_, err := h.kubeCli.CoreV1().PersistentVolumeClaims(pvc.GetNamespace()).Update(pvc)
	return err
}

func (h *ha) realPVCGetFn(ns, pvcName string) (*apiv1.PersistentVolumeClaim, error) {
	return h.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
}

func (h *ha) realTCGetFn(ns, tcName string) (*v1alpha1.TidbCluster, error) {
	return h.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
}

func (h *ha) setCurrentPodScheduling(pvc *apiv1.PersistentVolumeClaim) error {
	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	pvc.Annotations[label.AnnPVCPodScheduling] = time.Now().Format(time.RFC3339)
	return h.updatePVCFn(pvc)
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

func pvcName(component, podName string) string {
	return fmt.Sprintf("%s-%s", component, podName)
}

func GetNodeNames(nodes []apiv1.Node) []string {
	nodeNames := make([]string, 0)
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.GetName())
	}
	sort.Strings(nodeNames)
	return nodeNames
}

func getPodNameFromPVC(pvc *apiv1.PersistentVolumeClaim) string {
	return strings.TrimPrefix(pvc.Name, fmt.Sprintf("%s-", pvc.Labels[label.ComponentLabelKey]))
}
