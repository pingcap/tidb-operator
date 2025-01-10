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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination mock_generated.go -package=volumes ${GO_MODULE}/pkg/volumes Modifier
package volumes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Modifier interface {
	GetActualVolume(ctx context.Context, expect, current *corev1.PersistentVolumeClaim) (*ActualVolume, error)

	ShouldModify(ctx context.Context, actual *ActualVolume) bool

	Modify(ctx context.Context, vol *ActualVolume) error
}

type DesiredVolume struct {
	Size resource.Quantity
	// it may be nil if there is no permission to get storage class
	StorageClass *storagev1.StorageClass
	// it is sc name specified by user
	// the sc may not exist
	StorageClassName *string
	// VACName is the name of VolumeAttributesClass specified by user.
	// The VAC may not exist.
	VACName *string
	VAC     *storagev1beta1.VolumeAttributesClass
}

// GetStorageClassName may return empty because SC is unset or no permission to verify the existence of SC.
func (v *DesiredVolume) GetStorageClassName() string {
	if v.StorageClassName == nil {
		return ""
	}
	return *v.StorageClassName
}

func (v *DesiredVolume) GetStorageSize() resource.Quantity {
	return v.Size
}

func (v *DesiredVolume) String() string {
	return fmt.Sprintf("[Size: %v, StorageClass: %s, VAC: %v]", v.Size, v.GetStorageClassName(), v.VACName)
}

type ActualVolume struct {
	Desired *DesiredVolume
	PVC     *corev1.PersistentVolumeClaim
	Phase   VolumePhase
	// PV may be nil if there is no permission to get pvc
	PV *corev1.PersistentVolume
	// StorageClass may be nil if there is no permission to get storage class
	StorageClass *storagev1.StorageClass

	VACName *string
	VAC     *storagev1beta1.VolumeAttributesClass
}

func (v *ActualVolume) GetStorageClassName() string {
	return getStorageClassNameFromPVC(v.PVC)
}

func (v *ActualVolume) GetStorageSize() resource.Quantity {
	return getStorageSize(v.PVC.Status.Capacity)
}

func (v *ActualVolume) String() string {
	return fmt.Sprintf("[PVC: %s/%s, Phase: %s, StorageClass: %s, Size: %v, VAC: %v; desired: %s]",
		v.PVC.Namespace, v.PVC.Name, v.Phase, v.GetStorageClassName(),
		v.GetStorageSize(), v.VACName, v.Desired.String())
}

type VolumePhase int

const (
	VolumePhaseUnknown VolumePhase = iota
	// VolumePhasePending will be set when:
	//   1. isPVCRevisionChanged: false
	//   2. needModify: true
	//   3. waitForNextTime: true
	VolumePhasePending
	// VolumePhasePreparing will be set when:
	//   1. isPVCRevisionChanged: false
	//   2. needModify: true
	//   3. waitForNextTime: false
	VolumePhasePreparing
	// VolumePhaseModifying will be set when:
	//   1. isPVCRevisionChanged: true
	//   2. needModify: true/false
	//   3. waitForNextTime: true/false
	VolumePhaseModifying
	// VolumePhaseModified will be set when:
	//   1. isPVCRevisionChanged: false
	//   2. needModify: false
	//   3. waitForNextTime: true/false
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
	default:
		return "Unknown"
	}
}
