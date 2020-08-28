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
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

const (
	skipReasonScalerPVCNotFound             = "scaler: pvc is not found"
	skipReasonScalerAnnIsNil                = "scaler: pvc annotations is nil"
	skipReasonScalerAnnDeferDeletingIsEmpty = "scaler: pvc annotations defer deleting is empty"
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type Scaler interface {
	// Scale scales the cluster. It does nothing if scaling is not needed.
	Scale(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleOut scales out the cluster
	ScaleOut(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// SyncAutoScalerAnn would sync Ann created by AutoScaler
	SyncAutoScalerAnn(meta metav1.Object, actual *apps.StatefulSet) error
}

type generalScaler struct {
	pvcLister  corelisters.PersistentVolumeClaimLister
	pvcControl controller.PVCControlInterface
}

func (gs *generalScaler) deleteDeferDeletingPVC(controller runtime.Object,
	setName string, memberType v1alpha1.MemberType, ordinal int32) (map[string]string, error) {
	meta := controller.(metav1.Object)
	ns := meta.GetNamespace()
	// for unit test
	skipReason := map[string]string{}
	var podName, kind string
	var l label.Label
	switch controller.(type) {
	case *v1alpha1.TidbCluster:
		podName = ordinalPodName(memberType, meta.GetName(), ordinal)
		l = label.New().Instance(meta.GetName())
		l[label.AnnPodNameKey] = podName
		kind = v1alpha1.TiDBClusterKind
	case *v1alpha1.TiKVGroup:
		podName = fmt.Sprintf("%s-%s-group-%d", meta.GetName(), memberType, ordinal)
		l = label.NewGroup().Instance(meta.GetName())
		// TODO: support sync meta info into TiKVGroup resources (pod/pvc)
		kind = v1alpha1.TiKVGroupKind
	case *v1alpha1.DMCluster:
		podName = ordinalPodName(memberType, meta.GetName(), ordinal)
		l = label.New().Instance(meta.GetName())
		l[label.AnnPodNameKey] = podName
		kind = v1alpha1.DMClusterKind
	default:
		kind = controller.GetObjectKind().GroupVersionKind().Kind
		return nil, fmt.Errorf("%s[%s/%s] has unknown controller", kind, ns, meta.GetName())
	}
	selector, err := l.Selector()
	if err != nil {
		return skipReason, fmt.Errorf("%s %s/%s assemble label selector failed, err: %v", kind, ns, meta.GetName(), err)
	}

	pvcs, err := gs.pvcLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		msg := fmt.Sprintf("%s %s/%s list pvc failed, selector: %s, err: %v", kind, ns, meta.GetName(), selector, err)
		klog.Errorf(msg)
		return skipReason, fmt.Errorf(msg)
	}
	if len(pvcs) == 0 {
		klog.Infof("%s %s/%s list pvc not found, selector: %s", kind, ns, meta.GetName(), selector)
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

		err = gs.pvcControl.DeletePVC(controller, pvc)
		if err != nil {
			klog.Errorf("Scale out: failed to delete pvc %s/%s, %v", ns, pvcName, err)
			return skipReason, err
		}
		klog.Infof("Scale out: delete pvc %s/%s successfully", ns, pvcName)
	}
	return skipReason, nil
}

func (gs *generalScaler) updateDeferDeletingPVC(tc *v1alpha1.TidbCluster,
	memberType v1alpha1.MemberType, ordinal int32) error {
	ns := tc.GetNamespace()
	podName := ordinalPodName(memberType, tc.Name, ordinal)

	l := label.New().Instance(tc.GetInstanceName())
	l[label.AnnPodNameKey] = podName
	selector, err := l.Selector()
	if err != nil {
		return fmt.Errorf("cluster %s/%s assemble label selector failed, err: %v", ns, tc.Name, err)
	}

	pvcs, err := gs.pvcLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		msg := fmt.Sprintf("Cluster %s/%s list pvc failed, selector: %s, err: %v", ns, tc.Name, selector, err)
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}
	if len(pvcs) == 0 {
		msg := fmt.Sprintf("Cluster %s/%s list pvc not found, selector: %s", ns, tc.Name, selector)
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}

	for _, pvc := range pvcs {
		pvcName := pvc.Name
		if pvc.Annotations == nil {
			pvc.Annotations = map[string]string{}
		}
		now := time.Now().Format(time.RFC3339)
		pvc.Annotations[label.AnnPVCDeferDeleting] = now
		_, err = gs.pvcControl.UpdatePVC(tc, pvc)
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

func resetReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		helper.SetDeleteSlots(newSet, helper.GetDeleteSlots(oldSet))
	}
}

func setReplicasAndDeleteSlots(newSet *apps.StatefulSet, replicas int32, deleteSlots sets.Int32) {
	oldReplicas := *newSet.Spec.Replicas
	*newSet.Spec.Replicas = replicas
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		helper.SetDeleteSlots(newSet, deleteSlots)
		klog.Infof("scale statefulset: %s/%s replicas from %d to %d (delete slots: %v)",
			newSet.GetNamespace(), newSet.GetName(), oldReplicas, replicas, deleteSlots.List())
		return
	}
	klog.Infof("scale statefulset: %s/%s replicas from %d to %d",
		newSet.GetNamespace(), newSet.GetName(), oldReplicas, replicas)
}

