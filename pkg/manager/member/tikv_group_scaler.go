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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type TiKVGroupScaler interface {
	// Scale scales the cluster. It does nothing if scaling is not needed.
	Scale(tc *v1alpha1.TiKVGroup, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleOut scales out the cluster
	ScaleOut(tc *v1alpha1.TiKVGroup, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(tc *v1alpha1.TiKVGroup, actual *apps.StatefulSet, desired *apps.StatefulSet) error
}

type tikvGroupScaler struct {
	generalScaler
	podLister corelisters.PodLister
}

// NewTiKVScaler returns a tikvgroup Scaler
func NewTiKVGroupScaler(pdControl pdapi.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface,
	podLister corelisters.PodLister) TiKVGroupScaler {
	return &tikvGroupScaler{generalScaler{pdControl, pvcLister, pvcControl}, podLister}
}

func (tsd *tikvGroupScaler) Scale(tc *v1alpha1.TiKVGroup, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return tsd.ScaleOut(tc, oldSet, newSet)
	} else if scaling < 0 {
		return tsd.ScaleIn(tc, oldSet, newSet)
	}
	return nil
}

func (tsd *tikvGroupScaler) ScaleOut(tg *v1alpha1.TiKVGroup, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	fmt.Println("here")
	pvcName := fmt.Sprintf("tikv-%s-tikv-group-%d", tg.Name, ordinal)
	_, err := tsd.pvcLister.PersistentVolumeClaims(tg.Namespace).Get(pvcName)
	if err == nil {
		klog.Infof("scaling out tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
		_, err := tsd.deleteDeferDeletingPVC(tg, oldSet.GetName(), v1alpha1.TiKVMemberType, ordinal)
		if err != nil {
			return err
		}
		return controller.RequeueErrorf("TiKVGroup[%s/%s] ready to scale-out, wait for next round", tg.Namespace, tg.Name)
	}
	if !errors.IsNotFound(err) {
		return err
	}
	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func (tsd *tikvGroupScaler) ScaleIn(tg *v1alpha1.TiKVGroup, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	// we can only remove one member at a time when scaling in
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	klog.Infof("scaling in tikv statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	if controller.PodWebhookEnabled {
		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}
	return fmt.Errorf("TiKVGroup[%s/%s] failed to scaling-in, need to enable Pod Webhook feature", tg.Namespace, tg.Name)
}
