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
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

const (
	// annAutoFailureRecovery in TidbCluster indicates whether auto failure recovery is enabled for the components
	annAutoFailureRecovery = "kubernetes.io/auto-failure-recovery"

	podPhaseRunning = "Running"
	podPhaseUnknown = "Unknown"

	// Reason for host down being true
	hdReasonNodeFailure           = "NodeFailure"
	hdReasonRODiskFound           = "RODiskFound"
	hdReasonStoreDownTimeExceeded = "StoreDownTimeExceeded"

	restartToDeleteStoreGap = 10 * time.Minute
)

// FailureObjectAccess contains the common set of functions to access the properties of a failure object
type FailureObjectAccess interface {
	GetMemberType() v1alpha1.MemberType
	GetFailureObjects(tc *v1alpha1.TidbCluster) map[string]v1alpha1.EmptyStruct
	IsFailing(tc *v1alpha1.TidbCluster, objectId string) bool
	GetPodName(tc *v1alpha1.TidbCluster, objectId string) string
	IsHostDownForFailurePod(tc *v1alpha1.TidbCluster) bool
	IsHostDown(tc *v1alpha1.TidbCluster, objectId string) bool
	SetHostDown(tc *v1alpha1.TidbCluster, objectId string, hostDown bool)
	GetCreatedAt(tc *v1alpha1.TidbCluster, objectId string) metav1.Time
	GetLastTransitionTime(tc *v1alpha1.TidbCluster, objectId string) metav1.Time
	GetPvcUIDSet(tc *v1alpha1.TidbCluster, objectId string) map[types.UID]v1alpha1.EmptyStruct
}

// commonStatefulFailureRecovery has the common logic to handle the failure recovery of a stateful component
type commonStatefulFailureRecovery struct {
	deps                *controller.Dependencies
	failureObjectAccess FailureObjectAccess
}

