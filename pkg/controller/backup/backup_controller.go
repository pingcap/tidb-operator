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

package backup

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/backup"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller controls backup.
type Controller struct {
	deps *controller.Dependencies
	// control returns an interface capable of syncing a backup.
	// Abstracted out for testing.
	control ControlInterface
	// backups that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a backup controller.
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultBackupControl(deps.Clientset, backup.NewBackupManager(deps)),
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"backup",
		),
	}

	backupInformer := deps.InformerFactory.Pingcap().V1alpha1().Backups()
	jobInformer := deps.KubeInformerFactory.Batch().V1().Jobs()
	backupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updateBackup,
		UpdateFunc: func(old, cur interface{}) {
			c.updateBackup(cur)
		},
		DeleteFunc: c.updateBackup,
	})
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteJob,
	})

	return c
}

// Name returns backup controller name.
func (c *Controller) Name() string {
	return "backup"
}

// Run runs the backup controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting backup controller")
	defer klog.Info("Shutting down backup controller")

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (c *Controller) processNextWorkItem() bool {
	metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(1)
	defer metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(-1)

	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("Backup: %v, still need sync: %v, requeuing", key.(string), err)
			c.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.V(4).Infof("Backup: %v, ignore err: %v", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("Backup: %v, sync failed, err: %v, requeuing", key.(string), err))
			c.queue.AddRateLimited(key)
		}
	} else {
		c.queue.Forget(key)
	}
	return true
}

