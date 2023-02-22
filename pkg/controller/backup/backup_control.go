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
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/slice"
)

// ControlInterface implements the control logic for updating Backup
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateBackup implements the control logic for backup job and backup clean job's creation, deletion
	UpdateBackup(backup *v1alpha1.Backup) error
	// UpdateCondition updates the condition for a Backup.
	UpdateCondition(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error
	// UpdateBackupStatus updates the status for a Backup.
	UpdateBackupStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *controller.BackupUpdateStatus) error
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
func (c *defaultBackupControl) UpdateBackup(backup *v1alpha1.Backup) error {
	backup.SetGroupVersionKind(controller.BackupControllerKind)
	if err := c.addProtectionFinalizer(backup); err != nil {
		return err
	}

	if err := c.removeProtectionFinalizer(backup); err != nil {
		return err
	}

	return c.updateBackup(backup)
}

// UpdateCondition updates the condition for a Backup.
func (c *defaultBackupControl) UpdateCondition(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error {
	return c.backupManager.UpdateCondition(backup, condition)
}

// UpdateBackupStatus updates the status for a Backup.
func (c *defaultBackupControl) UpdateBackupStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *controller.BackupUpdateStatus) error {
	return c.backupManager.UpdateBackupStatus(backup, condition, newStatus)
}

func (c *defaultBackupControl) updateBackup(backup *v1alpha1.Backup) error {
	return c.backupManager.Sync(backup)
}

// addProtectionFinalizer will be called when the Backup CR is created
func (c *defaultBackupControl) addProtectionFinalizer(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()

	if needToAddFinalizer(backup) {
		backup.Finalizers = append(backup.Finalizers, label.BackupProtectionFinalizer)
		_, err := c.cli.PingcapV1alpha1().Backups(ns).Update(context.TODO(), backup, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("add backup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
	}
	return nil
}

func (c *defaultBackupControl) removeProtectionFinalizer(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()

	if needToRemoveFinalizer(backup) {
		backup.Finalizers = slice.RemoveString(backup.Finalizers, label.BackupProtectionFinalizer, nil)
		_, err := c.cli.PingcapV1alpha1().Backups(ns).Update(context.TODO(), backup, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("remove backup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
		klog.Infof("remove backup %s/%s protection finalizers success", ns, name)
		return controller.RequeueErrorf(fmt.Sprintf("backup %s/%s has been cleaned up", ns, name))
	}
	return nil
}

func needToAddFinalizer(backup *v1alpha1.Backup) bool {
	return backup.DeletionTimestamp == nil && v1alpha1.IsCleanCandidate(backup) && !slice.ContainsString(backup.Finalizers, label.BackupProtectionFinalizer, nil)
}

func needToRemoveFinalizer(backup *v1alpha1.Backup) bool {
	return v1alpha1.IsCleanCandidate(backup) && isDeletionCandidate(backup) &&
		(v1alpha1.IsBackupClean(backup) || v1alpha1.NeedNotClean(backup))
}

func isDeletionCandidate(backup *v1alpha1.Backup) bool {
	return backup.DeletionTimestamp != nil && slice.ContainsString(backup.Finalizers, label.BackupProtectionFinalizer, nil)
}

var _ ControlInterface = &defaultBackupControl{}

// FakeBackupControl is a fake BackupControlInterface
type FakeBackupControl struct {
	backupIndexer       cache.Indexer
	updateBackupTracker controller.RequestTracker
	condition           *v1alpha1.BackupCondition
}

// NewFakeBackupControl returns a FakeBackupControl
func NewFakeBackupControl(backupInformer informers.BackupInformer) *FakeBackupControl {
	return &FakeBackupControl{
		backupInformer.Informer().GetIndexer(),
		controller.RequestTracker{},
		nil,
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (c *FakeBackupControl) SetUpdateBackupError(err error, after int) {
	c.updateBackupTracker.SetError(err).SetAfter(after)
}

// UpdateBackup adds the backup to BackupIndexer
func (c *FakeBackupControl) UpdateBackup(backup *v1alpha1.Backup) error {
	defer c.updateBackupTracker.Inc()
	if c.updateBackupTracker.ErrorReady() {
		defer c.updateBackupTracker.Reset()
		return c.updateBackupTracker.GetError()
	}

	return c.backupIndexer.Add(backup)
}

// UpdateCondition updates the condition for a Backup.
func (c *FakeBackupControl) UpdateCondition(_ *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error {
	c.condition = condition
	return nil
}

// UpdateBackupStatus updates the status for a Backup.
func (c *FakeBackupControl) UpdateBackupStatus(_ *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *controller.BackupUpdateStatus) error {
	c.condition = condition
	return nil
}

var _ ControlInterface = &FakeBackupControl{}
