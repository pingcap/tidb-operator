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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const (
	// progressDebounceDuration is the minimum time interval between two progress updates for a backup
	progressDebounceDuration = 3 * time.Second
)

type Progress struct {
	// MetaCompleted is the number of meta files compacted
	MetaCompleted uint64 `json:"meta_completed"`
	// MetaTotal is the total number of meta files
	MetaTotal uint64 `json:"meta_total"`
	// BytesToCompact is the number of bytes to compact
	BytesToCompact uint64 `json:"bytes_to_compact"`
	// BytesCompacted is the number of bytes compacted
	BytesCompacted uint64 `json:"bytes_compacted"`
}

type CompactStatusUpdaterInterface interface {
	OnSchedule(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	OnCreateJob(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	OnStart(ctx context.Context, compact *v1alpha1.CompactBackup) error
	OnProgress(ctx context.Context, compact *v1alpha1.CompactBackup, p *Progress, endTs string) error
	OnFinish(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	// OnJobComplete / OnJobFailed take sharded indexes directly. Callers pass empty
	// strings when the terminal transition has no observed Job (e.g. create-job retries
	// exhausted before any pod ran).
	OnJobComplete(ctx context.Context, compact *v1alpha1.CompactBackup, completedIndexes, failedIndexes string) error
	OnJobFailed(ctx context.Context, compact *v1alpha1.CompactBackup, reason, completedIndexes, failedIndexes string) error
	UpdateStatus(compact *v1alpha1.CompactBackup, newStatus v1alpha1.CompactStatus) error
	UpdateShardIndexes(compact *v1alpha1.CompactBackup, jobStatus batchv1.JobStatus) error
}

type CompactStatusUpdater struct {
	recorder           record.EventRecorder
	lister             listers.CompactBackupLister
	cli                versioned.Interface
	progressLastUpdate time.Time
}

func NewCompactStatusUpdater(recorder record.EventRecorder, lister listers.CompactBackupLister, cli versioned.Interface) *CompactStatusUpdater {
	return &CompactStatusUpdater{
		recorder: recorder,
		lister:   lister,
		cli:      cli,
	}
}

func (r *CompactStatusUpdater) Event(compact *v1alpha1.CompactBackup, ty, reason, msg string) {
	r.recorder.Event(compact, ty, reason, msg)
}

func (r *CompactStatusUpdater) UpdateStatus(compact *v1alpha1.CompactBackup, newStatus v1alpha1.CompactStatus) error {
	ns := compact.GetNamespace()
	compactName := compact.GetName()

	now := time.Now()
	canUpdateProgress := true
	if newStatus.Progress != "" {
		if now.Sub(r.progressLastUpdate) < progressDebounceDuration {
			canUpdateProgress = false
		}
	}

	// Update the status
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		// Always get the latest CompactBackup before updating
		if updated, err := r.lister.CompactBackups(ns).Get(compactName); err == nil {
			*compact = *(updated.DeepCopy())
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated compact %s/%s from lister: %v", ns, compactName, err))
			return err
		}

		updated := false
		if newStatus.State != "" && compact.Status.State != newStatus.State {
			compact.Status.State = newStatus.State
			updated = true
			canUpdateProgress = true
		}
		if newStatus.Message != "" && compact.Status.Message != newStatus.Message {
			compact.Status.Message = newStatus.Message
			updated = true
		}
		if canUpdateProgress && newStatus.Progress != "" && compact.Status.Progress != newStatus.Progress {
			compact.Status.Progress = newStatus.Progress
			updated = true
			r.progressLastUpdate = now
		}
		if newStatus.RetryStatus != nil && newStatus.RetryStatus[0].RetryNum == len(compact.Status.RetryStatus) {
			compact.Status.RetryStatus = append(compact.Status.RetryStatus, newStatus.RetryStatus[0])
			updated = true
		}
		if newStatus.EndTs != "" && compact.Status.EndTs < newStatus.EndTs {
			compact.Status.EndTs = newStatus.EndTs
			updated = true
		}
		// Empty string means "caller didn't observe a value for this field" (mirroring the
		// convention used by State/Message/Progress above). Non-terminal sharded-index
		// writes go through UpdateShardIndexes, which unconditionally overwrites.
		if newStatus.CompletedIndexes != "" && compact.Status.CompletedIndexes != newStatus.CompletedIndexes {
			compact.Status.CompletedIndexes = newStatus.CompletedIndexes
			updated = true
		}
		if newStatus.FailedIndexes != "" && compact.Status.FailedIndexes != newStatus.FailedIndexes {
			compact.Status.FailedIndexes = newStatus.FailedIndexes
			updated = true
		}
		// Apply the update if any field changed
		if updated {
			_, updateErr := r.cli.PingcapV1alpha1().CompactBackups(ns).Update(context.TODO(), compact, metav1.UpdateOptions{})
			if updateErr == nil {
				klog.Infof("Compact: [%s/%s] updated successfully", ns, compactName)
				return nil
			}
			klog.Errorf("Failed to update Compact [%s/%s], error: %v", ns, compactName, updateErr)
			return updateErr
		}
		return nil
	})
	return err
}

func (r *CompactStatusUpdater) UpdateShardIndexes(compact *v1alpha1.CompactBackup, jobStatus batchv1.JobStatus) error {
	ns := compact.GetNamespace()
	compactName := compact.GetName()

	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		if updated, err := r.lister.CompactBackups(ns).Get(compactName); err == nil {
			*compact = *(updated.DeepCopy())
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated compact %s/%s from lister: %v", ns, compactName, err))
			return err
		}

		updated := false
		if compact.Status.CompletedIndexes != jobStatus.CompletedIndexes {
			compact.Status.CompletedIndexes = jobStatus.CompletedIndexes
			updated = true
		}
		failedIndexes := ""
		if jobStatus.FailedIndexes != nil {
			failedIndexes = *jobStatus.FailedIndexes
		}
		if compact.Status.FailedIndexes != failedIndexes {
			compact.Status.FailedIndexes = failedIndexes
			updated = true
		}

		if updated {
			_, updateErr := r.cli.PingcapV1alpha1().CompactBackups(ns).Update(context.TODO(), compact, metav1.UpdateOptions{})
			if updateErr == nil {
				klog.Infof("Compact: [%s/%s] shard indexes updated successfully", ns, compactName)
				return nil
			}
			klog.Errorf("Failed to update shard indexes for Compact [%s/%s], error: %v", ns, compactName, updateErr)
			return updateErr
		}
		return nil
	})
	return err
}

