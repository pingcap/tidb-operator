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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

// ControlInterface implements the control logic for updating Restore
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateRestore implements the control logic for restore job creation, update, and deletion
	UpdateRestore(schedule *v1alpha1.Restore) error
}

// NewDefaultRestoreControl returns a new instance of the default implementation RestoreControlInterface that
// implements the documented semantics for Restore.
func NewDefaultRestoreControl(
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreManager backup.RestoreManager,
	recorder record.EventRecorder) ControlInterface {
	return &defaultRestoreControl{
		statusUpdater,
		restoreManager,
		recorder,
	}
}

type defaultRestoreControl struct {
	statusUpdater  controller.RestoreConditionUpdaterInterface
	restoreManager backup.RestoreManager
	recorder       record.EventRecorder
}

// UpdateRestore executes the core logic loop for a Restore.
func (rc *defaultRestoreControl) UpdateRestore(restore *v1alpha1.Restore) error {
	var errs []error
	oldStatus := restore.Status.DeepCopy()

	if err := rc.updateRestore(restore); err != nil {
		errs = append(errs, err)
	}
	if apiequality.Semantic.DeepEqual(&restore.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}

	return errorutils.NewAggregate(errs)
}

func (rc *defaultRestoreControl) updateRestore(restore *v1alpha1.Restore) error {
	return rc.restoreManager.Sync(restore)
}
