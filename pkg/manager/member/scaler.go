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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	skipReasonPVCNotFound             = "pvc is not found"
	skipReasonAnnIsNil                = "pvc annotations is nil"
	skipReasonAnnDeferDeletingIsEmpty = "pvc annotations defer deleting is empty"
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type Scaler interface {
	// ScaleOut scales out the cluster
	ScaleOut(*v1alpha1.TidbCluster, *apps.StatefulSet, *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(*v1alpha1.TidbCluster, *apps.StatefulSet, *apps.StatefulSet) error
}

type generalScaler struct {
	pdControl  controller.PDControlInterface
	pvcLister  corelisters.PersistentVolumeClaimLister
	pvcControl controller.PVCControlInterface
}

func (gs *generalScaler) deleteAllDeferDeletingPVC(tc *v1alpha1.TidbCluster,
	setName string, memberType v1alpha1.MemberType, from, to int32) (map[int32]string, error) {
	ns := tc.GetNamespace()
	// for unit test
	skipReason := map[int32]string{}

	for i := from; i < to; i++ {
		pvcName := ordinalPVCName(memberType, setName, i)
		pvc, err := gs.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
		if errors.IsNotFound(err) {
			skipReason[i] = skipReasonPVCNotFound
			continue
		}
		if err != nil {
			return skipReason, err
		}

		if pvc.Annotations == nil {
			skipReason[i] = skipReasonAnnIsNil
			continue
		}
		if _, ok := pvc.Annotations[label.AnnPVCDeferDeleting]; !ok {
			skipReason[i] = skipReasonAnnDeferDeletingIsEmpty
			continue
		}

		err = gs.pvcControl.DeletePVC(tc, pvc)
		if err != nil {
			return skipReason, err
		}
	}

	return skipReason, nil
}

func resetReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas
}
func increaseReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas + 1
}
func decreaseReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas - 1
}

func ordinalPVCName(memberType v1alpha1.MemberType, setName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", memberType, setName, ordinal)
}

func ordinalPodName(memberType v1alpha1.MemberType, tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", tcName, memberType, ordinal)
}
