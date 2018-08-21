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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// TODO add e2e test specs

type pdScaler struct {
	generalScaler
}

// NewPDScaler returns a Scaler
func NewPDScaler(pdControl controller.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface) Scaler {
	return &pdScaler{generalScaler{pdControl, pvcLister, pvcControl}}
}

func (psd *pdScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if tc.PDUpgrading() {
		resetReplicas(newSet, oldSet)
		return nil
	}

	err := psd.deleteAllDeferDeletingPVC(tc, oldSet.GetName(), v1alpha1.PDMemberType, *oldSet.Spec.Replicas, *newSet.Spec.Replicas)
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}

	increaseReplicas(newSet, oldSet)
	return nil
}

// We need remove member from cluster before reducing statefulset replicas
// only remove one member at a time when scale down
func (psd *pdScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	ordinal := *oldSet.Spec.Replicas - 1
	memberName := fmt.Sprintf("%s-pd-%d", tc.GetName(), ordinal)
	setName := oldSet.GetName()

	if tc.PDUpgrading() {
		resetReplicas(newSet, oldSet)
		return nil
	}

	err := psd.pdControl.GetPDClient(tc).DeleteMember(memberName)
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}

	pvcName := ordinalPVCName(v1alpha1.PDMemberType, setName, ordinal)
	pvc, err := psd.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	pvc.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)

	err = psd.pvcControl.UpdatePVC(tc, pvc)
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}

	decreaseReplicas(newSet, oldSet)
	return nil
}

type fakePDScaler struct{}

// NewFakePDScaler returns a fake Scaler
func NewFakePDScaler() Scaler {
	return &fakePDScaler{}
}

func (fsd *fakePDScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	increaseReplicas(newSet, oldSet)
	return nil
}

func (fsd *fakePDScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	decreaseReplicas(newSet, oldSet)
	return nil
}
