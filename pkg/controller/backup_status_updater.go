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
	"fmt"

	"k8s.io/klog"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

// BackupConditionUpdaterInterface enables updating Backup conditions.
type BackupConditionUpdaterInterface interface {
	Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error
}

type realBackupConditionUpdater struct {
	deps *Dependencies
}

// returns a BackupConditionUpdaterInterface that updates the Status of a Backup,
func NewRealBackupConditionUpdater(deps *Dependencies) BackupConditionUpdaterInterface {
	return &realBackupConditionUpdater{deps: deps}
}

func (u *realBackupConditionUpdater) Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error {
	ns := backup.GetNamespace()
	backupName := backup.GetName()
	oldStatus := backup.Status.DeepCopy()
	var isUpdate bool
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		isUpdate = v1alpha1.UpdateBackupCondition(&backup.Status, condition)
		if isUpdate {
			_, updateErr := u.deps.Clientset.PingcapV1alpha1().Backups(ns).Update(backup)
			if updateErr == nil {
				klog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
				return nil
			}
			if updated, err := u.deps.BackupLister.Backups(ns).Get(backupName); err == nil {
				// make a copy so we don't mutate the shared cache
				backup = updated.DeepCopy()
				backup.Status = *oldStatus
			} else {
				utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
			}
			return updateErr
		}
		return nil
	})
	return err
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
		BackupLister:  backupInformer.Lister(),
		BackupIndexer: backupInformer.Informer().GetIndexer(),
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (u *FakeBackupConditionUpdater) SetUpdateBackupError(err error, after int) {
	u.updateBackupTracker.SetError(err).SetAfter(after)
}

// UpdateBackup updates the Backup
func (u *FakeBackupConditionUpdater) Update(backup *v1alpha1.Backup, _ *v1alpha1.BackupCondition) error {
	defer u.updateBackupTracker.Inc()
	if u.updateBackupTracker.ErrorReady() {
		defer u.updateBackupTracker.Reset()
		return u.updateBackupTracker.GetError()
	}

	return u.BackupIndexer.Update(backup)
}

var _ BackupConditionUpdaterInterface = &FakeBackupConditionUpdater{}
