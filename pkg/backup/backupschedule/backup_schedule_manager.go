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

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type nowFn func() time.Time

type backupScheduleManager struct {
	deps *controller.Dependencies
	now  nowFn
}

// NewBackupScheduleManager return a *backupScheduleManager
func NewBackupScheduleManager(deps *controller.Dependencies) backup.BackupScheduleManager {
	return &backupScheduleManager{
		deps: deps,
		now:  time.Now,
	}
}

func (bm *backupScheduleManager) doCompact(bs *v1alpha1.BackupSchedule, startTime time.Time, endTime time.Time) error {
	compact := buildCompactBackup(bs, startTime, endTime)
	_, err := bm.deps.CompactControl.CreateCompactBackup(compact)
	bs.Status.LastCompact = compact.Name
	return err
}

func calEndTs(bs *v1alpha1.BackupSchedule, startTs time.Time, interval time.Duration, scheduleTime *time.Time) time.Time {
	lastBackup := bs.Status.LastBackupTime
	nextCompact := bs.Status.NextCompactTime
	var endTs time.Time

	fastCompactLimit := startTs.Add(3 * interval)

	var scheduleMetaTime *metav1.Time
	if scheduleTime != nil {
		scheduleMetaTime = &metav1.Time{Time: *scheduleTime}
	}

	targets := []*metav1.Time{
		nextCompact,
		lastBackup,
		scheduleMetaTime,
	}

	for _, target := range targets {
		if target == nil || target.Time.Compare(startTs) <= 0 {
			continue
		}

		if target.After(fastCompactLimit) {
			endTs = fastCompactLimit
			nextCompact = target
			break
		} else {
			endTs = target.Time
			nextCompact = nil
			break
		}
	}
	bs.Status.NextCompactTime = nextCompact

	if endTs.IsZero() {
		endTs = startTs.Add(interval)
	}
	return endTs
}

func (bm *backupScheduleManager) performCompact(bs *v1alpha1.BackupSchedule, scheduleTime *time.Time, nowFn nowFn) error {
	if bs.Spec.CompactInterval == nil || bs.Status.LogBackup == nil {
		return nil
	}

	var startTs, endTs time.Time
	switch {
	case bs.Status.LogBackupStartTs == nil:
		return fmt.Errorf("Compact failed: %s/%s, please start a log backup before compact it", bs.GetNamespace(), bs.GetName())
	case bs.Status.LastCompactTime == nil:
		startTs = bs.Status.LogBackupStartTs.Time
	default:
		startTs = bs.Status.LastCompactTime.Time
	}
	klog.Infof("backupSchedule %s/%s startTs is %v", bs.GetNamespace(), bs.GetName(), startTs)

	interval, err := time.ParseDuration(*bs.Spec.CompactInterval)
	if err != nil {
		return fmt.Errorf("failed to parse compact interval: %w", err)
	}
	now := nowFn()

	endTs = calEndTs(bs, startTs, interval, scheduleTime)
	klog.Infof("backupSchedule %s/%s endTs is %v", bs.GetNamespace(), bs.GetName(), endTs)

	if endTs.After(now) {
		klog.Infof("backupSchedule %s/%s next compact time is not reached: %v", bs.GetNamespace(), bs.GetName(), endTs)
		return nil
	}

	if err := bm.doCompact(bs, startTs, endTs); err != nil {
		return err
	}

	return nil
}