// sync syncs the given backup.
func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing Backup %q (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	backup, err := c.deps.BackupLister.Backups(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("Backup has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return c.syncBackup(backup.DeepCopy())
}

func (c *Controller) syncBackup(backup *v1alpha1.Backup) error {
	return c.control.UpdateBackup(backup)
}

func (c *Controller) updateBackup(cur interface{}) {
	newBackup := cur.(*v1alpha1.Backup)
	ns := newBackup.GetNamespace()
	name := newBackup.GetName()

	if newBackup.DeletionTimestamp != nil {
		// the backup is being deleted, we need to do some cleanup work, enqueue backup.
		klog.Infof("backup %s/%s is being deleted", ns, name)
		c.enqueueBackup(newBackup)
		return
	}

	if v1alpha1.IsBackupInvalid(newBackup) {
		klog.V(4).Infof("backup %s/%s is invalid, skipping.", ns, name)
		return
	}

	if v1alpha1.IsBackupComplete(newBackup) {
		klog.V(4).Infof("backup %s/%s is Complete, skipping.", ns, name)
		return
	}

	// when volume backup failed, we should delete initialing job to resume GC and PD schedules, not skip it
	if v1alpha1.IsBackupFailed(newBackup) && newBackup.Spec.Mode != v1alpha1.BackupModeVolumeSnapshot {
		klog.V(4).Infof("backup %s/%s is Failed, skipping.", ns, name)
		return
	}

	if v1alpha1.IsVolumeBackupInitializeFailed(newBackup) {
		klog.V(4).Infof("backup %s/%s is VolumeBackupInitializeFailed, skipping.", ns, name)
		return
	}

	// volume backup has multiple phases, and it can create multiple jobs, not skip it
	if (v1alpha1.IsBackupScheduled(newBackup) || v1alpha1.IsBackupRunning(newBackup) || v1alpha1.IsBackupPrepared(newBackup) || v1alpha1.IsLogBackupStopped(newBackup)) &&
		newBackup.Spec.Mode != v1alpha1.BackupModeVolumeSnapshot {
		klog.V(4).Infof("backup %s/%s is already Scheduled, Running, Preparing or Failed, skipping.", ns, name)
		// TODO: log backup check all subcommand job's pod status
		if newBackup.Spec.Mode == v1alpha1.BackupModeLog {
			return
		}

		// we will create backup job when we mark backup as scheduled status,
		// but the backup job or its pod may failed due to insufficient resources or other reasons in k8s,
		// we should detect this kind of failure and try to restart backup according to spec.backoffRetryPolicy.
		podOrJobFailed, reason, originalReason, err := c.detectBackupJobOrPodFailure(newBackup)
		if err != nil {
			klog.Errorf("Fail to detect backup %s/%s failure, error %v", ns, name, err)
		}
		if !podOrJobFailed {
			return
		}

		// retry backup after detect failure
		err = c.retryAfterFailureDetected(newBackup, reason, originalReason)
		if err != nil {
			klog.Errorf("Fail to restart snapshot backup %s/%s, error %v", ns, name, err)
			return
		}
		return
	}

	klog.V(4).Infof("backup object %s/%s enqueue", ns, name)
	c.enqueueBackup(newBackup)
}

func (c *Controller) deleteJob(obj interface{}) {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return
	}

	ns := job.GetNamespace()
	jobName := job.GetName()
	backup := c.resolveBackupFromJob(ns, job)
	if backup == nil {
		return
	}
	klog.V(4).Infof("Job %s/%s deleted through %v.", ns, jobName, utilruntime.GetCaller())
	c.updateBackup(backup)
}

func (c *Controller) resolveBackupFromJob(namespace string, job *batchv1.Job) *v1alpha1.Backup {
	owner := metav1.GetControllerOf(job)
	if owner == nil {
		return nil
	}

	if owner.Kind != controller.BackupControllerKind.Kind {
		return nil
	}

	backup, err := c.deps.BackupLister.Backups(namespace).Get(owner.Name)
	if err != nil {
		return nil
	}
	if owner.UID != backup.UID {
		return nil
	}
	return backup
}

// enqueueBackup enqueues the given backup in the work queue.
func (c *Controller) enqueueBackup(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}

// detectBackupJobOrPodFailure detect backup job or pod failure.
// it will record failure info to backup status, it is to realize reentrant logic for spec.backoffRetryPolicy. so it only record snapshot backup failed now.
func (c *Controller) detectBackupJobOrPodFailure(backup *v1alpha1.Backup) (
	podOrJobFailed bool, reason string, originalReason string, err error) {
	var (
		ns   = backup.GetNamespace()
		name = backup.GetName()
	)

	// if failure was recorded, get reason from backup status
	if c.isFailureAlreadyRecorded(backup) {
		reason = backup.Status.BackoffRetryStatus[len(backup.Status.BackoffRetryStatus)-1].RetryReason
		originalReason = backup.Status.BackoffRetryStatus[len(backup.Status.BackoffRetryStatus)-1].OriginalReason
		return true, reason, originalReason, nil
	}

	// check whether backup pod and job failed by checking their status
	podOrJobFailed, reason, originalReason, err = c.isBackupPodOrJobFailed(backup)
	if err != nil {
		klog.Errorf("Fail to check backup %s/%s pod or job status, %v", ns, name, err)
		return false, "", "", err
	}
	// not failed, make sure reason and originalReason are empty when not failed
	if !podOrJobFailed {
		return false, "", "", nil
	}

	klog.Infof("Detect backup %s/%s pod or job failed, will retry, reason %s, original reason %s ", ns, name, reason, originalReason)
	// record failure when detect failure
	err = c.recordDetectedFailure(backup, reason, originalReason)
	if err != nil {
		klog.Errorf("failed to record detected failed %s for backup %s/%s", reason, ns, name)
	}
	return podOrJobFailed, reason, originalReason, nil
}

func (c *Controller) isFailureAlreadyRecorded(backup *v1alpha1.Backup) bool {
	// just snapshot backup record failure now
	if backup.Spec.Mode != v1alpha1.BackupModeSnapshot {
		return false
	}
	// retrying
	if isBackoffRetrying(backup) {
		return true
	}
	// no record
	if len(backup.Status.BackoffRetryStatus) == 0 {
		return false
	}
	// latest failure is done, this failure should be a new one and no records
	if isCurrentBackoffRetryDone(backup) {
		return false
	}
	return true
}

func (c *Controller) recordDetectedFailure(backup *v1alpha1.Backup, reason, originalReason string) error {
	ns := backup.GetNamespace()
	name := backup.GetName()

	if backup.Spec.Mode != v1alpha1.BackupModeSnapshot {
		return nil
	}

	retryNum := len(backup.Status.BackoffRetryStatus) + 1
	detectFailedAt := metav1.Now()
	minDuration, err := time.ParseDuration(backup.Spec.BackoffRetryPolicy.MinRetryDuration)
	if err != nil {
		klog.Errorf("fail to parse minRetryDuration %s of backup %s/%s, %v", backup.Spec.BackoffRetryPolicy.MinRetryDuration, ns, name, err)
		return err
	}
	duration := time.Duration(minDuration.Nanoseconds() << (retryNum - 1))
	expectedRetryAt := metav1.NewTime(detectFailedAt.Add(duration))

	// update
	newStatus := &controller.BackupUpdateStatus{
		RetryNum:        &retryNum,
		DetectFailedAt:  &detectFailedAt,
		ExpectedRetryAt: &expectedRetryAt,
		RetryReason:     &reason,
		OriginalReason:  &originalReason,
	}

	klog.Infof("Record backup %s/%s retry status, %v", ns, name, newStatus)
	if err := c.control.UpdateStatus(backup, nil, newStatus); err != nil {
		klog.Errorf("Fail to update the retry status of backup %s/%s, %v", ns, name, err)
		return err
	}

	return nil
}

// retryAfterFailureDetected retry detect failure according to spec.backoffRetryPolicy.
// it only retry snapshot backup now. for others, it will marking failed directly.
func (c *Controller) retryAfterFailureDetected(backup *v1alpha1.Backup, reason, originalReason string) error {
	var (
		ns   = backup.GetNamespace()
		name = backup.GetName()
		err  error
	)

	// not snapshot backup, just mark as failed
	if backup.Spec.Mode != v1alpha1.BackupModeSnapshot {
		err = c.control.UpdateStatus(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "AlreadyFailed",
			Message: fmt.Sprintf("reason %s, original reason %s", reason, originalReason),
		}, nil)
		if err != nil {
			klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
			return err
		}
		return nil
	}

	// retry snapshot backup according to backoff policy
	err = c.retrySnapshotBackupAccordingToBackoffPolicy(backup)
	if err != nil {
		klog.Errorf("Fail to retry snapshot backup %s/%s according to backoff policy , %v", ns, name, err)
		return err
	}
	klog.Infof("Retry snapshot backup %s/%s according to backoff policy", ns, name)
	return nil
}

