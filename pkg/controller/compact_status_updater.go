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
	OnSchedule(ctx context.Context, compact *v1alpha1.CompactBackup) error
	OnCreateJob(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	OnStart(ctx context.Context, compact *v1alpha1.CompactBackup) error
	OnProgress(ctx context.Context, compact *v1alpha1.CompactBackup, p Progress) error
	OnFinish(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	OnJobFailed(ctx context.Context, compact *v1alpha1.CompactBackup, reason string) error
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
	backupName := compact.GetName()

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
		if updated, err := r.lister.CompactBackups(ns).Get(backupName); err == nil {
			*compact = *(updated.DeepCopy())
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
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

		// Apply the update if any field changed
		if updated {
			_, updateErr := r.cli.PingcapV1alpha1().CompactBackups(ns).Update(context.TODO(), compact, metav1.UpdateOptions{})
			if updateErr == nil {
				klog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
				return nil
			}
			klog.Errorf("Failed to update backup [%s/%s], error: %v", ns, backupName, updateErr)
			return updateErr
		}
		return nil
	})
	return err
}

func (r *CompactStatusUpdater) OnSchedule(ctx context.Context, compact *v1alpha1.CompactBackup) error {
	newStatus := v1alpha1.CompactStatus{
		State: string(v1alpha1.BackupScheduled),
	}
	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnCreateJob(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error {
	newStatus := v1alpha1.CompactStatus{}
	if err != nil {
		newStatus.State = string(v1alpha1.BackupFailed)
		newStatus.Message = err.Error()
	} else {
		newStatus.State = string(v1alpha1.BackupRunning)
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

func (r *CompactStatusUpdater) OnProgress(ctx context.Context, compact *v1alpha1.CompactBackup, p Progress) error {
	progress := fmt.Sprintf("[READ_META(%d/%d),COMPACT_WORK(%d/%d)]",
		p.MetaCompleted, p.MetaTotal, p.BytesCompacted, p.BytesToCompact)

	newStatus := v1alpha1.CompactStatus{
		Progress: progress,
	}
	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnFinish(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error {
	newStatus := v1alpha1.CompactStatus{}
	if err != nil {
		newStatus.State = string(v1alpha1.BackupFailed)
		newStatus.Message = err.Error()
		r.Event(compact, corev1.EventTypeWarning, "Failed", err.Error())
	} else {
		newStatus.State = string(v1alpha1.BackupComplete)
		r.Event(compact, corev1.EventTypeNormal, "Finished", "The compaction process has finished successfully.")
	}
	return r.UpdateStatus(compact, newStatus)
}

func (r *CompactStatusUpdater) OnJobFailed(ctx context.Context, compact *v1alpha1.CompactBackup, reason string) error {
	newStatus := v1alpha1.CompactStatus{
		State: string(v1alpha1.BackupFailed),
	}
	r.Event(compact, corev1.EventTypeWarning, "The compact job is failed.", reason)
	return r.UpdateStatus(compact, newStatus)
}
