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
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
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
}

// NewHA returns a Predicate
func NewHA(kubeCli kubernetes.Interface, cli versioned.Interface) Predicate {
	h := &ha{
		kubeCli: kubeCli,
		cli:     cli,
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
	return "HAScheduling"
}

// 1. return the node to kube-scheduler if there is only one feasible node and the pod's pvc is bound
// 2. if there are more than two feasible nodes, we are trying to distribute TiKV/PD pods across the nodes for the best HA
//  a) for PD (one raft group, copies of data equals to replicas), no more than majority of replicas pods on one node, otherwise majority of replicas may lose when a node is lost.
//     e.g. when replicas is 3, we requires no more than 1 pods per node.
//  b) for TiKV (multiple raft groups, in each raft group, copies of data is hard-coded to 3)
//     when replicas is less than 3, no HA is forced because HA is impossible
//     when replicas is equal or greater than 3, we require TiKV pods are running on more than 3 nodes and no more than ceil(replicas / 3) per node
//  for PD/TiKV, we both try to balance the number of pods acorss the nodes
// 3. let kube-scheduler to make the final decision
func (h *ha) Filter(instanceName string, pod *apiv1.Pod, nodes []apiv1.Node) ([]apiv1.Node, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	ns := pod.GetNamespace()
	podName := pod.GetName()
	component := pod.Labels[label.ComponentLabelKey]
	tcName := getTCNameFromPod(pod, component)

	if component != label.PDLabelVal && component != label.TiKVLabelVal {
		klog.V(4).Infof("component %s is ignored in HA predicate", component)
		return nodes, nil
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available to schedule pods %s/%s", ns, podName)
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
	klog.Infof("ha: tidbcluster %s/%s component %s replicas %d", ns, tcName, component, replicas)

	allNodes := make(sets.String)
	nodeMap := make(map[string][]string)
	for _, node := range nodes {
		nodeMap[node.GetName()] = make([]string, 0)
	}
	for _, pod := range podList.Items {
		pName := pod.GetName()
		nodeName := pod.Spec.NodeName
		if nodeName != "" {
			allNodes.Insert(nodeName)
		}
		if nodeName == "" || nodeMap[nodeName] == nil {
			continue
		}

		nodeMap[nodeName] = append(nodeMap[nodeName], pName)
	}
	klog.V(4).Infof("nodeMap: %+v", nodeMap)

	min := -1
	minNodeNames := make([]string, 0)
	maxPodsPerNode := 0

	if component == label.PDLabelVal {
		/**
		 * replicas     maxPodsPerNode
		 * ---------------------------
		 * 1            1
		 * 2            1
		 * 3            1
		 * 4            1
		 * 5            2
		 * ...
		 */
		maxPodsPerNode = int((replicas+1)/2) - 1
		if maxPodsPerNode <= 0 {
			maxPodsPerNode = 1
		}
	} else {
		// 1. TiKV instances must run on at least 3 nodes, otherwise HA is not possible
		if allNodes.Len() < 3 {
			maxPodsPerNode = 1
		} else {
			/**
			 * 2. we requires TiKV instances to run on at least 3 nodes, so max
			 * allowed pods on each node is ceil(replicas / 3)
			 *
			 * replicas     maxPodsPerNode   best HA on three nodes
			 * ---------------------------------------------------
			 * 3            1                1, 1, 1
			 * 4            2                1, 1, 2
			 * 5            2                1, 2, 2
			 * 6            2                2, 2, 2
			 * 7            3                2, 2, 3
			 * 8            3                2, 3, 3
			 * ...
			 */
			maxPodsPerNode = int(math.Ceil(float64(replicas) / 3))
		}
	}

	for nodeName, podNames := range nodeMap {
		podsCount := len(podNames)

		// tikv replicas less than 3 cannot achieve high availability
		if component == label.TiKVLabelVal && replicas < 3 {
			minNodeNames = append(minNodeNames, nodeName)
			klog.Infof("replicas is %d, add node %s to minNodeNames", replicas, nodeName)
			continue
		}

		if podsCount+1 > maxPodsPerNode {
			// pods on this node exceeds the limit, skip
			klog.Infof("node %s has %d instances of component %s, max allowed is %d, skipping",
				nodeName, podsCount, component, maxPodsPerNode)
			continue
		}

		// Choose nodes which has minimum count of the component
		if min == -1 {
			min = podsCount
		}
		if podsCount > min {
			klog.Infof("node %s podsCount %d > min %d, skipping", nodeName, podsCount, min)
			continue
		}
		if podsCount < min {
			min = podsCount
			minNodeNames = make([]string, 0)
		}
		minNodeNames = append(minNodeNames, nodeName)
	}

	if len(minNodeNames) == 0 {
		nodesStrArr := []string{}
		for nodeName, podNameArr := range nodeMap {
			s := fmt.Sprintf("%s (%d %s pods)",
				nodeName, len(podNameArr), strings.ToLower(component))
			nodesStrArr = append(nodesStrArr, s)
		}
		sort.Strings(nodesStrArr)

		// example: unable to schedule to nodes: kube-node-1 (1 pd pods), kube-node-2 (1 pd pods), max pods per node: 1
		errMsg := fmt.Sprintf("unable to schedule to nodes: %s, max pods per node: %d",
			strings.Join(nodesStrArr, ", "), maxPodsPerNode)
		return nil, errors.New(errMsg)
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
			return schedulingPVC, currentPVC, fmt.Errorf("waiting for Pod %s/%s scheduling", ns, schedulingPodName)
		}
	}

	delete(schedulingPVC.Annotations, label.AnnPVCPodScheduling)
	err = h.updatePVCFn(schedulingPVC)
	if err != nil {
		klog.Errorf("ha: failed to delete pvc %s/%s annotation %s, %v",
			ns, schedulingPVC.GetName(), label.AnnPVCPodScheduling, err)
		return schedulingPVC, currentPVC, err
	}
	klog.Infof("ha: delete pvc %s/%s annotation %s successfully",
		ns, schedulingPVC.GetName(), label.AnnPVCPodScheduling)
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
	ns := pvc.GetNamespace()
	tcName := pvc.Labels[label.InstanceLabelKey]
	pvcName := pvc.GetName()

	labels := pvc.GetLabels()
	ann := pvc.GetAnnotations()
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, updateErr := h.kubeCli.CoreV1().PersistentVolumeClaims(ns).Update(pvc)
		if updateErr == nil {
			klog.Infof("update PVC: [%s/%s] successfully, TidbCluster: %s", ns, pvcName, tcName)
			return nil
		}
		klog.Errorf("failed to update PVC: [%s/%s], TidbCluster: %s, error: %v", ns, pvcName, tcName, updateErr)

		if updated, err := h.pvcGetFn(ns, pvcName); err == nil {
			// make a copy so we don't mutate the shared cache
			pvc = updated.DeepCopy()
			pvc.Labels = labels
			pvc.Annotations = ann
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated PVC %s/%s from lister: %v", ns, pvcName, err))
		}

		return updateErr
	})
}

func (h *ha) realPVCGetFn(ns, pvcName string) (*apiv1.PersistentVolumeClaim, error) {
	return h.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
}

func (h *ha) realTCGetFn(ns, tcName string) (*v1alpha1.TidbCluster, error) {
	return h.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
}

func (h *ha) setCurrentPodScheduling(pvc *apiv1.PersistentVolumeClaim) error {
	ns := pvc.GetNamespace()
	pvcName := pvc.GetName()
	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCPodScheduling] = now
	err := h.updatePVCFn(pvc)
	if err != nil {
		klog.Errorf("ha: failed to set pvc %s/%s annotation %s to %s, %v",
			ns, pvcName, label.AnnPVCPodScheduling, now, err)
		return err
	}
	klog.Infof("ha: set pvc %s/%s annotation %s to %s successfully",
		ns, pvcName, label.AnnPVCPodScheduling, now)
	return nil
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
