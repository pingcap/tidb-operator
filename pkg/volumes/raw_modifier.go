// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package volumes

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	timeutils "github.com/pingcap/tidb-operator/v2/pkg/utils/time"
	"github.com/pingcap/tidb-operator/v2/pkg/volumes/cloud"
	"github.com/pingcap/tidb-operator/v2/pkg/volumes/cloud/aws"
	"github.com/pingcap/tidb-operator/v2/pkg/volumes/cloud/azure"
)

var _ Modifier = &rawModifier{}

// rawModifier modifies volumes by calling cloud provider API.
type rawModifier struct {
	k8sClient       client.Client
	logger          logr.Logger
	volumeModifiers map[string]cloud.VolumeModifier
	clock           timeutils.Clock
}

func NewRawModifier(k8sClient client.Client, logger logr.Logger) Modifier {
	rw := &rawModifier{
		k8sClient:       k8sClient,
		logger:          logger,
		clock:           &timeutils.RealClock{},
		volumeModifiers: make(map[string]cloud.VolumeModifier, 2),
	}

	// aws
	ebsModifier := aws.NewEBSModifier(logger)
	rw.volumeModifiers[ebsModifier.Name()] = ebsModifier

	// azure
	diskModifier := azure.NewDiskModifier(logger)
	rw.volumeModifiers[diskModifier.Name()] = diskModifier

	return rw
}

func (m *rawModifier) GetActualVolume(ctx context.Context, expect, current *corev1.PersistentVolumeClaim) (*ActualVolume, error) {
	pv, err := getBoundPVFromPVC(ctx, m.k8sClient, current)
	if err != nil {
		return nil, fmt.Errorf("failed to get bound PV from PVC %s/%s: %w", current.Namespace, current.Name, err)
	}

	curSC, err := getStorageClassFromPVC(ctx, m.k8sClient, current)
	if err != nil {
		return nil, fmt.Errorf("failed to get StorageClass from PVC %s/%s: %w", current.Namespace, current.Name, err)
	}

	desired := &DesiredVolume{
		Size:             getStorageSize(expect.Spec.Resources.Requests),
		StorageClassName: expect.Spec.StorageClassName,
	}
	if desired.StorageClassName != nil {
		var sc storagev1.StorageClass
		if err = m.k8sClient.Get(ctx, client.ObjectKey{Name: *desired.StorageClassName}, &sc); err != nil {
			return nil, fmt.Errorf("failed to get StorageClass %s: %w", *desired.StorageClassName, err)
		}
		desired.StorageClass = &sc
	}

	actual := ActualVolume{
		Desired:      desired,
		PVC:          current,
		PV:           pv,
		StorageClass: curSC,
	}
	return &actual, nil
}

func (m *rawModifier) ShouldModify(_ context.Context, actual *ActualVolume) bool {
	actual.Phase = m.getVolumePhase(actual)
	return actual.Phase == VolumePhasePreparing || actual.Phase == VolumePhaseModifying
}

func (m *rawModifier) Modify(ctx context.Context, vol *ActualVolume) error {
	m.logger.Info("try to sync volume", "namespace", vol.PVC.Namespace, "name", vol.PVC.Name, "phase", vol.Phase)

	switch vol.Phase {
	case VolumePhasePreparing:
		if err := m.modifyPVCAnnoSpec(ctx, vol); err != nil {
			return err
		}

		fallthrough
	case VolumePhaseModifying:
		pvc := vol.PVC.DeepCopy()
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = vol.Desired.Size

		modifier, ok := m.volumeModifiers[vol.StorageClass.Provisioner]
		if !ok {
			return fmt.Errorf("no cloud volume modifier for storage class provisioner %s", vol.StorageClass.Provisioner)
		}
		wait, err := modifier.Modify(ctx, pvc, vol.PV, vol.Desired.StorageClass)
		if err != nil {
			return err
		}
		if wait {
			return &WaitError{Message: fmt.Sprintf("wait for volume %s/%s modification completed", vol.PVC.Namespace, vol.PVC.Name)}
		}

		// try to resize fs
		synced, err := m.syncPVCSize(ctx, vol)
		if err != nil {
			return err
		}
		if !synced {
			return &WaitError{Message: "wait for fs resize completed"}
		}
		if err := m.modifyPVCAnnoStatus(ctx, vol); err != nil {
			return err
		}
	default:
		return fmt.Errorf("volume %s/%s is in phase %s, cannot modify", vol.PVC.Namespace, vol.PVC.Name, vol.Phase)
	}
	return nil
}

func (m *rawModifier) modifyPVCAnnoStatus(ctx context.Context, vol *ActualVolume) error {
	pvc := vol.PVC.DeepCopy()

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	pvc.Annotations[annoKeyPVCStatusRevision] = pvc.Annotations[annoKeyPVCSpecRevision]
	if scName := pvc.Annotations[annoKeyPVCSpecStorageClass]; scName != "" {
		pvc.Annotations[annoKeyPVCStatusStorageClass] = scName
	}
	pvc.Annotations[annoKeyPVCStatusStorageSize] = pvc.Annotations[annoKeyPVCSpecStorageSize]

	err := m.k8sClient.Update(ctx, pvc)
	if err != nil {
		return err
	}

	vol.PVC = pvc
	return nil
}