func (bm *backupScheduleManager) Sync(bs *v1alpha1.BackupSchedule) error {
	defer bm.backupGC(bs)

	if bs.Spec.Pause {
		return controller.IgnoreErrorf("backupSchedule %s/%s has been paused", bs.GetNamespace(), bs.GetName())
	}

	if err := bm.performLogBackupIfNeeded(bs); err != nil {
		return err
	}

	if err := bm.canPerformNextBackup(bs); err != nil {
		return err
	}

	scheduledTime, err := getLastScheduledTime(bs, bm.now)
	if err != nil {
		return err
	}

	klog.Infof("backupSchedule %s/%s next scheduled time is %v", bs.GetNamespace(), bs.GetName(), scheduledTime)

	if err := bm.canPerformCompact(bs); err != nil {
		klog.Errorf("backupSchedule %s/%s can not perform next compact, err: %v", bs.GetNamespace(), bs.GetName(), err)
	} else if err := bm.performCompact(bs, scheduledTime, bm.now); err != nil {
		klog.Errorf("backupSchedule %s/%s perform compact failed, err: %v", bs.GetNamespace(), bs.GetName(), err)
	}

	if scheduledTime == nil {
		return nil
	}

	// Delete the last backup job for releasing the backup PVC
	if err := bm.deleteLastBackupJob(bs); err != nil {
		return nil
	}

	backup, err := createBackup(bm.deps.BackupControl, bs, *scheduledTime)
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

	backup, err := bm.deps.BackupLister.Backups(ns).Get(bs.Status.LastBackup)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, bs.Status.LastBackup, err)
	}

	jobName := backup.GetBackupJobName()
	job, err := bm.deps.JobLister.Jobs(ns).Get(jobName)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s job %s failed, err: %v", ns, bsName, backup.GetName(), jobName, err)
	}

	backup.SetGroupVersionKind(controller.BackupControllerKind)
	return bm.deps.JobControl.DeleteJob(backup, job)
}

func (bm *backupScheduleManager) canPerformNextBackup(bs *v1alpha1.BackupSchedule) error {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	// If this backup schedule has specified label of backup schedule group, then we need to check the last backup of the group.
	// Otherwise, check its own last backup.
	bsGroupName := bs.GetLabels()[label.BackupScheduleGroupLabelKey]

	if bsGroupName == "" {
		backup, err := bm.deps.BackupLister.Backups(ns).Get(bs.Status.LastBackup)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, bs.Status.LastBackup, err)
		}

		if v1alpha1.IsBackupComplete(backup) || (v1alpha1.IsBackupScheduled(backup) && v1alpha1.IsBackupFailed(backup)) {
			return nil
		}
		// skip this sync round of the backup schedule and waiting the last backup.
		return controller.RequeueErrorf("backup schedule %s/%s, the last backup %s is still running", ns, bsName, bs.Status.LastBackup)
	}

	// Check the last backup of the group
	backupScheduleGroupLabels := label.NewBackupScheduleGroup(bsGroupName)
	selector, err := backupScheduleGroupLabels.Selector()
	if err != nil {
		return fmt.Errorf("generate backup schedule group %s label selector failed, err: %v", bsGroupName, err)
	}

	bss, err := bm.deps.BackupScheduleLister.BackupSchedules(ns).List(selector)
	if err != nil {
		return fmt.Errorf("backup schedule %s/%s, list backup schedules failed, err: %v", ns, bsName, err)
	}

	for _, bsMember := range bss {
		// The check is not safe in fact since we don't have strict serialization
		backup, err := bm.deps.BackupLister.Backups(ns).Get(bsMember.Status.LastBackup)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, bsMember.Status.LastBackup, err)
		}

		if v1alpha1.IsBackupComplete(backup) || (v1alpha1.IsBackupScheduled(backup) && v1alpha1.IsBackupFailed(backup)) {
			continue
		}
		// skip this sync round of the backup schedule and waiting the last backup.
		return controller.RequeueErrorf("backup schedule %s/%s, the last backup %s is still running", ns, bsName, bsMember.Status.LastBackup)
	}

	return nil
}

func (bm *backupScheduleManager) deleteLastcompactJob(bs *v1alpha1.BackupSchedule) error {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	compact, err := bm.deps.CompactBackupLister.CompactBackups(ns).Get(bs.Status.LastCompact)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s failed, err: %v", ns, bsName, bs.Status.LastBackup, err)
	}

	jobName := compact.GetName()
	job, err := bm.deps.JobLister.Jobs(ns).Get(jobName)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s, get backup %s job %s failed, err: %v", ns, bsName, compact.GetName(), jobName, err)
	}

	compact.SetGroupVersionKind(controller.CompactBackupControllerKind)
	return bm.deps.JobControl.DeleteJob(compact, job)
}

