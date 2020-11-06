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
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// TODO add e2e test specs

type pdScaler struct {
	generalScaler
}

// NewPDScaler returns a Scaler
func NewPDScaler(deps *controller.Dependencies) Scaler {
	return &pdScaler{generalScaler: generalScaler{deps: deps}}
}

func (s *pdScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return s.SyncAutoScalerAnn(meta, oldSet)
}

func (s *pdScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return nil
	}

	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Infof("scaling out pd statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	_, err := s.deleteDeferDeletingPVC(tc, oldSet.GetName(), v1alpha1.PDMemberType, ordinal)
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
		targetPdName := PdName(tcName, i, tc.Namespace, tc.Spec.ClusterDomain)
		if member, ok := tc.Status.PD.Members[targetPdName]; ok && member.Health {
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
func (s *pdScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
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

	if s.deps.CLIConfig.PodWebhookEnabled {
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}

	pdClient := controller.GetPDClient(s.deps.PDControl, tc)
	leader, err := pdClient.GetPDLeader()
	if err != nil {
		return err
	}
	// If the PD pod was PD leader during scale-in, we would transfer PD leader first
	// If the PD StatefulSet would be scale-in to zero and no other members in the PD cluster,
	// we would directly delete the member without the leader transferring
	if leader.Name == memberName || leader.Name == pdPodName {
		if *newSet.Spec.Replicas > 1 {
			minOrdinal := helper.GetMinPodOrdinal(*newSet.Spec.Replicas, newSet)
			targetOrdinal := helper.GetMaxPodOrdinal(*newSet.Spec.Replicas, newSet)
			if ordinal > minOrdinal {
				targetOrdinal = minOrdinal
			}
			targetPdName := PdName(tcName, targetOrdinal, tc.Namespace, tc.Spec.ClusterDomain)
			if _, exist := tc.Status.PD.Members[targetPdName]; exist {
				err = pdClient.TransferPDLeader(targetPdName)
			} else {
				err = pdClient.TransferPDLeader(PdPodName(tcName, targetOrdinal))
			}
			if err != nil {
				return err
			}
		} else {
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
	}

	err = pdClient.DeleteMember(memberName)
	if err != nil {
		klog.Errorf("pd scale in: failed to delete member %s, %v", memberName, err)
		return err
	}
	klog.Infof("pd scale in: delete member %s successfully", memberName)

	pvcName := ordinalPVCName(v1alpha1.PDMemberType, setName, ordinal)
	pvc, err := s.deps.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return fmt.Errorf("pdScaler.ScaleIn: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now

	_, err = s.deps.PVCControl.UpdatePVC(tc, pvc)
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

func (s *pdScaler) SyncAutoScalerAnn(meta metav1.Object, actual *apps.StatefulSet) error {
	return nil
}

type fakePDScaler struct{}

// NewFakePDScaler returns a fake Scaler
func NewFakePDScaler() Scaler {
	return &fakePDScaler{}
}

func (s *fakePDScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

func (s *fakePDScaler) ScaleOut(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (s *fakePDScaler) ScaleIn(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}

func (s *fakePDScaler) SyncAutoScalerAnn(tc metav1.Object, actual *apps.StatefulSet) error {
	return nil
}
