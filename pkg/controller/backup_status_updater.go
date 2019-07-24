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

	"github.com/golang/glog"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap.com/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// BackupStatusUpdaterInterface is an interface used to update the BackupStatus associated with a Backup.
// For any use other than testing, clients should create an instance using NewRealBackupStatusUpdater.
type BackupStatusUpdaterInterface interface {
	// UpdateBackupStatus sets the backup's Status to status. Implementations are required to retry on conflicts,
	// but fail on other errors. If the returned error is nil backup's Status has been successfully set to status.
	UpdateBackupStatus(*v1alpha1.Backup, *v1alpha1.BackupStatus, *v1alpha1.BackupStatus) error
}

// returns a BackupStatusUpdaterInterface that updates the Status of a Backup,
// using the supplied client and backupLister.
func NewRealBackupStatusUpdater(
	cli versioned.Interface,
	backupLister listers.BackupLister,
	recorder record.EventRecorder) BackupStatusUpdaterInterface {
	return &realBackupStatusUpdater{
		cli,
		backupLister,
		recorder,
	}
}

type realBackupStatusUpdater struct {
	cli          versioned.Interface
	backupLister listers.BackupLister
	recorder     record.EventRecorder
}

func (bsu *realBackupStatusUpdater) UpdateBackupStatus(
	backup *v1alpha1.Backup,
	newStatus *v1alpha1.BackupStatus,
	oldStatus *v1alpha1.BackupStatus) error {

	ns := backup.GetNamespace()
	backupName := backup.GetName()
	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, updateErr := bsu.cli.PingcapV1alpha1().Backups(ns).Update(backup)
		if updateErr == nil {
			glog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
			return nil
		}
		if updated, err := bsu.backupLister.Backups(ns).Get(backupName); err == nil {
			// make a copy so we don't mutate the shared cache
			backup = updated.DeepCopy()
			backup.Status = *newStatus
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
		}

		return updateErr
	})
	if !apiequality.Semantic.DeepEqual(newStatus, oldStatus) {
		bsu.recordBackupEvent("update", backup, err)
	}
	return err
}

func (bsu *realBackupStatusUpdater) recordBackupEvent(verb string, backup *v1alpha1.Backup, err error) {
	backupName := backup.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Backup %s successful",
			strings.ToLower(verb), backupName)
		bsu.recorder.Event(backup, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Backup %s failed error: %s",
			strings.ToLower(verb), backupName, err)
		bsu.recorder.Event(backup, corev1.EventTypeWarning, reason, msg)
	}
}

var _ BackupStatusUpdaterInterface = &realBackupStatusUpdater{}

// FakeBackupStatusUpdater is a fake BackupStatusUpdaterInterface
type FakeBackupStatusUpdater struct {
	BackupLister        listers.BackupLister
	BackupIndexer       cache.Indexer
	updateBackupTracker requestTracker
}

// NewFakeBackupStatusUpdater returns a FakeBackupStatusUpdater
func NewFakeBackupStatusUpdater(backupInformer informers.BackupInformer) *FakeBackupStatusUpdater {
	return &FakeBackupStatusUpdater{
		backupInformer.Lister(),
		backupInformer.Informer().GetIndexer(),
		requestTracker{0, nil, 0},
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (fbs *FakeBackupStatusUpdater) SetUpdateBackupError(err error, after int) {
	fbs.updateBackupTracker.err = err
	fbs.updateBackupTracker.after = after
}

// UpdateBackup updates the Backup
func (fbs *FakeBackupStatusUpdater) UpdateBackupStatus(backup *v1alpha1.Backup, _ *v1alpha1.BackupStatus, _ *v1alpha1.BackupStatus) error {
	defer fbs.updateBackupTracker.inc()
	if fbs.updateBackupTracker.errorReady() {
		defer fbs.updateBackupTracker.reset()
		return fbs.updateBackupTracker.err
	}

	return fbs.BackupIndexer.Update(backup)
}

var _ BackupStatusUpdaterInterface = &FakeBackupStatusUpdater{}
