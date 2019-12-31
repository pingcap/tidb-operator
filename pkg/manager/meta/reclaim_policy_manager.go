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

package meta

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type reclaimPolicyManager struct {
	pvcLister corelisters.PersistentVolumeClaimLister
	pvLister  corelisters.PersistentVolumeLister
	pvControl controller.PVControlInterface
}

// NewReclaimPolicyManager returns a *reclaimPolicyManager
func NewReclaimPolicyManager(pvcLister corelisters.PersistentVolumeClaimLister,
	pvLister corelisters.PersistentVolumeLister,
	pvControl controller.PVControlInterface) manager.Manager {
	return &reclaimPolicyManager{
		pvcLister,
		pvLister,
		pvControl,
	}
}

func (rpm *reclaimPolicyManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	instanceName := tc.GetInstanceName()

	l, err := label.New().Instance(instanceName).Selector()
	if err != nil {
		return err
	}
	pvcs, err := rpm.pvcLister.PersistentVolumeClaims(ns).List(l)
	if err != nil {
		return err
	}

	for _, pvc := range pvcs {
		if pvc.Spec.VolumeName == "" {
			continue
		}
		if tc.IsPVReclaimEnabled() && len(pvc.Annotations[label.AnnPVCDeferDeleting]) != 0 {
			// If the pv reclaim function is turned on, and when pv is the candidate pv to be reclaimed, skip patch this pv.
			continue
		}
		pv, err := rpm.pvLister.Get(pvc.Spec.VolumeName)
		if err != nil {
			return err
		}

		if pv.Spec.PersistentVolumeReclaimPolicy == tc.Spec.PVReclaimPolicy {
			continue
		}

		err = rpm.pvControl.PatchPVReclaimPolicy(tc, pv, tc.Spec.PVReclaimPolicy)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ manager.Manager = &reclaimPolicyManager{}

type FakeReclaimPolicyManager struct {
	err error
}

func NewFakeReclaimPolicyManager() *FakeReclaimPolicyManager {
	return &FakeReclaimPolicyManager{}
}

func (frpm *FakeReclaimPolicyManager) SetSyncError(err error) {
	frpm.err = err
}

func (frpm *FakeReclaimPolicyManager) Sync(_ *v1alpha1.TidbCluster) error {
	return frpm.err
}
