// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package restore

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/util"
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
	// ProgressStep the step name of progress.
	ProgressStep *string
	// Progress is the step's progress value.
	Progress *int
	// ProgressUpdateTime is the progress update time.
	ProgressUpdateTime *metav1.Time
}

// RestoreConditionUpdaterInterface enables updating Restore conditions.
type RestoreConditionUpdaterInterface interface {
	Update(ctx context.Context, restore *v1alpha1.Restore, condition *metav1.Condition, newStatus *RestoreUpdateStatus) error
}

type realRestoreConditionUpdater struct {
	cli      client.Client
	recorder record.EventRecorder
}

// returns a RestoreConditionUpdaterInterface that updates the Status of a Restore,
func NewRealRestoreConditionUpdater(
	cli client.Client,
	recorder record.EventRecorder) RestoreConditionUpdaterInterface {
	return &realRestoreConditionUpdater{
		cli:      cli,
		recorder: recorder,
	}
}

func (u *realRestoreConditionUpdater) Update(ctx context.Context, restore *v1alpha1.Restore, condition *metav1.Condition, newStatus *RestoreUpdateStatus) error {
	logger := log.FromContext(ctx)
	// reason is required so that we do set if it's empty
	if condition != nil {
		if condition.Reason == "" {
			condition.Reason = condition.Type
		}
	}

	ns := restore.GetNamespace()
	restoreName := restore.GetName()
	var isStatusUpdate bool
	var isConditionUpdate bool
	// try best effort to guarantee restore is updated.
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		updated := &v1alpha1.Restore{}
		// Always get the latest restore before update.
		if err := u.cli.Get(ctx, client.ObjectKey{Namespace: ns, Name: restoreName}, updated); err == nil {
			// make a copy so we don't mutate the shared cache
			restore = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated restore %s/%s from lister: %w", ns, restoreName, err))
			return err
		}
		isStatusUpdate = updateRestoreStatus(&restore.Status, newStatus)
		isConditionUpdate = v1alpha1.UpdateRestoreCondition(&restore.Status, condition)
		if isStatusUpdate || isConditionUpdate {
			updateErr := u.cli.Status().Update(ctx, restore)
			if updateErr == nil {
				logger.Info("Restore updated successfully", "namespace", ns, "restore", restoreName)
				return nil
			}
			logger.Error(updateErr, "Failed to update restore", "namespace", ns, "restore", restoreName)
			return updateErr
		}
		return nil
	})
	return err
}

// updateRestoreStatus updates existing Restore status
// from the fields in RestoreUpdateStatus.
func updateRestoreStatus(status *v1alpha1.RestoreStatus, newStatus *RestoreUpdateStatus) bool {
	if newStatus == nil {
		return false
	}
	isUpdate := false
	if newStatus.TimeStarted != nil && status.TimeStarted != *newStatus.TimeStarted {
		status.TimeStarted = *newStatus.TimeStarted
		isUpdate = true
	}
	if newStatus.TimeCompleted != nil && status.TimeCompleted != *newStatus.TimeCompleted {
		status.TimeCompleted = *newStatus.TimeCompleted
		status.TimeTaken = status.TimeCompleted.Sub(status.TimeStarted.Time).Round(time.Second).String()
		isUpdate = true
	}
	if newStatus.CommitTs != nil && status.CommitTs != *newStatus.CommitTs {
		status.CommitTs = *newStatus.CommitTs
		isUpdate = true
	}
	if newStatus.ProgressStep != nil {
		progresses, updated := util.UpdateBRProgress(status.Progresses, newStatus.ProgressStep, newStatus.Progress, newStatus.ProgressUpdateTime)
		if updated {
			status.Progresses = progresses
			isUpdate = true
		}
	}

	return isUpdate
}

var _ RestoreConditionUpdaterInterface = &realRestoreConditionUpdater{}
