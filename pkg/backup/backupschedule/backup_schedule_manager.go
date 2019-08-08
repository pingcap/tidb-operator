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

package backupschedule

import (
	"fmt"
	"sort"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type backupScheduleManager struct {
	backupControl controller.BackupControlInterface
	backupLister  listers.BackupLister
}

// NewBackupScheduleManager return a *backupScheduleManager
func NewBackupScheduleManager(
	backupControl controller.BackupControlInterface,
	backupLister listers.BackupLister,
) backup.BackupScheduleManager {
	return &backupScheduleManager{
		backupControl,
		backupLister,
	}
}

func (bm *backupScheduleManager) Sync(bs *v1alpha1.BackupSchedule) error {
	if bs.Spec.MaxBackups > 0 {
		defer bm.backupGC(bs)
	}

	if err := bm.canPerformNextBackup(bs); err != nil {
		return err
	}

	scheduledTime, err := getLastScheduledTime(bs)
	if scheduledTime != nil {
		return err
	}

	backup, err := bm.createBackup(bs, *scheduledTime)
	if err != nil {
		return err
	}

	bs.Status.LastBackup = backup.GetName()
	bs.Status.LastBackupTime = &metav1.Time{Time: *scheduledTime}
	return nil
}

func (bm *backupScheduleManager) canPerformNextBackup(bs *v1alpha1.BackupSchedule) error {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backup, err := bm.backupLister.Backups(ns).Get(bs.Status.LastBackup)
	if err != nil {
		return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, bs.Status.LastBackup, err)
	}

	if v1alpha1.IsBackupComplete(backup) || v1alpha1.IsBackupFailed(backup) {
		return nil
	}

	return fmt.Errorf("backup schedule %s/%s, the last backup %s is still running", ns, bsName, bs.Status.LastBackup)
}

func getLastScheduledTime(bs *v1alpha1.BackupSchedule) (*time.Time, error) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	sched, err := cron.ParseStandard(bs.Spec.Schedule)
	if err != nil {
		return nil, fmt.Errorf("parse backup schedule %s/%s cron format %s failed, err: %v", ns, bsName, bs.Spec.Schedule, err)
	}
	var earliestTime time.Time
	if bs.Status.LastBackupTime != nil {
		earliestTime = bs.Status.LastBackupTime.Time
	} else {
		// If none found, then this is either a recently created backupSchedule,
		// or the backupSchedule status info was somehow lost,
		// or that we have started a backup, but have not update backupSchedule status yet
		// (distributed systems can have arbitrary delays).
		// In any case, use the creation time of the backupSchedule as last known start time.
		earliestTime = bs.ObjectMeta.CreationTimestamp.Time
	}

	now := time.Now()
	if earliestTime.After(now) {
		// timestamp fallback, waiting for the next backup schedule period
		glog.Errorf("backup schedule %s/%s timestamp fallback, lastBackupTime: %s, now: %s",
			ns, bsName, earliestTime.Format(time.RFC3339), now.Format(time.RFC3339))
		return nil, nil
	}

	scheduledTimes := []time.Time{}
	for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		scheduledTimes = append(scheduledTimes, t)
		// If there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		//
		// I've somewhat arbitrarily picked 100, as more than 80,
		// but less than "lots".
		if len(scheduledTimes) > 100 {
			// We can't get the last backup schedule time
			glog.Errorf("Too many missed start backup schedule time (> 100). Check the clock.")
			return nil, nil
		}
	}
	if len(scheduledTimes) == 0 {
		glog.V(4).Infof("unmet backup schedule %s/%s start time, waiting for the next backup schedule period", ns, bsName)
		return nil, nil
	}
	scheduledTime := scheduledTimes[len(scheduledTimes)-1]
	return &scheduledTime, nil
}

func (bm *backupScheduleManager) createBackup(bs *v1alpha1.BackupSchedule, timestamp time.Time) (*v1alpha1.Backup, error) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupSpec := bs.Spec.BackupTemplate
	if backupSpec.StorageClassName == "" {
		if bs.Spec.StorageClassName != "" {
			backupSpec.StorageClassName = bs.Spec.StorageClassName
		} else {
			backupSpec.StorageClassName = controller.DefaultBackupStorageClassName
		}
	}

	bsLabel := label.NewBackupSchedule().Instance(bs.Spec.BackupTemplate.Cluster).BackupSchedule(bsName)

	backup := &v1alpha1.Backup{
		Spec: backupSpec,
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      bs.GetBackupCRDName(timestamp),
			Labels:    bsLabel.Labels(),
		},
	}

	return bm.backupControl.CreateBackup(backup)
}

func (bm *backupScheduleManager) backupGC(bs *v1alpha1.BackupSchedule) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupLables := label.NewBackupSchedule().Instance(bs.Spec.BackupTemplate.Cluster).BackupSchedule(bsName)
	selector, err := backupLables.Selector()
	if err != nil {
		glog.Errorf("generate backup schedule %s/%s label selector failed, err: %v", ns, bsName, err)
		return
	}
	backupsList, err := bm.backupLister.Backups(ns).List(selector)
	if err != nil {
		glog.Errorf("get backup schedule %s/%s backup list failed, selector: %s, err: %v", ns, bsName, selector, err)
	}

	// sort backups by creation time before removing extra backups
	sort.Sort(byCreateTime(backupsList))

	for i, backup := range backupsList {
		if i >= bs.Spec.MaxBackups {
			// delete the backup
			if err := bm.backupControl.DeleteBackup(backup); err != nil {
				return
			}
		}
	}
}

type byCreateTime []*v1alpha1.Backup

func (b byCreateTime) Len() int      { return len(b) }
func (b byCreateTime) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCreateTime) Less(i, j int) bool {
	return b[j].ObjectMeta.CreationTimestamp.Before(&b[i].ObjectMeta.CreationTimestamp)
}

var _ backup.BackupScheduleManager = &backupScheduleManager{}
