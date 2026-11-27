// Copyright 2020 PingCAP, Inc.
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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

type helper struct {
	testutils.Helper
}

func newHelper(t *testing.T) *helper {
	h := testutils.NewHelper(t)
	return &helper{*h}
}

func (h *helper) createJob(job *batchv1.Job) {
	g := NewGomegaWithT(h.T)
	deps := h.Deps
	_, err := deps.KubeClientset.BatchV1().Jobs(job.GetNamespace()).Create(context.TODO(), job, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())

	g.Eventually(func() error {
		_, err := deps.JobLister.Jobs(job.GetNamespace()).Get(job.GetName())
		return err
	}, time.Second*10).Should(BeNil())
}

func (h *helper) deleteJob(job *batchv1.Job) {
	g := NewGomegaWithT(h.T)
	deps := h.Deps
	err := deps.KubeClientset.BatchV1().Jobs(job.GetNamespace()).Delete(context.TODO(), job.GetName(), metav1.DeleteOptions{})
	g.Expect(err).Should(BeNil())

	g.Eventually(func() error {
		_, err := deps.JobLister.Jobs(job.GetNamespace()).Get(job.GetName())
		if errors.IsNotFound(err) {
			return nil
		} else {
			return fmt.Errorf("err: %v", err)
		}
	}, time.Second*10).Should(BeNil())
}

func (h *helper) getJob(namespace, name string) *batchv1.Job {
	g := NewGomegaWithT(h.T)
	deps := h.Deps
	var job *batchv1.Job
	var err error

	g.Eventually(func() error {
		job, err = deps.JobLister.Jobs(namespace).Get(name)
		return err
	}, time.Second*5).Should(BeNil())
	return job
}

func (h *helper) updateJob(job *batchv1.Job, compare func(old, new *batchv1.Job) bool) {
	g := NewGomegaWithT(h.T)
	deps := h.Deps
	_, err := deps.KubeClientset.BatchV1().Jobs(job.GetNamespace()).Update(context.TODO(), job, metav1.UpdateOptions{})
	g.Expect(err).Should(BeNil())

	g.Eventually(func() error {
		newJob, err := deps.JobLister.Jobs(job.GetNamespace()).Get(job.GetName())
		if err != nil {
			return err
		}
		if compare(job, newJob) {
			return nil
		} else {
			return fmt.Errorf("compare failed")
		}
	}, time.Second*10).Should(BeNil())
}

// TODO: refactor to reduce duplicated code with restore tests
func (h *helper) hasCondition(ns string, name string, tp v1alpha1.BackupConditionType, reasonSub string) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	get, err := h.Deps.Clientset.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
	for _, c := range get.Status.Conditions {
		if c.Type == tp {
			if reasonSub == "" || strings.Contains(c.Reason, reasonSub) {
				return
			}
			h.T.Fatalf("%s do not match reason %s", reasonSub, c.Reason)
		}
	}
	h.T.Fatalf("%s/%s do not has condition type: %s, cur conds: %v", ns, name, tp, get.Status.Conditions)
}

func invalidBackup() *v1alpha1.Backup {
	b := &v1alpha1.Backup{}
	b.Namespace = "ns"
	b.Name = "invalid_name"
	return b
}

func validDumplingBackup() *v1alpha1.Backup {
	b := &v1alpha1.Backup{
		Spec: v1alpha1.BackupSpec{
			From: &v1alpha1.TiDBAccessConfig{
				Host:                "localhost",
				SecretName:          "secretName",
				TLSClientSecretName: pointer.StringPtr("secretName"),
			},
			StorageSize: "1G",
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Bucket: "s3",
					Prefix: "prefix-",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "env_name",
					Value: "env_value",
				},
				// existing env name will be overwritten for backup
				{
					Name:  "S3_PROVIDER",
					Value: "fake_provider",
				},
			},
		},
	}

	b.Namespace = "ns"
	b.Name = "dump_name"

	return b
}

func genValidBRBackups() []*v1alpha1.Backup {
	var bs []*v1alpha1.Backup

	for i, sp := range testutils.GenValidStorageProviders() {
		b := &v1alpha1.Backup{
			Spec: v1alpha1.BackupSpec{
				From: &v1alpha1.TiDBAccessConfig{
					Host:       "localhost",
					SecretName: fmt.Sprintf("backup_secret_%d", i),
				},
				StorageSize:     "1G",
				StorageProvider: sp,
				Type:            v1alpha1.BackupTypeDB,
				BR: &v1alpha1.BRConfig{
					ClusterNamespace: "ns",
					Cluster:          fmt.Sprintf("tidb_%d", i),
					DB:               "dbName",
				},
				Env: []corev1.EnvVar{
					{
						Name:  fmt.Sprintf("env_name_%d", i),
						Value: fmt.Sprintf("env_value_%d", i),
					},
					// existing env name will be overwritten for backup
					{
						Name:  "BR_LOG_TO_TERM",
						Value: "value",
					},
					// existing env name will be overwritten for cleaner
					{
						Name:  "S3_PROVIDER",
						Value: "value",
					},
				},
			},
		}
		b.Namespace = "ns"
		b.Name = fmt.Sprintf("backup_name_%d", i)
		bs = append(bs, b)
	}

	return bs
}