func (c *Controller) isBackupPodOrJobFailed(backup *v1alpha1.Backup) (
	podOrJobFailed bool, reason string, originalReason string, err error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	jobName := backup.GetBackupJobName()

	// check pod status
	selector, err := label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name).Selector()
	if err != nil {
		klog.Errorf("Fail to generate selector for backup %s/%s, error %v", ns, name, err)
		return false, "", "", err
	}
	pods, err := c.deps.PodLister.Pods(ns).List(selector)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Fail to list pod for backup %s/%s with selector %s, error %v", ns, name, selector, err)
		return false, "", "", err
	}
	// quick return when find one pod failed, basically there should be only one pod
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodFailed {
			klog.Infof("backup %s/%s has failed pod %s. original reason %s", ns, name, pod.Name, pod.Status.Reason)
			reason = fmt.Sprintf("Pod %s has failed", pod.Name)
			originalReason = pod.Status.Reason
			return true, reason, originalReason, nil
		}
	}

	// to avoid missing pod failed events, double check job status
	job, err := c.deps.JobLister.Jobs(ns).Get(jobName)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Fail to get job %s for backup %s/%s, error %v ", jobName, ns, name, err)
		return false, "", "", err
	}
	if job != nil {
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				reason = fmt.Sprintf("Job %s has failed", jobName)
				originalReason = condition.Reason
				return true, reason, originalReason, nil
			}
		}
	}
	return false, "", "", nil
}