func (m *rawModifier) getVolumePhase(vol *ActualVolume) VolumePhase {
	if err := m.validate(vol); err != nil {
		m.logger.Info("volume modification is not allowed", "namespace", vol.PVC.Namespace, "name", vol.PVC.Name, "error", err)
		return VolumePhaseCannotModify
	}
	if isPVCRevisionChanged(vol.PVC) {
		return VolumePhaseModifying
	}

	if !needModify(vol.PVC, vol.Desired) {
		return VolumePhaseModified
	}

	if m.waitForNextTime(vol.PVC, vol.StorageClass.Provisioner) {
		return VolumePhasePending
	}

	return VolumePhasePreparing
}

func (m *rawModifier) validate(vol *ActualVolume) error {
	if vol.Desired == nil {
		return fmt.Errorf("can't match desired volume")
	}
	desired := vol.Desired.GetStorageSize()
	actual := vol.GetStorageSize()
	result := desired.Cmp(actual)
	if result < 0 {
		return fmt.Errorf("can't shrunk size from %s to %s", &actual, &desired)
	}
	if result > 0 {
		supported, err := isVolumeExpansionSupported(vol.StorageClass)
		if err != nil {
			m.logger.Info("volume expansion may be not supported, but it will be tried",
				"namespace", vol.PVC.Namespace, "name", vol.PVC.Name,
				"storageclass", vol.GetStorageClassName(), "error", err)
		}
		if !supported {
			return fmt.Errorf("volume expansion is not supported by storageclass %s", vol.StorageClass.Name)
		}
	}

	// if no pv permission but have sc permission: cannot change sc
	if isStorageClassChanged(vol.GetStorageClassName(), vol.Desired.GetStorageClassName()) && vol.PV == nil {
		return fmt.Errorf("cannot change storage class (%s to %s), because there is no permission to get persistent volume",
			vol.GetStorageClassName(), vol.Desired.GetStorageClassName())
	}

	desiredPVC := vol.PVC.DeepCopy()
	desiredPVC.Spec.Resources.Requests[corev1.ResourceStorage] = desired

	modifier, ok := m.volumeModifiers[vol.StorageClass.Provisioner]
	if !ok {
		return fmt.Errorf("no cloud volume modifier for storage class provisioner %s", vol.StorageClass.Provisioner)
	}
	return modifier.Validate(vol.PVC, desiredPVC, vol.StorageClass, vol.Desired.StorageClass)
}

func (m *rawModifier) waitForNextTime(pvc *corev1.PersistentVolumeClaim, provisioner string) bool {
	str, ok := pvc.Annotations[annoKeyPVCLastTransitionTimestamp]
	if !ok {
		return false
	}
	timestamp, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return false
	}

	waitDur := defaultModifyWaitingDuration
	if modifier, ok := m.volumeModifiers[provisioner]; ok {
		waitDur = modifier.MinWaitDuration()
	}

	if d := m.clock.Since(timestamp); d < waitDur {
		m.logger.Info("volume modification is pending, should wait",
			"namespace", pvc.Namespace, "name", pvc.Name, "duration", waitDur-d)
		return true
	}

	return false
}

func getStorageClassFromPVC(ctx context.Context, cli client.Client, pvc *corev1.PersistentVolumeClaim) (*storagev1.StorageClass, error) {
	scName := getStorageClassNameFromPVC(pvc)
	if scName == "" {
		return nil, fmt.Errorf("StorageClass of pvc %s is not set", pvc.Name)
	}
	var sc storagev1.StorageClass
	if err := cli.Get(ctx, client.ObjectKey{Name: scName}, &sc); err != nil {
		return nil, fmt.Errorf("failed to get StorageClass %s: %w", scName, err)
	}
	return &sc, nil
}

func (m *rawModifier) syncPVCSize(ctx context.Context, vol *ActualVolume) (bool, error) {
	capacity := vol.PVC.Status.Capacity.Storage()
	requestSize := vol.PVC.Spec.Resources.Requests.Storage()
	if requestSize.Cmp(vol.Desired.Size) == 0 && capacity.Cmp(vol.Desired.Size) == 0 {
		return true, nil
	}

	if requestSize.Cmp(vol.Desired.Size) == 0 {
		return false, nil
	}

	pvc := vol.PVC.DeepCopy()
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = vol.Desired.Size
	err := m.k8sClient.Update(ctx, pvc)
	if err != nil {
		return false, err
	}

	vol.PVC = pvc
	return false, nil
}

func (m *rawModifier) modifyPVCAnnoSpec(ctx context.Context, vol *ActualVolume) error {
	pvc := vol.PVC.DeepCopy()
	size := vol.Desired.Size
	scName := vol.Desired.GetStorageClassName()

	if isChanged := snapshotStorageClassAndSize(pvc, scName, size); isChanged {
		upgradeRevision(pvc)
	}

	setLastTransitionTimestamp(pvc)
	if err := m.k8sClient.Update(ctx, pvc); err != nil {
		return fmt.Errorf("failed to update PVC %s/%s: %w", pvc.Namespace, pvc.Name, err)
	}
	vol.PVC = pvc
	return nil
}
