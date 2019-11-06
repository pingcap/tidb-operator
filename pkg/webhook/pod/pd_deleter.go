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
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func (pc *PodAdmissionControl) admitDeletePdPods(pod *corev1.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name
	ordinal, err := operatorUtils.GetOrdinalFromPodName(pod.Name)
	if err != nil {
		return util.ARFail(err)
	}

	isMember, err := IsPodInPdMembers(tc, pod, pdClient)
	if err != nil {
		return util.ARFail(err)
	}

	isOutOfOrdinal, err := operatorUtils.IsPodOutOfOrdinal(pod, *ownerStatefulSet.Spec.Replicas)
	if err != nil {
		return util.ARFail(err)
	}

	isUpgrading := IsStatefulSetUpgrading(ownerStatefulSet)

	klog.Infof("receive delete pd pod[%s/%s] of tc[%s/%s],isMember=%v,isOutOfOrdinal=%v,isUpgrading=%v", namespace, name, namespace, tcName, isMember, isOutOfOrdinal, isUpgrading)

	// each pd instance must be member deleted from tc-cluster before deleting its pod.
	if !isMember {
		// when pd scale in,we should delete member first and finally edit its pvc and admit to delete pod.
		// If we edit pvc first and finally delete pd member during scale in,
		// it can be error that pd scale in From 4 to 3,the pvc was edited successfully and
		// the pd member was fail to deleted. If the pd scale was recover to 4 at that time,
		// it would be existed an pd-3 instance with its deferDeleting label Annotations PVC.
		// And the pvc can be deleted during upgrading if we use create pod webhook in future.
		if isOutOfOrdinal {
			err = addDeferDeletingToPVC(pc, tc, ownerStatefulSet.Name, namespace, ordinal)
			if err != nil {
				klog.Infof("tc[%s/%s]'s pod[%s/%s] failed to update pvc,%v", namespace, tcName, namespace, name, err)
				return util.ARFail(err)
			}
		}
		klog.Infof("pd pod[%s/%s] is not member of tc[%s/%s],admit to delete", namespace, name, namespace, tcName)
		return util.ARSuccess()
	}

	if isOutOfOrdinal {
		err = pdClient.DeleteMember(name)
		if err != nil {
			return util.ARFail(err)
		}
		klog.Infof("tc[%s/%s]'s pd[%s/%s] is being deleted from pd-cluster,refuse to delete it.", namespace, tcName, namespace, name)
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}

	if isUpgrading {
		klog.Infof("receive delete pd pod[%s/%s] of tc[%s/%s] is upgrading,make sure former pd upgraded status was health", namespace, name, namespace, tcName)
		err = checkFormerPDPodStatus(pc.kubeCli, pdClient, tc, namespace, ordinal, *ownerStatefulSet.Spec.Replicas)
		if err != nil {
			return util.ARFail(err)
		}
	}

	leader, err := pdClient.GetPDLeader()
	if err != nil {
		klog.Errorf("tc[%s/%s] fail to get pd leader %v,refuse to delete pod[%s/%s]", namespace, tc.Name, err, namespace, name)
		return util.ARFail(err)
	}

	klog.Infof("tc[%s/%s]'s pd leader is pod[%s/%s] during deleting pod[%s/%s]", namespace, tc.Name, namespace, leader.Name, namespace, name)
	if leader.Name == name {
		lastOrdinal := tc.Status.PD.StatefulSet.Replicas - 1
		var targetName string
		if ordinal == lastOrdinal {
			targetName = pdutil.PdPodName(tc.Name, 0)
		} else {
			targetName = pdutil.PdPodName(tc.Name, lastOrdinal)
		}
		err = pdClient.TransferPDLeader(targetName)
		if err != nil {
			klog.Errorf("tc[%s/%s] failed to transfer pd leader to pod[%s/%s],%v", namespace, tc.Name, namespace, name, err)
			return util.ARFail(err)
		}
		klog.Infof("tc[%s/%s] start to transfer pd leader to pod[%s/%s],refuse to delete pod[%s/%s]", namespace, tc.Name, namespace, targetName, namespace, name)
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}

	klog.Infof("pod[%s/%s] is not pd-leader,admit to delete", namespace, name)
	return util.ARSuccess()
}
