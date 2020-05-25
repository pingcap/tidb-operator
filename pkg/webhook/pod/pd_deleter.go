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

package pod

import (
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	pdutil "github.com/pingcap/tidb-operator/pkg/manager/member"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func (pc *PodAdmissionControl) admitDeletePdPods(payload *admitPayload) *admission.AdmissionResponse {

	name := payload.pod.Name
	namespace := payload.pod.Namespace
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}

	// If the pd pod is deleted by restarter, it is necessary to check former pd restart status
	if _, exist := payload.pod.Annotations[label.AnnPodDeferDeleting]; exist {
		existed, err := checkFormerPodRestartStatus(pc.kubeCli, v1alpha1.PDMemberType, payload, ordinal)
		if err != nil {
			return util.ARFail(err)
		}
		if existed {
			return &admission.AdmissionResponse{
				Allowed: false,
			}
		}
	}

	isInOrdinal, err := operatorUtils.IsPodOrdinalNotExceedReplicas(payload.pod, payload.ownerStatefulSet)
	if err != nil {
		return util.ARFail(err)
	}
	tcName := payload.tc.Name
	isUpgrading := operatorUtils.IsStatefulSetUpgrading(payload.ownerStatefulSet)
	IsDeferDeleting := IsPodWithPDDeferDeletingAnnotations(payload.pod)

	isMember, err := IsPodInPdMembers(payload.tc, payload.pod, payload.pdClient)
	if err != nil {
		return util.ARFail(err)
	}

	isLeader, err := isPDLeader(payload.pdClient, payload.pod)
	if err != nil {
		return util.ARFail(err)
	}

	klog.Infof("receive delete pd pod[%s/%s] of tc[%s/%s],isMember=%v,isInOrdinal=%v,isUpgrading=%v,isDeferDeleting=%v,isLeader=%v", namespace, name, namespace, tcName, isMember, isInOrdinal, isUpgrading, IsDeferDeleting, isLeader)

	// NotMember represents this pod is deleted from pd cluster or haven't register to pd cluster yet.
	// We should ensure this pd pod wouldn't be pd member any more.
	if !isMember {
		return pc.admitDeleteNonPDMemberPod(payload)
	}

	// NotInOrdinal represents this is an scale-in operation, we need to delete this member in pd cluster
	if !isInOrdinal {
		return pc.admitDeleteExceedReplicasPDPod(payload, isLeader)
	}

	// If there is an pd pod deleting operation during upgrading, we should
	// check the pd pods which have been upgraded before were all health
	if isUpgrading {
		klog.Infof("receive delete pd pod[%s/%s] of tc[%s/%s] is upgrading, make sure former pd upgraded status was health", namespace, name, namespace, tcName)
		err = checkFormerPDPodStatus(pc.kubeCli, payload.pdClient, payload.tc, payload.ownerStatefulSet, ordinal)
		if err != nil {
			return util.ARFail(err)
		}
	}

	if isLeader {
		return pc.transferPDLeader(payload)
	}

	if isUpgrading {
		pc.recorder.Event(payload.tc, corev1.EventTypeNormal, pdUpgradeReason, podDeleteEventMessage(name))
	}

	klog.Infof("pod[%s/%s] is not pd-leader,admit to delete", namespace, name)
	return util.ARSuccess()
}