func TestBackupManagerDumpling(t *testing.T) {
	g := NewGomegaWithT(t)

	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	var err error

	bm := NewBackupManager(deps).(*backupManager)

	// create backup
	backup := validDumplingBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())

	// create relate secret
	helper.CreateSecret(backup)

	err = bm.syncBackupJob(backup)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupScheduled, "")
	job, err := deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(context.TODO(), backup.GetBackupJobName(), metav1.GetOptions{})
	g.Expect(err).Should(BeNil())

	// check pod env are set correctly
	env1 := corev1.EnvVar{
		Name:  "env_name",
		Value: "env_value",
	}
	env2Yes := corev1.EnvVar{
		Name:  "S3_PROVIDER",
		Value: "fake_provider",
	}
	env2No := corev1.EnvVar{
		Name:  "S3_PROVIDER",
		Value: "",
	}
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElement(env1))
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElement(env2Yes))
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).NotTo(ContainElement(env2No))
}

func TestBackupManagerBR(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	var err error

	bm := NewBackupManager(deps).(*backupManager)

	// test invalid Backup spec
	backup := invalidBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())
	err = bm.syncBackupJob(backup)
	g.Expect(err).ShouldNot(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupInvalid, "")

	// test valid backups
	for i, backup := range genValidBRBackups() {
		_, err := deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
		g.Expect(err).Should(BeNil())

		// create relate secret
		helper.CreateSecret(backup)

		// failed to get relate tc
		err = bm.syncBackupJob(backup)
		g.Expect(err).ShouldNot(BeNil())
		helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupRetryTheFailed, "failed to fetch tidbcluster")

		// create relate tc and try again should success and job created.
		helper.CreateTC(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, false, false)
		err = bm.syncBackupJob(backup)
		g.Expect(err).Should(BeNil())
		helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupScheduled, "")
		job, err := deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(context.TODO(), backup.GetBackupJobName(), metav1.GetOptions{})
		g.Expect(err).Should(BeNil())

		// check pod env are set correctly
		env1 := corev1.EnvVar{
			Name:  fmt.Sprintf("env_name_%d", i),
			Value: fmt.Sprintf("env_value_%d", i),
		}
		env2Yes := corev1.EnvVar{
			Name:  "BR_LOG_TO_TERM",
			Value: "value",
		}
		env2No := corev1.EnvVar{
			Name:  "BR_LOG_TO_TERM",
			Value: string(rune(1)),
		}
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElement(env1))
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElement(env2Yes))
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).NotTo(ContainElement(env2No))
	}
}

func TestClean(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps

	for _, backup := range genValidBRBackups() {
		// make the backup need to be clean
		backup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete

		_, err := deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
		g.Expect(err).Should(BeNil())
		helper.CreateSecret(backup)
		helper.CreateTC(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, false, false)

		statusUpdater := controller.NewRealBackupConditionUpdater(deps.Clientset, deps.BackupLister, deps.Recorder)
		bc := NewBackupCleaner(deps, statusUpdater)

		// test empty backup.Status.BackupPath
		backup.Status.BackupPath = ""
		err = bc.Clean(backup)
		g.Expect(err).Should(BeNil())
		helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupClean, "")

		// test clean job created
		backup.Status.BackupPath = "/path"
		err = bc.Clean(backup)
		g.Expect(err).Should(BeNil())
		helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupClean, "")
		_, err = deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(context.TODO(), backup.GetCleanJobName(), metav1.GetOptions{})
		g.Expect(err).Should(BeNil())

		// test already have a clean job running
		g.Eventually(func() error {
			_, err := deps.JobLister.Jobs(backup.Namespace).Get(backup.GetCleanJobName())
			return err
		}, time.Second*10).Should(BeNil())
		err = bc.Clean(backup)
		g.Expect(err).Should(BeNil())

		// test have a backup job completed
		completedJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backup.GetBackupJobName(),
				Namespace: backup.Namespace,
			},
			Status: batchv1.JobStatus{
				CompletionTime: &metav1.Time{},
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
		helper.createJob(completedJob)
		err = bc.Clean(backup)
		g.Expect(err).Should(BeNil())
		helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupClean, "")
		_, err = deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(context.TODO(), backup.GetCleanJobName(), metav1.GetOptions{})
		g.Expect(err).Should(BeNil())  // job shouldn't be deleted
		helper.deleteJob(completedJob) // clean job after test

		// test have a backup job running
		runningJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backup.GetBackupJobName(),
				Namespace: backup.Namespace,
			},
			Status: batchv1.JobStatus{},
		}
		helper.createJob(runningJob)
		err = bc.Clean(backup)
		g.Expect(err).Should(BeNil())
		g.Eventually(func() bool {
			_, err = deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(context.TODO(), backup.GetBackupJobName(), metav1.GetOptions{})
			return errors.IsNotFound(err)
		}, time.Second*10).Should(BeTrue()) // job should be deleted

		// test have a backup job deleting
		deletingJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              backup.GetBackupJobName(),
				Namespace:         backup.Namespace,
				DeletionTimestamp: &metav1.Time{},
			},
		}
		helper.createJob(deletingJob)
		err = bc.Clean(backup)
		g.Expect(err).Should(BeNil())
	}

}

