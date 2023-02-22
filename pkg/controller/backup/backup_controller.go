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
		klog.V(4).Infof("Finished syncing Backup %q (%v)", key, time.Since(startTime))
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

	if v1alpha1.IsBackupFailed(newBackup) {
		klog.V(4).Infof("backup %s/%s is Failed, skipping.", ns, name)
		return
	}

	if v1alpha1.IsBackupScheduled(newBackup) || v1alpha1.IsBackupRunning(newBackup) || v1alpha1.IsBackupPrepared(newBackup) || v1alpha1.IsLogBackupStopped(newBackup) {
		klog.V(4).Infof("backup %s/%s is already Scheduled, Running, Preparing or Failed, skipping.", ns, name)
		// TODO: log backup check all subcommand job's pod status
		if newBackup.Spec.Mode == v1alpha1.BackupModeLog {
			return
		}

		// backup retry after it has been scheduled, and scheduled means already create k8s job
		if newBackup.Status.Phase == v1alpha1.BackupRetryFailed {
			klog.V(4).Infof("backup %s/%s is retrying after it has been scheduled", ns, name)
			c.enqueueBackup(newBackup)
			return
		}

		// check whether backup pod and job failed
		isPodOrJobFailed, reason, err := c.isBackupPodOrJobFailed(newBackup)
		if err != nil {
			klog.Errorf("Fail to check backup %s/%s pod or job status, %v", ns, name, err)
		}

		if !isPodOrJobFailed {
			return
		}

		// not snapshot backup, just mark as failed
		if newBackup.Spec.Mode != v1alpha1.BackupModeSnapshot {
			err = c.control.UpdateCondition(newBackup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "AlreadyFailed",
				Message: reason,
			})
			if err != nil {
				klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
			}
			return
		}

		// restart snapshot backup after pod or job failed
		err = c.tryToRestartSnapshotBackup(newBackup, reason)
		if err != nil {
			klog.Errorf("Fail to restart snapshot backup %s/%s, %v", ns, name, err)
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

func (c *Controller) isBackupPodOrJobFailed(backup *v1alpha1.Backup) (bool, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	jobName := backup.GetBackupJobName()

	// check pod status
	selector, err := label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name).Selector()
	if err != nil {
		klog.Errorf("Fail to generate selector for backup %s/%s, error %v", ns, name, err)
		return false, "", err
	}
	pods, err := c.deps.PodLister.Pods(ns).List(selector)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Fail to list pod for backup %s/%s with selector %s, error %v", ns, name, selector, err)
		return false, "", err
	}
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodFailed {
			klog.Infof("backup %s/%s has failed pod %s.", ns, name, pod.Name)
			return true, fmt.Sprintf("Pod %s has failed", pod.Name), nil
		}
	}

	// to avoid missing pod failed events, double check job status
	job, err := c.deps.JobLister.Jobs(ns).Get(jobName)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Fail to get job %s for backup %s/%s, error %v ", jobName, ns, name, err)
		return false, "", err
	}
	if job != nil {
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				klog.Infof("backup %s/%s has failed job %s", ns, name, jobName)
				return true, fmt.Sprintf("Job %s has failed", jobName), nil
			}
		}
	}
	return false, "", nil
}

func (c *Controller) tryToRestartSnapshotBackup(backup *v1alpha1.Backup, reason string) error {
	var (
		ns                 = backup.GetNamespace()
		name               = backup.GetName()
		isExceedRetryLimit = backup.Status.ExponentialBackoffRetry.Count > 2
		isTimeToRetry      bool
		err                error
	)
	klog.V(4).Infof("try to restart backup %s/%s, failed reason %s", ns, name, reason)

	if isExceedRetryLimit {
		klog.Errorf("backup %s/%s exceed retry limit, current %d, max %d, failed reason %s", ns, name, backup.Status.ExponentialBackoffRetry.Count, 2, reason)
		err = c.control.UpdateCondition(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "AlreadyFailed",
			Message: fmt.Sprintf("exceed retry limit, current %d, max %d, failed reason %s", backup.Status.ExponentialBackoffRetry.Count, 2, reason),
		})
		if err != nil {
			klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
			return err
		}
		return nil
	}

	isTimeToRetry, err = c.isTimeToRetry(backup)
	if err != nil {
		klog.Errorf("Fail to check whether is the time to retry of backup %s/%s", ns, name)
		return err
	}

	if !isTimeToRetry {
		klog.V(4).Infof("backup %s/%s is not the time to retry, next retry time is %s", ns, name, backup.Status.ExponentialBackoffRetry.NextRetryAt)
		return nil
	}

	err = c.cleanBackupOldJob(backup)
	if err != nil {
		klog.Errorf("Fail to clean job of backup %s/%s, error is %v", ns, name, err)
		return err
	}

	err = c.resetRetryRecordToNext(backup)
	if err != nil {
		klog.Errorf("Fail to refresh retry record of backup %s/%s, error is %v", ns, name, err)
		return err
	}

	// retry
	err = c.control.UpdateCondition(backup, &v1alpha1.BackupCondition{
		Type:    v1alpha1.BackupRetryFailed,
		Status:  corev1.ConditionTrue,
		Reason:  "RetryFailedBackup",
		Message: reason,
	})

	if err != nil {
		klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
		return err
	}

	// add restart contidion to clean data before run br command
	err = c.control.UpdateCondition(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRestart,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
		return err
	}

	c.enqueueBackup(backup)
	return nil
}

func (c *Controller) isTimeToRetry(backup *v1alpha1.Backup) (bool, error) {
	var (
		ns             = backup.GetNamespace()
		name           = backup.GetName()
		now            = metav1.Now()
		count          = backup.Status.ExponentialBackoffRetry.Count
		currentRetryAt = backup.Status.ExponentialBackoffRetry.CurrentRetryAt
		nextRetryAt    = backup.Status.ExponentialBackoffRetry.NextRetryAt
	)

	if count > v1alpha1.MaxRetryTimes {
		return false, nil
	}

	if count == 0 || (currentRetryAt != nil && nextRetryAt != nil && currentRetryAt.After(nextRetryAt.Time)) {
		next := metav1.NewTime(now.Add(v1alpha1.RetryDurationMap[count]))
		nextCount := count + 1
		newStatus := &controller.BackupUpdateStatus{
			RetryCount:  &nextCount,
			NextRetryAt: &next,
		}
		nextRetryAt = &next
		if err := c.control.UpdateBackupStatus(backup, nil, newStatus); err != nil {
			klog.Errorf("Fail to update the retry status of backup %s/%s, %v", ns, name, err)
			return false, err
		}
	}

	if now.After(nextRetryAt.Time) {
		return true, nil
	}

	return false, nil
}

func (c *Controller) resetRetryRecordToNext(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	now := metav1.Now()

	newStatus := &controller.BackupUpdateStatus{
		CurrentRetryAt: &now,
	}
	if err := c.control.UpdateBackupStatus(backup, nil, newStatus); err != nil {
		klog.Errorf("Fail to update the retry status of backup %s/%s, %v", ns, name, err)
		return err
	}
	return nil
}

func (c *Controller) cleanBackupOldJob(backup *v1alpha1.Backup) error {
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
