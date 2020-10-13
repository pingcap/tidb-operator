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
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	"k8s.io/klog"
)

type backupScheduleManager struct {
	backupLister  listers.BackupLister
	backupControl controller.BackupControlInterface
	jobLister     batchlisters.JobLister
	jobControl    controller.JobControlInterface
}

// NewBackupScheduleManager return a *backupScheduleManager
func NewBackupScheduleManager(
	backupLister listers.BackupLister,
	backupControl controller.BackupControlInterface,
	jobLister batchlisters.JobLister,
	jobControl controller.JobControlInterface,
) backup.BackupScheduleManager {
	return &backupScheduleManager{
		backupLister,
		backupControl,
		jobLister,
		jobControl,
	}
}

func (bm *backupScheduleManager) Sync(bs *v1alpha1.BackupSchedule) error {
	defer bm.backupGC(bs)

	if bs.Spec.Pause {
		return controller.IgnoreErrorf("backupSchedule %s/%s has been paused", bs.GetNamespace(), bs.GetName())
	}

	if err := bm.canPerformNextBackup(bs); err != nil {
		return err
	}

	scheduledTime, err := getLastScheduledTime(bs)
	if scheduledTime == nil {
		return err
	}

	// delete the last backup job for release the backup PVC
	if err := bm.deleteLastBackupJob(bs); err != nil {
		return nil
	}

	backup, err := bm.createBackup(bs, *scheduledTime)
	if err != nil {
		return err
	}

	bs.Status.LastBackup = backup.GetName()
	bs.Status.LastBackupTime = &metav1.Time{Time: *scheduledTime}
	bs.Status.AllBackupCleanTime = nil
	return nil
}

func (bm *backupScheduleManager) deleteLastBackupJob(bs *v1alpha1.BackupSchedule) error {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backup, err := bm.backupLister.Backups(ns).Get(bs.Status.LastBackup)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, bs.Status.LastBackup, err)
	}

	jobName := backup.GetBackupJobName()
	job, err := bm.jobLister.Jobs(ns).Get(jobName)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s job %s failed, err: %v", ns, bsName, backup.GetName(), jobName, err)
	}

	backup.SetGroupVersionKind(controller.BackupControllerKind)
	return bm.jobControl.DeleteJob(backup, job)
}

func (bm *backupScheduleManager) canPerformNextBackup(bs *v1alpha1.BackupSchedule) error {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backup, err := bm.backupLister.Backups(ns).Get(bs.Status.LastBackup)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, bs.Status.LastBackup, err)
	}

	if v1alpha1.IsBackupComplete(backup) || (v1alpha1.IsBackupScheduled(backup) && v1alpha1.IsBackupFailed(backup)) {
		return nil
	}
	// If the last backup is in a failed state, but it is not scheduled yet,
	// skip this sync round of the backup schedule and waiting the last backup.
	return controller.RequeueErrorf("backup schedule %s/%s, the last backup %s is still running", ns, bsName, bs.Status.LastBackup)
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
	} else if bs.Status.AllBackupCleanTime != nil {
		// Recovery from a long paused backup schedule may cause problem like "incorrect clock",
		// so we introduce AllBackupCleanTime field to solve this problem.
		earliestTime = bs.Status.AllBackupCleanTime.Time
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
		klog.Errorf("backup schedule %s/%s timestamp fallback, lastBackupTime: %s, now: %s",
			ns, bsName, earliestTime.Format(time.RFC3339), now.Format(time.RFC3339))
		return nil, nil
	}

	var scheduledTimes []time.Time
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
			if bs.Status.LastBackupTime == nil && bs.Status.AllBackupCleanTime != nil {
				// Recovery backup schedule from pause status, should refresh AllBackupCleanTime to avoid unschedulable problem
				bs.Status.AllBackupCleanTime = &metav1.Time{Time: time.Now()}
				return nil, controller.RequeueErrorf("recovery backup schedule %s/%s from pause status, refresh AllBackupCleanTime.", ns, bsName)
			}
			klog.Error("Too many missed start backup schedule time (> 100). Check the clock.")
			return nil, nil
		}
	}

	if len(scheduledTimes) == 0 {
		klog.V(4).Infof("unmet backup schedule %s/%s start time, waiting for the next backup schedule period", ns, bsName)
		return nil, nil
	}
	scheduledTime := scheduledTimes[len(scheduledTimes)-1]
	return &scheduledTime, nil
}

func (bm *backupScheduleManager) createBackup(bs *v1alpha1.BackupSchedule, timestamp time.Time) (*v1alpha1.Backup, error) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupSpec := *bs.Spec.BackupTemplate.DeepCopy()
	if backupSpec.BR == nil {
		if backupSpec.StorageClassName == nil || *backupSpec.StorageClassName == "" {
			backupSpec.StorageClassName = bs.Spec.StorageClassName
		}

		if backupSpec.StorageSize == "" {
			if bs.Spec.StorageSize != "" {
				backupSpec.StorageSize = bs.Spec.StorageSize
			} else {
				backupSpec.StorageSize = constants.DefaultStorageSize
			}
		}
	} else {
		var pdAddress, clusterNamespace string
		clusterNamespace = backupSpec.BR.ClusterNamespace
		if backupSpec.BR.ClusterNamespace == "" {
			clusterNamespace = ns
		}
		pdAddress = fmt.Sprintf("%s-pd.%s:2379", backupSpec.BR.Cluster, clusterNamespace)

		if backupSpec.S3 != nil {
			backupSpec.S3.Prefix = path.Join(backupSpec.S3.Prefix,
				strings.ReplaceAll(pdAddress, ":", "-")+"-"+timestamp.UTC().Format(constants.TimeFormat))
		} else if backupSpec.Gcs != nil {
			backupSpec.Gcs.Prefix = path.Join(backupSpec.Gcs.Prefix,
				strings.ReplaceAll(pdAddress, ":", "-")+"-"+timestamp.UTC().Format(constants.TimeFormat))
		}
	}

	if bs.Spec.ImagePullSecrets != nil {
		backupSpec.ImagePullSecrets = bs.Spec.ImagePullSecrets
	}

	bsLabel := label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName)

	backup := &v1alpha1.Backup{
		Spec: backupSpec,
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ns,
			Name:        bs.GetBackupCRDName(timestamp),
			Labels:      bsLabel.Labels(),
			Annotations: bs.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupScheduleOwnerRef(bs),
			},
		},
	}

	return bm.backupControl.CreateBackup(backup)
}

