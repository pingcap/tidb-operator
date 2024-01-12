// Copyright 2023 PingCAP, Inc.
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
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

type nowFn func() time.Time

type backupScheduleManager struct {
	deps *controller.BrFedDependencies
	now  nowFn
}

// NewBackupScheduleManager return backupScheduleManager
func NewBackupScheduleManager(deps *controller.BrFedDependencies) fedvolumebackup.BackupScheduleManager {
	return &backupScheduleManager{
		deps: deps,
		now:  time.Now,
	}
}

func (bm *backupScheduleManager) Sync(vbs *v1alpha1.VolumeBackupSchedule) error {
	defer bm.backupGC(vbs)

	if vbs.Spec.Pause {
		return controller.IgnoreErrorf("backupSchedule %s/%s has been paused", vbs.GetNamespace(), vbs.GetName())
	}

	if err := bm.canPerformNextBackup(vbs); err != nil {
		return err
	}

	scheduledTime, err := getLastScheduledTime(vbs, bm.now)
	if scheduledTime == nil {
		return err
	}

	backup, err := createBackup(bm.deps.FedVolumeBackupControl, vbs, *scheduledTime)
	if err != nil {
		return err
	}

	vbs.Status.LastBackup = backup.GetName()
	vbs.Status.LastBackupTime = &metav1.Time{Time: *scheduledTime}
	vbs.Status.AllBackupCleanTime = nil
	return nil
}

// getLastScheduledTime return the newest time need to be scheduled according last backup time.
// the return time is not before now and return nil if there's no such time.
func getLastScheduledTime(vbs *v1alpha1.VolumeBackupSchedule, nowFn nowFn) (*time.Time, error) {
	ns := vbs.GetNamespace()
	bsName := vbs.GetName()

	sched, err := cron.ParseStandard(vbs.Spec.Schedule)
	if err != nil {
		return nil, fmt.Errorf("parse backup schedule %s/%s cron format %s failed, err: %v", ns, bsName, vbs.Spec.Schedule, err)
	}

	var earliestTime time.Time
	if vbs.Status.LastBackupTime != nil {
		earliestTime = vbs.Status.LastBackupTime.Time
	} else if vbs.Status.AllBackupCleanTime != nil {
		// Recovery from a long paused backup schedule may cause problem like "incorrect clock",
		// so we introduce AllBackupCleanTime field to solve this problem.
		earliestTime = vbs.Status.AllBackupCleanTime.Time
	} else {
		// If none found, then this is either a recently created backupSchedule,
		// or the backupSchedule status info was somehow lost,
		// or that we have started a backup, but have not update backupSchedule status yet
		// (distributed systems can have arbitrary delays).
		// In any case, use the creation time of the backupSchedule as last known start time.
		earliestTime = vbs.ObjectMeta.CreationTimestamp.Time
	}

	now := nowFn()
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
		// by decades or more). So, we need to set LastBackupTime to now() in order to let
		// next reconcile succeed.
		if len(scheduledTimes) > 1000 {
			// We can't get the last backup schedule time
			if vbs.Status.LastBackupTime == nil && vbs.Status.AllBackupCleanTime != nil {
				// Recovery backup schedule from pause status, should refresh AllBackupCleanTime to avoid unschedulable problem
				vbs.Status.AllBackupCleanTime = &metav1.Time{Time: nowFn()}
				return nil, controller.RequeueErrorf("recovery backup schedule %s/%s from pause status, refresh AllBackupCleanTime.", ns, bsName)
			}

			klog.Warning("Too many missed start backup schedule time (> 1000). Fail current one.")
			offset := sched.Next(t).Sub(t)
			vbs.Status.LastBackupTime = &metav1.Time{Time: time.Now().Add(-offset)}
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

func buildBackup(vbs *v1alpha1.VolumeBackupSchedule, timestamp time.Time) *v1alpha1.VolumeBackup {
	ns := vbs.GetNamespace()
	bsName := vbs.GetName()

	backupSpec := *vbs.Spec.BackupTemplate.DeepCopy()

	bsLabel := util.CombineStringMap(label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName), vbs.Labels)

	if backupSpec.Template.BR == nil {
		klog.Errorf("Information on BR missing in template")
		return nil
	}

	if backupSpec.Template.S3 != nil {
		backupSpec.Template.S3.Prefix = path.Join(backupSpec.Template.S3.Prefix, "-"+timestamp.UTC().Format(pingcapv1alpha1.BackupNameTimeFormat))
	} else {
		klog.Errorf("Information on S3 missing in template")
		return nil
	}

	if vbs.Spec.BackupTemplate.Template.ImagePullSecrets != nil {
		backupSpec.Template.ImagePullSecrets = vbs.Spec.BackupTemplate.Template.ImagePullSecrets
	}
	backup := &v1alpha1.VolumeBackup{
		Spec: backupSpec,
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ns,
			Name:        vbs.GetBackupCRDName(timestamp),
			Labels:      bsLabel,
			Annotations: vbs.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetFedVolumeBackupScheduleOwnerRef(vbs),
			},
		},
	}

	klog.V(4).Infof("created backup is [%v], at time %v", backup, timestamp)

	return backup
}

