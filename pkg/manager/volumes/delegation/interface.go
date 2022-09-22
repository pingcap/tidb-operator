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

package delegation

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

type VolumeModifier interface {
	MinWaitDuration() time.Duration
	// ModifyVolume modifies the underlay volume of pvc to match the args of storageclass
	ModifyVolume(ctx context.Context, pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) (bool, error)

	Validate(spvc, dpvc *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error

	Name() string
}

var _ VolumeModifier = &MockVolumeModifier{}

type ModifyVolumeFunc func(ctx context.Context, pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) (bool, error)

type ValidateFunc func(spvc, dpvc *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error

type MockVolumeModifier struct {
	name            string
	minWaitDuration time.Duration

	ValidateFunc     ValidateFunc
	ModifyVolumeFunc ModifyVolumeFunc
}

func NewMockVolumeModifier(name string, minWaitDuration time.Duration) *MockVolumeModifier {
	return &MockVolumeModifier{
		name:            name,
		minWaitDuration: minWaitDuration,
	}
}

func (m *MockVolumeModifier) Name() string {
	return m.name
}

func (m *MockVolumeModifier) MinWaitDuration() time.Duration {
	return m.minWaitDuration
}

func (m *MockVolumeModifier) ModifyVolume(ctx context.Context, pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) (bool, error) {
	if m.ModifyVolumeFunc == nil {
		return false, nil
	}
	return m.ModifyVolumeFunc(ctx, pvc, pv, sc)
}

func (m *MockVolumeModifier) Validate(spvc, dpvc *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error {
	if m.ValidateFunc == nil {
		return nil
	}
	return m.ValidateFunc(spvc, dpvc, ssc, dsc)
}
