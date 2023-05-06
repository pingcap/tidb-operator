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
	"k8s.io/client-go/tools/cache"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
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
	// UpdateCondition updates the condition for a VolumeRestore
	UpdateCondition(volumeRestore *v1alpha1.VolumeRestore, condition *v1alpha1.VolumeRestoreCondition) error
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
	return c.restoreManager.Sync(volumeRestore)
}

// UpdateCondition updates the condition for a VolumeRestore
func (c *defaultRestoreControl) UpdateCondition(volumeRestore *v1alpha1.VolumeRestore, condition *v1alpha1.VolumeRestoreCondition) error {
	return c.restoreManager.UpdateCondition(volumeRestore, condition)
}

var _ ControlInterface = &defaultRestoreControl{}

// FakeRestoreControl is a fake RestoreControlInterface
type FakeRestoreControl struct {
	restoreIndexer       cache.Indexer
	updateRestoreTracker controller.RequestTracker
	condition            *v1alpha1.VolumeRestoreCondition
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
func (c *FakeRestoreControl) UpdateCondition(_ *v1alpha1.VolumeRestore, condition *v1alpha1.VolumeRestoreCondition) error {
	c.condition = condition
	return nil
}

var _ ControlInterface = &FakeRestoreControl{}