func createBackup(bkController controller.FedVolumeBackupControlInterface, vbs *v1alpha1.VolumeBackupSchedule, timestamp time.Time) (*v1alpha1.VolumeBackup, error) {
	bk := buildBackup(vbs, timestamp)
	if bk == nil {
		return nil, controller.IgnoreErrorf("Invalid backup template for volume backup schedule [%s], BR or S3 information missing", vbs.GetName())
	}

	return bkController.CreateVolumeBackup(bk)
}

func (bm *backupScheduleManager) canPerformNextBackup(vbs *v1alpha1.VolumeBackupSchedule) error {
	ns := vbs.GetNamespace()
	bsName := vbs.GetName()

	backup, err := bm.deps.VolumeBackupLister.VolumeBackups(ns).Get(vbs.Status.LastBackup)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, vbs.Status.LastBackup, err)
	}

	if v1alpha1.IsVolumeBackupComplete(backup) || v1alpha1.IsVolumeBackupFailed(backup) {
		return nil
	}
	// skip this sync round of the backup schedule and waiting the last backup.
	return controller.RequeueErrorf("backup schedule %s/%s, the last backup %s is still running", ns, bsName, vbs.Status.LastBackup)
}

func (bm *backupScheduleManager) backupGC(vbs *v1alpha1.VolumeBackupSchedule) {
	ns := vbs.GetNamespace()
	bsName := vbs.GetName()

	// if MaxBackups and MaxReservedTime are set at the same time, MaxReservedTime is preferred.
	if vbs.Spec.MaxReservedTime != nil {
		bm.backupGCByMaxReservedTime(vbs)
		return
	}

	if vbs.Spec.MaxBackups != nil && *vbs.Spec.MaxBackups > 0 {
		bm.backupGCByMaxBackups(vbs)
		return
	}
	klog.Warningf("backup schedule %s/%s does not set backup gc policy", ns, bsName)
}

func (bm *backupScheduleManager) backupGCByMaxReservedTime(vbs *v1alpha1.VolumeBackupSchedule) {
	ns := vbs.GetNamespace()
	bsName := vbs.GetName()

	reservedTime, err := time.ParseDuration(*vbs.Spec.MaxReservedTime)
	if err != nil {
		klog.Errorf("backup schedule %s/%s, invalid MaxReservedTime %s", ns, bsName, *vbs.Spec.MaxReservedTime)
		return
	}

	backupsList, err := bm.getBackupList(vbs)
	if err != nil {
		klog.Errorf("backupGCByMaxReservedTime, err: %s", err)
		return
	}

	ascBackups := sortSnapshotBackups(backupsList)
	if len(ascBackups) == 0 {
		return
	}

	var expiredBackups []*v1alpha1.VolumeBackup

	expiredBackups, err = calculateExpiredBackups(ascBackups, reservedTime)
	if err != nil {
		klog.Errorf("calculate expired backups without log backup, err: %s", err)
		return
	}

	// In order to avoid throttling, we choose to do delete volumebackup one by one.
	// Delete the oldest expired backup
	if len(expiredBackups) > 0 {
		backup := expiredBackups[0]
		if backup.DeletionTimestamp != nil {
			klog.Infof("Deletion is ongoing for backup schedule %s/%s, backup %s", ns, bsName, backup.GetName())
			return
		} else {
			if err = bm.deps.FedVolumeBackupControl.DeleteVolumeBackup(backup); err != nil {
				klog.Errorf("backup schedule %s/%s gc backup %s failed, err %v", ns, bsName, backup.GetName(), err)
				return
			}
			klog.Infof("backup schedule %s/%s gc backup %s success", ns, bsName, backup.GetName())

			if len(expiredBackups) == 1 && len(backupsList) == 1 {
				// All backups have been deleted, so the last backup information in the backupSchedule should be reset
				bm.resetLastBackup(vbs)
			}
		}
	}
}

// sortSnapshotBackups return snapshot backups to be GCed order by create time asc
func sortSnapshotBackups(backupsList []*v1alpha1.VolumeBackup) []*v1alpha1.VolumeBackup {
	var ascBackupList = make([]*v1alpha1.VolumeBackup, 0)

	for _, backup := range backupsList {
		// Only try to GC Completed or Failed VolumeBackup
		if !(v1alpha1.IsVolumeBackupFailed(backup) || v1alpha1.IsVolumeBackupComplete(backup)) {
			continue
		}
		ascBackupList = append(ascBackupList, backup)
	}

	sort.Slice(ascBackupList, func(i, j int) bool {
		return ascBackupList[i].CreationTimestamp.Unix() < ascBackupList[j].CreationTimestamp.Unix()
	})
	return ascBackupList
}

