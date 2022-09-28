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

package snapshotter

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
)

type NoneSnapshotter struct {
}

func (s *NoneSnapshotter) Init(deps *controller.Dependencies, conf map[string]string) error {
	return nil
}

func (s *NoneSnapshotter) GetVolumeID(pv *corev1.PersistentVolume) (string, error) {
	return "", nil
}

func (s *NoneSnapshotter) PrepareBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster) (string, error) {
	return "", nil
}

func (s *NoneSnapshotter) SetVolumeID(pv *corev1.PersistentVolume, volumeID string) error {
	return nil
}

func (s *NoneSnapshotter) PrepareRestoreMetadata(r *v1alpha1.Restore) (string, error) {
	return "", nil
}