func (r *CompactStatusUpdater) OnSchedule(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error {
	newStatus := v1alpha1.CompactStatus{}
	if err != nil {
		newStatus.State = string(v1alpha1.BackupFailed)
		newStatus.Message = err.Error()
	} else {
		newStatus.State = string(v1alpha1.BackupScheduled)
	}
	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnCreateJob(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error {
	newStatus := v1alpha1.CompactStatus{}
	if err != nil {
		newStatus.State = string(v1alpha1.BackupRetryTheFailed)
		newStatus.Message = err.Error()
		newStatus.RetryStatus = []v1alpha1.CompactRetryRecord{
			{
				RetryNum:       len(compact.Status.RetryStatus),
				DetectFailedAt: metav1.NewTime(time.Now()),
				RetryReason:    err.Error(),
			},
		}
	} else {
		newStatus.State = string(v1alpha1.BackupPrepare)
	}
	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnStart(ctx context.Context, compact *v1alpha1.CompactBackup) error {
	r.Event(compact, corev1.EventTypeNormal, "Started", "The compaction process has started successfully.")

	newStauts := v1alpha1.CompactStatus{
		State: string(v1alpha1.BackupRunning),
	}
	return r.UpdateStatus(compact, newStauts)
}

func (r *CompactStatusUpdater) OnProgress(ctx context.Context, compact *v1alpha1.CompactBackup, p *Progress, endTs string) error {
	newStatus := v1alpha1.CompactStatus{}
	if endTs != "" {
		newStatus.EndTs = endTs
	}
	if p != nil {
		progress := fmt.Sprintf("[READ_META(%d/%d),COMPACT_WORK(%d/%d)]",
			p.MetaCompleted, p.MetaTotal, p.BytesCompacted, p.BytesToCompact)

		newStatus.Progress = progress
	}

	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnFinish(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error {
	newStatus := v1alpha1.CompactStatus{}
	if err != nil {
		newStatus.State = string(v1alpha1.BackupRetryTheFailed)
		newStatus.Message = err.Error()
		newStatus.RetryStatus = []v1alpha1.CompactRetryRecord{
			{
				RetryNum:       len(compact.Status.RetryStatus),
				DetectFailedAt: metav1.NewTime(time.Now()),
				RetryReason:    err.Error(),
			},
		}
		r.Event(compact, corev1.EventTypeWarning, "Failed(Retryable)", err.Error())
	} else {
		newStatus.State = string(v1alpha1.BackupComplete)
		r.Event(compact, corev1.EventTypeNormal, "Finished", "The compaction process has finished successfully.")
	}
	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnJobComplete(ctx context.Context, compact *v1alpha1.CompactBackup, completedIndexes, failedIndexes string) error {
	r.Event(compact, corev1.EventTypeNormal, "Finished", "The compaction process has finished successfully.")

	newStatus := v1alpha1.CompactStatus{
		State:            string(v1alpha1.BackupComplete),
		CompletedIndexes: completedIndexes,
		FailedIndexes:    failedIndexes,
	}
	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnJobFailed(ctx context.Context, compact *v1alpha1.CompactBackup, reason, completedIndexes, failedIndexes string) error {
	newStatus := v1alpha1.CompactStatus{
		State:            string(v1alpha1.BackupFailed),
		Message:          reason,
		CompletedIndexes: completedIndexes,
		FailedIndexes:    failedIndexes,
	}
	r.Event(compact, corev1.EventTypeWarning, "The compact job is failed.", reason)
	return r.UpdateStatus(compact, newStatus)
}

// ShardedCompactStatusUpdater wraps a real CompactStatusUpdater for use by the
// backup-manager worker in sharded mode. Forwards OnStart (writing BackupRunning is
// idempotent and per-pod Events help observability); suppresses OnProgress (per-shard
// percentages would race over a single CR field) and OnFinish (any shard finishing
// would prematurely push terminal state—terminal writes are owned by the controller
// observing the Indexed Job). Other methods are not called from the worker; they
// default to no-op as defensive stubs.
type ShardedCompactStatusUpdater struct {
	real CompactStatusUpdaterInterface
}

func NewShardedCompactStatusUpdater(real CompactStatusUpdaterInterface) *ShardedCompactStatusUpdater {
	return &ShardedCompactStatusUpdater{real: real}
}

func (s *ShardedCompactStatusUpdater) OnSchedule(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}

func (s *ShardedCompactStatusUpdater) OnCreateJob(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}

func (s *ShardedCompactStatusUpdater) OnStart(ctx context.Context, compact *v1alpha1.CompactBackup) error {
	return s.real.OnStart(ctx, compact)
}

func (s *ShardedCompactStatusUpdater) OnProgress(_ context.Context, _ *v1alpha1.CompactBackup, _ *Progress, _ string) error {
	return nil
}

func (s *ShardedCompactStatusUpdater) OnFinish(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}

func (s *ShardedCompactStatusUpdater) OnJobComplete(_ context.Context, _ *v1alpha1.CompactBackup, _, _ string) error {
	return nil
}

func (s *ShardedCompactStatusUpdater) OnJobFailed(_ context.Context, _ *v1alpha1.CompactBackup, _, _, _ string) error {
	return nil
}

func (s *ShardedCompactStatusUpdater) UpdateStatus(_ *v1alpha1.CompactBackup, _ v1alpha1.CompactStatus) error {
	return nil
}

func (s *ShardedCompactStatusUpdater) UpdateShardIndexes(_ *v1alpha1.CompactBackup, _ batchv1.JobStatus) error {
	return nil
}
