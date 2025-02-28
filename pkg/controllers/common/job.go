// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	rtClient "sigs.k8s.io/controller-runtime/pkg/client"

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
)

var (
	JobLifecycleManager = &jobLifecycleManager{}
)

type jobLifecycleManager struct {
}

func (m *jobLifecycleManager) Sync(ctx context.Context, job runtime.Job, c client.Client) error {
	jobNs, jobName := job.Object().GetNamespace(), job.Object().GetName()

	// TODO(ideascf): how to handle if the job is deleted, wait for the job to be Completed or Failed?

	// finalizer management
	if job.NeedAddFinalizer() {
		if err := k8s.EnsureFinalizer(ctx, c, job.Object()); err != nil {
			return fmt.Errorf("failed to ensure finalizer for job %s: %w", job.Object(), err)
		}
	}
	if job.NeedRemoveFinalizer() {
		if err := k8s.RemoveFinalizer(ctx, c, job.Object()); err != nil {
			return fmt.Errorf("failed to remove finalizer for job %s: %w", job.Object(), err)
		}
	}

	if job.Completed() || job.Failed() || job.Invalid() {
		klog.Infof("job %s/%s is Finished, skipping retry.", jobNs, jobName)
		return nil
	}

	// do retry if needed
	retriable, ok := job.(runtime.RetriableJob)
	if !ok {
		return nil
	}
	if !retriable.NeedRetry() {
		return nil
	}
	klog.Infof("do retry for %s/%s", jobNs, jobName)
	jobFailed, reason, originalReason, err := m.detectK8sJobFailure(ctx, c, retriable)
	if err != nil {
		klog.Errorf("Fail to detect backup %s/%s failure, error %v", jobNs, jobName, err)
		return err
	}
	if !jobFailed {
		return nil
	}
	if err := m.retryWithBackoffPolicy(ctx, c, retriable, reason, originalReason); err != nil {
		klog.Errorf("Fail to restart snapshot backup %s/%s: %v", jobNs, jobName, err)
		return err
	}
	return nil
}

func (m *jobLifecycleManager) detectK8sJobFailure(ctx context.Context, c client.Client, job runtime.RetriableJob) (
	jobFailed bool, reason string, originalReason string, err error) {
	var (
		j = job.Object()
	)

	var (
		ns   = j.GetNamespace()
		name = j.GetName()
	)

	// if failure was recorded, get reason from backup status
	if m.isFailureAlreadyRecorded(job) {
		lastRetryRecord, _ := job.LastRetryRecord()
		reason = lastRetryRecord.RetryReason
		originalReason = lastRetryRecord.OriginalReason
		return true, reason, originalReason, nil
	}

	// check whether backup job failed by checking their status
	jobFailed, reason, originalReason, err = m.isK8sJobFailed(ctx, c, job)
	if err != nil {
		klog.Errorf("Fail to check backup %s/%s job status, %v", ns, name, err)
		return false, "", "", err
	}
	// not failed, make sure reason and originalReason are empty when not failed
	if !jobFailed {
		return false, "", "", nil
	}

	klog.Infof("Detect backup %s/%s job failed, will retry, reason %s, original reason %s ", ns, name, reason, originalReason)
	return jobFailed, reason, originalReason, nil
}

func (m *jobLifecycleManager) isFailureAlreadyRecorded(job runtime.RetriableJob) bool {
	// no record
	lastRetryRecord, ok := job.LastRetryRecord()
	if !ok {
		return false
	}
	return !lastRetryRecord.IsTriggered()
}

func (m *jobLifecycleManager) isK8sJobFailed(ctx context.Context, cli client.Client, job runtime.Job) (
	jobFailed bool, reason string, originalReason string, err error) {
	j := job.Object()
	k8sJobKey := job.K8sJob()
	k8sJob := batchv1.Job{}
	if err := cli.Get(ctx, k8sJobKey, &k8sJob); err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Fail to get k8s job %s for backup %s/%s, error %v ", k8sJobKey, j.GetNamespace(), j.GetName(), err)
		return false, "", "", err
	}
	if errors.IsNotFound(err) || k8sJob.DeletionTimestamp != nil {
		return false, "", "", nil
	}

	for _, condition := range k8sJob.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			reason = fmt.Sprintf("Job %s has failed", k8sJobKey)
			originalReason = condition.Reason
			return true, reason, originalReason, nil
		}
	}
	return false, "", "", nil
}

