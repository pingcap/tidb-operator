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

package restore

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/client-go/tools/cache"
)

// ControlInterface implements the control logic for updating Restore
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateRestore implements the control logic for restore job creation, update, and deletion
	UpdateRestore(restore *v1alpha1.Restore) error
}

// NewDefaultRestoreControl returns a new instance of the default implementation RestoreControlInterface that
// implements the documented semantics for Restore.
func NewDefaultRestoreControl(restoreManager backup.RestoreManager) ControlInterface {
	return &defaultRestoreControl{
		restoreManager,
	}
}

type defaultRestoreControl struct {
	restoreManager backup.RestoreManager
}

var _ ControlInterface = &defaultRestoreControl{}

// UpdateRestore executes the core logic loop for a Restore.
func (rc *defaultRestoreControl) UpdateRestore(restore *v1alpha1.Restore) error {
	restore.SetGroupVersionKind(controller.RestoreControllerKind)
	return rc.restoreManager.Sync(restore)
}

// FakeRestoreControl is a fake RestoreControlInterface
type FakeRestoreControl struct {
	backupIndexer        cache.Indexer
	updateRestoreTracker controller.RequestTracker
}

// NewFakeRestoreControl returns a FakeRestoreControl
func NewFakeRestoreControl(restoreInformer informers.RestoreInformer) *FakeRestoreControl {
	return &FakeRestoreControl{
		restoreInformer.Informer().GetIndexer(),
		controller.RequestTracker{},
	}
}

// SetUpdateRestoreError sets the error attributes of updateRestoreTracker
func (fbc *FakeRestoreControl) SetUpdateRestoreError(err error, after int) {
	fbc.updateRestoreTracker.SetError(err).SetAfter(after)
}

// UpdateRestore adds the backup to RestoreIndexer
func (fbc *FakeRestoreControl) UpdateRestore(backup *v1alpha1.Restore) error {
	defer fbc.updateRestoreTracker.Inc()
	if fbc.updateRestoreTracker.ErrorReady() {
		defer fbc.updateRestoreTracker.Reset()
		return fbc.updateRestoreTracker.GetError()
	}

	return fbc.backupIndexer.Add(backup)
}

var _ ControlInterface = &FakeRestoreControl{}