// canPerformCompact handles the compact backup processing logic.
// It returns a controller.RequeueError if the backup is still running,
// otherwise it updates the LastCompactTime or returns any encountered error.
func (bm *backupScheduleManager) canPerformCompact(bs *v1alpha1.BackupSchedule) error {
	if bs.Status.LastCompact == "" {
		return nil
	}

	lastCompact := bs.Status.LastCompact
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	compact, err := bm.deps.CompactBackupLister.CompactBackups(ns).Get(lastCompact)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("backup schedule %s/%s: get compact backup %s failed: %w", ns, bsName, lastCompact, err)
	}

	var lastCompactTs string
	switch compact.Status.State {
	case string(v1alpha1.BackupComplete):
		lastCompactTs = compact.Spec.EndTs
	case string(v1alpha1.BackupFailed):
		lastCompactTs = compact.Spec.StartTs
	default:
		return controller.RequeueErrorf("backup schedule %s/%s: compact backup %s is still running", ns, bsName, lastCompact)
	}

	t, err := config.ParseTSStringToGoTime(lastCompactTs)
	if err != nil {
		return perrors.AddStack(err)
	}

	if err := bm.deps.CompactControl.DeleteCompactBackup(compact); err != nil {
		return fmt.Errorf("backup schedule %s/%s: delete compact backup %s failed: %w", ns, bsName, lastCompact, err)
	}

	if !t.IsZero() && t.Before(bs.Status.LogBackupStartTs.Time) {
		return fmt.Errorf("backupSchedule %s/%s last compact time can't rollback (from %v to %v)", bs.GetNamespace(), bs.GetName(), bs.Status.LogBackupStartTs, t)
	}
	if bs.Status.LastCompactTime != nil && t.Before(bs.Status.LastCompactTime.Time) {
		return fmt.Errorf("backupSchedule %s/%s last compact time can't rollback (from %v to %v)", bs.GetNamespace(), bs.GetName(), bs.Status.LastCompactTime, t)
	}
	bs.Status.LastCompactTime = &metav1.Time{Time: t}
	return nil
}

