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

	"github.com/pingcap/tidb-operator/pkg/apis/pdapi"
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
func NewPDScaler(pdControl pdapi.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface) Scaler {
	return &pdScaler{generalScaler{pdControl, pvcLister, pvcControl}}
}

func (psd *pdScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if tc.PDUpgrading() {
		resetReplicas(newSet, oldSet)
		return nil
	}

	_, err := psd.deleteDeferDeletingPVC(tc, oldSet.GetName(), v1alpha1.PDMemberType, *oldSet.Spec.Replicas)
	if err != nil {
		resetReplicas(newSet, oldSet)
		return err
	}

	if !tc.Status.PD.Synced {
		resetReplicas(newSet, oldSet)
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed,can't scale out now", ns, tcName)
	}

	if len(tc.Status.PD.FailureMembers) != 0 {
		increaseReplicas(newSet, oldSet)
		return nil
	}

	var i int32 = 0
	healthCount := 0
	totalCount := *oldSet.Spec.Replicas
	for ; i < totalCount; i++ {
		podName := ordinalPodName(v1alpha1.PDMemberType, tcName, i)
		if member, ok := tc.Status.PD.Members[podName]; ok && member.Health {
			healthCount++
		}
	}
	if healthCount < int(totalCount) {
		resetReplicas(newSet, oldSet)
		return fmt.Errorf("TidbCluster: %s/%s's pd %d/%d is ready, can't scale out now",
			ns, tcName, healthCount, totalCount)
	}

	increaseReplicas(newSet, oldSet)
	return nil
}

// We need remove member from cluster before reducing statefulset replicas
// only remove one member at a time when scale down
func (psd *pdScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	ordinal := *oldSet.Spec.Replicas - 1
	memberName := fmt.Sprintf("%s-pd-%d", tc.GetName(), ordinal)
	setName := oldSet.GetName()

	if tc.PDUpgrading() {
		resetReplicas(newSet, oldSet)
		return nil
	}

	if !tc.Status.PD.Synced {
		resetReplicas(newSet, oldSet)
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed,can't scale in now", ns, tcName)
	}

	err := controller.GetPDClient(psd.pdControl, tc).DeleteMember(memberName)
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

	_, err = psd.pvcControl.UpdatePVC(tc, pvc)
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

func (fsd *fakePDScaler) ScaleOut(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	increaseReplicas(newSet, oldSet)
	return nil
}

func (fsd *fakePDScaler) ScaleIn(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	decreaseReplicas(newSet, oldSet)
	return nil
}
