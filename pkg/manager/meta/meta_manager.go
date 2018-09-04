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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

var PVCNotFound = errors.New("PVC is not found")

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
	tcName := tc.GetName()

	l, err := label.New().Cluster(tcName).Selector()
	if err != nil {
		return err
	}
	pods, err := pmm.podLister.List(l)
	if err != nil {
		return err
	}

	for _, pod := range pods {
		// update meta info for pod
		err := pmm.podControl.UpdateMetaInfo(tc, pod)
		if err != nil {
			return err
		}
		// update meta info for pvc
		pvc, _ := pmm.resolvePVCFromPod(pod)
		if pvc == nil {
			return nil
		}
		err = pmm.pvcControl.UpdateMetaInfo(tc, pvc, pod)
		if err != nil {
			return err
		}
		// update meta info for pv
		pv, err := pmm.pvLister.Get(pvc.Spec.VolumeName)
		if err != nil {
			return err
		}
		err = pmm.pvControl.UpdateMetaInfo(tc, pv)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pmm *metaManager) resolvePVCFromPod(pod *corev1.Pod) (*corev1.PersistentVolumeClaim, error) {
	var pvcName string
	for _, vol := range pod.Spec.Volumes {
		switch vol.Name {
		case v1alpha1.PDMemberType.String(), v1alpha1.TiKVMemberType.String():
			if vol.PersistentVolumeClaim != nil {
				pvcName = vol.PersistentVolumeClaim.ClaimName
				break
			}
		case v1alpha1.TiDBMemberType.String():
			return nil, nil
		default:
			return nil, nil
		}
	}
	if len(pvcName) == 0 {
		return nil, PVCNotFound
	}

	pvc, err := pmm.pvcLister.PersistentVolumeClaims(pod.Namespace).Get(pvcName)
	if err != nil {
		return nil, err
	}
	return pvc, nil
}

var _ manager.Manager = &metaManager{}
