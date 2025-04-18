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

	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env1))
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env2Yes))
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).NotTo(gomega.ContainElement(env2No))
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
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env1))
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env2Yes))
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).NotTo(gomega.ContainElement(env2No))
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
