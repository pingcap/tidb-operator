// Copyright 2022 PingCAP, Inc.
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

package volumes

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	klog "k8s.io/klog/v2"
)

type VolumePhase int

const (
	VolumePhaseUnknown VolumePhase = iota
	// 1. isPVCRevisionChanged: false
	// 2. needModify: true
	// 3. waitForNextTime: true
	VolumePhasePending
	// 1. isPVCRevisionChanged: false
	// 2. needModify: true
	// 3. waitForNextTime: false
	VolumePhasePreparing
	// 1. isPVCRevisionChanged: true
	// 2. needModify: true/false
	// 3. waitForNextTime: true/false
	VolumePhaseModifying
	// 1. isPVCRevisionChanged: false
	// 2. needModify: false
	// 3. waitForNextTime: true/false
	VolumePhaseModified

	VolumePhaseCannotModify
)

func (p VolumePhase) String() string {
	switch p {
	case VolumePhasePending:
		return "Pending"
	case VolumePhasePreparing:
		return "Preparing"
	case VolumePhaseModifying:
		return "Modifying"
	case VolumePhaseModified:
		return "Modified"
	case VolumePhaseCannotModify:
		return "CannotModify"
	}

	return "Unknown"
}

func (p *podVolModifier) getVolumePhase(vol *ActualVolume) VolumePhase {
	if err := p.validate(vol); err != nil {
		klog.Warningf("volume %s/%s modification is not allowed: %v", vol.PVC.Namespace, vol.PVC.Name, err)
		return VolumePhaseCannotModify
	}
	if isPVCRevisionChanged(vol.PVC) {
		return VolumePhaseModifying
	}

	if !needModify(vol.PVC, vol.Desired) {
		return VolumePhaseModified
	}

	if p.waitForNextTime(vol.PVC, vol.Desired.StorageClass) {
		return VolumePhasePending
	}

	return VolumePhasePreparing
}

func isVolumeExpansionSupported(sc *storagev1.StorageClass) bool {
	if sc.AllowVolumeExpansion == nil {
		return false
	}
	return *sc.AllowVolumeExpansion
}

func (p *podVolModifier) validate(vol *ActualVolume) error {
	if vol.Desired.StorageClass == nil {
		// TODO: support default storage class
		return fmt.Errorf("can't change storage class to the default one")
	}
	desired := vol.Desired.Size
	actual := getStorageSize(vol.PVC.Spec.Resources.Requests)
	result := desired.Cmp(actual)
	switch {
	case result == 0:
	case result < 0:
		return fmt.Errorf("can't shrunk size from %s to %s", &actual, &desired)
	case result > 0:
		if !isVolumeExpansionSupported(vol.StorageClass) {
			return fmt.Errorf("volume expansion is not supported by storageclass %s", vol.StorageClass.Name)
		}
	}
	m := p.getVolumeModifier(vol.Desired.StorageClass)
	if m == nil {
		return nil
	}
	desiredPVC := vol.PVC.DeepCopy()
	desiredPVC.Spec.Resources.Requests[corev1.ResourceStorage] = desired

	return m.Validate(vol.PVC, desiredPVC, vol.StorageClass, vol.Desired.StorageClass)
}

func isPVCRevisionChanged(pvc *corev1.PersistentVolumeClaim) bool {
	specRevision := pvc.Annotations[annoKeyPVCSpecRevision]
	statusRevision := pvc.Annotations[annoKeyPVCStatusRevision]

	return specRevision != statusRevision
}

func (p *podVolModifier) waitForNextTime(pvc *corev1.PersistentVolumeClaim, sc *storagev1.StorageClass) bool {
	str, ok := pvc.Annotations[annoKeyPVCLastTransitionTimestamp]
	if !ok {
		return false
	}
	timestamp, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return false
	}
	d := time.Since(timestamp)

	m := p.getVolumeModifier(sc)

	waitDur := defaultModifyWaitingDuration
	if m != nil {
		waitDur = m.MinWaitDuration()
	}

	if d < waitDur {
		klog.Warningf("volume %s/%s modification is pending, should wait %v", pvc.Namespace, pvc.Name, waitDur-d)
		return true
	}

	return false
}

func needModify(pvc *corev1.PersistentVolumeClaim, desired *DesiredVolume) bool {
	size := desired.Size
	scName := ""
	if desired.StorageClass != nil {
		scName = desired.StorageClass.Name
	}

	return isPVCStatusMatched(pvc, scName, size)
}

func isPVCStatusMatched(pvc *corev1.PersistentVolumeClaim, scName string, size resource.Quantity) bool {
	isChanged := false
	oldSc, ok := pvc.Annotations[annoKeyPVCStatusStorageClass]
	if !ok {
		oldSc = ignoreNil(pvc.Spec.StorageClassName)
	}
	if oldSc != scName {
		isChanged = true
	}

	oldSize, ok := pvc.Annotations[annoKeyPVCStatusStorageSize]
	if !ok {
		quantity := getStorageSize(pvc.Spec.Resources.Requests)
		oldSize = quantity.String()
	}
	if oldSize != size.String() {
		isChanged = true
	}
	if isChanged {
		klog.Infof("volume %s/%s is changed, sc (%s => %s), size (%s => %s)", pvc.Namespace, pvc.Name, oldSc, scName, oldSize, size.String())
	}

	return isChanged
}
