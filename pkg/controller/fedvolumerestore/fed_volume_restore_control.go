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

package fedvolumerestore

import (
	"context"
	"fmt"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
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

// ControlInterface implements the control logic for updating VolumeRestore
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateRestore implements the control logic for VolumeRestore's creation, deletion
	UpdateRestore(volumeRestore *v1alpha1.VolumeRestore) error
	// UpdateStatus updates the status for a VolumeRestore
	UpdateStatus(volumeRestore *v1alpha1.VolumeRestore, newStatus *v1alpha1.VolumeRestoreStatus) error
}

// NewDefaultVolumeRestoreControl returns a new instance of the default VolumeRestore ControlInterface implementation.
func NewDefaultVolumeRestoreControl(
	cli versioned.Interface,
	restoreManager fedvolumebackup.RestoreManager) ControlInterface {
	return &defaultRestoreControl{
		cli,
		restoreManager,
	}
}

type defaultRestoreControl struct {
	cli            versioned.Interface
	restoreManager fedvolumebackup.RestoreManager
}

// UpdateRestore executes the core logic loop for a VolumeRestore.
func (c *defaultRestoreControl) UpdateRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	volumeRestore.SetGroupVersionKind(controller.FedVolumeRestoreControllerKind)
	if err := c.addProtectionFinalizer(volumeRestore); err != nil {
		return err
	}

	if err := c.removeProtectionFinalizer(volumeRestore); err != nil {
		return err
	}

	return c.updateRestore(volumeRestore)
}

// UpdateStatus updates the status for a VolumeRestore
func (c *defaultRestoreControl) UpdateStatus(volumeRestore *v1alpha1.VolumeRestore, newStatus *v1alpha1.VolumeRestoreStatus) error {
	return c.restoreManager.UpdateStatus(volumeRestore, newStatus)
}

func (c *defaultRestoreControl) updateRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	oldStatus := volumeRestore.Status.DeepCopy()
	ns := volumeRestore.GetNamespace()
	name := volumeRestore.GetName()

	err := c.restoreManager.Sync(volumeRestore)
	if err != nil {
		klog.Warningf("VolumeRestore %s/%s sync error: %s", ns, name, err.Error())
	}

	if !apiequality.Semantic.DeepEqual(oldStatus, &volumeRestore.Status) {
		klog.Infof("VolumeRestore %s/%s update status, old: %+v, new: %+v", ns, name, oldStatus, volumeRestore.Status)
		if sErr := c.restoreManager.UpdateStatus(volumeRestore, &volumeRestore.Status); sErr != nil {
			klog.Warningf("VolumeRestore %s/%s update status error: %s", ns, name, err.Error())
			if err == nil {
				err = sErr
			}
		}
	}

	return err
}

// addProtectionFinalizer will be called when the VolumeBackup CR is created
func (c *defaultRestoreControl) addProtectionFinalizer(volumeRestore *v1alpha1.VolumeRestore) error {
	ns := volumeRestore.GetNamespace()
	name := volumeRestore.GetName()

	if needToAddFinalizer(volumeRestore) {
		volumeRestore.Finalizers = append(volumeRestore.Finalizers, label.VolumeRestoreFederationFinalizer)
		_, err := c.cli.FederationV1alpha1().VolumeRestores(ns).Update(context.TODO(), volumeRestore, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("add VolumeRestore %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
	}
	return nil
}

func (c *defaultRestoreControl) removeProtectionFinalizer(volumeRestore *v1alpha1.VolumeRestore) error {
	ns := volumeRestore.GetNamespace()
	name := volumeRestore.GetName()

	if needToRemoveFinalizer(volumeRestore) {
		volumeRestore.Finalizers = slice.RemoveString(volumeRestore.Finalizers, label.VolumeRestoreFederationFinalizer, nil)
		_, err := c.cli.FederationV1alpha1().VolumeRestores(ns).Update(context.TODO(), volumeRestore, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("remove VolumeRestore %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
		klog.Infof("remove VolumeRestore %s/%s protection finalizers success", ns, name)
		return controller.RequeueErrorf(fmt.Sprintf("VolumeRestore %s/%s has been cleaned up", ns, name))
	}
	return nil
}

func needToAddFinalizer(volumeRestore *v1alpha1.VolumeRestore) bool {
	return volumeRestore.DeletionTimestamp == nil &&
		!slice.ContainsString(volumeRestore.Finalizers, label.VolumeRestoreFederationFinalizer, nil)
}

func needToRemoveFinalizer(volumeRestore *v1alpha1.VolumeRestore) bool {
	return isDeletionCandidate(volumeRestore) && v1alpha1.IsVolumeRestoreCleaned(volumeRestore)
}

func isDeletionCandidate(volumeRestore *v1alpha1.VolumeRestore) bool {
	return volumeRestore.DeletionTimestamp != nil && slice.ContainsString(volumeRestore.Finalizers, label.VolumeRestoreFederationFinalizer, nil)
}

var _ ControlInterface = &defaultRestoreControl{}

// FakeRestoreControl is a fake RestoreControlInterface
type FakeRestoreControl struct {
	restoreIndexer       cache.Indexer
	updateRestoreTracker controller.RequestTracker
	status               *v1alpha1.VolumeRestoreStatus
}

// NewFakeRestoreControl returns a FakeRestoreControl
func NewFakeRestoreControl(restoreInformer informers.VolumeRestoreInformer) *FakeRestoreControl {
	return &FakeRestoreControl{
		restoreInformer.Informer().GetIndexer(),
		controller.RequestTracker{},
		nil,
	}
}

// SetUpdateRestoreError sets the error attributes of updateRestoreTracker
func (c *FakeRestoreControl) SetUpdateRestoreError(err error, after int) {
	c.updateRestoreTracker.SetError(err).SetAfter(after)
}

// UpdateRestore adds the VolumeRestore to RestoreIndexer
func (c *FakeRestoreControl) UpdateRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	defer c.updateRestoreTracker.Inc()
	if c.updateRestoreTracker.ErrorReady() {
		defer c.updateRestoreTracker.Reset()
		return c.updateRestoreTracker.GetError()
	}

	return c.restoreIndexer.Add(volumeRestore)
}

// UpdateStatus updates the condition for a VolumeRestore, include condition and status info
func (c *FakeRestoreControl) UpdateStatus(_ *v1alpha1.VolumeRestore, newStatus *v1alpha1.VolumeRestoreStatus) error {
	c.status = newStatus
	return nil
}

var _ ControlInterface = &FakeRestoreControl{}
