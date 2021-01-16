// Copyright 2021 PingCAP, Inc.
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
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func ComponentScale(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return ComponentScaleOut(context, oldSet, newSet)
	} else if scaling < 0 {
		return ComponentScaleIn(context, oldSet, newSet)
	}
	return ComponentSyncAutoScalerAnn(context, oldSet)
}

func ComponentScaleOut(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc := context.tc

	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Infof("scaling out pd statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	_, err := ComponentDeleteDeferDeletingPVC(context, oldSet.GetName(), ordinal)
	if err != nil {
		return err
	}

	if !tc.Status.PD.Synced {
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed, can't scale out now", ns, tcName)
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
func ComponentScaleIn(context *ComponentContext, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	// context deserialization
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	memberName := PdName(tcName, ordinal, tc.Namespace, tc.Spec.ClusterDomain)
	pdPodName := PdPodName(tcName, ordinal)
	setName := oldSet.GetName()

	if !tc.Status.PD.Synced {
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed, can't scale in now", ns, tcName)
	}

	klog.Infof("scaling in pd statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())

	if dependencies.CLIConfig.PodWebhookEnabled {
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}

	// limit scale in when multi-cluster is enabled
	if pass := ComponentPreCheckUpMembers(context, pdPodName); !pass {
		return nil
	}

	pdClient := controller.GetPDClient(dependencies.PDControl, tc)
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
	pvc, err := dependencies.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return fmt.Errorf("pdScaler.ScaleIn: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now

	_, err = dependencies.PVCControl.UpdatePVC(tc, pvc)
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

func ComponentSyncAutoScalerAnn(context *ComponentContext, actual *apps.StatefulSet) error {
	return nil
}

func ComponentPreCheckUpMembers(context *ComponentContext, podName string) bool {
	// context deserialization
	tc := context.tc
	dependencies := context.dependencies

	upComponents := 0

	upComponents += len(tc.Status.TiKV.Stores) + len(tc.Status.TiFlash.Stores) + len(tc.Status.TiDB.Members)

	if tc.Status.TiCDC.StatefulSet != nil {
		upComponents += int(tc.Status.TiCDC.StatefulSet.Replicas)
	}

	if tc.Status.Pump.StatefulSet != nil {
		upComponents += int(tc.Status.Pump.StatefulSet.Replicas)
	}

	if upComponents != 0 && tc.Spec.PD.Replicas == 0 {
		errMsg := fmt.Sprintf("The PD is in use by TidbCluster [%s/%s], can't scale in PD, podname %s", tc.GetNamespace(), tc.GetName(), podName)
		klog.Error(errMsg)
		dependencies.Recorder.Event(tc, v1.EventTypeWarning, "FailedScaleIn", errMsg)
		return false
	}

	return true
}

func ComponentDeleteDeferDeletingPVC(context *ComponentContext, setName string, ordinal int32) (map[string]string, error) {
	tc := context.tc
	dependencies := context.dependencies
	memberType := v1alpha1.PDMemberType	
	
	ns := tc.GetNamespace()
	// for unit test
	skipReason := map[string]string{}
	var podName, kind string
	var l label.Label
	podName = ordinalPodName(memberType, tc.GetName(), ordinal)
	l = label.New().Instance(tc.GetName())
	l[label.AnnPodNameKey] = podName
	kind = v1alpha1.TiDBClusterKind
	selector, err := l.Selector()
	if err != nil {
		return skipReason, fmt.Errorf("%s %s/%s assemble label selector failed, err: %v", kind, ns, tc.GetName(), err)
	}

	pvcs, err := dependencies.PVCLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		msg := fmt.Sprintf("%s %s/%s list pvc failed, selector: %s, err: %v", kind, ns, tc.GetName(), selector, err)
		klog.Error(msg)
		return skipReason, fmt.Errorf(msg)
	}
	if len(pvcs) == 0 {
		klog.Infof("%s %s/%s list pvc not found, selector: %s", kind, ns, tc.GetName(), selector)
		skipReason[podName] = skipReasonScalerPVCNotFound
		return skipReason, nil
	}

	for _, pvc := range pvcs {
		pvcName := pvc.Name
		if pvc.Annotations == nil {
			skipReason[pvcName] = skipReasonScalerAnnIsNil
			continue
		}
		if _, ok := pvc.Annotations[label.AnnPVCDeferDeleting]; !ok {
			skipReason[pvcName] = skipReasonScalerAnnDeferDeletingIsEmpty
			continue
		}

		err = dependencies.PVCControl.DeletePVC(tc, pvc)
		if err != nil {
			klog.Errorf("Scale out: failed to delete pvc %s/%s, %v", ns, pvcName, err)
			return skipReason, err
		}
		klog.Infof("Scale out: delete pvc %s/%s successfully", ns, pvcName)
	}
	return skipReason, nil
}

func ComponentUpdateDeferDeletingPVC(context *ComponentContext, ordinal int32) error {
	tc := context.tc
	dependencies := context.dependencies
	memberType := v1alpha1.PDMemberType

	ns := tc.GetNamespace()
	podName := ordinalPodName(memberType, tc.Name, ordinal)

	l := label.New().Instance(tc.GetInstanceName())
	l[label.AnnPodNameKey] = podName
	selector, err := l.Selector()
	if err != nil {
		return fmt.Errorf("cluster %s/%s assemble label selector failed, err: %v", ns, tc.Name, err)
	}

	pvcs, err := dependencies.PVCLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		msg := fmt.Sprintf("Cluster %s/%s list pvc failed, selector: %s, err: %v", ns, tc.Name, selector, err)
		klog.Error(msg)
		return fmt.Errorf(msg)
	}
	if len(pvcs) == 0 {
		msg := fmt.Sprintf("Cluster %s/%s list pvc not found, selector: %s", ns, tc.Name, selector)
		klog.Error(msg)
		return fmt.Errorf(msg)
	}

	for _, pvc := range pvcs {
		pvcName := pvc.Name
		if pvc.Annotations == nil {
			pvc.Annotations = map[string]string{}
		}
		now := time.Now().Format(time.RFC3339)
		pvc.Annotations[label.AnnPVCDeferDeleting] = now
		_, err = dependencies.PVCControl.UpdatePVC(tc, pvc)
		if err != nil {
			klog.Errorf("Scale in: failed to set pvc %s/%s annotation: %s to %s, error: %v",
				ns, pvcName, label.AnnPVCDeferDeleting, now, err)
			return err
		}
		klog.Infof("Scale in: set pvc %s/%s annotation: %s to %s",
			ns, pvcName, label.AnnPVCDeferDeleting, now)
	}
	return nil
}