func (m *jobLifecycleManager) retryWithBackoffPolicy(ctx context.Context, c client.Client, job runtime.RetriableJob, reason, originalReason string) error {
	var (
		ns   = job.Object().GetNamespace()
		name = job.Object().GetName()
		now  = time.Now()
	)
	klog.Infof("retry job %s/%s, retry reason %s, original reason %s", ns, name, reason, originalReason)

	// retry is already created
	lastRetryRecord, ok := job.LastRetryRecord()
	if ok && !lastRetryRecord.IsTriggered() {
		if !m.isTimeToRetry(lastRetryRecord, now) {
			klog.Infof("backup %s/%s is not the time to retry, expected retry time is %s, now is %s", ns, name, lastRetryRecord.ExpectedRetryAt, now)
			return RequeueErrorf(lastRetryRecord.ExpectedRetryAt.Time.Sub(now)+time.Second, "retry backup %s/%s after %s", ns, name, now.Sub(lastRetryRecord.ExpectedRetryAt.Time))
		}
		return m.doRetryLogic(ctx, c, job, now)
	}

	// create next retry
	if err := m.createNextRetry(ctx, c, job, reason, originalReason, now); err != nil {
		return fmt.Errorf("create next retry: %w", err)
	}

	return nil
}

// createNextRetry is used to create the next retry record. The real retry will happen at `ExpectedRetryAt`.
func (m *jobLifecycleManager) createNextRetry(ctx context.Context, c client.Client, job runtime.RetriableJob, reason, originalReason string, now time.Time) error {
	j := job.Object()
	ns := j.GetNamespace()
	name := j.GetName()

	klog.Infof("create the next retry for %s/%s", ns, name)
	nextRunAfter, err := job.Retry(reason, originalReason, now)
	if err != nil {
		klog.Errorf("Fail to retry backup %s/%s, %v", ns, name, err)
		return err
	}
	if err := c.Status().Update(ctx, job.Object()); err != nil {
		klog.Errorf("Fail to update the retry status of backup %s/%s, %v", ns, name, err)
		return err
	}
	if nextRunAfter != nil {
		return RequeueErrorf(*nextRunAfter, "retry backup %s/%s after %s", ns, name, *nextRunAfter)
	}
	return nil
}

// doRetryLogic is used to do the retry logic when the `ExpectedRetryAt` is exceeded.
// For example, it will clean the k8s job and do post retry.
func (m *jobLifecycleManager) doRetryLogic(ctx context.Context, c client.Client, job runtime.RetriableJob, now time.Time) error {
	var (
		ns   = job.Object().GetNamespace()
		name = job.Object().GetName()
	)

	klog.Infof("job %s/%s is retrying after it has been scheduled", ns, name)

	// clean job
	k8sJobKey := job.K8sJob()
	k8sJob := &batchv1.Job{}
	k8sJob.Namespace = k8sJobKey.Namespace
	k8sJob.Name = k8sJobKey.Name
	if err := c.Delete(ctx, k8sJob, rtClient.PropagationPolicy(metav1.DeletePropagationForeground)); client.IgnoreNotFound(err) != nil {
		klog.Errorf("faled to clean k8s Job %s/%s for %s: %s", job.GetNamespace(), job.GetName(), job.K8sJob(), err)
		return err
	}

	klog.Infof("k8s job %s/%s is deleted", ns, k8sJobKey.Name)
	// do post retry
	if err := job.PostRetry(now); err != nil {
		klog.Errorf("Fail to post retry for backup %s/%s, %v", ns, name, err)
		return err
	}
	if err := c.Status().Update(ctx, job.Object()); err != nil {
		klog.Errorf("Fail to update the condition of backup %s/%s, %v", ns, name, err)
		return err
	}

	return nil
}

func (m *jobLifecycleManager) isTimeToRetry(retryRecord brv1alpha1.BackoffRetryRecord, now time.Time) bool {
	return now.After(retryRecord.ExpectedRetryAt.Time)
}