// CheckHostDownAndRestartPod checks for HostDown for any failure store or member then does a restart of pod
func (fr *commonStatefulFailureRecovery) CheckHostDownAndRestartPod(tc *v1alpha1.TidbCluster) error {
	if fr.deps.CLIConfig.DetectNodeFailure {
		if fr.failureObjectAccess.IsHostDownForFailurePod(tc) {
			if canAutoFailureRecovery(tc) {
				if err := fr.restartPodForHostDown(tc); err != nil {
					if controller.IsIgnoreError(err) {
						return nil
					}
					return err
				}
			}
		} else {
			if err := fr.checkAndMarkHostDown(tc); err != nil {
				if controller.IsIgnoreError(err) {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

// checkAndMarkHostDown tries to detect node failure if HostDown is not set for any failure store or member
func (fr *commonStatefulFailureRecovery) checkAndMarkHostDown(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for objectId := range fr.failureObjectAccess.GetFailureObjects(tc) {
		if fr.failureObjectAccess.IsFailing(tc, objectId) {
			// for backward compatibility, if there exists failure stores and user upgrades operator to newer version there
			// will be failure store structures with empty PVCUIDSet set from api server, we should not handle those failure
			// stores for failure recovery (or failure member with empty PVCUIDSet)
			if !fr.failureObjectAccess.IsHostDown(tc, objectId) && len(fr.failureObjectAccess.GetPvcUIDSet(tc, objectId)) > 0 {
				pod, err := fr.deps.PodLister.Pods(ns).Get(fr.failureObjectAccess.GetPodName(tc, objectId))
				if err != nil && !errors.IsNotFound(err) {
					return fmt.Errorf("%s failover [checkAndMarkHostDown]: failed to get pod %s for tc %s/%s, error: %s", fr.failureObjectAccess.GetMemberType(), fr.failureObjectAccess.GetPodName(tc, objectId), ns, tcName, err)
				}
				if pod == nil {
					continue
				}

				// Check node and pod conditions and set HostDown in FailureStore
				var reason string
				if fr.deps.CLIConfig.PodHardRecoveryPeriod > 0 && time.Now().After(fr.failureObjectAccess.GetLastTransitionTime(tc, objectId).Add(fr.deps.CLIConfig.PodHardRecoveryPeriod)) {
					reason = hdReasonStoreDownTimeExceeded
				}
				if len(reason) == 0 {
					nodeAvailabilityStatus, detectErr := fr.getNodeAvailabilityStatus(pod)
					if detectErr != nil {
						return detectErr
					}
					if nodeAvailabilityStatus.NodeUnavailable {
						reason = hdReasonNodeFailure
					}
					if nodeAvailabilityStatus.ReadOnlyDiskFound {
						reason = hdReasonRODiskFound
					}
				}

				if len(reason) > 0 {
					fr.failureObjectAccess.SetHostDown(tc, objectId, true)
					return controller.IgnoreErrorf("Marked host down true for pod %s of tc %s/%s. Host down reason: %s", fr.failureObjectAccess.GetPodName(tc, objectId), ns, tcName, reason)
				}
			}
		}
	}
	return nil
}

// getNodeAvailabilityStatus returns the node availability status information
func (fr *commonStatefulFailureRecovery) getNodeAvailabilityStatus(pod *corev1.Pod) (NodeAvailabilityStatus, error) {
	ns := pod.Namespace
	name := strings.Split(pod.Name, ".")[0]

	// 1. Check the pod Status, if the pod has Unknown status phase, then it means node is not available
	// 2. Check the node Status, if the Ready condition of node is False or Unknown, then it means node is not available
	// 3. Check the node Status, if the RODiskFound condition of node is True, then it means disk has become read only
	nodeUnavailable := pod.Status.Phase == podPhaseUnknown
	var roDiskFound bool
	if pod.Status.Phase == podPhaseRunning {
		// If the Ready condition of pod is False, then detect whether the K8s node hosting the pod is no more available
		podReadyCond := getPodConditionFromList(pod.Status.Conditions, corev1.PodReady)
		klog.Infof("failover[getNodeAvailabilityStatus]: pod ready condition of node %s of failure pod %s/%s = %v", pod.Spec.NodeName, ns, pod.Name, podReadyCond)
		if podReadyCond != nil && fr.deps.NodeLister != nil {
			podNode, err := fr.deps.NodeLister.Get(pod.Spec.NodeName)
			if err != nil {
				return NodeAvailabilityStatus{}, fmt.Errorf("failover[getNodeAvailabilityStatus]: failed to get node for pod %s/%s, error: %s", ns, name, err)
			}
			if podReadyCond.Status == corev1.ConditionFalse {
				nodeUnavailable = IsNodeReadyConditionFalseOrUnknown(podNode.Status)
			}
			if podReadyCond.Status == corev1.ConditionTrue {
				roDiskFound = IsNodeRODiskFoundConditionTrue(podNode.Status)
			}
		}
	}
	klog.Infof("failover[getNodeAvailabilityStatus]: nodeUnavailable=%t, roDiskFound=%t for %s of failure pod %s/%s", nodeUnavailable, roDiskFound, pod.Spec.NodeName, ns, pod.Name)
	return NodeAvailabilityStatus{NodeUnavailable: nodeUnavailable, ReadOnlyDiskFound: roDiskFound}, nil
}

// restartPodForHostDown restarts pod of a failure store or member if HostDown is marked true
func (fr *commonStatefulFailureRecovery) restartPodForHostDown(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	memberType := fr.failureObjectAccess.GetMemberType()

	for objectId := range fr.failureObjectAccess.GetFailureObjects(tc) {
		if fr.failureObjectAccess.IsFailing(tc, objectId) && fr.failureObjectAccess.IsHostDown(tc, objectId) {
			pod, err := fr.deps.PodLister.Pods(ns).Get(fr.failureObjectAccess.GetPodName(tc, objectId))
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("%s failover [restartPodForHostDown]: failed to get pod %s for tc %s/%s, error: %s", memberType, fr.failureObjectAccess.GetPodName(tc, objectId), ns, tcName, err)
			}
			if pod != nil {
				// If the failed pod has already been restarted once, its CreationTimestamp will be after FailureMember.CreatedAt
				if fr.failureObjectAccess.GetCreatedAt(tc, objectId).After(pod.CreationTimestamp.Time) {
					// Use force option to delete the pod
					if err = fr.deps.PodControl.ForceDeletePod(tc, pod); err != nil {
						return err
					}
					msg := fmt.Sprintf("Failed %s pod %s/%s is force deleted for recovery", memberType, ns, fr.failureObjectAccess.GetPodName(tc, objectId))
					klog.Infof(msg)
					return controller.IgnoreErrorf(msg)
				}
			}
		}
	}
	return nil
}

// canDoCleanUpNow checks if it is ok to do the cleanup of the failure store or member and its pod and pvcs now
func (fr *commonStatefulFailureRecovery) canDoCleanUpNow(tc *v1alpha1.TidbCluster, objectId string) bool {
	// If HostDown is set for the pod of the failure store or member and the pod was restarted after that, then give some time gap
	// (for it to come up and be ready again) before deleting the failure store or member
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	failurePodName := fr.failureObjectAccess.GetPodName(tc, objectId)
	if fr.failureObjectAccess.IsHostDown(tc, objectId) {
		pod, err := fr.deps.PodLister.Pods(ns).Get(failurePodName)
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorf("%s failover[canDoCleanUpNow]: failed to get pod %s for tc %s/%s, error: %s", fr.failureObjectAccess.GetMemberType(), failurePodName, ns, tcName, err)
			return false
		}
		if pod == nil || fr.failureObjectAccess.GetCreatedAt(tc, objectId).After(pod.CreationTimestamp.Time) || pod.CreationTimestamp.Add(restartToDeleteStoreGap).After(time.Now()) {
			return false
		}
	}
	return true
}

// deletePodAndPvcs deletes the pod and the pvcs of the failure store or member
func (fr *commonStatefulFailureRecovery) deletePodAndPvcs(tc *v1alpha1.TidbCluster, objectId string) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	memberType := fr.failureObjectAccess.GetMemberType()
	failurePodName := fr.failureObjectAccess.GetPodName(tc, objectId)
	pod, pvcs, err := fr.getPodAndPvcs(tc, failurePodName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("%s failover[deletePodAndPvcs]: failed to get pod %s for tc %s/%s, error: %s", memberType, failurePodName, ns, tcName, err)
	}
	if pod == nil {
		klog.Infof("%s failover[deletePodAndPvcs]: failure pod %s/%s not found, skip", memberType, ns, failurePodName)
		return nil
	}
	if pod.DeletionTimestamp == nil {
		// In case of K8s node failure, it is expected that scheduling be disabled on the K8s node by cordoning it.
		// Or else, after restart the pod would use the old PVC and then clean up of pvc will not happen.
		// The Scheduled condition of pod if true can confirm that the K8s node is not cordoned.
		podScheduled := isPodConditionScheduledTrue(pod.Status.Conditions)
		klog.Infof("%s failover[deletePodAndPvcs]: Scheduled condition of pod %s of tc %s/%s: %t", memberType, failurePodName, ns, tcName, podScheduled)
		if deleteErr := fr.deps.PodControl.DeletePod(tc, pod); deleteErr != nil {
			return deleteErr
		}
	} else {
		klog.Infof("pod %s/%s has DeletionTimestamp set to %s", ns, pod.Name, pod.DeletionTimestamp)
	}

	pvcUIDSet := fr.failureObjectAccess.GetPvcUIDSet(tc, objectId)
	pvcUids := make([]types.UID, 0, len(pvcs))
	for p := range pvcs {
		pvcUids = append(pvcUids, pvcs[p].ObjectMeta.UID)
	}
	klog.Infof("%s failover[deletePodAndPvcs]: PVCs used in cluster %s/%s: %s", memberType, ns, tcName, pvcUids)
	for p := range pvcs {
		pvc := pvcs[p]
		if _, pvcUIDExist := pvcUIDSet[pvc.ObjectMeta.UID]; pvcUIDExist {
			if pvc.DeletionTimestamp == nil /*&& pvcUIDExist*/ {
				if deleteErr := fr.deps.PVCControl.DeletePVC(tc, pvc); deleteErr != nil {
					klog.Errorf("%s failover[deletePodAndPvcs]: failed to delete PVC: %s/%s, error: %s", memberType, ns, pvc.Name, deleteErr)
					return deleteErr
				}
				klog.Infof("%s failover[deletePodAndPvcs]: delete PVC %s/%s successfully", memberType, ns, pvc.Name)
			} else {
				klog.Infof("pvc %s/%s has DeletionTimestamp set to %s", ns, pvc.Name, pvc.DeletionTimestamp)
			}
		}
	}
	return nil
}

// getPodAndPvcs returns the pod and pvcs of a component pod in a Tidb cluster
func (fr *commonStatefulFailureRecovery) getPodAndPvcs(tc *v1alpha1.TidbCluster, podName string) (*corev1.Pod, []*corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	pod, err := fr.deps.PodLister.Pods(ns).Get(podName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s failover: failed to get pod %s for tc %s/%s, error: %s", fr.failureObjectAccess.GetMemberType(), podName, ns, tcName, err)
	}
	pvcs, err := fr.getPodPvcs(tc, podName)
	if err != nil {
		return pod, nil, err
	}
	return pod, pvcs, nil
}

// getPodPvcs returns the pvcs of a component pod in a Tidb cluster
func (fr *commonStatefulFailureRecovery) getPodPvcs(tc *v1alpha1.TidbCluster, podName string) ([]*corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	ordinal, err := util.GetOrdinalFromPodName(podName)
	memberType := fr.failureObjectAccess.GetMemberType()
	if err != nil {
		return nil, fmt.Errorf("%s failover: failed to parse ordinal from pod name for %s/%s, error: %s", memberType, ns, podName, err)
	}
	pvcSelector, err := GetPVCSelectorForPod(tc, memberType, ordinal)
	if err != nil {
		return nil, fmt.Errorf("%s failover: failed to get PVC selector for pod %s/%s, error: %s", memberType, ns, podName, err)
	}
	pvcs, err := fr.deps.PVCLister.PersistentVolumeClaims(ns).List(pvcSelector)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("%s failover: failed to get PVCs for pod %s/%s, error: %s", memberType, ns, podName, err)
	}
	return pvcs, nil
}

// canAutoFailureRecovery checks whether auto recovery of failure pods and pvc should be done
func canAutoFailureRecovery(tc *v1alpha1.TidbCluster) bool {
	// TODO Use a boolean in TidbCluster spec instead of annotation for this
	return tc.Annotations[annAutoFailureRecovery] == strconv.FormatBool(true)
}

// isPodConditionScheduledTrue returns true if "PodScheduled" condition of pod is True
func isPodConditionScheduledTrue(conditions []corev1.PodCondition) bool {
	podSchCond := getPodConditionFromList(conditions, corev1.PodScheduled)
	return podSchCond != nil && podSchCond.Status == corev1.ConditionTrue
}

// getPodConditionFromList extracts the provided condition from the given list of condition and returns that.
// Returns nil if the condition is not present.
func getPodConditionFromList(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) *corev1.PodCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
