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
	"errors"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

var errPVCNotFound = errors.New("PVC is not found")

type metaManager struct {
	pvcLister  corelisters.PersistentVolumeClaimLister
	pvcControl controller.PVCControlInterface
	pvLister   corelisters.PersistentVolumeLister
	pvControl  controller.PVControlInterface
	podLister  corelisters.PodLister
	podControl controller.PodControlInterface
}

// NewMetaManager returns a *metaManager
func NewMetaManager(
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.PVCControlInterface,
	pvLister corelisters.PersistentVolumeLister,
	pvControl controller.PVControlInterface,
	podLister corelisters.PodLister,
	podControl controller.PodControlInterface,
) manager.Manager {
	return &metaManager{
		pvcLister:  pvcLister,
		pvcControl: pvcControl,
		pvLister:   pvLister,
		pvControl:  pvControl,
		podLister:  podLister,
		podControl: podControl,
	}
}

func (pmm *metaManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	instanceName := tc.GetInstanceName()

	l, err := label.New().Instance(instanceName).Selector()
	if err != nil {
		return err
	}
	pods, err := pmm.podLister.Pods(ns).List(l)
	if err != nil {
		return fmt.Errorf("metaManager.Sync: failed to list pods for cluster %s/%s, selector: %s, error: %v", ns, instanceName, l, err)
	}

	for _, pod := range pods {
		// update meta info for pod
		_, err := pmm.podControl.UpdateMetaInfo(tc, pod)
		if err != nil {
			return err
		}

		if component := pod.Labels[label.ComponentLabelKey]; component != label.PDLabelVal && component != label.TiKVLabelVal && component != label.TiFlashLabelVal {
			// Skip syncing meta info for pod that doesn't use PV
			// Currently only PD/TiKV/TiFlash uses PV
			continue
		}
		// update meta info for pvc
		pvcs, err := pmm.resolvePVCFromPod(pod)
		if err != nil {
			return err
		}
		for _, pvc := range pvcs {
			_, err = pmm.pvcControl.UpdateMetaInfo(tc, pvc, pod)
			if err != nil {
				return err
			}
			if pvc.Spec.VolumeName == "" {
				continue
			}
			// update meta info for pv
			pv, err := pmm.pvLister.Get(pvc.Spec.VolumeName)
			if err != nil {
				klog.Errorf("Get PV %s error: %v", pvc.Spec.VolumeName, err)
				return err
			}
			_, err = pmm.pvControl.UpdateMetaInfo(tc, pv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (pmm *metaManager) resolvePVCFromPod(pod *corev1.Pod) ([]*corev1.PersistentVolumeClaim, error) {
	var pvcs []*corev1.PersistentVolumeClaim
	var pvcName string
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil {
			pvcName = vol.PersistentVolumeClaim.ClaimName
			if len(pvcName) == 0 {
				continue
			}
			pvc, err := pmm.pvcLister.PersistentVolumeClaims(pod.Namespace).Get(pvcName)
			if err != nil {
				klog.Errorf("Get PVC %s/%s error: %v", pod.Namespace, pvcName, err)
				continue
			}
			pvcs = append(pvcs, pvc)
		}
	}
	if len(pvcs) == 0 {
		return nil, errPVCNotFound
	}
	return pvcs, nil
}

var _ manager.Manager = &metaManager{}

type FakeMetaManager struct {
	err error
}

func NewFakeMetaManager() *FakeMetaManager {
	return &FakeMetaManager{}
}

func (fmm *FakeMetaManager) SetSyncError(err error) {
	fmm.err = err
}

func (fmm *FakeMetaManager) Sync(_ *v1alpha1.TidbCluster) error {
	return fmm.err
}