func (bm *backupScheduleManager) backupGC(bs *v1alpha1.BackupSchedule) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	// if MaxBackups and MaxReservedTime are set at the same time, MaxReservedTime is preferred.
	if bs.Spec.MaxReservedTime != nil {
		bm.backupGCByMaxReservedTime(bs)
		return
	}

	if bs.Spec.MaxBackups != nil && *bs.Spec.MaxBackups > 0 {
		bm.backupGCByMaxBackups(bs)
		return
	}
	// TODO: When the backup schedule gc policy is not set, we should set a default backup gc policy.
	klog.Warningf("backup schedule %s/%s does not set backup gc policy", ns, bsName)
}

func (bm *backupScheduleManager) backupGCByMaxReservedTime(bs *v1alpha1.BackupSchedule) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	reservedTime, err := time.ParseDuration(*bs.Spec.MaxReservedTime)
	if err != nil {
		klog.Errorf("backup schedule %s/%s, invalid MaxReservedTime %s", ns, bsName, *bs.Spec.MaxReservedTime)
		return
	}

	backupsList, err := bm.getBackupList(bs, false)
	if err != nil {
		klog.Errorf("backupGCByMaxReservedTime, err: %s", err)
		return
	}

	var deleteCount int
	for _, backup := range backupsList {
		if backup.CreationTimestamp.Add(reservedTime).After(time.Now()) {
			continue
		}
		// delete the expired backup
		if err := bm.backupControl.DeleteBackup(backup); err != nil {
			klog.Errorf("backup schedule %s/%s gc backup %s failed, err %v", ns, bsName, backup.GetName(), err)
			return
		}
		deleteCount += 1
		klog.Infof("backup schedule %s/%s gc backup %s success", ns, bsName, backup.GetName())
	}

	if deleteCount == len(backupsList) {
		// All backups have been deleted, so the last backup information in the backupSchedule should be reset
		bs.Status.LastBackupTime = nil
		bs.Status.LastBackup = ""
		bs.Status.AllBackupCleanTime = &metav1.Time{Time: time.Now()}
	}
}

func (bm *backupScheduleManager) backupGCByMaxBackups(bs *v1alpha1.BackupSchedule) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupsList, err := bm.getBackupList(bs, true)
	if err != nil {
		klog.Errorf("backupGCByMaxBackups failed, err: %s", err)
		return
	}

	var deleteCount int
	for i, backup := range backupsList {
		if i < int(*bs.Spec.MaxBackups) {
			continue
		}
		// delete the backup
		if err := bm.backupControl.DeleteBackup(backup); err != nil {
			klog.Errorf("backup schedule %s/%s gc backup %s failed, err %v", ns, bsName, backup.GetName(), err)
			return
		}
		deleteCount += 1
		klog.Infof("backup schedule %s/%s gc backup %s success", ns, bsName, backup.GetName())
	}

	if deleteCount == len(backupsList) {
		// All backups have been deleted, so the last backup information in the backupSchedule should be reset
		bs.Status.LastBackupTime = nil
		bs.Status.LastBackup = ""
		bs.Status.AllBackupCleanTime = &metav1.Time{Time: time.Now()}
	}
}

func (bm *backupScheduleManager) getBackupList(bs *v1alpha1.BackupSchedule, needSort bool) ([]*v1alpha1.Backup, error) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupLabels := label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName)
	selector, err := backupLabels.Selector()
	if err != nil {
		return nil, fmt.Errorf("generate backup schedule %s/%s label selector failed, err: %v", ns, bsName, err)
	}
	backupsList, err := bm.backupLister.Backups(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("get backup schedule %s/%s backup list failed, selector: %s, err: %v", ns, bsName, selector, err)
	}

	if needSort {
		// sort backups by creation time before removing expired backups
		sort.Sort(byCreateTime(backupsList))
	}
	return backupsList, nil
}

type byCreateTime []*v1alpha1.Backup

func (b byCreateTime) Len() int      { return len(b) }
func (b byCreateTime) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCreateTime) Less(i, j int) bool {
	return b[j].ObjectMeta.CreationTimestamp.Before(&b[i].ObjectMeta.CreationTimestamp)
}

var _ backup.BackupScheduleManager = &backupScheduleManager{}

type FakeBackupScheduleManager struct {
	err error
}

func NewFakeBackupScheduleManager() *FakeBackupScheduleManager {
	return &FakeBackupScheduleManager{}
}

func (fbsm *FakeBackupScheduleManager) SetSyncError(err error) {
	fbsm.err = err
}

func (fbsm *FakeBackupScheduleManager) Sync(bs *v1alpha1.BackupSchedule) error {
	if fbsm.err != nil {
		return fbsm.err
	}

	if bs.Status.LastBackupTime != nil {
		// simulate status update
		bs.Status.LastBackupTime = &metav1.Time{Time: bs.Status.LastBackupTime.Add(1 * time.Hour)}
	}
	return nil
}

var _ backup.BackupScheduleManager = &FakeBackupScheduleManager{}
