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

package cloud

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

type VolumeModifier interface {
	// Name returns the name of the volume modifier.
	Name() string

	// Modify modifies the underlay volume of pvc to match the args of storageclass.
	// If no PV permission (e.g `-cluster-permission-pv=false`), the `pv` may be nil and will return `false, nil`.
	Modify(ctx context.Context, pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) (bool, error)

	MinWaitDuration() time.Duration

	Validate(spvc, dpvc *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error
}

// FakeVolumeModifier is a fake implementation of the VolumeModifier interface for unit testing.
type FakeVolumeModifier struct {
	name          string
	modifyResult  bool
	modifyError   error
	minWait       time.Duration
	validateError error
}

func (f *FakeVolumeModifier) Name() string {
	return f.name
}

func (f *FakeVolumeModifier) Modify(_ context.Context, _ *corev1.PersistentVolumeClaim,
	_ *corev1.PersistentVolume, _ *storagev1.StorageClass) (bool, error) {
	return f.modifyResult, f.modifyError
}

func (f *FakeVolumeModifier) MinWaitDuration() time.Duration {
	return f.minWait
}

func (f *FakeVolumeModifier) Validate(_, _ *corev1.PersistentVolumeClaim, _, _ *storagev1.StorageClass) error {
	return f.validateError
}
