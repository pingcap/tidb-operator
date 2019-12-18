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

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	corelisters "k8s.io/client-go/listers/core/v1"
	glog "k8s.io/klog"
)

const (
	skipReasonScalerPVCNotFound             = "scaler: pvc is not found"
	skipReasonScalerAnnIsNil                = "scaler: pvc annotations is nil"
	skipReasonScalerAnnDeferDeletingIsEmpty = "scaler: pvc annotations defer deleting is empty"
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type Scaler interface {
	// Scale scales the cluster. It does nothing if scaling is not needed.
	Scale(tc *v1alpha1.TidbCluster, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleOut scales out the cluster
	ScaleOut(tc *v1alpha1.TidbCluster, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(tc *v1alpha1.TidbCluster, actual *apps.StatefulSet, desired *apps.StatefulSet) error
}

type generalScaler struct {
	pdControl  pdapi.PDControlInterface
	pvcLister  corelisters.PersistentVolumeClaimLister
	pvcControl controller.PVCControlInterface
}

func (gs *generalScaler) deleteDeferDeletingPVC(tc *v1alpha1.TidbCluster,
	setName string, memberType v1alpha1.MemberType, ordinal int32) (map[int32]string, error) {
	ns := tc.GetNamespace()
	// for unit test
	skipReason := map[int32]string{}

	pvcName := ordinalPVCName(memberType, setName, ordinal)
	pvc, err := gs.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if errors.IsNotFound(err) {
		skipReason[ordinal] = skipReasonScalerPVCNotFound
		return skipReason, nil
	}
	if err != nil {
		return skipReason, err
	}

	if pvc.Annotations == nil {
		skipReason[ordinal] = skipReasonScalerAnnIsNil
		return skipReason, nil
	}
	if _, ok := pvc.Annotations[label.AnnPVCDeferDeleting]; !ok {
		skipReason[ordinal] = skipReasonScalerAnnDeferDeletingIsEmpty
		return skipReason, nil
	}

	err = gs.pvcControl.DeletePVC(tc, pvc)
	if err != nil {
		glog.Errorf("scale out: failed to delete pvc %s/%s, %v", ns, pvcName, err)
		return skipReason, err
	}
	glog.Infof("scale out: delete pvc %s/%s successfully", ns, pvcName)

	return skipReason, nil
}

func resetReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		helper.SetDeleteSlots(newSet, helper.GetDeleteSlots(oldSet))
	}
}

func setReplicasAndDeleteSlots(newSet *apps.StatefulSet, replicas int32, deleteSlots sets.Int) {
	oldReplicas := *newSet.Spec.Replicas
	*newSet.Spec.Replicas = replicas
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		helper.SetDeleteSlots(newSet, deleteSlots)
		glog.Infof("scale statefulset: %s/%s replicas from %d to %d (delete slots: %v)",
			newSet.GetNamespace(), newSet.GetName(), oldReplicas, replicas, deleteSlots.List())
		return
	}
	glog.Infof("scale statefulset: %s/%s replicas from %d to %d",
		newSet.GetNamespace(), newSet.GetName(), oldReplicas, replicas)
}

func ordinalPVCName(memberType v1alpha1.MemberType, setName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", memberType, setName, ordinal)
}

func ordinalPodName(memberType v1alpha1.MemberType, tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", tcName, memberType, ordinal)
}

func validateScaling(actual *apps.StatefulSet, desired *apps.StatefulSet) error {
	if !features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		return nil
	}
	// Following cases are not supported yet.
	// TODO support it or validate in validation phase
	actualDeleteSlots := helper.GetDeleteSlots(actual)
	desiredDeleteSlots := helper.GetDeleteSlots(desired)
	if *actual.Spec.Replicas == *desired.Spec.Replicas {
		// no scaling
		if actualDeleteSlots.Equal(desiredDeleteSlots) {
			return nil
		}
	} else if *actual.Spec.Replicas > *desired.Spec.Replicas {
		// scaling in
		if actualDeleteSlots.Equal(desiredDeleteSlots) || desiredDeleteSlots.IsSuperset(actualDeleteSlots) {
			return nil
		}
	} else {
		// scaling out
		if actualDeleteSlots.Equal(desiredDeleteSlots) || actualDeleteSlots.IsSuperset(desiredDeleteSlots) {
			return nil
		}
	}
	return fmt.Errorf("unsupported scaling operation for sts %s/%s (actual: %d, %v; desired: %d, %v)", actual.Namespace, actual.Name,
		*actual.Spec.Replicas, actualDeleteSlots.List(),
		*desired.Spec.Replicas, desiredDeleteSlots.List(),
	)
}

