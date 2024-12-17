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

	storagev1 "k8s.io/api/storage/v1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"

	"github.com/pingcap/tidb-operator/pkg/client"
)

var _ Modifier = &nativeModifier{}

// nativeModifier modifies volumes via K8s native API - VolumeAttributesClass.
type nativeModifier struct {
	k8sClient client.Client
	logger    logr.Logger
}

func NewNativeModifier(k8sClient client.Client, logger logr.Logger) Modifier {
	return &nativeModifier{
		k8sClient: k8sClient,
		logger:    logger,
	}
}

func (m *nativeModifier) GetActualVolume(ctx context.Context, expect, current *corev1.PersistentVolumeClaim) (*ActualVolume, error) {
	pv, err := getBoundPVFromPVC(ctx, m.k8sClient, current)
	if err != nil {
		return nil, fmt.Errorf("failed to get bound PV from PVC %s/%s: %w", current.Namespace, current.Name, err)
	}

	desired := &DesiredVolume{
		Size:    getStorageSize(expect.Spec.Resources.Requests),
		VACName: expect.Spec.VolumeAttributesClassName,
	}
	if desired.VACName != nil {
		var vac storagev1beta1.VolumeAttributesClass
		if err = m.k8sClient.Get(ctx, client.ObjectKey{Name: *desired.VACName}, &vac); err != nil {
			return nil, fmt.Errorf("failed to get desired VolumeAttributesClass %s: %w", *desired.VACName, err)
		}
		desired.VAC = &vac
	}

	actual := ActualVolume{
		Desired: desired,
		PVC:     current,
		PV:      pv,
		VACName: current.Status.CurrentVolumeAttributesClassName,
	}
	if actual.VACName != nil {
		var vac storagev1beta1.VolumeAttributesClass
		if err = m.k8sClient.Get(ctx, client.ObjectKey{Name: *actual.VACName}, &vac); err != nil {
			return nil, fmt.Errorf("failed to get current VolumeAttributesClass %s: %w", *actual.VACName, err)
		}
		actual.VAC = &vac
	}
	if current.Spec.StorageClassName != nil {
		var sc storagev1.StorageClass
		if err = m.k8sClient.Get(ctx, client.ObjectKey{Name: *current.Spec.StorageClassName}, &sc); err != nil {
			return nil, fmt.Errorf("failed to get current StorageClass %s: %w", *current.Spec.StorageClassName, err)
		}
		actual.StorageClass = &sc
	}
	return &actual, nil
}

func (m *nativeModifier) ShouldModify(_ context.Context, actual *ActualVolume) (modify bool) {
	if actual.PVC.Status.ModifyVolumeStatus != nil {
		m.logger.Info("there is a ModifyVolume operation being attempted",
			"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
			"target", actual.PVC.Status.ModifyVolumeStatus.TargetVolumeAttributesClassName,
			"status", actual.PVC.Status.ModifyVolumeStatus.Status)
		return false
	}

	if isStorageClassChanged(actual.GetStorageClassName(), actual.Desired.GetStorageClassName()) {
		m.logger.Info("cannot change storage class",
			"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
			"current", actual.GetStorageClassName(), "desired", actual.Desired.GetStorageClassName())
		return false
	}

	vacChanged := isVolumeAttributesClassChanged(actual)
	if vacChanged && actual.Desired.VAC != nil && actual.StorageClass != nil &&
		actual.Desired.VAC.DriverName != actual.StorageClass.Provisioner {
		m.logger.Info("the drive name in VAC should be same as the providioner in StorageClass",
			"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
			"in VAC", actual.Desired.VAC.DriverName, "in StorageClass", actual.StorageClass.Provisioner)
		return false
	}

	desiredSize := actual.Desired.GetStorageSize()
	actualSize := actual.GetStorageSize()
	sizeChanged := false
	result := desiredSize.Cmp(actualSize)
	if result < 0 {
		m.logger.Info("can't shrink volume size",
			"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
			"current", &actualSize, "desired", &desiredSize)
		return false
	} else if result > 0 {
		supported, err := isVolumeExpansionSupported(actual.StorageClass)
		if err != nil {
			m.logger.Error(err, "volume expansion of storage class may be not supported, but it will be tried",
				"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
				"storageclass", actual.GetStorageClassName())
		}
		if !supported {
			m.logger.Info("volume expansion is not supported by storage class",
				"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
				"storageclass", actual.GetStorageClassName())
			return false
		}
		sizeChanged = true
	}

	defer func() {
		// Log PVC conditions if not modified for debugging
		if !modify {
			m.logger.Info("PVC not modified",
				"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
				"size changed", sizeChanged, "vac changed", vacChanged)
			for _, cond := range actual.PVC.Status.Conditions {
				m.logger.Info("PVC condition",
					"namespace", actual.PVC.Namespace, "name", actual.PVC.Name,
					"type", cond.Type, "status", cond.Status, "reason", cond.Reason, "message", cond.Message)
			}
		}
	}()
	return sizeChanged || vacChanged
}

func (m *nativeModifier) Modify(ctx context.Context, vol *ActualVolume) error {
	desiredPVC := vol.PVC.DeepCopy()
	desiredPVC.Spec.Resources.Requests[corev1.ResourceStorage] = vol.Desired.GetStorageSize()
	desiredPVC.Spec.VolumeAttributesClassName = vol.Desired.VACName
	return m.k8sClient.Update(ctx, desiredPVC)
}
