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

package fedvolumebackup

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/slice"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

// ControlInterface implements the control logic for updating VolumeBackup
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateBackup implements the control logic for VolumeBackup's creation, deletion
	UpdateBackup(volumeBackup *v1alpha1.VolumeBackup) error
	// UpdateStatus updates the status for a VolumeBackup, include condition and status info
	// NOTE(federation): add a separate struct for newStatus as non-federation backup control did if needed
	UpdateStatus(volumeBackup *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error
}

// NewDefaultVolumeBackupControl returns a new instance of the default VolumeBackup ControlInterface implementation.
func NewDefaultVolumeBackupControl(
	cli versioned.Interface,
	backupManager fedvolumebackup.BackupManager) ControlInterface {
	return &defaultBackupControl{
		cli,
		backupManager,
	}
}

type defaultBackupControl struct {
	cli           versioned.Interface
	backupManager fedvolumebackup.BackupManager
}

// UpdateBackup executes the core logic loop for a VolumeBackup.
func (c *defaultBackupControl) UpdateBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	volumeBackup.SetGroupVersionKind(controller.FedVolumeBackupControllerKind)
	if err := c.addProtectionFinalizer(volumeBackup); err != nil {
		return err
	}

	if err := c.removeProtectionFinalizer(volumeBackup); err != nil {
		return err
	}

	return c.updateBackup(volumeBackup)
}

// UpdateStatus updates the status for a VolumeBackup, include condition and status info
func (c *defaultBackupControl) UpdateStatus(volumeBackup *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	return c.backupManager.UpdateStatus(volumeBackup, newStatus)
}

func (c *defaultBackupControl) updateBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	return c.backupManager.Sync(volumeBackup)
}

// addProtectionFinalizer will be called when the VolumeBackup CR is created
func (c *defaultBackupControl) addProtectionFinalizer(volumeBackup *v1alpha1.VolumeBackup) error {
	ns := volumeBackup.GetNamespace()
	name := volumeBackup.GetName()

	if needToAddFinalizer(volumeBackup) {
		volumeBackup.Finalizers = append(volumeBackup.Finalizers, label.BackupProtectionFinalizer)
		_, err := c.cli.FederationV1alpha1().VolumeBackups(ns).Update(context.TODO(), volumeBackup, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("add VolumeBackup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
	}
	return nil
}

func (c *defaultBackupControl) removeProtectionFinalizer(volumeBackup *v1alpha1.VolumeBackup) error {
	ns := volumeBackup.GetNamespace()
	name := volumeBackup.GetName()

	if needToRemoveFinalizer(volumeBackup) {
		volumeBackup.Finalizers = slice.RemoveString(volumeBackup.Finalizers, label.BackupProtectionFinalizer, nil)
		_, err := c.cli.FederationV1alpha1().VolumeBackups(ns).Update(context.TODO(), volumeBackup, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("remove VolumeBackup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
		klog.Infof("remove VolumeBackup %s/%s protection finalizers success", ns, name)
		return controller.RequeueErrorf(fmt.Sprintf("VolumeBackup %s/%s has been cleaned up", ns, name))
	}
	return nil
}

func needToAddFinalizer(volumeBackup *v1alpha1.VolumeBackup) bool {
	// TODO(federation): add something like non-federation's `IsCleanCandidate` check if needed
	return volumeBackup.DeletionTimestamp == nil &&
		!slice.ContainsString(volumeBackup.Finalizers, label.BackupProtectionFinalizer, nil)
}

func needToRemoveFinalizer(volumeBackup *v1alpha1.VolumeBackup) bool {
	// TODO(federation): add something like non-federation's
	// `IsCleanCandidate`, `IsBackupClean`, `NeedNotClean` check if needed
	return isDeletionCandidate(volumeBackup)
}

func isDeletionCandidate(volumeBackup *v1alpha1.VolumeBackup) bool {
	return volumeBackup.DeletionTimestamp != nil && slice.ContainsString(volumeBackup.Finalizers, label.BackupProtectionFinalizer, nil)
}

var _ ControlInterface = &defaultBackupControl{}

// FakeBackupControl is a fake BackupControlInterface
type FakeBackupControl struct {
	backupIndexer       cache.Indexer
	updateBackupTracker controller.RequestTracker
	condition           *v1alpha1.VolumeBackupCondition
}

// NewFakeBackupControl returns a FakeBackupControl
func NewFakeBackupControl(backupInformer informers.VolumeBackupInformer) *FakeBackupControl {
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

// UpdateBackup adds the VolumeBackup to BackupIndexer
func (c *FakeBackupControl) UpdateBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	defer c.updateBackupTracker.Inc()
	if c.updateBackupTracker.ErrorReady() {
		defer c.updateBackupTracker.Reset()
		return c.updateBackupTracker.GetError()
	}

	return c.backupIndexer.Add(volumeBackup)
}

// UpdateStatus updates the status for a VolumeBackup, include condition and status info
func (c *FakeBackupControl) UpdateStatus(_ *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	if len(newStatus.Conditions) > 0 {
		c.condition = &newStatus.Conditions[0]
	}
	return nil
}

var _ ControlInterface = &FakeBackupControl{}
