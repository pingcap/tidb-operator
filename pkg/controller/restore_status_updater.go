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

// RestoreConditionUpdaterInterface enables updating Restore conditions.
type RestoreConditionUpdaterInterface interface {
	Update(restore *v1alpha1.Restore, condition *v1alpha1.RestoreCondition) error
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
		cli,
		restoreLister,
		recorder,
	}
}

func (rcu *realRestoreConditionUpdater) Update(restore *v1alpha1.Restore, condition *v1alpha1.RestoreCondition) error {
	ns := restore.GetNamespace()
	restoreName := restore.GetName()
	oldStatus := restore.Status.DeepCopy()
	var isUpdate bool
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		isUpdate = v1alpha1.UpdateRestoreCondition(&restore.Status, condition)
		if isUpdate {
			_, updateErr := rcu.cli.PingcapV1alpha1().Restores(ns).Update(restore)
			if updateErr == nil {
				klog.Infof("Restore: [%s/%s] updated successfully", ns, restoreName)
				return nil
			}
			if updated, err := rcu.restoreLister.Restores(ns).Get(restoreName); err == nil {
				// make a copy so we don't mutate the shared cache
				restore = updated.DeepCopy()
				restore.Status = *oldStatus
			} else {
				utilruntime.HandleError(perrors.Errorf("error getting updated restore %s/%s from lister: %v", ns, restoreName, err))
			}
			return updateErr
		}
		return nil
	})
	return err
}

func (rcu *realRestoreConditionUpdater) recordRestoreEvent(verb string, restore *v1alpha1.Restore, err error) {
	restoreName := restore.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Restore %s successful",
			strings.ToLower(verb), restoreName)
		rcu.recorder.Event(restore, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Restore %s failed error: %s",
			strings.ToLower(verb), restoreName, err)
		rcu.recorder.Event(restore, corev1.EventTypeWarning, reason, msg)
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
func (frc *FakeRestoreConditionUpdater) SetUpdateRestoreError(err error, after int) {
	frc.updateRestoreTracker.SetError(err).SetAfter(after)
}

// UpdateRestore updates the Restore
func (frc *FakeRestoreConditionUpdater) Update(restore *v1alpha1.Restore, _ *v1alpha1.RestoreCondition) error {
	defer frc.updateRestoreTracker.Inc()
	if frc.updateRestoreTracker.ErrorReady() {
		defer frc.updateRestoreTracker.Reset()
		return frc.updateRestoreTracker.GetError()
	}

	return frc.RestoreIndexer.Update(restore)
}

var _ RestoreConditionUpdaterInterface = &FakeRestoreConditionUpdater{}
