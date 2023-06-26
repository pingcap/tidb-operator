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
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/federation/listers/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// VolumeBackupScheduleStatusUpdaterInterface is an interface used to update the VolumeBackupScheduleStatus associated with a VolumeBackupSchedule.
// For any use other than testing, clients should create an instance using NewRealBackupScheduleStatusUpdater.
type VolumeBackupScheduleStatusUpdaterInterface interface {
	// UpdateBackupScheduleStatus sets the backupSchedule's Status to status. Implementations are required to retry on conflicts,
	// but fail on other errors. If the returned error is nil backup's Status has been successfully set to status.
	UpdateBackupScheduleStatus(*v1alpha1.VolumeBackupSchedule, *v1alpha1.VolumeBackupScheduleStatus, *v1alpha1.VolumeBackupScheduleStatus) error
}

// NewRealVolumeBackupScheduleStatusUpdater returns a VolumeBackupScheduleStatusUpdaterInterface that updates the Status of a VolumeBackupScheduleStatus,
// using the supplied client and bsLister.
func NewRealVolumeBackupScheduleStatusUpdater(deps *BrFedDependencies) VolumeBackupScheduleStatusUpdaterInterface {
	return &realVolumeBackupScheduleStatusUpdater{
		deps: deps,
	}
}

type realVolumeBackupScheduleStatusUpdater struct {
	deps *BrFedDependencies
}

func (u *realVolumeBackupScheduleStatusUpdater) UpdateBackupScheduleStatus(
	bs *v1alpha1.VolumeBackupSchedule,
	newStatus *v1alpha1.VolumeBackupScheduleStatus,
	oldStatus *v1alpha1.VolumeBackupScheduleStatus) error {

	ns := bs.GetNamespace()
	bsName := bs.GetName()
	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, updateErr := u.deps.Clientset.FederationV1alpha1().VolumeBackupSchedules(ns).Update(context.TODO(), bs, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("VolumeBackupSchedule: [%s/%s] updated successfully", ns, bsName)
			return nil
		}
		if updated, err := u.deps.VolumeBackupScheduleLister.VolumeBackupSchedules(ns).Get(bsName); err == nil {
			// make a copy so we don't mutate the shared cache
			bs = updated.DeepCopy()
			bs.Status = *newStatus
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated backupSchedule %s/%s from lister: %v", ns, bsName, err))
		}

		return updateErr
	})
	return err
}

var _ VolumeBackupScheduleStatusUpdaterInterface = &realVolumeBackupScheduleStatusUpdater{}

// FakeVolumeBackupScheduleStatusUpdater is a fake VolumeBackupScheduleStatusUpdaterInterface
type FakeVolumeBackupScheduleStatusUpdater struct {
	BsLister        listers.VolumeBackupScheduleLister
	BsIndexer       cache.Indexer
	updateBsTracker RequestTracker
}

// NewFakeVolumeBackupScheduleStatusUpdater returns a FakeVolumeBackupScheduleStatusUpdater
func NewFakeVolumeBackupScheduleStatusUpdater(bsInformer informers.VolumeBackupScheduleInformer) *FakeVolumeBackupScheduleStatusUpdater {
	return &FakeVolumeBackupScheduleStatusUpdater{
		bsInformer.Lister(),
		bsInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateBackupScheduleError sets the error attributes of updateBackupScheduleTracker
func (u *FakeVolumeBackupScheduleStatusUpdater) SetUpdateBackupScheduleError(err error, after int) {
	u.updateBsTracker.err = err
	u.updateBsTracker.after = after
	u.updateBsTracker.SetError(err).SetAfter(after)
}

// UpdateBackupScheduleStatus updates the BackupSchedule
func (u *FakeVolumeBackupScheduleStatusUpdater) UpdateBackupScheduleStatus(bs *v1alpha1.VolumeBackupSchedule, _ *v1alpha1.VolumeBackupScheduleStatus, _ *v1alpha1.VolumeBackupScheduleStatus) error {
	defer u.updateBsTracker.Inc()
	if u.updateBsTracker.ErrorReady() {
		defer u.updateBsTracker.Reset()
		return u.updateBsTracker.GetError()
	}

	return u.BsIndexer.Update(bs)
}

var _ VolumeBackupScheduleStatusUpdaterInterface = &FakeVolumeBackupScheduleStatusUpdater{}