func (bm *backupScheduleManager) performLogBackupIfNeeded(bs *v1alpha1.BackupSchedule) error {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	// no log backup or already run no need to perform log backup again
	if bs.Spec.LogBackupTemplate == nil || bs.Status.LogBackup != nil {
		return nil
	}

	startTs := bm.now()
	logBackup := buildLogBackup(bs, startTs)
	oldLogBackup, err := bm.deps.BackupControl.GetBackup(logBackup)
	if err == nil {
		if err := bm.deps.BackupControl.DeleteBackup(oldLogBackup); err != nil {
			return fmt.Errorf("backup schedule %s/%s, delete log backup %s failed, err: %v", ns, bsName, oldLogBackup.Name, err)
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("backup schedule %s/%s, get log backup %s failed, err: %v", ns, bsName, oldLogBackup.Name, err)
		return err
	}

	// create log backup
	_, err = bm.deps.BackupControl.CreateBackup(logBackup)
	if err != nil {
		return fmt.Errorf("backup schedule %s/%s, create log backup %s failed, err: %v", ns, bsName, logBackup.Name, err)
	}

	klog.Infof("backup schedule %s/%s, create log backup %s successfully", ns, bsName, logBackup.Name)

	bs.Status.LogBackup = &logBackup.Name
	bs.Status.LogBackupStartTs = &metav1.Time{Time: startTs}
	return nil
}

// getLastScheduledTime return the newest time need to be scheduled according last backup time.
// the return time is not before now and return nil if there's no such time.
func getLastScheduledTime(bs *v1alpha1.BackupSchedule, nowFn nowFn) (*time.Time, error) {
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
			if bs.Status.LastBackupTime == nil && bs.Status.AllBackupCleanTime != nil {
				// Recovery backup schedule from pause status, should refresh AllBackupCleanTime to avoid unschedulable problem
				bs.Status.AllBackupCleanTime = &metav1.Time{Time: nowFn()}
				return nil, controller.RequeueErrorf("recovery backup schedule %s/%s from pause status, refresh AllBackupCleanTime.", ns, bsName)
			}
			klog.Warning("Too many missed start backup schedule time (> 1000). Fail current one.")
			offset := sched.Next(t).Sub(t)
			bs.Status.LastBackupTime = &metav1.Time{Time: time.Now().Add(-offset)}
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

func buildBackup(bs *v1alpha1.BackupSchedule, timestamp time.Time) *v1alpha1.Backup {
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
		pdAddress = fmt.Sprintf("%s-pd.%s:%d", backupSpec.BR.Cluster, clusterNamespace, v1alpha1.DefaultPDClientPort)

		backupPrefix := strings.ReplaceAll(pdAddress, ":", "-") + "-" + timestamp.UTC().Format(v1alpha1.BackupNameTimeFormat)
		if backupSpec.S3 != nil {
			backupSpec.S3.Prefix = path.Join(backupSpec.S3.Prefix, backupPrefix)
		} else if backupSpec.Gcs != nil {
			backupSpec.Gcs.Prefix = path.Join(backupSpec.Gcs.Prefix, backupPrefix)
		} else if backupSpec.Azblob != nil {
			backupSpec.Azblob.Prefix = path.Join(backupSpec.Azblob.Prefix, backupPrefix)
		} else if backupSpec.Local != nil {
			backupSpec.Local.Prefix = path.Join(backupSpec.Local.Prefix, backupPrefix)
		}
	}

	if bs.Spec.ImagePullSecrets != nil {
		backupSpec.ImagePullSecrets = bs.Spec.ImagePullSecrets
	}

	bsLabel := util.CombineStringMap(label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName), bs.Labels)
	backup := &v1alpha1.Backup{
		Spec: backupSpec,
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ns,
			Name:        bs.GetBackupCRDName(timestamp),
			Labels:      bsLabel,
			Annotations: bs.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupScheduleOwnerRef(bs),
			},
		},
	}

	return backup
}

func buildLogBackup(bs *v1alpha1.BackupSchedule, timestamp time.Time) *v1alpha1.Backup {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	if bs.Spec.LogBackupTemplate == nil {
		return nil
	}

	logBackupSpec := *bs.Spec.LogBackupTemplate.DeepCopy()

	logBackupPrefix := "log" + "-" + timestamp.UTC().Format(v1alpha1.BackupNameTimeFormat)
	if logBackupSpec.S3 != nil {
		logBackupSpec.S3.Prefix = path.Join(logBackupSpec.S3.Prefix, logBackupPrefix)
	} else if logBackupSpec.Gcs != nil {
		logBackupSpec.Gcs.Prefix = path.Join(logBackupSpec.Gcs.Prefix, logBackupPrefix)
	} else if logBackupSpec.Azblob != nil {
		logBackupSpec.Azblob.Prefix = path.Join(logBackupSpec.Azblob.Prefix, logBackupPrefix)
	} else if logBackupSpec.Local != nil {
		logBackupSpec.Local.Prefix = path.Join(logBackupSpec.Local.Prefix, logBackupPrefix)
	}

	if bs.Spec.ImagePullSecrets != nil {
		logBackupSpec.ImagePullSecrets = bs.Spec.ImagePullSecrets
	}

	bsLabel := util.CombineStringMap(label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName), bs.Labels)
	logBackup := &v1alpha1.Backup{
		Spec: logBackupSpec,
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ns,
			Name:        bs.GetLogBackupCRDName(),
			Labels:      bsLabel,
			Annotations: bs.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupScheduleOwnerRef(bs),
			},
		},
	}

	return logBackup
}

