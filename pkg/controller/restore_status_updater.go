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

// RestoreStatusUpdaterInterface is an interface used to update the RestoreStatus associated with a Restore.
// For any use other than testing, clients should create an instance using NewRealRestoreStatusUpdater.
type RestoreStatusUpdaterInterface interface {
	// UpdateRestoreStatus sets the restore's Status to status. Implementations are required to retry on conflicts,
	// but fail on other errors. If the returned error is nil restore's Status has been successfully set to status.
	UpdateRestoreStatus(*v1alpha1.Restore, *v1alpha1.RestoreStatus, *v1alpha1.RestoreStatus) error
}

// returns a RestoreStatusUpdaterInterface that updates the Status of a Restore,
// using the supplied client and restoreLister.
func NewRealRestoreStatusUpdater(
	cli versioned.Interface,
	restoreLister listers.RestoreLister,
	recorder record.EventRecorder) RestoreStatusUpdaterInterface {
	return &realRestoreStatusUpdater{
		cli,
		restoreLister,
		recorder,
	}
}

type realRestoreStatusUpdater struct {
	cli           versioned.Interface
	restoreLister listers.RestoreLister
	recorder      record.EventRecorder
}

func (bsu *realRestoreStatusUpdater) UpdateRestoreStatus(
	restore *v1alpha1.Restore,
	newStatus *v1alpha1.RestoreStatus,
	oldStatus *v1alpha1.RestoreStatus) error {

	ns := restore.GetNamespace()
	restoreName := restore.GetName()
	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, updateErr := bsu.cli.PingcapV1alpha1().Restores(ns).Update(restore)
		if updateErr == nil {
			glog.Infof("Restore: [%s/%s] updated successfully", ns, restoreName)
			return nil
		}
		if updated, err := bsu.restoreLister.Restores(ns).Get(restoreName); err == nil {
			// make a copy so we don't mutate the shared cache
			restore = updated.DeepCopy()
			restore.Status = *newStatus
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated restore %s/%s from lister: %v", ns, restoreName, err))
		}

		return updateErr
	})
	if !apiequality.Semantic.DeepEqual(newStatus, oldStatus) {
		bsu.recordRestoreEvent("update", restore, err)
	}
	return err
}

func (bsu *realRestoreStatusUpdater) recordRestoreEvent(verb string, restore *v1alpha1.Restore, err error) {
	restoreName := restore.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Restore %s successful",
			strings.ToLower(verb), restoreName)
		bsu.recorder.Event(restore, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Restore %s failed error: %s",
			strings.ToLower(verb), restoreName, err)
		bsu.recorder.Event(restore, corev1.EventTypeWarning, reason, msg)
	}
}

var _ RestoreStatusUpdaterInterface = &realRestoreStatusUpdater{}

// FakeRestoreStatusUpdater is a fake RestoreStatusUpdaterInterface
type FakeRestoreStatusUpdater struct {
	RestoreLister        listers.RestoreLister
	RestoreIndexer       cache.Indexer
	updateRestoreTracker requestTracker
}

// NewFakeRestoreStatusUpdater returns a FakeRestoreStatusUpdater
func NewFakeRestoreStatusUpdater(restoreInformer informers.RestoreInformer) *FakeRestoreStatusUpdater {
	return &FakeRestoreStatusUpdater{
		restoreInformer.Lister(),
		restoreInformer.Informer().GetIndexer(),
		requestTracker{0, nil, 0},
	}
}

// SetUpdateRestoreError sets the error attributes of updateRestoreTracker
func (frs *FakeRestoreStatusUpdater) SetUpdateRestoreError(err error, after int) {
	frs.updateRestoreTracker.err = err
	frs.updateRestoreTracker.after = after
}

// UpdateRestore updates the Restore
func (frs *FakeRestoreStatusUpdater) UpdateRestoreStatus(restore *v1alpha1.Restore, _ *v1alpha1.RestoreStatus, _ *v1alpha1.RestoreStatus) error {
	defer frs.updateRestoreTracker.inc()
	if frs.updateRestoreTracker.errorReady() {
		defer frs.updateRestoreTracker.reset()
		return frs.updateRestoreTracker.err
	}

	return frs.RestoreIndexer.Update(restore)
}

var _ RestoreStatusUpdaterInterface = &FakeRestoreStatusUpdater{}