func TestCleanLogBackupWaitsForStop(t *testing.T) {
	for _, tc := range []struct {
		name, startFailure, commitTS, region                  string
		startPhase, historicalPhase                           v1alpha1.BackupConditionType
		clusterExists, wantStop, alreadyStopped               bool
		startInFlight, startJobFailed, stopSubcommandComplete bool
	}{
		{name: "never started"},
		{name: "start metadata failed", startPhase: v1alpha1.BackupFailed, startFailure: "GetLogBackupMetadataFailed", clusterExists: true, wantStop: true},
		{name: "start metadata failed and cluster deleted", startPhase: v1alpha1.BackupFailed, startFailure: "GetLogBackupMetadataFailed"},
		{name: "start reached prepare", startPhase: v1alpha1.BackupPrepare, clusterExists: true, wantStop: true},
		{name: "start reached running", startPhase: v1alpha1.BackupRetryTheFailed, historicalPhase: v1alpha1.BackupRunning, clusterExists: true, wantStop: true},
		{name: "start reached complete", startPhase: v1alpha1.BackupComplete, clusterExists: true, wantStop: true},
		{name: "start job exists before status was written", startInFlight: true, clusterExists: true, wantStop: true},
		{name: "start job finished without start status", startJobFailed: true, clusterExists: true, wantStop: true},
		{name: "missing secret before start job creation", startPhase: v1alpha1.BackupRetryTheFailed, startFailure: "SecretNotExist", clusterExists: true},
		{name: "BR rejected start and no job remains", startPhase: v1alpha1.BackupFailed, historicalPhase: v1alpha1.BackupRunning, startFailure: "StartLogBackupFailed", clusterExists: true},
		{name: "BR rejected start and job is terminal", startPhase: v1alpha1.BackupFailed, historicalPhase: v1alpha1.BackupRunning, startFailure: "StartLogBackupFailed", startJobFailed: true, clusterExists: true},
		{name: "BR rejected start before the job became terminal", startPhase: v1alpha1.BackupFailed, historicalPhase: v1alpha1.BackupRunning, startFailure: "StartLogBackupFailed", startInFlight: true, clusterExists: true},
		{name: "started", clusterExists: true, wantStop: true, commitTS: "100", region: "us-west-2"},
		{name: "started with an earlier conflict condition", clusterExists: true, wantStop: true, commitTS: "100", startPhase: v1alpha1.BackupFailed, startFailure: "LogBackupTaskAlreadyExists"},
		{name: "already stopped", commitTS: "100", alreadyStopped: true},
		{name: "truncate replaced top-level stop completion", commitTS: "100", stopSubcommandComplete: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := newHelper(t)
			defer helper.Close()
			deps := helper.Deps
			backup := genValidBRBackups()[0]
			backup.UID = "log-backup"
			backup.Spec.BR.ClusterNamespace = "cluster-ns"
			backup.Spec.Mode = v1alpha1.BackupModeLog
			backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
			backup.Spec.From = nil
			deletionTime := metav1.Now()
			backup.DeletionTimestamp = &deletionTime
			backup.Status.BackupPath = "s3://s3/prefix-"
			backup.Status.CommitTs = tc.commitTS
			backup.Spec.S3.Region = tc.region
			if tc.startFailure == "SecretNotExist" {
				backup.Spec.S3.SecretName = "missing-s3-secret"
				backup.Status.BackupPath = ""
			}
			if tc.alreadyStopped {
				backup.Status.Phase = v1alpha1.BackupStopped
				backup.Status.Conditions = []v1alpha1.BackupCondition{{Type: v1alpha1.BackupStopped, Status: corev1.ConditionTrue}}
			}
			if tc.startPhase != "" {
				conditions := []v1alpha1.BackupCondition{{Type: tc.startPhase, Reason: tc.startFailure, Status: corev1.ConditionTrue}}
				if tc.historicalPhase != "" {
					conditions = append(conditions, v1alpha1.BackupCondition{Type: tc.historicalPhase, Status: corev1.ConditionTrue})
				}
				backup.Status.LogSubCommandStatuses = map[v1alpha1.LogSubCommandType]v1alpha1.LogSubCommandStatus{
					v1alpha1.LogStartCommand: {Phase: tc.startPhase, Conditions: conditions},
				}
			}
			if tc.stopSubcommandComplete {
				backup.Status.Phase = v1alpha1.BackupComplete
				backup.Status.Conditions = []v1alpha1.BackupCondition{
					{Command: v1alpha1.LogTruncateCommand, Type: v1alpha1.BackupComplete, Status: corev1.ConditionTrue},
				}
				backup.Status.LogSubCommandStatuses = map[v1alpha1.LogSubCommandType]v1alpha1.LogSubCommandStatus{
					v1alpha1.LogStopCommand: {Phase: v1alpha1.BackupComplete},
				}
			}
			ctx := context.Background()
			backups := deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace)
			_, err := backups.Create(ctx, backup, metav1.CreateOptions{})
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				_, err := deps.BackupLister.Backups(backup.Namespace).Get(backup.Name)
				return err == nil
			}, time.Second*10, time.Millisecond*10)
			if tc.clusterExists {
				helper.CreateTC(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, false, false)
			}
			statusUpdater := controller.NewRealBackupConditionUpdater(deps.Clientset, deps.BackupLister, deps.Recorder)
			cleaner := NewBackupCleaner(deps, statusUpdater)
			jobs := deps.KubeClientset.BatchV1().Jobs(backup.Namespace)
			stopName := backup.GetStopLogBackupJobName()
			startJobMeta := metav1.ObjectMeta{
				Name:            backup.GetBackupJobName(),
				Namespace:       backup.Namespace,
				OwnerReferences: []metav1.OwnerReference{controller.GetBackupOwnerRef(backup)},
			}
			if tc.startJobFailed {
				helper.createJob(&batchv1.Job{
					ObjectMeta: startJobMeta,
					Status:     batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}}},
				})
			}
			if tc.startInFlight {
				startJob := &batchv1.Job{ObjectMeta: startJobMeta}
				// Reconcile against the API without waiting for the informer or a Scheduled status.
				_, err := jobs.Create(ctx, startJob, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			if (tc.startInFlight || tc.startJobFailed) && tc.startFailure == "" {
				err = cleaner.Clean(backup)
				require.True(t, controller.IsRequeueError(err), "observed start Job should be recorded before proceeding: %v", err)
				require.Eventually(t, func() bool {
					current, err := deps.BackupLister.Backups(backup.Namespace).Get(backup.Name)
					if err != nil {
						return false
					}
					start := current.Status.LogSubCommandStatuses[v1alpha1.LogStartCommand]
					if len(start.Conditions) != 1 || start.Conditions[0].Reason != startLogBackupJobObserved {
						return false
					}
					backup = current.DeepCopy()
					return true
				}, time.Second*10, time.Millisecond*10)
			}
			if tc.startInFlight {
				err = cleaner.Clean(backup)
				require.True(t, controller.IsRequeueError(err), "stop should wait for start Job deletion: %v", err)
				_, err = jobs.Get(ctx, backup.GetBackupJobName(), metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "the running start Job should be deleted: %v", err)
				_, err = jobs.Get(ctx, stopName, metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "stop should wait for start Job deletion: %v", err)
				// The replacement cleaner should continue from the persisted status after the Job is gone.
				cleaner = NewBackupCleaner(deps, statusUpdater)
			}

			err = cleaner.Clean(backup)
			if tc.wantStop {
				require.True(t, controller.IsRequeueError(err), "stop Job should be monitored after creation: %v", err)
				stopJob, err := jobs.Get(ctx, stopName, metav1.GetOptions{})
				require.NoError(t, err)
				require.EqualValues(t, constants.DefaultBackoffLimit, *stopJob.Spec.BackoffLimit)
				podEnv := stopJob.Spec.Template.Spec.Containers[0].Env
				if tc.region == "" {
					for _, env := range podEnv {
						require.NotEqual(t, constants.AWSRegionEnv, env.Name)
					}
				} else {
					require.Contains(t, podEnv, corev1.EnvVar{Name: constants.AWSRegionEnv, Value: tc.region})
				}
				err = cleaner.Clean(backup)
				require.True(t, controller.IsRequeueError(err), "nonterminal stop Job should remain monitored: %v", err)
				_, err = jobs.Get(ctx, backup.GetCleanJobName(), metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "clean Job was created before stop completed: %v", err)
				stopJob.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}}
				helper.updateJob(stopJob, func(_, updated *batchv1.Job) bool {
					return len(updated.Status.Conditions) == 1 && updated.Status.Conditions[0].Type == batchv1.JobFailed
				})
				err = cleaner.Clean(backup)
				require.True(t, controller.IsRequeueError(err), "failed stop Job should be deleted and retried: %v", err)
				_, err = jobs.Get(ctx, stopName, metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "failed stop Job should be deleted: %v", err)
				_, err = jobs.Get(ctx, backup.GetCleanJobName(), metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "clean Job was created after a failed stop: %v", err)
				require.Eventually(t, func() bool {
					_, err := deps.JobLister.Jobs(backup.Namespace).Get(stopName)
					return errors.IsNotFound(err)
				}, time.Second*10, time.Millisecond*10)
				err = cleaner.Clean(backup)
				require.True(t, controller.IsRequeueError(err), "stop Job should be recreated and monitored: %v", err)
				stopJob, err = jobs.Get(ctx, stopName, metav1.GetOptions{})
				require.NoError(t, err)
				require.Empty(t, stopJob.Status.Conditions)
				require.NoError(t, statusUpdater.Update(backup, &v1alpha1.BackupCondition{
					Command: v1alpha1.LogStopCommand, Type: v1alpha1.BackupComplete, Status: corev1.ConditionTrue,
				}, nil))
				if tc.startInFlight {
					require.Eventually(t, func() bool {
						current, err := deps.BackupLister.Backups(backup.Namespace).Get(backup.Name)
						return err == nil && current.Status.LogSubCommandStatuses[v1alpha1.LogStopCommand].Phase == v1alpha1.BackupComplete
					}, time.Second*10, time.Millisecond*10)
					truncatedUntil := "42"
					require.NoError(t, statusUpdater.Update(backup, &v1alpha1.BackupCondition{
						Command: v1alpha1.LogTruncateCommand, Type: v1alpha1.BackupComplete, Status: corev1.ConditionTrue,
					}, &controller.BackupUpdateStatus{LogSuccessTruncateUntil: &truncatedUntil}))
					current, err := backups.Get(ctx, backup.Name, metav1.GetOptions{})
					require.NoError(t, err)
					_, complete := v1alpha1.GetBackupCondition(&current.Status, v1alpha1.BackupComplete)
					require.NotNil(t, complete)
					require.Equal(t, v1alpha1.LogTruncateCommand, complete.Command)
					require.True(t, IsLogBackupStoppedForCleanup(current))
				}
			} else {
				require.NoError(t, err)
				_, err = jobs.Get(ctx, stopName, metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "unexpected stop Job: %v", err)
			}
			var stoppedBackup *v1alpha1.Backup
			require.Eventually(t, func() bool {
				current, err := deps.BackupLister.Backups(backup.Namespace).Get(backup.Name)
				if err != nil || !IsLogBackupStoppedForCleanup(current) {
					return false
				}
				stoppedBackup = current.DeepCopy()
				return true
			}, time.Second*10, time.Millisecond*10)
			if !tc.alreadyStopped && !tc.stopSubcommandComplete {
				require.NoError(t, cleaner.Clean(stoppedBackup))
			}
			stored, err := backups.Get(ctx, backup.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.True(t, IsLogBackupStoppedForCleanup(stored))
			if tc.startFailure == "SecretNotExist" {
				require.True(t, v1alpha1.IsBackupClean(stored))
			} else {
				_, err = jobs.Get(ctx, backup.GetCleanJobName(), metav1.GetOptions{})
				require.NoError(t, err)
			}
		})
	}
}