func buildCompactBackup(bs *v1alpha1.BackupSchedule, startTs time.Time, endTs time.Time) *v1alpha1.CompactBackup {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	if bs.Spec.CompactBackupTemplate == nil {
		return nil
	}

	compactSpec := *bs.Spec.CompactBackupTemplate.DeepCopy()
	compactSpec.StartTs = startTs.UTC().Format(v1alpha1.BackupTimestampFormat)
	compactSpec.EndTs = endTs.UTC().Format(v1alpha1.BackupTimestampFormat)

	logBackupPrefix := "log" + "-" + bs.Status.LogBackupStartTs.UTC().Format(v1alpha1.BackupNameTimeFormat)
	if compactSpec.S3 != nil {
		compactSpec.S3.Prefix = path.Join(compactSpec.S3.Prefix, logBackupPrefix)
	} else if compactSpec.Gcs != nil {
		compactSpec.Gcs.Prefix = path.Join(compactSpec.Gcs.Prefix, logBackupPrefix)
	} else if compactSpec.Azblob != nil {
		compactSpec.Azblob.Prefix = path.Join(compactSpec.Azblob.Prefix, logBackupPrefix)
	} else if compactSpec.Local != nil {
		compactSpec.Local.Prefix = path.Join(compactSpec.Local.Prefix, logBackupPrefix)
	}

	if bs.Spec.ImagePullSecrets != nil {
		compactSpec.ImagePullSecrets = bs.Spec.ImagePullSecrets
	}

	bsLabel := util.CombineStringMap(label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName), bs.Labels)
	compactBackup := &v1alpha1.CompactBackup{
		Spec: compactSpec,
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ns,
			Name:        bs.GetCompactBackupCRDName(),
			Labels:      bsLabel,
			Annotations: bs.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupScheduleOwnerRef(bs),
			},
		},
	}
	klog.Infof("build compact backup %s/%s", ns, bsName)
	return compactBackup
}

func createBackup(bkController controller.BackupControlInterface, bs *v1alpha1.BackupSchedule, timestamp time.Time) (*v1alpha1.Backup, error) {
	bk := buildBackup(bs, timestamp)
	return bkController.CreateBackup(bk)
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

	backupsList, err := bm.getBackupList(bs)
	if err != nil {
		klog.Errorf("backupGCByMaxReservedTime, err: %s", err)
		return
	}

	ascBackups, logBackup := separateSnapshotBackupsAndLogBackup(backupsList)
	if len(ascBackups) == 0 {
		return
	}

	var (
		expiredBackups []*v1alpha1.Backup
		truncateTSO    uint64
	)

	if logBackup != nil {
		expiredBackups, truncateTSO, err = calExpiredBackupsAndLogBackup(ascBackups, logBackup, reservedTime)
		if err != nil {
			klog.Errorf("caculate expired backups with log backup, err: %s", err)
			return
		}
	} else {
		expiredBackups, err = calculateExpiredBackups(ascBackups, reservedTime)
		if err != nil {
			klog.Errorf("caculate expired backups without log backup, err: %s", err)
			return
		}
	}

	for _, backup := range expiredBackups {
		// delete the expired backup
		if err = bm.deps.BackupControl.DeleteBackup(backup); err != nil {
			klog.Errorf("backup schedule %s/%s gc backup %s failed, err %v", ns, bsName, backup.GetName(), err)
			return
		}
		klog.Infof("backup schedule %s/%s gc backup %s success", ns, bsName, backup.GetName())
	}

	if truncateTSO > 0 {
		// truncate the log backup
		if err = bm.deps.BackupControl.TruncateLogBackup(logBackup, truncateTSO); err != nil {
			klog.Errorf("backup schedule %s/%s truncate log backup %s failed, truncateTSO %d, err %v", ns, bsName, logBackup.GetName(), truncateTSO, err)
			return
		}
		klog.Infof("backup schedule %s/%s truncate log backup %s success, truncateTSO %d", ns, bsName, logBackup.GetName(), truncateTSO)
	}

	if len(expiredBackups) == len(backupsList) && len(expiredBackups) > 0 {
		// All backups have been deleted, so the last backup information in the backupSchedule should be reset
		bm.resetLastBackup(bs)
	}
}

