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
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type reclaimPolicyManager struct {
	deps *controller.Dependencies
}

// NewReclaimPolicyManager returns a *reclaimPolicyManager
func NewReclaimPolicyManager(deps *controller.Dependencies) *reclaimPolicyManager {
	return &reclaimPolicyManager{
		deps: deps,
	}
}

func (rpm *reclaimPolicyManager) Sync(tc *v1alpha1.TidbCluster) error {
	return rpm.sync(v1alpha1.TiDBClusterKind, tc.GetNamespace(), tc.GetInstanceName(), tc.IsPVReclaimEnabled(), *tc.Spec.PVReclaimPolicy, tc)
}

func (rpm *reclaimPolicyManager) SyncMonitor(tm *v1alpha1.TidbMonitor) error {
	return rpm.sync(v1alpha1.TiDBMonitorKind, tm.GetNamespace(), tm.GetName(), false, *tm.Spec.PVReclaimPolicy, tm)
}

func (rpm *reclaimPolicyManager) sync(kind, ns, instanceName string, isPVReclaimEnabled bool, policy corev1.PersistentVolumeReclaimPolicy, obj runtime.Object) error {
	selector, err := label.New().Instance(instanceName).Selector()
	if err != nil {
		return err
	}

	pvcs, err := rpm.deps.PVCLister.PersistentVolumeClaims(ns).List(selector)
	if err != nil {
		return fmt.Errorf("reclaimPolicyManager.sync: failed to list pvc for cluster %s/%s, selector %s, error: %s", ns, instanceName, selector, err)
	}

	for _, pvc := range pvcs {
		if pvc.Spec.VolumeName == "" {
			continue
		}
		l := label.Label(pvc.Labels)
		switch kind {
		case v1alpha1.TiDBClusterKind:
			if !l.IsPD() && !l.IsTiDB() && !l.IsTiKV() && !l.IsTiFlash() && !l.IsPump() {
				continue
			}
		case v1alpha1.TiDBMonitorKind:
			if !l.IsMonitor() {
				continue
			}
		default:
			continue
		}

		if isPVReclaimEnabled && len(pvc.Annotations[label.AnnPVCDeferDeleting]) != 0 {
			// If the pv reclaim function is turned on, and when pv is the candidate pv to be reclaimed, skip patch this pv.
			continue
		}
		pv, err := rpm.deps.PVLister.Get(pvc.Spec.VolumeName)
		if err != nil {
			return fmt.Errorf("reclaimPolicyManager.sync: failed to get pvc %s for cluster %s/%s, error: %s", pvc.Spec.VolumeName, ns, instanceName, err)
		}

		if pv.Spec.PersistentVolumeReclaimPolicy == policy {
			continue
		}
		err = rpm.deps.PVControl.PatchPVReclaimPolicy(obj, pv, policy)
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