func ordinalPVCName(memberType v1alpha1.MemberType, setName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", memberType, setName, ordinal)
}

func ordinalPodName(memberType v1alpha1.MemberType, tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", tcName, memberType, ordinal)
}

// scaleOne calculates desired replicas and delete slots from actual/desired
// stateful sets by allowing only one pod to be deleted or created
// it returns following values:
// - scaling:
//   - 0: no scaling required
//   - 1: scaling out
//   - -1: scaling in
// - ordinal: pod ordinal to create or delete
// - replicas/deleteSlots: desired replicas and deleteSlots by allowing only one pod to be deleted or created
func scaleOne(actual *apps.StatefulSet, desired *apps.StatefulSet) (scaling int, ordinal int32, replicas int32, deleteSlots sets.Int32) {
	actualPodOrdinals := helper.GetPodOrdinals(*actual.Spec.Replicas, actual)
	desiredPodOrdinals := helper.GetPodOrdinals(*desired.Spec.Replicas, desired)
	additions := desiredPodOrdinals.Difference(actualPodOrdinals)
	deletions := actualPodOrdinals.Difference(desiredPodOrdinals)
	scaling = 0
	ordinal = -1
	replicas = *actual.Spec.Replicas
	actualDeleteSlots := helper.GetDeleteSlots(actual)
	desiredDeleteSlots := helper.GetDeleteSlots(desired)
	// copy delete slots from desired delete slots if not in actual pod ordinals
	for i := range desiredDeleteSlots {
		if !actualPodOrdinals.Has(i) {
			actualDeleteSlots.Insert(i)
		}
	}
	if additions.Len() > 0 {
		// we always do scaling out before scaling in to maintain maximum avaiability
		scaling = 1
		ordinal = additions.List()[0]
		replicas += 1
		if !desiredDeleteSlots.Has(ordinal) {
			// not in desired delete slots, remove it from actual delete slots
			actualDeleteSlots.Delete(ordinal)
		}
		actualDeleteSlots = normalizeDeleteSlots(replicas, actualDeleteSlots, desiredDeleteSlots)
	} else if deletions.Len() > 0 {
		scaling = -1
		deletionsList := deletions.List()
		ordinal = deletionsList[len(deletionsList)-1]
		replicas -= 1
		if desiredDeleteSlots.Has(ordinal) {
			// in desired delete slots, add it in actual delete slots
			actualDeleteSlots.Insert(ordinal)
		}
		actualDeleteSlots = normalizeDeleteSlots(replicas, actualDeleteSlots, desiredDeleteSlots)
	}
	deleteSlots = actualDeleteSlots
	return
}

// normalizeDeleteSlots
// - add redundant data if in desired delete slots
// - remove redundant data if not in desired delete slots
// this is necessary to reach the desired state
func normalizeDeleteSlots(replicas int32, deleteSlots sets.Int32, desiredDeleteSlots sets.Int32) sets.Int32 {
	maxReplicaCount, _ := helper.GetMaxReplicaCountAndDeleteSlots(replicas, deleteSlots)
	for ordinal := range deleteSlots {
		if ordinal >= maxReplicaCount && !desiredDeleteSlots.Has(ordinal) {
			deleteSlots.Delete(ordinal)
		}
	}
	return deleteSlots
}