// retrySnapshotBackupAccordingToBackoffPolicy retry snapshot backup according to spec.backoffRetryPolicy.
// the main logic is reentrant:
//  1. check whether is retrying which is marked as BackupRetryFailed,
//     if true, clean job and mark RealRetryAt which means current retry is done.
//  2. check whether is retry done, skip.
//  3. check whether exceed retry limit, if true, mark as failed.
//  4. check whether exceed the retry interval which is recorded as ExpectedRetryAt,
//     the value is the time detect failure + MinRetryDuration << (retry num -1),
//     if true, mark as BackupRetryFailed, if not, wait to next loop.
//  5. after mark as BackupRetryFailed, the logic will back to 1 in next loop.
func (c *Controller) retrySnapshotBackupAccordingToBackoffPolicy(backup *v1alpha1.Backup) error {
	if len(backup.Status.BackoffRetryStatus) == 0 {
		return nil
	}
	var (
		ns           = backup.GetNamespace()
		name         = backup.GetName()
		now          = time.Now()
		err          error
		failedReason string
		retryRecord  = backup.Status.BackoffRetryStatus[len(backup.Status.BackoffRetryStatus)-1]
	)
	klog.V(4).Infof("retry backup %s/%s, retry reason %s, original reason %s", ns, name, retryRecord.RetryReason, retryRecord.OriginalReason)

	// check retrying
	if isBackoffRetrying(backup) {
		return c.doRetryFailedBackup(backup)
	}

	// check retry done
	if isCurrentBackoffRetryDone(backup) {
		return nil
	}

	// check is exceed retry limit
	isExceedRetryTimes := isExceedRetryTimes(backup)
	if isExceedRetryTimes {
		failedReason = fmt.Sprintf("exceed retry times, max is %d, failed reason %s, original reason %s", backup.Spec.BackoffRetryPolicy.MaxRetryTimes, retryRecord.RetryReason, retryRecord.OriginalReason)
	}
	isRetryTimeout, err := isRetryTimeout(backup, &now)
	if err != nil {
		klog.Errorf("fail to check whether the retry is timeout, backup is %s/%s, %v", ns, name, err)
		return err
	}
	if isRetryTimeout {
		failedReason = fmt.Sprintf("retry timeout, max is %s, failed reason %s, original reason %s", backup.Spec.BackoffRetryPolicy.RetryTimeout, retryRecord.RetryReason, retryRecord.OriginalReason)
	}

	if isExceedRetryTimes || isRetryTimeout {
		klog.Infof("backup %s/%s exceed retry limit, failed reason %s", ns, name, failedReason)
		err = c.control.UpdateStatus(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "AlreadyFailed",
			Message: failedReason,
		}, nil)
		if err != nil {
			klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
			return err
		}
		return nil
	}

	// check is time to retry
	if !isTimeToRetry(backup, &now) {
		klog.V(4).Infof("backup %s/%s is not the time to retry, expected retry time is %s, now is %s", ns, name, retryRecord.ExpectedRetryAt, now)
		return nil
	}

	klog.V(4).Infof("backup %s/%s is the time to retry, expected retry time is %s, now is %s", ns, name, retryRecord.ExpectedRetryAt, now)

	// update retry status
	err = c.control.UpdateStatus(backup, &v1alpha1.BackupCondition{
		Type:    v1alpha1.BackupRetryTheFailed,
		Status:  corev1.ConditionTrue,
		Reason:  "RetryFailedBackup",
		Message: fmt.Sprintf("reason %s, original reason %s", retryRecord.RetryReason, retryRecord.OriginalReason),
	}, nil)
	if err != nil {
		klog.Errorf("Fail to update the retry status of backup %s/%s, %v", ns, name, err)
		return err
	}

	return nil
}

