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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

var _ PodVolumeModifier = &FakePodVolumeModifier{}

type FakePodVolumeModifier struct {
	ModifyFunc            func(actual []ActualVolume) error
	ShouldModifyFunc      func(actual []ActualVolume) bool
	GetDesiredVolumesFunc func(tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) ([]DesiredVolume, error)
	GetActualVolumesFunc  func(pod *corev1.Pod, vs []DesiredVolume) ([]ActualVolume, error)
}

func (pvm *FakePodVolumeModifier) SetResult(shouldModify bool, err error) {
	pvm.ModifyFunc = func(actual []ActualVolume) error {
		return err
	}
	pvm.ShouldModifyFunc = func(actual []ActualVolume) bool {
		return shouldModify
	}
}

func (pvm *FakePodVolumeModifier) ShouldModify(actual []ActualVolume) bool {
	if pvm.ShouldModifyFunc == nil {
		return false
	}
	return pvm.ShouldModifyFunc(actual)
}

func (pvm *FakePodVolumeModifier) Modify(actual []ActualVolume) error {
	if pvm.ModifyFunc == nil {
		return nil
	}
	return pvm.ModifyFunc(actual)
}

func (pvm *FakePodVolumeModifier) GetDesiredVolumes(tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) ([]DesiredVolume, error) {
	if pvm.GetDesiredVolumesFunc == nil {
		return nil, nil
	}
	return pvm.GetDesiredVolumesFunc(tc, mt)
}

func (pvm *FakePodVolumeModifier) GetActualVolumes(pod *corev1.Pod, vs []DesiredVolume) ([]ActualVolume, error) {
	if pvm.GetActualVolumesFunc == nil {
		return nil, nil
	}
	return pvm.GetActualVolumesFunc(pod, vs)
}
