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

package member

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

const (
	// AnnAutoFailureRecovery in TidbCluster indicates whether auto failure recovery is enabled for the components
	AnnAutoFailureRecovery = "kubernetes.io/auto-failure-recovery"

	podPhaseRunning = "Running"
	podPhaseUnknown = "Unknown"

	// Reason for host down being true
	hdReasonNodeFailure           = "NodeFailure"
	hdReasonRODiskFound           = "RODiskFound"
	hdReasonStoreDownTimeExceeded = "StoreDownTimeExceeded"

	// restartToDeleteStoreGap
	restartToDeleteStoreGap = 10 * time.Minute
)

// canAutoFailureRecovery checks whether auto recovery of failure pods and pvc should be done
func canAutoFailureRecovery(tc *v1alpha1.TidbCluster) bool {
	// TODO Use a boolean in TidbCluster spec instead of annotation for this
	return tc.Annotations[AnnAutoFailureRecovery] == strconv.FormatBool(true)
}

// isHostingNodeUnavailable checks whether the hosting node is unavailable
// 1. Check the pod Status, if the pod has Unknown status phase, then it means node is not available
// 2. Check the node Status, if the Ready condition of node is False or Unknown, then it means node is not available
func isHostingNodeUnavailable(nodeLister corelisterv1.NodeLister, pod *corev1.Pod) (bool, bool, error) {
	ns := pod.Namespace
	name := strings.Split(pod.Name, ".")[0]

	nodeUnavailable := pod.Status.Phase == podPhaseUnknown
	var roDiskFound bool
	if pod.Status.Phase == podPhaseRunning {
		// If the Ready condition of pod is False, then detect whether the K8s node hosting the pod is no more available
		_, podReadyCond := getPodConditionFromList(pod.Status.Conditions, corev1.PodReady)
		klog.Infof("failover[isHostingNodeUnavailable]: pod ready condition of %s of failure pod %s/%s = %v", pod.Spec.NodeName, ns, pod.Name, podReadyCond)
		if podReadyCond != nil && nodeLister != nil {
			podNode, err := nodeLister.Get(pod.Spec.NodeName)
			if err != nil {
				return false, false, fmt.Errorf("failover[isHostingNodeUnavailable]: failed to get node for pod %s/%s, error: %s", ns, name, err)
			}
			if podReadyCond.Status == corev1.ConditionFalse {
				nodeUnavailable = IsNodeReadyConditionFalseOrUnknown(podNode.Status)
			}
			if podReadyCond.Status == corev1.ConditionTrue {
				roDiskFound = IsNodeRODiskFoundConditionTrue(podNode.Status)
			}
		}
	}
	klog.Infof("failover[isHostingNodeUnavailable]: nodeUnavailable=%t, roDiskFound=%t for %s of failure pod %s/%s", nodeUnavailable, roDiskFound, pod.Spec.NodeName, ns, pod.Name)
	return nodeUnavailable, roDiskFound, nil
}

// getPodAndPvcs returns the pod and pvcs of a component pod in a Tidb cluster
func getPodAndPvcs(tc *v1alpha1.TidbCluster, podName string, memberType v1alpha1.MemberType, podLister corelisterv1.PodLister, pvcLister corelisterv1.PersistentVolumeClaimLister) (*corev1.Pod, []*corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	pod, err := podLister.Pods(ns).Get(podName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s failover: failed to get pod %s for tc %s/%s, error: %s", memberType, podName, ns, tcName, err)
	}
	pvcs, err := getPodPvcs(tc, podName, memberType, pvcLister)
	if err != nil {
		return pod, nil, err
	}
	return pod, pvcs, nil
}

// getPodPvcs returns the pvcs of a component pod in a Tidb cluster
func getPodPvcs(tc *v1alpha1.TidbCluster, podName string, memberType v1alpha1.MemberType, pvcLister corelisterv1.PersistentVolumeClaimLister) ([]*corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		return nil, fmt.Errorf("%s failover: failed to parse ordinal from pod name for %s/%s, error: %s", memberType, ns, podName, err)
	}
	pvcSelector, err := GetPVCSelectorForPod(tc, memberType, ordinal)
	if err != nil {
		return nil, fmt.Errorf("%s failover: failed to get PVC selector for pod %s/%s, error: %s", memberType, ns, podName, err)
	}
	pvcs, err := pvcLister.PersistentVolumeClaims(ns).List(pvcSelector)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("%s failover: failed to get PVCs for pod %s/%s, error: %s", memberType, ns, podName, err)
	}
	return pvcs, nil
}

// isPodConditionScheduledTrue
func isPodConditionScheduledTrue(conditions []corev1.PodCondition) bool {
	_, podSchCond := getPodConditionFromList(conditions, corev1.PodScheduled)
	return podSchCond != nil && podSchCond.Status == corev1.ConditionTrue
}

// getPodConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func getPodConditionFromList(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) (int, *corev1.PodCondition) {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}