func isExceedRetryTimes(backup *v1alpha1.Backup) bool {
	records := backup.Status.BackoffRetryStatus
	if len(records) == 0 {
		return false
	}

	currentRetryNum := records[len(records)-1].RetryNum
	return currentRetryNum > backup.Spec.BackoffRetryPolicy.MaxRetryTimes
}

func isRetryTimeout(backup *v1alpha1.Backup, now *time.Time) (bool, error) {
	records := backup.Status.BackoffRetryStatus
	if len(records) == 0 {
		return false, nil
	}
	firstDetectAt := records[0].DetectFailedAt
	retryTimeout, err := time.ParseDuration(backup.Spec.BackoffRetryPolicy.RetryTimeout)
	if err != nil {
		klog.Errorf("fail to parse retryTimeout %s of backup %s/%s, %v", backup.Spec.BackoffRetryPolicy.RetryTimeout, backup.Namespace, backup.Name, err)
		return false, err
	}
	return now.Unix()-firstDetectAt.Unix() > int64(retryTimeout)/int64(time.Second), nil
}

func isTimeToRetry(backup *v1alpha1.Backup, now *time.Time) bool {
	if len(backup.Status.BackoffRetryStatus) == 0 {
		return false
	}
	retryRecord := backup.Status.BackoffRetryStatus[len(backup.Status.BackoffRetryStatus)-1]
	return time.Now().Unix() > retryRecord.ExpectedRetryAt.Unix()

}

func (c *Controller) cleanBackupOldJobIfExist(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	jobName := backup.GetBackupJobName()
	oldJob, err := c.deps.JobLister.Jobs(ns).Get(jobName)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Fail to get job %s  for backup %s/%s, error %v ", jobName, ns, name, err)
		return nil
	}
	if oldJob != nil {
		if err := c.deps.JobControl.DeleteJob(backup, oldJob); err != nil {
			klog.Errorf("backup %s/%s delete job %s failed, err: %v", ns, name, jobName, err)
			return nil
		}
	}
	return nil
}

func (c *Controller) doRetryFailedBackup(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	klog.V(4).Infof("backup %s/%s is retrying after it has been scheduled", ns, name)

	// retry done
	if isCurrentBackoffRetryDone(backup) {
		// clean job is asynchronous, we need enqueue again,
		// the backup status will be scheduled after create new job, and then this reconcile is really done.
		c.enqueueBackup(backup)
		return nil
	}

	// clean job
	err := c.cleanBackupOldJobIfExist(backup)
	if err != nil {
		klog.Errorf("Fail to clean job of backup %s/%s, error is %v", ns, name, err)
		return err
	}

	// retry done, mark RealRetryAt
	RealRetryStatus := &controller.BackupUpdateStatus{
		RealRetryAt: &metav1.Time{Time: time.Now()},
	}

	// add restart condition to clean data before run br command
	err = c.control.UpdateStatus(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRestart,
		Status: corev1.ConditionTrue,
	}, RealRetryStatus)
	if err != nil {
		klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
		return err
	}

	c.enqueueBackup(backup)
	return nil
}

func isBackoffRetrying(backup *v1alpha1.Backup) bool {
	if backup.Spec.Mode != v1alpha1.BackupModeSnapshot {
		return false
	}
	return backup.Status.Phase == v1alpha1.BackupRetryTheFailed
}

func isCurrentBackoffRetryDone(backup *v1alpha1.Backup) bool {
	if backup.Spec.Mode != v1alpha1.BackupModeSnapshot {
		return false
	}
	if len(backup.Status.BackoffRetryStatus) == 0 {
		return false
	}
	return backup.Status.BackoffRetryStatus[len(backup.Status.BackoffRetryStatus)-1].RealRetryAt != nil
}
