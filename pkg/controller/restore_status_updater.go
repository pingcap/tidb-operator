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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// RestoreUpdateStatus represents the status of a restore to be updated.
// This structure should keep synced with the fields in `RestoreStatus`
// except for `Phase` and `Conditions`.
type RestoreUpdateStatus struct {
	// TimeStarted is the time at which the restore was started.
	TimeStarted *metav1.Time
	// TimeCompleted is the time at which the restore was completed.
	TimeCompleted *metav1.Time
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs *string
}

// RestoreConditionUpdaterInterface enables updating Restore conditions.
type RestoreConditionUpdaterInterface interface {
	Update(restore *v1alpha1.Restore, condition *v1alpha1.RestoreCondition, newStatus *RestoreUpdateStatus) error
}

type realRestoreConditionUpdater struct {
	cli           versioned.Interface
	restoreLister listers.RestoreLister
	recorder      record.EventRecorder
}

// returns a RestoreConditionUpdaterInterface that updates the Status of a Restore,
func NewRealRestoreConditionUpdater(
	cli versioned.Interface,
	restoreLister listers.RestoreLister,
	recorder record.EventRecorder) RestoreConditionUpdaterInterface {
	return &realRestoreConditionUpdater{
		cli:           cli,
		restoreLister: restoreLister,
		recorder:      recorder,
	}
}

func (u *realRestoreConditionUpdater) Update(restore *v1alpha1.Restore, condition *v1alpha1.RestoreCondition, newStatus *RestoreUpdateStatus) error {
	ns := restore.GetNamespace()
	restoreName := restore.GetName()
	var isUpdate bool
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updateRestoreStatus(&restore.Status, newStatus)
		isUpdate = v1alpha1.UpdateRestoreCondition(&restore.Status, condition)
		if isUpdate {
			_, updateErr := u.cli.PingcapV1alpha1().Restores(ns).Update(restore)
			if updateErr == nil {
				klog.Infof("Restore: [%s/%s] updated successfully", ns, restoreName)
				return nil
			}
			klog.Errorf("Failed to update resotre [%s/%s], error: %v", ns, restoreName, updateErr)
			if updated, err := u.restoreLister.Restores(ns).Get(restoreName); err == nil {
				// make a copy so we don't mutate the shared cache
				restore = updated.DeepCopy()
			} else {
				utilruntime.HandleError(fmt.Errorf("error getting updated restore %s/%s from lister: %v", ns, restoreName, err))
			}
			return updateErr
		}
		return nil
	})
	return err
}

// updateRestoreStatus updates existing Restore status
// from the fields in RestoreUpdateStatus.
func updateRestoreStatus(status *v1alpha1.RestoreStatus, newStatus *RestoreUpdateStatus) {
	if newStatus == nil {
		return
	}
	if newStatus.TimeStarted != nil {
		status.TimeStarted = *newStatus.TimeStarted
	}
	if newStatus.TimeCompleted != nil {
		status.TimeCompleted = *newStatus.TimeCompleted
	}
	if newStatus.CommitTs != nil {
		status.CommitTs = *newStatus.CommitTs
	}
}

var _ RestoreConditionUpdaterInterface = &realRestoreConditionUpdater{}

// FakeRestoreConditionUpdater is a fake RestoreConditionUpdaterInterface
type FakeRestoreConditionUpdater struct {
	RestoreLister        listers.RestoreLister
	RestoreIndexer       cache.Indexer
	updateRestoreTracker RequestTracker
}

// NewFakeRestoreConditionUpdater returns a FakeRestoreConditionUpdater
func NewFakeRestoreConditionUpdater(restoreInformer informers.RestoreInformer) *FakeRestoreConditionUpdater {
	return &FakeRestoreConditionUpdater{
		restoreInformer.Lister(),
		restoreInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateRestoreError sets the error attributes of updateRestoreTracker
func (u *FakeRestoreConditionUpdater) SetUpdateRestoreError(err error, after int) {
	u.updateRestoreTracker.SetError(err).SetAfter(after)
}

// UpdateRestore updates the Restore
func (u *FakeRestoreConditionUpdater) Update(restore *v1alpha1.Restore, _ *v1alpha1.RestoreCondition, _ *RestoreUpdateStatus) error {
	defer u.updateRestoreTracker.Inc()
	if u.updateRestoreTracker.ErrorReady() {
		defer u.updateRestoreTracker.Reset()
		return u.updateRestoreTracker.GetError()
	}

	return u.RestoreIndexer.Update(restore)
}

var _ RestoreConditionUpdaterInterface = &FakeRestoreConditionUpdater{}
