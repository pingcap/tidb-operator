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

package backup

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

type backupManager struct {
	deps *controller.BrFedDependencies
}

// NewBackupManager return backupManager
func NewBackupManager(deps *controller.BrFedDependencies) fedvolumebackup.BackupManager {
	return &backupManager{
		deps: deps,
	}
}

func (bm *backupManager) Sync(volumeBackup *v1alpha1.VolumeBackup) error {
	// because a finalizer is installed on the VolumeBackup on creation, when the VolumeBackup is deleted,
	// volumeBackup.DeletionTimestamp will be set, controller will be informed with an onUpdate event,
	// this is the moment that we can do clean up work.
	// TODO(federation): do clean up work

	if volumeBackup.DeletionTimestamp != nil {
		// volumeBackup is being deleted, don't do more things, return directly.
		return nil
	}

	return bm.syncBackup(volumeBackup)
}

// UpdateStatus updates the status for a Backup, include condition and status info.
func (bm *backupManager) UpdateStatus(backup *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	// TODO(federation): update status
	return nil
}

func (bm *backupManager) syncBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	ns := volumeBackup.GetNamespace()
	name := volumeBackup.GetName()

	// TODO(federation): implement the main logic of backup
	klog.Infof("sync VolumeBackup %s/%s", ns, name)

	// TODO(federation): remove the following code
	for k8sName, k8sClient := range bm.deps.FedClientset {
		bkList, err := k8sClient.PingcapV1alpha1().Backups("default").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Errorf("failed to list backups in %s: %v", k8sName, err)
			continue
		}
		for _, bk := range bkList.Items {
			klog.Infof("get backup %s/%s", bk.Namespace, bk.Name)
		}
	}

	return nil
}

var _ fedvolumebackup.BackupManager = &backupManager{}

type FakeBackupManager struct {
	err error
}

func NewFakeBackupManager() *FakeBackupManager {
	return &FakeBackupManager{}
}

func (m *FakeBackupManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeBackupManager) Sync(_ *v1alpha1.VolumeBackup) error {
	return m.err
}

// UpdateStatus updates the status for a Backup, include condition and status info.
func (m *FakeBackupManager) UpdateStatus(_ *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	return nil
}

var _ fedvolumebackup.BackupManager = &FakeBackupManager{}
