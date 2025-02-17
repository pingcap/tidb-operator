package restore

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
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
	Update(restore *v1alpha1.Restore, condition *metav1.Condition, newStatus *RestoreUpdateStatus) error
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

func (u *realRestoreConditionUpdater) Update(restore *v1alpha1.Restore, condition *metav1.Condition, newStatus *RestoreUpdateStatus) error {
	ns := restore.GetNamespace()
	restoreName := restore.GetName()
	var isStatusUpdate bool
	var isConditionUpdate bool
	// try best effort to guarantee restore is updated.
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		updated := &v1alpha1.Restore{}
		// Always get the latest restore before update.
		if err := u.cli.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: restoreName}, updated); err == nil {
			// make a copy so we don't mutate the shared cache
			restore = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated restore %s/%s from lister: %v", ns, restoreName, err))
			return err
		}
		isStatusUpdate = updateRestoreStatus(&restore.Status, newStatus)
		isConditionUpdate = v1alpha1.UpdateRestoreCondition(&restore.Status, condition)
		if isStatusUpdate || isConditionUpdate {
			updateErr := u.cli.Update(context.TODO(), restore)
			if updateErr == nil {
				klog.Infof("Restore: [%s/%s] updated successfully", ns, restoreName)
				return nil
			}
			klog.Errorf("Failed to update restore [%s/%s], error: %v", ns, restoreName, updateErr)
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
		progresses, updated := updateBRProgress(status.Progresses, newStatus.ProgressStep, newStatus.Progress, newStatus.ProgressUpdateTime)
		if updated {
			status.Progresses = progresses
			isUpdate = true
		}
	}

	return isUpdate
}

// FIXME(ideascf): this function is copied from backup_status_updater.go, we should refactor it.
// updateBRProgress updates progress for backup/restore.
func updateBRProgress(progresses []v1alpha1.Progress, step *string, progress *int, updateTime *metav1.Time) ([]v1alpha1.Progress, bool) {
	var oldProgress *v1alpha1.Progress
	for i, p := range progresses {
		if p.Step == *step {
			oldProgress = &progresses[i]
			break
		}
	}

	makeSureLastProgressOver := func() {
		size := len(progresses)
		if size == 0 || progresses[size-1].Progress >= 100 {
			return
		}
		progresses[size-1].Progress = 100
		progresses[size-1].LastTransitionTime = metav1.Time{Time: time.Now()}
	}

	// no such progress, will new
	if oldProgress == nil {
		makeSureLastProgressOver()
		progresses = append(progresses, v1alpha1.Progress{
			Step:               *step,
			Progress:           *progress,
			LastTransitionTime: *updateTime,
		})
		return progresses, true
	}

	isUpdate := false
	if oldProgress.Progress < *progress {
		oldProgress.Progress = *progress
		isUpdate = true
	}

	if oldProgress.LastTransitionTime != *updateTime {
		oldProgress.LastTransitionTime = *updateTime
		isUpdate = true
	}

	return progresses, isUpdate
}

var _ RestoreConditionUpdaterInterface = &realRestoreConditionUpdater{}
