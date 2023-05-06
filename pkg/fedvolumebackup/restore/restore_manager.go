// Copyright 2023 PingCAP, Inc.
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

package restore

import (
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

type restoreManager struct {
	deps *controller.BrFedDependencies
}

// NewRestoreManager return restoreManager
func NewRestoreManager(deps *controller.BrFedDependencies) fedvolumebackup.RestoreManager {
	return &restoreManager{
		deps: deps,
	}
}

func (bm *restoreManager) Sync(volumeRestore *v1alpha1.VolumeRestore) error {
	return bm.syncRestore(volumeRestore)
}

// UpdateCondition updates the condition for a Restore.
func (bm *restoreManager) UpdateCondition(volumeRestore *v1alpha1.VolumeRestore, condition *v1alpha1.VolumeRestoreCondition) error {
	// TODO(federation): update condition
	return nil
}

func (bm *restoreManager) syncRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	ns := volumeRestore.GetNamespace()
	name := volumeRestore.GetName()

	// TODO(federation): implement the main logic of restore
	klog.Infof("sync VolumeRestore %s/%s", ns, name)

	return nil
}

var _ fedvolumebackup.RestoreManager = &restoreManager{}

type FakeRestoreManager struct {
	err error
}

func NewFakeRestoreManager() *FakeRestoreManager {
	return &FakeRestoreManager{}
}

func (m *FakeRestoreManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeRestoreManager) Sync(_ *v1alpha1.VolumeRestore) error {
	return m.err
}

// UpdateStatus updates the condition for a Restore.
func (m *FakeRestoreManager) UpdateCondition(_ *v1alpha1.VolumeRestore, condition *v1alpha1.VolumeRestoreCondition) error {
	return nil
}

var _ fedvolumebackup.RestoreManager = &FakeRestoreManager{}