// this pod is not a member of pd cluster currently, it could be deleted from pd cluster already or haven't registered in pd cluster
// we need to check whether this pd pod has been ensured wouldn't be a member in pd cluster
func (pc *PodAdmissionControl) admitDeleteNonPDMemberPod(payload *admitPayload) *admission.AdmissionResponse {

	name := payload.pod.Name
	namespace := payload.pod.Namespace
	isInOrdinal, err := operatorUtils.IsPodOrdinalNotExceedReplicas(payload.pod, payload.ownerStatefulSet)
	if err != nil {
		return util.ARFail(err)
	}
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}
	tcName := payload.tc.Name
	IsDeferDeleting := IsPodWithPDDeferDeletingAnnotations(payload.pod)

	// check whether this pod has been ensured wouldn't be a member in pd cluster
	if IsDeferDeleting {
		// when pd scale in, we should delete member first and finally edit its pvc and admit to delete pod.
		// If we edit pvc first and finally delete pd member during scale in,
		// it can be error that pd scale in From 4 to 3,the pvc was edited successfully and
		// the pd member was fail to deleted. If the pd scale was recover to 4 at that time,
		// it would be existed an pd-3 instance with its deferDeleting label Annotations PVC.
		// And the pvc can be deleted during upgrading if we use create pod webhook in future.
		if !isInOrdinal {
			pvcName := operatorUtils.OrdinalPVCName(v1alpha1.PDMemberType, payload.ownerStatefulSet.Name, ordinal)
			pvc, err := pc.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, meta.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					pc.recorder.Event(payload.tc, corev1.EventTypeNormal, pdScaleInReason, podDeleteEventMessage(name))
					return util.ARSuccess()
				}
				return util.ARFail(err)
			}
			err = addDeferDeletingToPVC(pvc, pc.kubeCli, payload.tc)
			if err != nil {
				klog.Infof("tc[%s/%s]'s pod[%s/%s] failed to update pvc,%v", namespace, tcName, namespace, name, err)
				return util.ARFail(err)
			}
			pc.recorder.Event(payload.tc, corev1.EventTypeNormal, pdScaleInReason, podDeleteEventMessage(name))
		}
		klog.Infof("pd pod[%s/%s] is not member of tc[%s/%s],admit to delete", namespace, name, namespace, tcName)
		return util.ARSuccess()
	}
	err = payload.pdClient.DeleteMember(name)
	if err != nil {
		return util.ARFail(err)
	}
	err = addDeferDeletingToPDPod(pc.kubeCli, payload.pod)
	if err != nil {
		return util.ARFail(err)
	}

	// make sure this pd pod won't be a member of pd cluster any more
	return &admission.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteExceedReplicasPDPod(payload *admitPayload, isPdLeader bool) *admission.AdmissionResponse {

	name := payload.pod.Name
	namespace := payload.pod.Namespace
	tcName := payload.tc.Name

	if isPdLeader {
		return pc.transferPDLeader(payload)
	}

	err := payload.pdClient.DeleteMember(name)
	if err != nil {
		return util.ARFail(err)
	}
	// we should add deferDeleting Annotation when we delete member successfully.
	err = addDeferDeletingToPDPod(pc.kubeCli, payload.pod)
	if err != nil {
		return util.ARFail(err)
	}

	klog.Infof("tc[%s/%s]'s pd[%s/%s] is being deleted from pd-cluster,refuse to delete it.", namespace, tcName, namespace, name)
	return &admission.AdmissionResponse{
		Allowed: false,
	}
}

// this pod is a pd leader, we should transfer pd leader to other pd pod before it gets deleted before.
func (pc *PodAdmissionControl) transferPDLeader(payload *admitPayload) *admission.AdmissionResponse {

	name := payload.pod.Name
	namespace := payload.pod.Namespace
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}
	tcName := payload.tc.Name
	var targetName string

	lastOrdinal := helper.GetMaxPodOrdinal(*payload.ownerStatefulSet.Spec.Replicas, payload.ownerStatefulSet)
	if ordinal == lastOrdinal {
		targetName = pdutil.PdPodName(tcName, helper.GetMinPodOrdinal(*payload.ownerStatefulSet.Spec.Replicas, payload.ownerStatefulSet))
	} else {
		targetName = pdutil.PdPodName(tcName, lastOrdinal)
	}

	err = payload.pdClient.TransferPDLeader(targetName)
	if err != nil {
		klog.Errorf("tc[%s/%s] failed to transfer pd leader to pod[%s/%s],%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}
	klog.Infof("tc[%s/%s] start to transfer pd leader to pod[%s/%s],refuse to delete pod[%s/%s]", namespace, tcName, namespace, targetName, namespace, name)
	return &admission.AdmissionResponse{
		Allowed: false,
	}
}
