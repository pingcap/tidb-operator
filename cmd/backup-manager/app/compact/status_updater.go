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

package compact

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

func (r *CompactStatusUpdater) UpdateStatus(compact *v1alpha1.CompactBackup, newState string, progress string, message string) error {
	ns := compact.GetNamespace()
	backupName := compact.GetName()
	key := fmt.Sprintf("%s/%s", ns, backupName) // Unique key for the backup

	now := time.Now()
	updateProgress := true
	if progress != "" {
		if now.Sub(r.progressLastUpdate) < progressDebounceDuration {
			klog.Infof("Skipping progress update for [%s] due to debounce interval", key)
			updateProgress = false
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
		if newState !="" && compact.Status.State != newState {
			compact.Status.State = newState
			updated = true
			updateProgress = true
		}
		if message != "" && compact.Status.Message != message {
			compact.Status.Message = message
			updated = true
		}
		if updateProgress && progress != "" && compact.Status.Progress != progress {
			compact.Status.Progress = progress
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

func (r *CompactStatusUpdater) OnStart(ctx context.Context, compact *v1alpha1.CompactBackup) {
	r.UpdateStatus(compact, string(v1alpha1.BackupRunning), "", "")
	r.Event(compact, corev1.EventTypeNormal, "Started", "The compaction process has started successfully.")
}

func (r *CompactStatusUpdater) OnProgress(ctx context.Context, p Progress, compact *v1alpha1.CompactBackup) {
	progress := fmt.Sprintf("[READ_META(%d/%d),COMPACT_WORK(%d/%d)]",
		p.MetaCompleted, p.MetaTotal, p.BytesCompacted, p.BytesToCompact)
	r.UpdateStatus(compact, "", progress, "")
}

func (r *CompactStatusUpdater) OnFinish(ctx context.Context, err error, compact *v1alpha1.CompactBackup) {
	if err != nil {
		r.Event(compact, corev1.EventTypeWarning, "Failed", err.Error())
		r.UpdateStatus(compact, string(v1alpha1.BackupFailed), "", err.Error())
	} else {
		r.Event(compact, corev1.EventTypeNormal, "Finished", "The compaction process has finished successfully.")
		r.UpdateStatus(compact, string(v1alpha1.BackupComplete), "", "")
	}
}
