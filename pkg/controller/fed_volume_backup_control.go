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

package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/federation/listers/pingcap/v1alpha1"
)

// FedVolumeBackupControlInterface manages federation VolumeBackups used in VolumeBackupSchedule
type FedVolumeBackupControlInterface interface {
	CreateVolumeBackup(backup *v1alpha1.VolumeBackup) (*v1alpha1.VolumeBackup, error)
	DeleteVolumeBackup(backup *v1alpha1.VolumeBackup) error
}

type realFedVolumeBackupControl struct {
	cli      versioned.Interface
	recorder record.EventRecorder
}

// NewRealFedVolumeBackupControl creates a new FedVolumeBackupControlInterface
func NewRealFedVolumeBackupControl(
	cli versioned.Interface,
	recorder record.EventRecorder,
) FedVolumeBackupControlInterface {
	return &realFedVolumeBackupControl{
		cli:      cli,
		recorder: recorder,
	}
}

func (c *realFedVolumeBackupControl) CreateVolumeBackup(volumeBackup *v1alpha1.VolumeBackup) (*v1alpha1.VolumeBackup, error) {
	ns := volumeBackup.GetNamespace()
	backupName := volumeBackup.GetName()

	bsName := volumeBackup.GetLabels()[label.BackupScheduleLabelKey]
	volumeBackup, err := c.cli.FederationV1alpha1().VolumeBackups(ns).Create(context.TODO(), volumeBackup, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("failed to create VolumeBackup: [%s/%s] for volumeBackupSchedule/%s, err: %v", ns, backupName, bsName, err)
	} else {
		klog.V(4).Infof("create VolumeBackup: [%s/%s] for volumeBackupSchedule/%s successfully", ns, backupName, bsName)
	}
	c.recordVolumeBackupEvent("create", volumeBackup, err)
	return volumeBackup, err
}

func (c *realFedVolumeBackupControl) DeleteVolumeBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	ns := volumeBackup.GetNamespace()
	backupName := volumeBackup.GetName()

	bsName := volumeBackup.GetLabels()[label.BackupScheduleLabelKey]
	err := c.cli.FederationV1alpha1().VolumeBackups(ns).Delete(context.TODO(), backupName, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("failed to delete VolumeBackup: [%s/%s] for volumeBackupSchedule/%s, err: %v", ns, backupName, bsName, err)
	} else {
		klog.V(4).Infof("delete Volumebackup: [%s/%s] successfully, volumeBackupSchedule/%s", ns, backupName, bsName)
	}
	c.recordVolumeBackupEvent("delete", volumeBackup, err)
	return err
}

func (c *realFedVolumeBackupControl) recordVolumeBackupEvent(verb string, volumeBackup *v1alpha1.VolumeBackup, err error) {
	backupName := volumeBackup.GetName()
	ns := volumeBackup.GetNamespace()

	bsName := volumeBackup.GetLabels()[label.BackupScheduleLabelKey]
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s VolumeBackup %s/%s for volumeBackupSchedule/%s successful",
			strings.ToLower(verb), ns, backupName, bsName)
		c.recorder.Event(volumeBackup, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s VolumeBackup %s/%s for volumeBackupSchedule/%s failed error: %s",
			strings.ToLower(verb), ns, backupName, bsName, err)
		c.recorder.Event(volumeBackup, corev1.EventTypeWarning, reason, msg)
	}
}

var _ FedVolumeBackupControlInterface = &realFedVolumeBackupControl{}

// FakeFedVolumeBackupControl is a fake FedVolumeBackupControlInterface
type FakeFedVolumeBackupControl struct {
	volumeBackupLister        listers.VolumeBackupLister
	volumeBackupIndexer       cache.Indexer
	createVolumeBackupTracker RequestTracker
	deleteVolumeBackupTracker RequestTracker
}

// NewFakeFedVolumeBackupControl returns a FakeFedVolumeBackupControl
func NewFakeFedVolumeBackupControl(volumeBackupInformer informers.VolumeBackupInformer) *FakeFedVolumeBackupControl {
	return &FakeFedVolumeBackupControl{
		volumeBackupInformer.Lister(),
		volumeBackupInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
	}
}

// SetCreateVolumeBackupError sets the error attributes of createVolumeBackupTracker
func (fbc *FakeFedVolumeBackupControl) SetCreateVolumeBackupError(err error, after int) {
	fbc.createVolumeBackupTracker.SetError(err).SetAfter(after)
}

// SetDeleteVolumeBackupError sets the error attributes of deleteVolumeBackupTracker
func (fbc *FakeFedVolumeBackupControl) SetDeleteVolumeBackupError(err error, after int) {
	fbc.deleteVolumeBackupTracker.SetError(err).SetAfter(after)
}

// CreateVolumeBackup adds the volumeBackup to VolumeBackupIndexer
func (fbc *FakeFedVolumeBackupControl) CreateVolumeBackup(volumeBackup *v1alpha1.VolumeBackup) (*v1alpha1.VolumeBackup, error) {
	defer fbc.createVolumeBackupTracker.Inc()
	if fbc.createVolumeBackupTracker.ErrorReady() {
		defer fbc.createVolumeBackupTracker.Reset()
		return volumeBackup, fbc.createVolumeBackupTracker.GetError()
	}

	return volumeBackup, fbc.volumeBackupIndexer.Add(volumeBackup)
}

// DeleteVolumeBackup deletes the volumeBackup from VolumeBackupIndexer
func (fbc *FakeFedVolumeBackupControl) DeleteVolumeBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	defer fbc.createVolumeBackupTracker.Inc()
	if fbc.createVolumeBackupTracker.ErrorReady() {
		defer fbc.createVolumeBackupTracker.Reset()
		return fbc.createVolumeBackupTracker.GetError()
	}

	return fbc.volumeBackupIndexer.Delete(volumeBackup)
}

var _ BackupControlInterface = &FakeBackupControl{}