func TestCleanLogBackupRejectsForeignStopJob(t *testing.T) {
	for _, tc := range []struct {
		name      string
		condition batchv1.JobConditionType
		apiLookup bool
	}{
		{name: "failed job in informer", condition: batchv1.JobFailed},
		{name: "completed job from API", condition: batchv1.JobComplete, apiLookup: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := newHelper(t)
			defer helper.Close()
			deps := *helper.Deps
			backup := genValidBRBackups()[0]
			backup.UID = "log-backup"
			backup.Spec.Mode = v1alpha1.BackupModeLog
			backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
			backup.Spec.From = nil
			backup.Status.BackupPath = "s3://s3/logs"
			backup.Status.CommitTs = "100"
			deletionTime := metav1.Now()
			backup.DeletionTimestamp = &deletionTime
			helper.CreateTC(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, false, false)

			ctx := context.Background()
			backups := deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace)
			_, err := backups.Create(ctx, backup, metav1.CreateOptions{})
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				_, err := deps.BackupLister.Backups(backup.Namespace).Get(backup.Name)
				return err == nil
			}, time.Second*10, time.Millisecond*10)

			previousBackup := backup.DeepCopy()
			previousBackup.UID = "previous-log-backup"
			stopName := backup.GetStopLogBackupJobName()
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:            stopName,
					Namespace:       backup.Namespace,
					OwnerReferences: []metav1.OwnerReference{controller.GetBackupOwnerRef(previousBackup)},
				},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{{Type: tc.condition, Status: corev1.ConditionTrue}},
				},
			}
			helper.createJob(job)
			if tc.apiLookup {
				deps.JobLister = batchlisters.NewJobLister(cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{}))
			}

			updater := controller.NewRealBackupConditionUpdater(deps.Clientset, deps.BackupLister, deps.Recorder)
			cleaner := NewBackupCleaner(&deps, updater)
			require.EqualError(t, cleaner.Clean(backup),
				fmt.Sprintf("stop log backup job %s/%s is not controlled by backup %s/%s", backup.Namespace, stopName, backup.Namespace, backup.Name))

			jobs := deps.KubeClientset.BatchV1().Jobs(backup.Namespace)
			remainingJob, err := jobs.Get(ctx, stopName, metav1.GetOptions{})
			require.NoError(t, err)
			require.True(t, metav1.IsControlledBy(remainingJob, previousBackup))
			require.Nil(t, remainingJob.DeletionTimestamp)
			stored, err := backups.Get(ctx, backup.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.False(t, IsLogBackupStoppedForCleanup(stored))
			_, err = jobs.Get(ctx, backup.GetCleanJobName(), metav1.GetOptions{})
			require.True(t, errors.IsNotFound(err), "unexpected clean Job: %v", err)
		})
	}
}

