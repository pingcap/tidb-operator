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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type tiflashScaler struct {
	generalScaler
	podLister corelisters.PodLister
}

// NewTiFlashScaler returns a tiflash Scaler
func NewTiFlashScaler(pdControl pdapi.PDControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface,
	podLister corelisters.PodLister) Scaler {
	return &tiflashScaler{generalScaler{pdControl, pvcLister, pvcControl}, podLister}
}

// TODO: Finish the scaling logic
func (tfs *tiflashScaler) Scale(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return tfs.ScaleOut(tc, oldSet, newSet)
	} else if scaling < 0 {
		return tfs.ScaleIn(tc, oldSet, newSet)
	}
	// we only sync auto scaler annotations when we are finishing syncing scaling
	return tfs.SyncAutoScalerAnn(tc, oldSet)
}

func (tfs *tiflashScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {

	return nil
}

func (tfs *tiflashScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {

	return nil
}

// SyncAutoScalerAnn reclaims the auto-scaling-out slots if the target pods no longer exist
func (tfs *tiflashScaler) SyncAutoScalerAnn(tc *v1alpha1.TidbCluster, actual *apps.StatefulSet) error {

	return nil
}

type fakeTiFlashScaler struct{}

// NewFakeTiFlashScaler returns a fake tiflash Scaler
func NewFakeTiFlashScaler() Scaler {
	return &fakeTiFlashScaler{}
}

func (fsd *fakeTiFlashScaler) Scale(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return fsd.ScaleOut(tc, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return fsd.ScaleIn(tc, oldSet, newSet)
	}
	return nil
}

func (fsd *fakeTiFlashScaler) ScaleOut(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (fsd *fakeTiFlashScaler) ScaleIn(_ *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}

func (fsd *fakeTiFlashScaler) SyncAutoScalerAnn(tc *v1alpha1.TidbCluster, actual *apps.StatefulSet) error {
	return nil
}
