// Copyright 2020 PingCAP, Inc.
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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/label"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

type masterScaler struct {
	generalScaler
	masterControl dmapi.MasterControlInterface
}

// NewMasterScaler returns a DMScaler
func NewMasterScaler(masterControl dmapi.MasterControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface) Scaler {
	return &masterScaler{
		generalScaler: generalScaler{
			pvcLister:  pvcLister,
			pvcControl: pvcControl,
		},
		masterControl: masterControl,
	}
}

func (msd *masterScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return msd.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return msd.ScaleIn(meta, oldSet, newSet)
	}
	return msd.SyncAutoScalerAnn(meta, oldSet)
}

func (msd *masterScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	dc, ok := meta.(*v1alpha1.DMCluster)
	if !ok {
		return nil
	}

	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	klog.Infof("scaling out dm-master statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	_, err := msd.deleteDeferDeletingPVC(dc, oldSet.GetName(), v1alpha1.DMMasterMemberType, ordinal)
	if err != nil {
		return err
	}

	if !dc.Status.Master.Synced {
		return fmt.Errorf("DMCluster: %s/%s's dm-master status sync failed, can't scale out now", ns, dcName)
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// We need remove member from cluster before reducing statefulset replicas
// only remove one member at a time when scale down
func (msd *masterScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	dc, ok := meta.(*v1alpha1.DMCluster)
	if !ok {
		return nil
	}

	ns := dc.GetNamespace()
	dcName := dc.GetName()
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	memberName := ordinalPodName(v1alpha1.DMMasterMemberType, dcName, ordinal)
	setName := oldSet.GetName()

	if !dc.Status.Master.Synced {
		return fmt.Errorf("DMCluster: %s/%s's dm-master status sync failed, can't scale in now", ns, dcName)
	}

	klog.Infof("scaling in dm-master statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())

	//if controller.PodWebhookEnabled {
	//	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	//	return nil
	//}

	// If the dm-master pod was dm-master leader during scale-in, we would evict dm-master leader first
	// If the dm-master statefulSet would be scale-in to zero and the dm-master-0 was going to be deleted,
	// we would directly deleted the dm-master-0 without dm-master leader evict
	if ordinal > 0 {
		if dc.Status.Master.Leader.Name == memberName {
			masterPeerClient := controller.GetMasterPeerClient(msd.masterControl, dc, memberName)
			err := masterPeerClient.EvictLeader()
			if err != nil {
				return err
			}
			return controller.RequeueErrorf("dc [%s/%s]'s dm-master pod[%s/%s] is transferring dm-master leader, can't scale-in now", ns, dcName, ns, memberName)
		}
	}

	masterClient := controller.GetMasterClient(msd.masterControl, dc)
	err := masterClient.DeleteMaster(memberName)
	if err != nil {
		klog.Errorf("dm-master scale in: failed to delete member %s, %v", memberName, err)
		return err
	}
	klog.Infof("dm-master scale in: delete member %s successfully", memberName)

	// double check whether member deleted after delete member
	mastersInfo, err := masterClient.GetMasters()
	if err != nil {
		klog.Errorf("dm-master scale in: failed to get dm-masters %s, %v", memberName, err)
		return err
	}

	existed := false
	for _, member := range mastersInfo {
		if member.Name == memberName {
			existed = true
			break
		}
	}
	if existed {
		err = fmt.Errorf("dm-master scale in: dm-master %s still exist after being deleted", memberName)
		klog.Error(err)
		return err
	}

	pvcName := ordinalPVCName(v1alpha1.DMMasterMemberType, setName, ordinal)
	pvc, err := msd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return fmt.Errorf("dm-master.ScaleIn: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, dcName, err)
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now

	_, err = msd.pvcControl.UpdatePVC(dc, pvc)
	if err != nil {
		klog.Errorf("dm-master scale in: failed to set pvc %s/%s annotation: %s to %s",
			ns, pvcName, label.AnnPVCDeferDeleting, now)
		return err
	}
	klog.Infof("dm-master scale in: set pvc %s/%s annotation: %s to %s",
		ns, pvcName, label.AnnPVCDeferDeleting, now)

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func (msd *masterScaler) SyncAutoScalerAnn(meta metav1.Object, oldSet *apps.StatefulSet) error {
	return nil
}