func TestCleanLogBackupWithExistingTask(t *testing.T) {
	for _, policy := range []v1alpha1.CleanPolicyType{v1alpha1.CleanPolicyTypeDelete, v1alpha1.CleanPolicyTypeOnFailure} {
		for _, tc := range []struct {
			name, reason, message string
			topLevel, stale       bool
		}{
			{name: "start subcommand", reason: "LogBackupTaskAlreadyExists"},
			{name: "legacy top-level condition", reason: "StartLogBackupFailed", message: "Error: [BR:Stream:ErrStreamLogTaskExist]stream task already exists", topLevel: true},
			{name: "stale running cache", reason: "LogBackupTaskAlreadyExists", stale: true},
		} {
			t.Run(string(policy)+"/"+tc.name, func(t *testing.T) {
				helper := newHelper(t)
				defer helper.Close()
				deps := helper.Deps
				backup := genValidBRBackups()[0]
				backup.UID = "log-backup"
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.CleanPolicy = policy
				backup.Finalizers = []string{label.BackupProtectionFinalizer}
				deletionTime := metav1.Now()
				backup.DeletionTimestamp = &deletionTime
				backup.Status.BackupPath = "s3://s3/shared-prefix"
				backup.Status.Phase = v1alpha1.BackupFailed
				failure := v1alpha1.BackupCondition{
					Type: v1alpha1.BackupFailed, Status: corev1.ConditionTrue, Reason: tc.reason, Message: tc.message,
				}
				if tc.topLevel {
					backup.Status.Conditions = []v1alpha1.BackupCondition{failure}
				} else {
					backup.Status.LogSubCommandStatuses = map[v1alpha1.LogSubCommandType]v1alpha1.LogSubCommandStatus{
						v1alpha1.LogStartCommand: {Phase: v1alpha1.BackupFailed, Conditions: []v1alpha1.BackupCondition{failure}},
					}
				}

				cached := backup.DeepCopy()
				if tc.stale {
					cached.Status.Phase = v1alpha1.BackupRunning
					cached.Status.LogSubCommandStatuses[v1alpha1.LogStartCommand] = v1alpha1.LogSubCommandStatus{Phase: v1alpha1.BackupRunning}
				}
				ctx := context.Background()
				backups := deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace)
				_, err := backups.Create(ctx, cached, metav1.CreateOptions{})
				require.NoError(t, err)
				liveLister := deps.BackupLister
				require.Eventually(t, func() bool {
					_, err := liveLister.Backups(backup.Namespace).Get(backup.Name)
					return err == nil
				}, time.Second*10, time.Millisecond*10)
				helper.CreateTC(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, false, false)
				startJob := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name: backup.GetBackupJobName(), Namespace: backup.Namespace,
						OwnerReferences: []metav1.OwnerReference{controller.GetBackupOwnerRef(backup)},
					},
				}
				jobFailed := []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}}
				if !tc.topLevel {
					if !tc.stale {
						startJob.Status.Conditions = jobFailed
					}
					helper.createJob(startJob)
				}

				if tc.stale {
					staleCache := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
					require.NoError(t, staleCache.Add(cached.DeepCopy()))
					deps.BackupLister = listers.NewBackupLister(staleCache)
				}
				// Let the updater observe persistence while the cleaner reconciles the deliberately stale state.
				updater := controller.NewRealBackupConditionUpdater(deps.Clientset, liveLister, deps.Recorder)
				cleaner := NewBackupCleaner(deps, updater)
				jobs := deps.KubeClientset.BatchV1().Jobs(backup.Namespace)
				if tc.stale {
					err := cleaner.Clean(cached)
					require.True(t, controller.IsRequeueError(err), "cleanup should wait for BR to report its start result: %v", err)
					_, err = jobs.Get(ctx, startJob.Name, metav1.GetOptions{})
					require.NoError(t, err)
					_, err = backups.Update(ctx, backup, metav1.UpdateOptions{})
					require.NoError(t, err)
					require.Eventually(t, func() bool {
						current, err := liveLister.Backups(backup.Namespace).Get(backup.Name)
						return err == nil && isLogBackupTaskAlreadyExists(current)
					}, time.Second*10, time.Millisecond*10)
					startJob.Status.Conditions = jobFailed
					helper.updateJob(startJob, func(_, updated *batchv1.Job) bool {
						return len(updated.Status.Conditions) == 1 && updated.Status.Conditions[0].Type == batchv1.JobFailed
					})
				}
				require.NoError(t, cleaner.Clean(cached))
				_, err = jobs.Get(ctx, backup.GetStopLogBackupJobName(), metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "unexpected stop Job: %v", err)

				require.Eventually(t, func() bool {
					current, err := liveLister.Backups(backup.Namespace).Get(backup.Name)
					return err == nil && IsLogBackupStoppedForCleanup(current)
				}, time.Second*10, time.Millisecond*10)
				stored, err := backups.Get(ctx, backup.Name, metav1.GetOptions{})
				require.NoError(t, err)
				stop := stored.Status.LogSubCommandStatuses[v1alpha1.LogStopCommand]
				require.Equal(t, v1alpha1.BackupComplete, stop.Phase)
				if tc.stale {
					// The clean reconcile has seen the stop marker but still lacks the start failure.
					cached.Status.LogSubCommandStatuses[v1alpha1.LogStopCommand] = stop
					stored = cached
				}
				require.NoError(t, cleaner.Clean(stored))
				_, err = jobs.Get(ctx, backup.GetCleanJobName(), metav1.GetOptions{})
				require.True(t, errors.IsNotFound(err), "unexpected clean Job: %v", err)
				stored, err = backups.Get(ctx, backup.Name, metav1.GetOptions{})
				require.NoError(t, err)
				require.True(t, IsLogBackupStoppedForCleanup(stored))
				require.True(t, v1alpha1.IsBackupClean(stored))
			})
		}
	}
}