// separateSnapshotBackupsAndLogBackup return snapshot backups order by create time asc and log backup
func separateSnapshotBackupsAndLogBackup(backupsList []*v1alpha1.Backup) ([]*v1alpha1.Backup, *v1alpha1.Backup) {
	var (
		ascBackupList = make([]*v1alpha1.Backup, 0)
		logBackup     *v1alpha1.Backup
	)

	for _, backup := range backupsList {
		if backup.Spec.Mode == v1alpha1.BackupModeLog {
			logBackup = backup
			continue
		}
		// Completed or failed backups will be GC'ed
		if !(v1alpha1.IsBackupFailed(backup) || v1alpha1.IsBackupComplete(backup)) {
			continue
		}
		ascBackupList = append(ascBackupList, backup)
	}

	sort.Slice(ascBackupList, func(i, j int) bool {
		return ascBackupList[i].CreationTimestamp.Unix() < ascBackupList[j].CreationTimestamp.Unix()
	})
	return ascBackupList, logBackup
}

// separateAllSnapshotBackupsAndLogBackup return all snapshot backups order by create time asc and log backup,
// this function is for test only
func separateAllSnapshotBackupsAndLogBackup(backupsList []*v1alpha1.Backup) ([]*v1alpha1.Backup, *v1alpha1.Backup) {
	var (
		ascBackupList = make([]*v1alpha1.Backup, 0)
		logBackup     *v1alpha1.Backup
	)

	for _, backup := range backupsList {
		if backup.Spec.Mode == v1alpha1.BackupModeLog {
			logBackup = backup
			continue
		}
		ascBackupList = append(ascBackupList, backup)
	}

	sort.Slice(ascBackupList, func(i, j int) bool {
		return ascBackupList[i].CreationTimestamp.Unix() < ascBackupList[j].CreationTimestamp.Unix()
	})
	return ascBackupList, logBackup
}

// calExpiredBackupsAndLogBackup calculate what backups and log backup we can delete or truncate.
//
// snapshot1----snapshot2--------------snapshot3-----snapshot...----snapshot n-----------> snapshot backups
// ---------------------------------------------------------------------checkpointTS----> log backup
// ---t1-----------t2-------expiredTS-----t3------------t...-----------tn------------------> time
//
// supposed that expiredTS = checkpointTS - retentionPeriod, and expiredTS is between t2 and t3.
// we should promise user can restore to any time between expiredTS and checkpointTS,
// so we should reserve snapshots from snapshot2 to snapshot n.
// then we can delete snapshot1 and truncate log backup to t2.
// so the returned value is snapshot1 and t2.
func calExpiredBackupsAndLogBackup(backupsList []*v1alpha1.Backup, logBackup *v1alpha1.Backup, reservedTime time.Duration) ([]*v1alpha1.Backup, uint64, error) {
	var (
		lastestTSO     uint64
		expiredTSO     uint64
		expiredBackups []*v1alpha1.Backup
		truncateTSO    uint64
		err            error
	)

	// caculate latest ts
	lastestTSO, err = calculateLatestTSO(backupsList, logBackup)
	if err != nil {
		return nil, 0, perrors.Annotate(err, "calculate latest tso")
	}

	expiredTSO = calculateExpiredTSO(lastestTSO, reservedTime)

	// calculate expired backups
	expiredBackups, err = calExpiredBackupsWithLogBackupOn(backupsList, expiredTSO)
	if err != nil {
		return nil, 0, perrors.Annotate(err, "calculate expired backups which should be deleted")
	}

	// no expired backup should be deleted, we can skip and wait next time.
	// because log backup checkpoint ts is always changing, but we should't frequently delete or truncate.
	// we can delete or truncate log backup when there has backup should be deleted, and this is not frequent and will not trigger repeatedly.
	if len(expiredBackups) == 0 {
		return nil, 0, nil
	}

	// calculate truncate tso
	truncateTSO, err = calLogBackupTruncateTSO(backupsList, logBackup, expiredTSO)
	if err != nil {
		return nil, 0, perrors.Annotate(err, "calculate expired log backup tso which should be truncated")
	}

	return expiredBackups, truncateTSO, nil
}