// return the last ordinal in the list, -1 if the list is empty
func lastOrdinalInList(list []int) int {
	if len(list) == 0 {
		return -1
	}
	return list[len(list)-1]
}

// scaleOne scales actual set to desired set by deleting or creating a replica
func scaleOne(actual *apps.StatefulSet, desired *apps.StatefulSet) (int32, int32, sets.Int, error) {
	if *actual.Spec.Replicas == *desired.Spec.Replicas {
		return -1, -1, nil, fmt.Errorf("unsupported scaling operation for set %s/%s, actual (replicas: %d), desired (replicas: %d)",
			actual.Namespace, actual.Name, *actual.Spec.Replicas, *desired.Spec.Replicas)
	}
	if *desired.Spec.Replicas > *actual.Spec.Replicas {
		// scale out by one
		if !features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
			return *actual.Spec.Replicas, *actual.Spec.Replicas + 1, nil, nil
		}
		actualDeleteSlots := helper.GetDeleteSlots(actual)
		desiredDeleteSlots := helper.GetDeleteSlots(desired)
		if desiredDeleteSlots.Len()-actualDeleteSlots.Len() < 0 {
			slotList := actualDeleteSlots.Difference(desiredDeleteSlots).List()
			slot := slotList[0]
			actualDeleteSlots.Delete(slot)
			return int32(slot), *actual.Spec.Replicas + 1, actualDeleteSlots, nil
		} else if !desiredDeleteSlots.Equal(desiredDeleteSlots) {
			// unreachable because this is prohibited by validateScaling
			return -1, -1, nil, fmt.Errorf("unsupported scaling operation for set %s/%s, actual (replicas: %d, delete slots: %v), desired (replicas: %d, delete slots: %v)", actual.Namespace, actual.Name, *actual.Spec.Replicas, actualDeleteSlots, *desired.Spec.Replicas, desiredDeleteSlots)
		}
		// actualDeleteSlots equals to desiredDeleteSlots
		ordinalList := helper.GetDesiredPodOrdinals(int(*actual.Spec.Replicas), actual).List()
		return int32(lastOrdinalInList(ordinalList)) + 1, *actual.Spec.Replicas + 1, actualDeleteSlots, nil
	}
	// scale in by one
	if !features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		return *actual.Spec.Replicas - 1, *actual.Spec.Replicas - 1, nil, nil
	}
	actualDeleteSlots := helper.GetDeleteSlots(actual)
	desiredDeleteSlots := helper.GetDeleteSlots(desired)
	if desiredDeleteSlots.Len()-actualDeleteSlots.Len() > 0 {
		slotList := desiredDeleteSlots.Difference(actualDeleteSlots).List()
		slot := slotList[len(slotList)-1]
		if !helper.GetDesiredPodOrdinals(int(*actual.Spec.Replicas), actual).Has(slot) {
			// unreachable because this is prohibited by validateScaling
			return -1, -1, nil, fmt.Errorf("unsupported scaling operation for set %s/%s, actual (replicas: %d, delete slots: %v), desired (replicas: %d, delete slots: %v)", actual.Namespace, actual.Name, *actual.Spec.Replicas, actualDeleteSlots, *desired.Spec.Replicas, desiredDeleteSlots)
		}
		actualDeleteSlots.Insert(slot)
		return int32(slot), *actual.Spec.Replicas - 1, actualDeleteSlots, nil
	} else if !desiredDeleteSlots.Equal(desiredDeleteSlots) {
		// unreachable because this is prohibited by validateScaling
		return -1, -1, nil, fmt.Errorf("unsupported scaling operation for set %s/%s, actual (replicas: %d, delete slots: %v), desired (replicas: %d, delete slots: %v)", actual.Namespace, actual.Name, *actual.Spec.Replicas, actualDeleteSlots, *desired.Spec.Replicas, desiredDeleteSlots)
	}
	// actualDeleteSlots equals to desiredDeleteSlots
	ordinalList := helper.GetDesiredPodOrdinals(int(*actual.Spec.Replicas), actual).List()
	return int32(ordinalList[len(ordinalList)-1]), *actual.Spec.Replicas - 1, actualDeleteSlots, nil
}
