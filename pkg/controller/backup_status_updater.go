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

package controller

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// BackupUpdateStatus represents the status of a backup to be updated.
// This structure should keep synced with the fields in `BackupStatus`
// except for `Phase` and `Conditions`.
type BackupUpdateStatus struct {
	// BackupPath is the location of the backup.
	BackupPath *string
	// TimeStarted is the time at which the backup was started.
	TimeStarted *metav1.Time
	// TimeCompleted is the time at which the backup was completed.
	TimeCompleted *metav1.Time
	// BackupSizeReadable is the data size of the backup.
	// the difference with BackupSize is that its format is human readable
	BackupSizeReadable *string
	// BackupSize is the data size of the backup.
	BackupSize *int64
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs *string
	// LogStopped indicates whether the log backup has stopped.
	LogStopped *bool
	// LogCheckpointTs is the ts of log backup process.
	LogCheckpointTs *string
	// LogTruncateUntil is log backup truncate until timestamp.
	LogTruncateUntil *string
	// LogSafeTruncatedUntil is log backup safe truncate until timestamp.
	LogSafeTruncatedUntil *string
}

// BackupConditionUpdaterInterface enables updating Backup conditions.
type BackupConditionUpdaterInterface interface {
	Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error
}

type realBackupConditionUpdater struct {
	cli          versioned.Interface
	backupLister listers.BackupLister
	recorder     record.EventRecorder
}

// returns a BackupConditionUpdaterInterface that updates the Status of a Backup,
func NewRealBackupConditionUpdater(
	cli versioned.Interface,
	backupLister listers.BackupLister,
	recorder record.EventRecorder) BackupConditionUpdaterInterface {
	return &realBackupConditionUpdater{
		cli,
		backupLister,
		recorder,
	}
}

func (u *realBackupConditionUpdater) Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error {
	ns := backup.GetNamespace()
	backupName := backup.GetName()
	// try best effort to guarantee backup is updated.
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		isStatusUpdate := updateBackupStatus(&backup.Status, newStatus)
		isConditionUpdate := v1alpha1.UpdateBackupCondition(&backup.Status, condition)
		if isStatusUpdate || isConditionUpdate {
			_, updateErr := u.cli.PingcapV1alpha1().Backups(ns).Update(context.TODO(), backup, metav1.UpdateOptions{})
			if updateErr == nil {
				klog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
				return nil
			}
			klog.Errorf("Failed to update backup [%s/%s], error: %v", ns, backupName, updateErr)
			if updated, err := u.backupLister.Backups(ns).Get(backupName); err == nil {
				// make a copy so we don't mutate the shared cache
				backup = updated.DeepCopy()
			} else {
				utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
			}
			return updateErr
		}
		return nil
	})
	return err
}

// updateBackupStatus updates existing Backup status
// from the fields in BackupUpdateStatus.
func updateBackupStatus(status *v1alpha1.BackupStatus, newStatus *BackupUpdateStatus) bool {
	isUpdate := false
	if newStatus == nil {
		return isUpdate
	}
	if newStatus.BackupPath != nil && status.BackupPath != *newStatus.BackupPath {
		status.BackupPath = *newStatus.BackupPath
		isUpdate = true
	}
	if newStatus.TimeStarted != nil && status.TimeStarted != *newStatus.TimeStarted {
		status.TimeStarted = *newStatus.TimeStarted
		isUpdate = true
	}
	if newStatus.TimeCompleted != nil && status.TimeCompleted != *newStatus.TimeCompleted {
		status.TimeCompleted = *newStatus.TimeCompleted
		isUpdate = true
	}
	if newStatus.BackupSizeReadable != nil && status.BackupSizeReadable != *newStatus.BackupSizeReadable {
		status.BackupSizeReadable = *newStatus.BackupSizeReadable
		isUpdate = true
	}
	if newStatus.BackupSize != nil && status.BackupSize != *newStatus.BackupSize {
		status.BackupSize = *newStatus.BackupSize
		isUpdate = true
	}
	if newStatus.CommitTs != nil && status.CommitTs != *newStatus.CommitTs {
		status.CommitTs = *newStatus.CommitTs
		isUpdate = true
	}
	if newStatus.LogStopped != nil && status.LogStopped != *newStatus.LogStopped {
		status.LogStopped = *newStatus.LogStopped
		isUpdate = true
	}
	if newStatus.LogCheckpointTs != nil && status.LogCheckpointTs != *newStatus.LogCheckpointTs {
		status.LogCheckpointTs = *newStatus.LogCheckpointTs
		isUpdate = true
	}
	if newStatus.LogTruncateUntil != nil && status.LogTruncateUntil != *newStatus.LogTruncateUntil {
		status.LogTruncateUntil = *newStatus.LogTruncateUntil
		isUpdate = true
	}
	if newStatus.LogSafeTruncatedUntil != nil && status.LogSafeTruncatedUntil != *newStatus.LogSafeTruncatedUntil {
		status.LogSafeTruncatedUntil = *newStatus.LogSafeTruncatedUntil
		isUpdate = true
	}
	return isUpdate
}

var _ BackupConditionUpdaterInterface = &realBackupConditionUpdater{}

// FakeBackupConditionUpdater is a fake BackupConditionUpdaterInterface
type FakeBackupConditionUpdater struct {
	BackupLister        listers.BackupLister
	BackupIndexer       cache.Indexer
	updateBackupTracker RequestTracker
}

// NewFakeBackupConditionUpdater returns a FakeBackupConditionUpdater
func NewFakeBackupConditionUpdater(backupInformer informers.BackupInformer) *FakeBackupConditionUpdater {
	return &FakeBackupConditionUpdater{
		backupInformer.Lister(),
		backupInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (c *FakeBackupConditionUpdater) SetUpdateBackupError(err error, after int) {
	c.updateBackupTracker.SetError(err).SetAfter(after)
}

// UpdateBackup updates the Backup
func (c *FakeBackupConditionUpdater) Update(backup *v1alpha1.Backup, _ *v1alpha1.BackupCondition, _ *BackupUpdateStatus) error {
	defer c.updateBackupTracker.Inc()
	if c.updateBackupTracker.ErrorReady() {
		defer c.updateBackupTracker.Reset()
		return c.updateBackupTracker.GetError()
	}

	return c.BackupIndexer.Update(backup)
}

var _ BackupConditionUpdaterInterface = &FakeBackupConditionUpdater{}