func calculateExpiredBackups(backupsList []*v1alpha1.Backup, reservedTime time.Duration) ([]*v1alpha1.Backup, error) {
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

// calculateLatestTSO calculate the latest tso in auto backups and log backups
func calculateLatestTSO(backupsList []*v1alpha1.Backup, logBackup *v1alpha1.Backup) (uint64, error) {
	var (
		currentBackupTS    uint64
		latestBackupTS     uint64
		latestCheckPointTS uint64
		latestTS           uint64
		err                error
	)

	// get latestCheckPointTs
	latestCheckPointTS, err = config.ParseTSString(logBackup.Status.LogCheckpointTs)
	if err != nil {
		return 0, perrors.Annotatef(err, "parse checkpoint ts of log backup %s/%s", logBackup.Namespace, logBackup.Name)
	}

	for _, backup := range backupsList {
		currentBackupTS, err = config.ParseTSString(backup.Status.CommitTs)
		if err != nil {
			return 0, perrors.Annotatef(err, "parse backup ts of backup %s/%s", backup.Namespace, backup.Name)
		}
		if currentBackupTS > latestBackupTS {
			latestBackupTS = currentBackupTS
		}
	}

	// use the larger of backup and logBackup ts as the latestTS
	latestTS = latestBackupTS
	if latestCheckPointTS > latestTS {
		latestTS = latestCheckPointTS
	}

	return latestTS, nil
}

// calculateExpiredTSO calculate latestTSO - reservedTime
func calculateExpiredTSO(latestTSO uint64, reservedTime time.Duration) uint64 {
	latestTS := config.TSOToTS(latestTSO)
	expiredTS := time.Unix(latestTS, 0).Add(-1 * reservedTime).Unix()
	return config.TSToTSO(expiredTS)
}

// calExpiredBackupsWithLogBackupOn according to expired tso calculate expired backups when there is log backup.
// we will save latest expired snapshot backup to make sure it can be used as the starting point of PiTR Restore.
func calExpiredBackupsWithLogBackupOn(backupsList []*v1alpha1.Backup, expiredTSO uint64) ([]*v1alpha1.Backup, error) {
	var (
		i                int
		currentBackupTSO uint64
		err              error
	)

	for ; i < len(backupsList); i++ {
		currentBackupTSO, err = config.ParseTSString(backupsList[i].Status.CommitTs)
		if err != nil {
			return nil, perrors.Annotatef(err, "parse backup ts of backup %s/%s", backupsList[i].Namespace, backupsList[i].Name)
		}

		if currentBackupTSO > expiredTSO {
			break
		}
	}

	if i != 0 {
		return backupsList[:i-1], nil
	}

	return nil, nil
}

// calLogBackupTruncateTSO according to expired tso calculate log backup truncate tso
func calLogBackupTruncateTSO(backupsList []*v1alpha1.Backup, logBackup *v1alpha1.Backup, expiredTSO uint64) (uint64, error) {
	var truncateTSO uint64
	for i := 0; i < len(backupsList); i++ {
		currentBackupTSO, err := config.ParseTSString(backupsList[i].Status.CommitTs)
		if err != nil {
			return 0, perrors.Annotatef(err, "parse backup ts of backup %s/%s", backupsList[i].Namespace, backupsList[i].Name)
		}

		if currentBackupTSO > expiredTSO {
			break
		}

		truncateTSO = currentBackupTSO
	}

	// check truncate tso is within log backup's effective range
	isTruncateTSOInLogBackup, err := checkTruncateTSOWithinLogBackupRange(logBackup, truncateTSO)
	if err != nil {
		return 0, perrors.Annotate(err, "check truncate ts in log backup")
	}
	if isTruncateTSOInLogBackup {
		return truncateTSO, nil
	}

	return 0, nil
}

func checkTruncateTSOWithinLogBackupRange(logBackup *v1alpha1.Backup, truncateTSO uint64) (bool, error) {
	var (
		startTSO      uint64
		checkPointTSO uint64
		err           error
	)

	startTSO, err = config.ParseTSString(logBackup.Status.CommitTs)
	if err != nil {
		return false, perrors.Annotatef(err, "parse commit ts of log backup %s/%s", logBackup.Namespace, logBackup.Name)
	}

	checkPointTSO, err = config.ParseTSString(logBackup.Status.LogCheckpointTs)
	if err != nil {
		return false, perrors.Annotatef(err, "parse checkpoint ts of log backup %s/%s", logBackup.Namespace, logBackup.Name)
	}

	if truncateTSO > startTSO && truncateTSO <= checkPointTSO {
		return true, nil
	}

	return false, nil
}

func (bm *backupScheduleManager) backupGCByMaxBackups(bs *v1alpha1.BackupSchedule) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupsList, err := bm.getBackupList(bs)
	if err != nil {
		klog.Errorf("backupGCByMaxBackups failed, err: %s", err)
		return
	}

	sort.Sort(byCreateTimeDesc(backupsList))

	var deleteCount int
	for i, backup := range backupsList {
		if i < int(*bs.Spec.MaxBackups) {
			continue
		}
		// delete the backup
		if err := bm.deps.BackupControl.DeleteBackup(backup); err != nil {
			klog.Errorf("backup schedule %s/%s gc backup %s failed, err %v", ns, bsName, backup.GetName(), err)
			return
		}
		deleteCount += 1
		klog.Infof("backup schedule %s/%s gc backup %s success", ns, bsName, backup.GetName())
	}

	if deleteCount == len(backupsList) && deleteCount > 0 {
		// All backups have been deleted, so the last backup information in the backupSchedule should be reset
		bm.resetLastBackup(bs)
	}
}

