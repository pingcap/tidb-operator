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
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

// TODO add e2e test specs

type pdScaler struct {
	generalScaler
	pdControl pdapi.PDControlInterface
}

// NewPDScaler returns a Scaler
func NewPDScaler(pdControl pdapi.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface) Scaler {
	return &pdScaler{generalScaler{pvcLister, pvcControl}, pdControl}
}

func (psd *pdScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return psd.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return psd.ScaleIn(meta, oldSet, newSet)
	}
	return psd.SyncAutoScalerAnn(meta, oldSet)
}

func (psd *pdScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return nil
	}

	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Infof("scaling out pd statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	_, err := psd.deleteDeferDeletingPVC(tc, oldSet.GetName(), v1alpha1.PDMemberType, ordinal)
	if err != nil {
		return err
	}

	if !tc.Status.PD.Synced {
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed,can't scale out now", ns, tcName)
	}

	if len(tc.Status.PD.FailureMembers) != 0 {
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}

	healthCount := 0
	totalCount := *oldSet.Spec.Replicas
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _, i := range podOrdinals {
		podName := ordinalPodName(v1alpha1.PDMemberType, tcName, i)
		if member, ok := tc.Status.PD.Members[podName]; ok && member.Health {
			healthCount++
		}
	}
	if healthCount < int(totalCount) {
		return fmt.Errorf("TidbCluster: %s/%s's pd %d/%d is ready, can't scale out now",
			ns, tcName, healthCount, totalCount)
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// We need remove member from cluster before reducing statefulset replicas
// only remove one member at a time when scale down
func (psd *pdScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	memberName := PdName(tcName, ordinal, tc.Namespace, tc.Spec.ClusterDomain)
	pdPodName := PdPodName(tcName, ordinal)
	setName := oldSet.GetName()

	if !tc.Status.PD.Synced {
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed,can't scale in now", ns, tcName)
	}

	klog.Infof("scaling in pd statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())

	if controller.PodWebhookEnabled {
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}

	pdClient := controller.GetPDClient(psd.pdControl, tc)
	leader, err := pdClient.GetPDLeader()
	if err != nil {
		return err
	}
	// If the pd pod was pd leader during scale-in, we would transfer pd leader to pd-0 directly
	// If the pd statefulSet would be scale-in to zero and the pd-0 was going to be deleted,
	// we would directly deleted the pd-0 without pd leader transferring
	if leader.Name == memberName || leader.Name == pdPodName {
		if ordinal > 0 {
			targetPdName := PdName(tcName, 0, tc.Namespace, tc.Spec.ClusterDomain)
			if _, exist := tc.Status.PD.Members[targetPdName]; exist{
				err = pdClient.TransferPDLeader(targetPdName)
			} else {
				err = pdClient.TransferPDLeader(PdPodName(tcName, 0))
			}
			if err != nil {
				return err
			}
			return controller.RequeueErrorf("tc[%s/%s]'s pd pod[%s/%s] is transferring pd leader,can't scale-in now", ns, tcName, ns, memberName)
		}
		for _, member := range tc.Status.PD.PeerMembers {
			if member.Health && member.Name != memberName {
				err = pdClient.TransferPDLeader(member.Name)
				if err != nil {
					return err
				}
				return controller.RequeueErrorf("tc[%s/%s]'s pd pod[%s/%s] is transferring pd leader,can't scale-in now", ns, tcName, ns, memberName)
			}
		}
	}

	err = pdClient.DeleteMember(memberName)
	if err != nil {
		klog.Errorf("pd scale in: failed to delete member %s, %v", memberName, err)
		return err
	}
	klog.Infof("pd scale in: delete member %s successfully", memberName)

	// double check whether member deleted after delete member
	// The PD member could be remained after deleting API success, see https://github.com/pingcap/pd/issues/2541
	// The bug still need to be investigated.
	membersInfo, err := pdClient.GetMembers()
	if err != nil {
		klog.Errorf("pd scale in: failed to get members %s, %v", memberName, err)
		return err
	}

	existed := false
	for _, member := range membersInfo.Members {
		if member.Name == memberName {
			existed = true
			break
		}
	}
	if existed {
		err = fmt.Errorf("pd scale in: member %s still exist after being deleted", memberName)
		klog.Error(err)
		return err
	}

	pvcName := ordinalPVCName(v1alpha1.PDMemberType, setName, ordinal)
	pvc, err := psd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return fmt.Errorf("pdScaler.ScaleIn: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now

	_, err = psd.pvcControl.UpdatePVC(tc, pvc)
	if err != nil {
		klog.Errorf("pd scale in: failed to set pvc %s/%s annotation: %s to %s",
			ns, pvcName, label.AnnPVCDeferDeleting, now)
		return err
	}
	klog.Infof("pd scale in: set pvc %s/%s annotation: %s to %s",
		ns, pvcName, label.AnnPVCDeferDeleting, now)

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func (psd *pdScaler) SyncAutoScalerAnn(meta metav1.Object, actual *apps.StatefulSet) error {
	return nil
}

type fakePDScaler struct{}

// NewFakePDScaler returns a fake Scaler
func NewFakePDScaler() Scaler {
	return &fakePDScaler{}
}

func (fsd *fakePDScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return fsd.ScaleOut(meta, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return fsd.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

func (fsd *fakePDScaler) ScaleOut(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (fsd *fakePDScaler) ScaleIn(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}

func (fsd *fakePDScaler) SyncAutoScalerAnn(tc metav1.Object, actual *apps.StatefulSet) error {
	return nil
}
