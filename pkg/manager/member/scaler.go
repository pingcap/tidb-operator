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

// scaleOne calculates desired replicas and delete slots from actual/desired
// stateful sets by allowing only one pod to be deleted or created
// it returns following values:
// - scaling:
//   - 0: no scaling required
//   - 1: scaling out
//   - -1: scaling in
// - ordinal: pod ordinal to create or delete
// - replicas/deleteSlots: desired replicas and deleteSlots by allowing only one pod to be deleted or created
func scaleOne(actual *apps.StatefulSet, desired *apps.StatefulSet) (scaling int, ordinal int32, replicas int32, deleteSlots sets.Int) {
	actualDesiredPodOrdinals := helper.GetDesiredPodOrdinals(int(*actual.Spec.Replicas), actual)
	desiredDesiredPodOrdinals := helper.GetDesiredPodOrdinals(int(*desired.Spec.Replicas), desired)
	additions := desiredDesiredPodOrdinals.Difference(actualDesiredPodOrdinals)
	deletions := actualDesiredPodOrdinals.Difference(desiredDesiredPodOrdinals)
	scaling = 0
	ordinal = -1
	replicas = *actual.Spec.Replicas
	actualDeleteSlots := helper.GetDeleteSlots(actual)
	desiredDeleteSlots := helper.GetDeleteSlots(desired)
	if additions.Len() > 0 {
		// we always do scaling out before scaling in to maintain maximum avaiability
		scaling = 1
		ordinal = int32(additions.List()[0])
		replicas = *actual.Spec.Replicas + 1
		if !desiredDeleteSlots.Has(int(ordinal)) {
			// not in desired delete slots, remote it from actual delete slots
			actualDeleteSlots.Delete(int(ordinal))
		}
	} else if deletions.Len() > 0 {
		scaling = -1
		deletionsList := deletions.List()
		ordinal = int32(deletionsList[len(deletionsList)-1])
		replicas = *actual.Spec.Replicas - 1
		if desiredDeleteSlots.Has(int(ordinal)) {
			// in desired delete slots, add it in actual delete slots
			actualDeleteSlots.Insert(int(ordinal))
		}
	}
	deleteSlots = actualDeleteSlots
	return
}