// sortAllSnapshotBackups return all snapshot backups order by create time asc
// it's for test only now
func sortAllSnapshotBackups(backupsList []*v1alpha1.VolumeBackup) []*v1alpha1.VolumeBackup {
	var ascBackupList = make([]*v1alpha1.VolumeBackup, 0)
	ascBackupList = append(ascBackupList, backupsList...)

	sort.Slice(ascBackupList, func(i, j int) bool {
		return ascBackupList[i].CreationTimestamp.Unix() < ascBackupList[j].CreationTimestamp.Unix()
	})
	return ascBackupList
}

func calculateExpiredBackups(backupsList []*v1alpha1.VolumeBackup, reservedTime time.Duration) ([]*v1alpha1.VolumeBackup, error) {
	expiredTS := config.TSToTSO(time.Now().Add(-1 * reservedTime).Unix())
	i := 0
	for ; i < len(backupsList); i++ {
		startTS, err := config.ParseTSString(backupsList[i].Status.CommitTs)
		if err != nil {
			return nil, perrors.Annotatef(err, "parse start tso: %s", backupsList[i].Status.CommitTs)
		}
		if startTS >= expiredTS {
			break
		}
	}
	return backupsList[:i], nil
}

func (bm *backupScheduleManager) getBackupList(bs *v1alpha1.VolumeBackupSchedule) ([]*v1alpha1.VolumeBackup, error) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupLabels := label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName)
	selector, err := backupLabels.Selector()
	if err != nil {
		return nil, fmt.Errorf("generate backup schedule %s/%s label selector failed, err: %v", ns, bsName, err)
	}
	backupsList, err := bm.deps.VolumeBackupLister.VolumeBackups(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("get backup schedule %s/%s backup list failed, selector: %s, err: %v", ns, bsName, selector, err)
	}

	return backupsList, nil
}

func (bm *backupScheduleManager) backupGCByMaxBackups(vbs *v1alpha1.VolumeBackupSchedule) {
	ns := vbs.GetNamespace()
	bsName := vbs.GetName()

	backupsList, err := bm.getBackupList(vbs)
	if err != nil {
		klog.Errorf("backupGCByMaxBackups failed, err: %s", err)
		return
	}

	sort.Sort(byCreateTimeDesc(backupsList))

	// In order to avoid throttling, we choose to do delete volumebackup one by one.
	// Delete the oldest expired backup
	if len(backupsList) > int(*vbs.Spec.MaxBackups) {
		backup := backupsList[len(backupsList)-1]
		if backup.DeletionTimestamp != nil {
			klog.Infof("Deletion is ongoing for backup schedule %s/%s, backup %s", ns, bsName, backup.GetName())
			return
		} else {
			if err = bm.deps.FedVolumeBackupControl.DeleteVolumeBackup(backup); err != nil {
				klog.Errorf("backup schedule %s/%s gc backup %s failed, err %v", ns, bsName, backup.GetName(), err)
				return
			}
			klog.Infof("backup schedule %s/%s gc backup %s success", ns, bsName, backup.GetName())

			if len(backupsList) == 1 {
				// All backups have been deleted, so the last backup information in the backupSchedule should be reset
				bm.resetLastBackup(vbs)
			}
		}
	}
}

func (bm *backupScheduleManager) resetLastBackup(vbs *v1alpha1.VolumeBackupSchedule) {
	vbs.Status.LastBackupTime = nil
	vbs.Status.LastBackup = ""
	vbs.Status.AllBackupCleanTime = &metav1.Time{Time: bm.now()}
}

type byCreateTimeDesc []*v1alpha1.VolumeBackup

func (b byCreateTimeDesc) Len() int      { return len(b) }
func (b byCreateTimeDesc) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCreateTimeDesc) Less(i, j int) bool {
	return b[j].ObjectMeta.CreationTimestamp.Before(&b[i].ObjectMeta.CreationTimestamp)
}

var _ fedvolumebackup.BackupScheduleManager = &backupScheduleManager{}

type FakeBackupScheduleManager struct {
	err error
}

func NewFakeBackupScheduleManager() *FakeBackupScheduleManager {
	return &FakeBackupScheduleManager{}
}

func (m *FakeBackupScheduleManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeBackupScheduleManager) Sync(vbs *v1alpha1.VolumeBackupSchedule) error {
	if m.err != nil {
		return m.err
	}
	if vbs.Status.LastBackupTime != nil {
		// simulate status update
		vbs.Status.LastBackupTime = &metav1.Time{Time: vbs.Status.LastBackupTime.Add(1 * time.Hour)}
	}
	return nil
}

var _ fedvolumebackup.BackupScheduleManager = &FakeBackupScheduleManager{}
