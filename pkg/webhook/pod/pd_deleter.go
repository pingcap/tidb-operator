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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	pdutil "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1"
	"k8s.io/klog"
)

func (pc *PodAdmissionControl) admitDeletePdPods(payload *admitPayload) *admission.AdmissionResponse {

	pod, ownerStatefulSet, ordinal, name, namespace, tcName, isInOrdinal, isUpgrading, IsDeferDeleting, pdClient, err := fetchInfoFromPayload(payload)
	if err != nil {
		return util.ARFail(err)
	}

	isMember, err := IsPodInPdMembers(payload.tc, pod, pdClient)
	if err != nil {
		return util.ARFail(err)
	}

	isLeader, err := isPDLeader(pdClient, pod)
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
		err = checkFormerPDPodStatus(pc.podLister, pdClient, payload.tc, namespace, ordinal, *ownerStatefulSet.Spec.Replicas)
		if err != nil {
			return util.ARFail(err)
		}
	}

	if isLeader {
		return pc.transferPDLeader(payload)
	}

	klog.Infof("pod[%s/%s] is not pd-leader,admit to delete", namespace, name)
	return util.ARSuccess()
}

// this pod is not a member of pd cluster currently, it could be deleted from pd cluster already or haven't registered in pd cluster
// we need to check whether this pd pod has been ensured wouldn't be a member in pd cluster
func (pc *PodAdmissionControl) admitDeleteNonPDMemberPod(payload *admitPayload) *admission.AdmissionResponse {

	pod, ownerStatefulSet, ordinal, name, namespace, tcName, isInOrdinal, _, IsDeferDeleting, pdClient, err := fetchInfoFromPayload(payload)
	if err != nil {
		return util.ARFail(err)
	}

	// check whether this pod has been ensured wouldn't be a member in pd cluster
	if IsDeferDeleting {
		// when pd scale in, we should delete member first and finally edit its pvc and admit to delete pod.
		// If we edit pvc first and finally delete pd member during scale in,
		// it can be error that pd scale in From 4 to 3,the pvc was edited successfully and
		// the pd member was fail to deleted. If the pd scale was recover to 4 at that time,
		// it would be existed an pd-3 instance with its deferDeleting label Annotations PVC.
		// And the pvc can be deleted during upgrading if we use create pod webhook in future.
		if !isInOrdinal {
			err := addDeferDeletingToPVC(v1alpha1.PDMemberType, pc, payload.tc, ownerStatefulSet.Name, namespace, ordinal)
			if err != nil {
				klog.Infof("tc[%s/%s]'s pod[%s/%s] failed to update pvc,%v", namespace, tcName, namespace, name, err)
				return util.ARFail(err)
			}
		}
		klog.Infof("pd pod[%s/%s] is not member of tc[%s/%s],admit to delete", namespace, name, namespace, tcName)
		return util.ARSuccess()

	}
	err = pdClient.DeleteMember(name)
	if err != nil {
		return util.ARFail(err)
	}
	err = addDeferDeletingToPDPod(pc, pod)
	if err != nil {
		return util.ARFail(err)
	}

	// make sure this pd pod won't be a member of pd cluster any more
	return &admission.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteExceedReplicasPDPod(payload *admitPayload, isPdLeader bool) *admission.AdmissionResponse {

	pod, _, _, name, namespace, tcName, _, _, _, pdClient, err := fetchInfoFromPayload(payload)
	if err != nil {
		return util.ARFail(err)
	}

	if isPdLeader {
		return pc.transferPDLeader(payload)
	}

	err = pdClient.DeleteMember(name)
	if err != nil {
		return util.ARFail(err)
	}
	// we should add deferDeleting Annotation when we delete member successfully.
	err = addDeferDeletingToPDPod(pc, pod)
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

	_, _, ordinal, name, namespace, tcName, _, _, _, pdClient, err := fetchInfoFromPayload(payload)
	if err != nil {
		return util.ARFail(err)
	}

	lastOrdinal := payload.tc.Status.PD.StatefulSet.Replicas - 1
	var targetName string
	if ordinal == lastOrdinal {
		targetName = pdutil.PdPodName(tcName, 0)
	} else {
		targetName = pdutil.PdPodName(tcName, lastOrdinal)
	}

	err = pdClient.TransferPDLeader(targetName)
	if err != nil {
		klog.Errorf("tc[%s/%s] failed to transfer pd leader to pod[%s/%s],%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}
	klog.Infof("tc[%s/%s] start to transfer pd leader to pod[%s/%s],refuse to delete pod[%s/%s]", namespace, tcName, namespace, targetName, namespace, name)
	return &admission.AdmissionResponse{
		Allowed: false,
	}
}
