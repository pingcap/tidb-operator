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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"

	perrors "github.com/pingcap/errors"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// BackupScheduleStatusUpdaterInterface is an interface used to update the BackupScheduleStatus associated with a BackupSchedule.
// For any use other than testing, clients should create an instance using NewRealBackupScheduleStatusUpdater.
type BackupScheduleStatusUpdaterInterface interface {
	// UpdateBackupScheduleStatus sets the backupSchedule's Status to status. Implementations are required to retry on conflicts,
	// but fail on other errors. If the returned error is nil backup's Status has been successfully set to status.
	UpdateBackupScheduleStatus(*v1alpha1.BackupSchedule, *v1alpha1.BackupScheduleStatus, *v1alpha1.BackupScheduleStatus) error
}

// returns a BackupScheduleStatusUpdaterInterface that updates the Status of a BackupSchedule,
// using the supplied client and bsLister.
func NewRealBackupScheduleStatusUpdater(
	cli versioned.Interface,
	bsLister listers.BackupScheduleLister,
	recorder record.EventRecorder) BackupScheduleStatusUpdaterInterface {
	return &realBackupScheduleStatusUpdater{
		cli,
		bsLister,
		recorder,
	}
}

type realBackupScheduleStatusUpdater struct {
	cli      versioned.Interface
	bsLister listers.BackupScheduleLister
	recorder record.EventRecorder
}

func (bss *realBackupScheduleStatusUpdater) UpdateBackupScheduleStatus(
	bs *v1alpha1.BackupSchedule,
	newStatus *v1alpha1.BackupScheduleStatus,
	oldStatus *v1alpha1.BackupScheduleStatus) error {

	ns := bs.GetNamespace()
	bsName := bs.GetName()
	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, updateErr := bss.cli.PingcapV1alpha1().BackupSchedules(ns).Update(bs)
		if updateErr == nil {
			klog.Infof("BackupSchedule: [%s/%s] updated successfully", ns, bsName)
			return nil
		}
		if updated, err := bss.bsLister.BackupSchedules(ns).Get(bsName); err == nil {
			// make a copy so we don't mutate the shared cache
			bs = updated.DeepCopy()
			bs.Status = *newStatus
		} else {
			utilruntime.HandleError(perrors.Errorf("error getting updated backupSchedule %s/%s from lister: %v", ns, bsName, err))
		}

		return updateErr
	})
	return err
}

func (bss *realBackupScheduleStatusUpdater) recordBackupScheduleEvent(verb string, bs *v1alpha1.BackupSchedule, err error) {
	bsName := bs.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s BackupSchedule %s successful",
			strings.ToLower(verb), bsName)
		bss.recorder.Event(bs, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s BackupSchedule %s failed error: %s",
			strings.ToLower(verb), bsName, err)
		bss.recorder.Event(bs, corev1.EventTypeWarning, reason, msg)
	}
}

var _ BackupScheduleStatusUpdaterInterface = &realBackupScheduleStatusUpdater{}

// FakeBackupScheduleStatusUpdater is a fake BackupScheduleStatusUpdaterInterface
type FakeBackupScheduleStatusUpdater struct {
	BsLister        listers.BackupScheduleLister
	BsIndexer       cache.Indexer
	updateBsTracker RequestTracker
}

// NewFakeBackupScheduleStatusUpdater returns a FakeBackupScheduleStatusUpdater
func NewFakeBackupScheduleStatusUpdater(bsInformer informers.BackupScheduleInformer) *FakeBackupScheduleStatusUpdater {
	return &FakeBackupScheduleStatusUpdater{
		bsInformer.Lister(),
		bsInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupScheduleTracker
func (fbs *FakeBackupScheduleStatusUpdater) SetUpdateBackupScheduleError(err error, after int) {
	fbs.updateBsTracker.err = err
	fbs.updateBsTracker.after = after
	fbs.updateBsTracker.SetError(err).SetAfter(after)
}

// UpdateBackupSchedule updates the BackupSchedule
func (fbs *FakeBackupScheduleStatusUpdater) UpdateBackupScheduleStatus(bs *v1alpha1.BackupSchedule, _ *v1alpha1.BackupScheduleStatus, _ *v1alpha1.BackupScheduleStatus) error {
	defer fbs.updateBsTracker.Inc()
	if fbs.updateBsTracker.ErrorReady() {
		defer fbs.updateBsTracker.Reset()
		return fbs.updateBsTracker.GetError()
	}

	return fbs.BsIndexer.Update(bs)
}

var _ BackupScheduleStatusUpdaterInterface = &FakeBackupScheduleStatusUpdater{}
