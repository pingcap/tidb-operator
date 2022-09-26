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
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

const (
	skipReasonScalerPVCNotFound             = "scaler: pvc is not found"
	skipReasonScalerAnnIsNil                = "scaler: pvc annotations is nil"
	skipReasonScalerAnnDeferDeletingIsEmpty = "scaler: pvc annotations defer deleting is empty"
	scalingOutFlag                          = 1
	scalingInFlag                           = -1
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type Scaler interface {
	// Scale scales the cluster. It does nothing if scaling is not needed.
	Scale(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleOut scales out the cluster
	ScaleOut(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
}

type generalScaler struct {
	deps *controller.Dependencies
}

// TODO: change skipReason to event recorder as in TestPDFailoverFailover
func (s *generalScaler) deleteDeferDeletingPVC(controller runtime.Object, memberType v1alpha1.MemberType, ordinal int32) (map[string]string, error) {
	meta := controller.(metav1.Object)
	ns := meta.GetNamespace()
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	// for unit test
	skipReason := map[string]string{}

	selector, err := GetPVCSelectorForPod(controller, memberType, ordinal)
	if err != nil {
		return skipReason, fmt.Errorf("%s %s/%s assemble label selector failed, err: %v", kind, ns, meta.GetName(), err)
	}

	pvcs, err := s.deps.PVCLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		msg := fmt.Sprintf("%s %s/%s list pvc failed, selector: %s, err: %v", kind, ns, meta.GetName(), selector, err)
		klog.Error(msg)
		return skipReason, fmt.Errorf(msg)
	}
	if len(pvcs) == 0 {
		klog.Infof("%s %s/%s list pvc not found, selector: %s", kind, ns, meta.GetName(), selector)
		podName := ordinalPodName(memberType, meta.GetName(), ordinal)
		skipReason[podName] = skipReasonScalerPVCNotFound
		return skipReason, nil
	}

	for _, pvc := range pvcs {
		pvcName := pvc.Name
		if pvc.Annotations == nil {
			klog.Infof("Exists unexpected pvc %s/%s: %s", ns, pvcName, skipReasonScalerAnnIsNil)
			skipReason[pvcName] = skipReasonScalerAnnIsNil
			continue
		}
		if _, ok := pvc.Annotations[label.AnnPVCDeferDeleting]; !ok {
			klog.Infof("Exists unexpected pvc %s/%s: %s", ns, pvcName, skipReasonScalerAnnDeferDeletingIsEmpty)
			skipReason[pvcName] = skipReasonScalerAnnDeferDeletingIsEmpty
			continue
		}

		err = s.deps.PVCControl.DeletePVC(controller, pvc)
		if err != nil {
			klog.Errorf("Scale out: failed to delete pvc %s/%s, %v", ns, pvcName, err)
			return skipReason, err
		}
		klog.Infof("Scale out: delete pvc %s/%s successfully", ns, pvcName)
	}
	return skipReason, nil
}

func (s *generalScaler) updateDeferDeletingPVC(tc *v1alpha1.TidbCluster,
	memberType v1alpha1.MemberType, ordinal int32) error {
	ns := tc.GetNamespace()
	podName := ordinalPodName(memberType, tc.Name, ordinal)

	l := label.New().Instance(tc.GetInstanceName())
	l[label.AnnPodNameKey] = podName
	selector, err := l.Selector()
	if err != nil {
		return fmt.Errorf("cluster %s/%s assemble label selector failed, err: %v", ns, tc.Name, err)
	}

	pvcs, err := s.deps.PVCLister.PersistentVolumeClaims(ns).List(selector)
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
		_, err = s.deps.PVCControl.UpdatePVC(tc, pvc)
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
//
// - ordinal: pod ordinal to create or delete
// - replicas/deleteSlots: desired replicas and deleteSlots by allowing only one pod to be deleted or created
func scaleOne(actual *apps.StatefulSet, desired *apps.StatefulSet) (scaling int, ordinal int32, replicas int32, deleteSlots sets.Int32) {
	var ordinals []int32
	scaling, ordinals, replicas, deleteSlots = scaleMulti(actual, desired, 1)
	if len(ordinals) == 0 {
		ordinal = -1
	} else {
		ordinal = ordinals[0]
	}
	return
}

// scaleMulti calculates desired replicas and delete slots from actual/desired
// StatefulSets by allowing multiple pods to be deleted or created
// it returns following values:
// - scaling:
//   - 0: no scaling required
//   - 1: scaling out
//   - -1: scaling in
//
// - ordinals: pod ordinals to create or delete
// - replicas/deleteSlots: desired replicas and deleteSlots by allowing no more than maxCount pods to be deleted or created
func scaleMulti(actual *apps.StatefulSet, desired *apps.StatefulSet, maxCount int) (scaling int, ordinals []int32, replicas int32, deleteSlots sets.Int32) {
	actualPodOrdinals := helper.GetPodOrdinals(*actual.Spec.Replicas, actual)
	desiredPodOrdinals := helper.GetPodOrdinals(*desired.Spec.Replicas, desired)
	additions := desiredPodOrdinals.Difference(actualPodOrdinals)
	deletions := actualPodOrdinals.Difference(desiredPodOrdinals)
	scaling = 0
	ordinals = make([]int32, 0, maxCount)
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
		// we always do scaling out before scaling in to maintain maximum availability
		scaling = scalingOutFlag
		additionsList := additions.List()
		for i := 0; i < len(additionsList) && i < maxCount; i++ {
			ordinal := additionsList[i]
			replicas++
			ordinals = append(ordinals, ordinal)
			if !desiredDeleteSlots.Has(ordinal) {
				// not in desired delete slots, remove it from actual delete slots
				actualDeleteSlots.Delete(ordinal)
			}
		}
		actualDeleteSlots = normalizeDeleteSlots(replicas, actualDeleteSlots, desiredDeleteSlots)
	} else if deletions.Len() > 0 {
		scaling = scalingInFlag
		deletionsList := deletions.List()
		stop := len(deletionsList) - maxCount
		if stop < 0 {
			stop = 0
		}
		for i := len(deletionsList) - 1; i >= stop; i-- {
			ordinal := deletionsList[i]
			replicas--
			ordinals = append(ordinals, ordinal)
			if desiredDeleteSlots.Has(ordinal) {
				// in desired delete slots, add it in actual delete slots
				actualDeleteSlots.Insert(ordinal)
			}
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

// setReplicasAndDeleteSlotsByFinished sets the replica and deleteSlots according to the ordinals finished successfully.
// parameter means:
// - scaling:
//   - 1: scaling out
//   - -1: scaling in
//
// - ordinals: pod oridnals to create or delete this round, in processing order, which means increasing when scale out and decreasing when scale in.
// - finishedOrdinals: oridnals finished successfully in this round.
func setReplicasAndDeleteSlotsByFinished(scaling int, newSet, oldSet *apps.StatefulSet, ordinals []int32, finishedOrdinals sets.Int32) {
	klog.Infof("scale statefulset: ordinals: %v, finishedOrdinals: %v, StatefulSet: %s/%s", ordinals, finishedOrdinals, newSet.Namespace, newSet.Name)
	newReplicas := *oldSet.Spec.Replicas
	actualPodOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet)
	actualDeleteSlots := helper.GetDeleteSlots(oldSet)
	desiredDeleteSlots := helper.GetDeleteSlots(newSet)
	// copy delete slots from desired delete slots if not in actual pod ordinals
	for i := range desiredDeleteSlots {
		if !actualPodOrdinals.Has(i) {
			actualDeleteSlots.Insert(i)
		}
	}
	// we should count ordinal by order strict here.
	if scaling > 0 {
		for _, ordinal := range ordinals {
			if !finishedOrdinals.Has(ordinal) {
				break
			}
			newReplicas++
			if !desiredDeleteSlots.Has(ordinal) {
				actualDeleteSlots.Delete(ordinal)
			}
		}
	} else {
		for _, ordinal := range ordinals {
			if !finishedOrdinals.Has(ordinal) {
				break
			}
			newReplicas--
			if desiredDeleteSlots.Has(ordinal) {
				actualDeleteSlots.Insert(ordinal)
			}
		}
	}
	if newReplicas == *oldSet.Spec.Replicas {
		// reset if nothing changed.
		resetReplicas(newSet, oldSet)
	} else {
		actualDeleteSlots = normalizeDeleteSlots(newReplicas, actualDeleteSlots, desiredDeleteSlots)
		setReplicasAndDeleteSlots(newSet, newReplicas, actualDeleteSlots)
	}
}