func (bm *backupScheduleManager) resetLastBackup(bs *v1alpha1.BackupSchedule) {
	bs.Status.LastBackupTime = nil
	bs.Status.LastBackup = ""
	bs.Status.AllBackupCleanTime = &metav1.Time{Time: bm.now()}
}

func (bm *backupScheduleManager) getBackupList(bs *v1alpha1.BackupSchedule) ([]*v1alpha1.Backup, error) {
	ns := bs.GetNamespace()
	bsName := bs.GetName()

	backupLabels := label.NewBackupSchedule().Instance(bsName).BackupSchedule(bsName)
	selector, err := backupLabels.Selector()
	if err != nil {
		return nil, fmt.Errorf("generate backup schedule %s/%s label selector failed, err: %v", ns, bsName, err)
	}
	backupsList, err := bm.deps.BackupLister.Backups(ns).List(selector)
	if err != nil {
		return nil, fmt.Errorf("get backup schedule %s/%s backup list failed, selector: %s, err: %v", ns, bsName, selector, err)
	}

	return backupsList, nil
}

type byCreateTimeDesc []*v1alpha1.Backup

func (b byCreateTimeDesc) Len() int      { return len(b) }
func (b byCreateTimeDesc) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCreateTimeDesc) Less(i, j int) bool {
	return b[j].ObjectMeta.CreationTimestamp.Before(&b[i].ObjectMeta.CreationTimestamp)
}

var _ backup.BackupScheduleManager = &backupScheduleManager{}

type FakeBackupScheduleManager struct {
	err error
}

func NewFakeBackupScheduleManager() *FakeBackupScheduleManager {
	return &FakeBackupScheduleManager{}
}

func (m *FakeBackupScheduleManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeBackupScheduleManager) Sync(bs *v1alpha1.BackupSchedule) error {
	if m.err != nil {
		return m.err
	}

	if bs.Status.LastBackupTime != nil {
		// simulate status update
		bs.Status.LastBackupTime = &metav1.Time{Time: bs.Status.LastBackupTime.Add(1 * time.Hour)}
	}
	return nil
}

var _ backup.BackupScheduleManager = &FakeBackupScheduleManager{}
