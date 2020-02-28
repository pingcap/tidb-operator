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
// limitations under the License.i

package controller

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// BackupControlInterface manages Backups used in BackupSchedule
type BackupControlInterface interface {
	CreateBackup(backup *v1alpha1.Backup) (*v1alpha1.Backup, error)
	DeleteBackup(backup *v1alpha1.Backup) error
}

type realBackupControl struct {
	cli      versioned.Interface
	recorder record.EventRecorder
}

// NewRealBackupControl creates a new BackupControlInterface
func NewRealBackupControl(
	cli versioned.Interface,
	recorder record.EventRecorder,
) BackupControlInterface {
	return &realBackupControl{
		cli:      cli,
		recorder: recorder,
	}
}

func (rbc *realBackupControl) CreateBackup(backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	ns := backup.GetNamespace()
	backupName := backup.GetName()

	bsName := backup.GetLabels()[label.BackupScheduleLabelKey]
	backup, err := rbc.cli.PingcapV1alpha1().Backups(ns).Create(backup)
	if err != nil {
		klog.Errorf("failed to create Backup: [%s/%s] for backupSchedule/%s, err: %v", ns, backupName, bsName, err)
	} else {
		klog.V(4).Infof("create Backup: [%s/%s] for backupSchedule/%s successfully", ns, backupName, bsName)
	}
	rbc.recordBackupEvent("create", backup, err)
	return backup, err
}

func (rbc *realBackupControl) DeleteBackup(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	backupName := backup.GetName()

	bsName := backup.GetLabels()[label.BackupScheduleLabelKey]
	err := rbc.cli.PingcapV1alpha1().Backups(ns).Delete(backupName, nil)
	if err != nil {
		klog.Errorf("failed to delete Backup: [%s/%s] for backupSchedule/%s, err: %v", ns, backupName, bsName, err)
	} else {
		klog.V(4).Infof("delete backup: [%s/%s] successfully, backupSchedule/%s", ns, backupName, bsName)
	}
	rbc.recordBackupEvent("delete", backup, err)
	return err
}

func (rbc *realBackupControl) recordBackupEvent(verb string, backup *v1alpha1.Backup, err error) {
	backupName := backup.GetName()
	ns := backup.GetNamespace()

	bsName := backup.GetLabels()[label.BackupScheduleLabelKey]
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Backup %s/%s for backupSchedule/%s successful",
			strings.ToLower(verb), ns, backupName, bsName)
		rbc.recorder.Event(backup, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Backup %s/%s for backupSchedule/%s failed error: %s",
			strings.ToLower(verb), ns, backupName, bsName, err)
		rbc.recorder.Event(backup, corev1.EventTypeWarning, reason, msg)
	}
}

var _ BackupControlInterface = &realBackupControl{}

// FakeBackupControl is a fake BackupControlInterface
type FakeBackupControl struct {
	backupLister        listers.BackupLister
	backupIndexer       cache.Indexer
	createBackupTracker RequestTracker
	deleteBackupTracker RequestTracker
}

// NewFakeBackupControl returns a FakeBackupControl
func NewFakeBackupControl(backupInformer informers.BackupInformer) *FakeBackupControl {
	return &FakeBackupControl{
		backupInformer.Lister(),
		backupInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
	}
}

// SetCreateBackupError sets the error attributes of createBackupTracker
func (fbc *FakeBackupControl) SetCreateBackupError(err error, after int) {
	fbc.createBackupTracker.SetError(err).SetAfter(after)
}

// SetDeleteBackupError sets the error attributes of deleteBackupTracker
func (fbc *FakeBackupControl) SetDeleteBackupError(err error, after int) {
	fbc.deleteBackupTracker.SetError(err).SetAfter(after)
}

// CreateBackup adds the backup to BackupIndexer
func (fbc *FakeBackupControl) CreateBackup(backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	defer fbc.createBackupTracker.Inc()
	if fbc.createBackupTracker.ErrorReady() {
		defer fbc.createBackupTracker.Reset()
		return backup, fbc.createBackupTracker.GetError()
	}

	return backup, fbc.backupIndexer.Add(backup)
}

// DeleteBackup deletes the backup from BackupIndexer
func (fbc *FakeBackupControl) DeleteBackup(backup *v1alpha1.Backup) error {
	defer fbc.createBackupTracker.Inc()
	if fbc.createBackupTracker.ErrorReady() {
		defer fbc.createBackupTracker.Reset()
		return fbc.createBackupTracker.GetError()
	}

	return fbc.backupIndexer.Delete(backup)
}

var _ BackupControlInterface = &FakeBackupControl{}
