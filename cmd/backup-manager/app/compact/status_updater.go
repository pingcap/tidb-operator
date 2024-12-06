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

type CompactStatusUpdater struct {
	recorder record.EventRecorder
	lister   listers.CompactBackupLister
	cli      versioned.Interface
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

func (r *CompactStatusUpdater) UpdateStatus(compact *v1alpha1.CompactBackup, newState string) error {
	ns := compact.GetNamespace()
	backupName := compact.GetName()
	// try best effort to guarantee backup is updated.
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		// Always get the latest backup before update.
		if updated, err := r.lister.CompactBackups(ns).Get(backupName); err == nil {
			// make a copy so we don't mutate the shared cache
			*compact = *(updated.DeepCopy())
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
			return err
		}
		if compact.Status.State != newState {
			compact.Status.State = newState
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
	r.UpdateStatus(compact, "RUNNING")
	r.Event(compact, corev1.EventTypeNormal, "Started", "The compaction process has started successfully.")
}

func (r *CompactStatusUpdater) OnProgress(ctx context.Context, p Progress, compact *v1alpha1.CompactBackup) {
	message := fmt.Sprintf("RUNNING[READ_META(%d/%d),COMPACT_WORK(%d/%d)]",
		p.MetaCompleted, p.MetaTotal, p.BytesCompacted, p.BytesToCompact)
	r.UpdateStatus(compact, message)
}

func (r *CompactStatusUpdater) OnFinish(ctx context.Context, err error, compact *v1alpha1.CompactBackup) {
	if err != nil {
		r.Event(compact, corev1.EventTypeWarning, "Failed", err.Error())
		r.UpdateStatus(compact, "FAILED")
	} else {
		r.Event(compact, corev1.EventTypeNormal, "Finished", "The compaction process has finished successfully.")
		r.UpdateStatus(compact, "FINISHED")
	}
}