func TestCleanLogBackupIgnoresForeignStartJob(t *testing.T) {
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	backup := genValidBRBackups()[0]
	backup.Name = "foo"
	backup.UID = "log-backup"
	backup.Spec.Mode = v1alpha1.BackupModeLog
	backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
	backup.Status.BackupPath = "s3://s3/logs"
	deletionTime := metav1.Now()
	backup.DeletionTimestamp = &deletionTime

	snapshot := genValidBRBackups()[0]
	snapshot.Name = "foo-log-start"
	snapshot.UID = "snapshot-backup"
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: snapshot.GetBackupJobName(), Namespace: snapshot.Namespace,
		OwnerReferences: []metav1.OwnerReference{controller.GetBackupOwnerRef(snapshot)},
	}}
	require.Equal(t, backup.GetBackupJobName(), job.Name)
	helper.createJob(job)

	ctx := context.Background()
	backups := deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace)
	_, err := backups.Create(ctx, backup, metav1.CreateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, err := deps.BackupLister.Backups(backup.Namespace).Get(backup.Name)
		return err == nil
	}, time.Second*10, time.Millisecond*10)

	updater := controller.NewRealBackupConditionUpdater(deps.Clientset, deps.BackupLister, deps.Recorder)
	cleaner := NewBackupCleaner(deps, updater)
	require.NoError(t, cleaner.Clean(backup))
	stored, err := backups.Get(ctx, backup.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, IsLogBackupStoppedForCleanup(stored))
	require.NotContains(t, stored.Status.LogSubCommandStatuses, v1alpha1.LogStartCommand)
	require.NoError(t, cleaner.Clean(stored))
	jobs := deps.KubeClientset.BatchV1().Jobs(backup.Namespace)
	_, err = jobs.Get(ctx, backup.GetStopLogBackupJobName(), metav1.GetOptions{})
	require.True(t, errors.IsNotFound(err), "unexpected stop Job: %v", err)
	_, err = jobs.Get(ctx, backup.GetCleanJobName(), metav1.GetOptions{})
	require.NoError(t, err)
	remainingJob, err := jobs.Get(ctx, job.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, metav1.IsControlledBy(remainingJob, snapshot))
	require.Nil(t, remainingJob.DeletionTimestamp)
}

