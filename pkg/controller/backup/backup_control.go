// Copyright 2019 PingCAP, Inc.
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
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/util/slice"
)

// ControlInterface implements the control logic for updating Backup
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateBackup implements the control logic for backup job and backup clean job's creation, deletion
	UpdateBackup(backup *v1alpha1.Backup) error
}

// NewDefaultBackupControl returns a new instance of the default implementation BackupControlInterface that
// implements the documented semantics for Backup.
func NewDefaultBackupControl(
	cli versioned.Interface,
	backupManager backup.BackupManager) ControlInterface {
	return &defaultBackupControl{
		cli,
		backupManager,
	}
}

type defaultBackupControl struct {
	cli           versioned.Interface
	backupManager backup.BackupManager
}

// UpdateBackup executes the core logic loop for a Backup.
func (bc *defaultBackupControl) UpdateBackup(backup *v1alpha1.Backup) error {
	backup.SetGroupVersionKind(controller.BackupControllerKind)
	if err := bc.addProtectionFinalizer(backup); err != nil {
		return err
	}

	if err := bc.removeProtectionFinalizer(backup); err != nil {
		return err
	}

	return bc.updateBackup(backup)
}

func (bc *defaultBackupControl) updateBackup(backup *v1alpha1.Backup) error {
	return bc.backupManager.Sync(backup)
}

func (bc *defaultBackupControl) addProtectionFinalizer(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()

	if needToAddFinalizer(backup) {
		backup.Finalizers = append(backup.Finalizers, label.BackupProtectionFinalizer)
		_, err := bc.cli.PingcapV1alpha1().Backups(ns).Update(backup)
		if err != nil {
			return fmt.Errorf("add backup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
	}
	return nil
}

func (bc *defaultBackupControl) removeProtectionFinalizer(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()

	if v1alpha1.ShouldCleanData(backup) && isDeletionCandidate(backup) &&
		(v1alpha1.IsBackupClean(backup) || (backup.Spec.CleanPolicy == v1alpha1.CleanPolicyTypeOnFailure && !v1alpha1.IsBackupFailed(backup))) {
		backup.Finalizers = slice.RemoveString(backup.Finalizers, label.BackupProtectionFinalizer, nil)
		_, err := bc.cli.PingcapV1alpha1().Backups(ns).Update(backup)
		if err != nil {
			return fmt.Errorf("remove backup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
		return controller.RequeueErrorf(fmt.Sprintf("backup %s/%s has been cleaned up", ns, name))
	}
	return nil
}

func needToAddFinalizer(backup *v1alpha1.Backup) bool {
	return backup.DeletionTimestamp == nil && v1alpha1.ShouldCleanData(backup) && !slice.ContainsString(backup.Finalizers, label.BackupProtectionFinalizer, nil)
}

func isDeletionCandidate(backup *v1alpha1.Backup) bool {
	return backup.DeletionTimestamp != nil && slice.ContainsString(backup.Finalizers, label.BackupProtectionFinalizer, nil)
}

var _ ControlInterface = &defaultBackupControl{}

// FakeBackupControl is a fake BackupControlInterface
type FakeBackupControl struct {
	backupIndexer       cache.Indexer
	updateBackupTracker controller.RequestTracker
}

// NewFakeBackupControl returns a FakeBackupControl
func NewFakeBackupControl(backupInformer informers.BackupInformer) *FakeBackupControl {
	return &FakeBackupControl{
		backupInformer.Informer().GetIndexer(),
		controller.RequestTracker{},
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (fbc *FakeBackupControl) SetUpdateBackupError(err error, after int) {
	fbc.updateBackupTracker.SetError(err).SetAfter(after)
}

// UpdateBackup adds the backup to BackupIndexer
func (fbc *FakeBackupControl) UpdateBackup(backup *v1alpha1.Backup) error {
	defer fbc.updateBackupTracker.Inc()
	if fbc.updateBackupTracker.ErrorReady() {
		defer fbc.updateBackupTracker.Reset()
		return fbc.updateBackupTracker.GetError()
	}

	return fbc.backupIndexer.Add(backup)
}

var _ ControlInterface = &FakeBackupControl{}
