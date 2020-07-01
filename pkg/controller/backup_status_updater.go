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
	"strings"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// BackupConditionUpdaterInterface enables updating Backup conditions.
type BackupConditionUpdaterInterface interface {
	Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error
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

func (bcu *realBackupConditionUpdater) Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error {
	ns := backup.GetNamespace()
	backupName := backup.GetName()
	oldStatus := backup.Status.DeepCopy()
	var isUpdate bool
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		isUpdate = v1alpha1.UpdateBackupCondition(&backup.Status, condition)
		if isUpdate {
			_, updateErr := bcu.cli.PingcapV1alpha1().Backups(ns).Update(backup)
			if updateErr == nil {
				klog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
				return nil
			}
			if updated, err := bcu.backupLister.Backups(ns).Get(backupName); err == nil {
				// make a copy so we don't mutate the shared cache
				backup = updated.DeepCopy()
				backup.Status = *oldStatus
			} else {
				utilruntime.HandleError(perrors.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
			}
			return updateErr
		}
		return nil
	})
	return err
}

func (bcu *realBackupConditionUpdater) recordBackupEvent(verb string, backup *v1alpha1.Backup, err error) {
	backupName := backup.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Backup %s successful",
			strings.ToLower(verb), backupName)
		bcu.recorder.Event(backup, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Backup %s failed error: %s",
			strings.ToLower(verb), backupName, err)
		bcu.recorder.Event(backup, corev1.EventTypeWarning, reason, msg)
	}
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
func (fbc *FakeBackupConditionUpdater) SetUpdateBackupError(err error, after int) {
	fbc.updateBackupTracker.SetError(err).SetAfter(after)
}

// UpdateBackup updates the Backup
func (fbc *FakeBackupConditionUpdater) Update(backup *v1alpha1.Backup, _ *v1alpha1.BackupCondition) error {
	defer fbc.updateBackupTracker.Inc()
	if fbc.updateBackupTracker.ErrorReady() {
		defer fbc.updateBackupTracker.Reset()
		return fbc.updateBackupTracker.GetError()
	}

	return fbc.BackupIndexer.Update(backup)
}

var _ BackupConditionUpdaterInterface = &FakeBackupConditionUpdater{}