func genVolumeBackup() *v1alpha1.Backup {
	b := &v1alpha1.Backup{
		Spec: v1alpha1.BackupSpec{
			FederalVolumeBackupPhase: v1alpha1.FederalVolumeBackupInitialize,
			BR: &v1alpha1.BRConfig{
				ClusterNamespace: "ns",
				Cluster:          "tidb-test",
			},
			Mode: v1alpha1.BackupModeVolumeSnapshot,
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Bucket:   "s3",
					Prefix:   "prefix-",
					Endpoint: "s3://localhost:80",
				},
			},
		},
	}
	b.Namespace = "ns"
	b.Name = "test-volume-backup"
	return b
}

func TestVolumeBackupInitFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	var err error

	bm := NewBackupManager(deps).(*backupManager)
	backup := genVolumeBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())
	helper.CreateTC(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, true, false)

	err = bm.syncBackupJob(backup)
	g.Expect(err).Should(BeNil())

	backup, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Get(context.TODO(), backup.Name, metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
	g.Expect(backup.Status.Phase).ShouldNot(Equal(v1alpha1.VolumeBackupInitializeFailed))

	job := helper.getJob(backup.Namespace, backup.GetVolumeBackupInitializeJobName())
	job.Status.Conditions = append(job.Status.Conditions, batchv1.JobCondition{
		Type:   batchv1.JobFailed,
		Status: corev1.ConditionTrue,
	})
	helper.updateJob(job, func(old, new *batchv1.Job) bool {
		for _, condition := range new.Status.Conditions {
			if condition.Type == batchv1.JobFailed {
				return true
			}
		}
		return false
	})

	err = bm.syncBackupJob(backup)
	g.Expect(err).ShouldNot(BeNil())

	backup, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Get(context.TODO(), backup.Name, metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
	g.Expect(backup.Status.Phase).Should(Equal(v1alpha1.VolumeBackupInitializeFailed))
}

func TestVolumeBackupAgainstInvalidTC(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	var err error

	bm := NewBackupManager(deps).(*backupManager)
	backup := genVolumeBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())
	helper.CreateTC(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, false, false)

	err = bm.syncBackupJob(backup)
	g.Expect(err.Error()).Should(MatchRegexp("only support volume snapshot backup across k8s clusters"))
}

func TestVolumeBackupAgainstTCWithNoTiKV(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	var err error

	bm := NewBackupManager(deps).(*backupManager)
	backup := genVolumeBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())
	helper.CreateTCWithNoTiKV(backup.Spec.BR.ClusterNamespace, backup.Spec.BR.Cluster, true, false)

	err = bm.syncBackupJob(backup)
	g.Expect(err.Error()).Should(MatchRegexp("not support backup TiDB cluster with no tikv replica"))
}

func TestUnmarshalPauseV2Info(t *testing.T) {
	input := `{
        "severity": "manual",
        "operation_hostname": "host-01",
        "operation_pid": 1234,
        "operation_time": "2023-10-05T15:04:05Z",
        "payload_type": "text/plain",
        "payload": "SGVsbG8="
    }`
	var info PauseV2Info
	err := json.Unmarshal([]byte(input), &info)
	require.NoError(t, err)
}
