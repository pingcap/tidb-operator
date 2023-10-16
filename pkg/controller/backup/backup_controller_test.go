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
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestBackupControllerEnqueueBackup(t *testing.T) {
	g := NewGomegaWithT(t)
	backup := newBackup()
	bkc, _, _ := newFakeBackupController()
	bkc.enqueueBackup(backup)
	g.Expect(bkc.queue.Len()).To(Equal(1))
}

func TestBackupControllerEnqueueBackupFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	bkc, _, _ := newFakeBackupController()
	bkc.enqueueBackup(struct{}{})
	g.Expect(bkc.queue.Len()).To(Equal(0))
}

func TestBackupControllerUpdateBackup(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                 string
		backupHasBeenDeleted bool
		conditionType        v1alpha1.BackupConditionType // only one condition used now.
		beforeUpdateFn       func(*GomegaWithT, *Controller, *v1alpha1.Backup)
		expectFn             func(*GomegaWithT, *Controller)
		afterUpdateFn        func(*GomegaWithT, *Controller, *v1alpha1.Backup)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		backup := newBackup()
		bkc, _, _ := newFakeBackupController()

		if len(test.conditionType) > 0 {
			backup.Status.Conditions = []v1alpha1.BackupCondition{
				{
					Type:   test.conditionType,
					Status: corev1.ConditionTrue,
				},
			}
		}

		if test.backupHasBeenDeleted {
			backup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		}

		if test.beforeUpdateFn != nil {
			test.beforeUpdateFn(g, bkc, backup)
		}

		bkc.updateBackup(backup)
		if test.expectFn != nil {
			test.expectFn(g, bkc)
		}

		if test.afterUpdateFn != nil {
			test.afterUpdateFn(g, bkc, backup)
		}
	}

	// create a job with failed status in the job informer.
	createFailedJob := func(g *GomegaWithT, rtc *Controller, backup *v1alpha1.Backup) {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backup.GetBackupJobName(),
				Namespace: backup.Namespace,
			},
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{
					{
						Type:   batchv1.JobFailed,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
		_, err := rtc.deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
		g.Expect(err).To(Succeed())
		rtc.deps.KubeInformerFactory.Start(context.TODO().Done())
		cache.WaitForCacheSync(context.TODO().Done(), rtc.deps.KubeInformerFactory.Batch().V1().Jobs().Informer().HasSynced)
	}

	updatingToFail := func(g *GomegaWithT, rtc *Controller, backup *v1alpha1.Backup) {
		control := rtc.control.(*FakeBackupControl)
		condition := control.condition
		g.Expect(condition).NotTo(Equal(nil))
		g.Expect(condition.Type).To(Equal(v1alpha1.BackupFailed))
		g.Expect(condition.Reason).To(Equal("AlreadyFailed"))
	}

	tests := []testcase{
		{
			name:                 "backup has been deleted",
			backupHasBeenDeleted: true,
			conditionType:        "", // no condition
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(1))
			},
		},
		{
			name:                 "backup is invalid",
			backupHasBeenDeleted: false,
			conditionType:        v1alpha1.BackupInvalid,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                 "backup has been completed",
			backupHasBeenDeleted: false,
			conditionType:        v1alpha1.BackupComplete,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                 "backup has been scheduled",
			backupHasBeenDeleted: false,
			conditionType:        v1alpha1.BackupScheduled,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                 "backup is newly created",
			backupHasBeenDeleted: false,
			conditionType:        "", // no condition
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(1))
			},
		},
		{
			name:                 "backup has been failed",
			backupHasBeenDeleted: false,
			conditionType:        v1alpha1.BackupFailed,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                 "backup has been scheduled with failed job",
			backupHasBeenDeleted: false,
			conditionType:        v1alpha1.BackupScheduled,
			beforeUpdateFn:       createFailedJob,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
			afterUpdateFn: updatingToFail,
		},
		{
			name:                 "backup has been running with failed job",
			backupHasBeenDeleted: false,
			conditionType:        v1alpha1.BackupRunning,
			beforeUpdateFn:       createFailedJob,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
			afterUpdateFn: updatingToFail,
		},
		{
			name:                 "backup has been prepared with failed job",
			backupHasBeenDeleted: false,
			conditionType:        v1alpha1.BackupPrepare,
			beforeUpdateFn:       createFailedJob,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
			afterUpdateFn: updatingToFail,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestBackupControllerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		addBackupToIndexer  bool
		errWhenUpdateBackup bool
		invalidKeyFn        func(backup *v1alpha1.Backup) string
		errExpectFn         func(*GomegaWithT, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		backup := newBackup()
		bkc, backupIndexer, backupControl := newFakeBackupController()

		if test.addBackupToIndexer {
			err := backupIndexer.Add(backup)
			g.Expect(err).NotTo(HaveOccurred())
		}

		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(backup)
		if test.invalidKeyFn != nil {
			key = test.invalidKeyFn(backup)
		}

		if test.errWhenUpdateBackup {
			backupControl.SetUpdateBackupError(fmt.Errorf("update backup failed"), 0)
		}

		err := bkc.sync(key)

		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}

	tests := []testcase{
		{
			name:                "normal",
			addBackupToIndexer:  true,
			errWhenUpdateBackup: false,
			invalidKeyFn:        nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                "invalid backup key",
			addBackupToIndexer:  true,
			errWhenUpdateBackup: false,
			invalidKeyFn: func(backup *v1alpha1.Backup) string {
				return fmt.Sprintf("test/demo/%s", backup.GetName())
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
		},
		{
			name:                "can't found backup",
			addBackupToIndexer:  false,
			errWhenUpdateBackup: false,
			invalidKeyFn:        nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                "update backup failed",
			addBackupToIndexer:  true,
			errWhenUpdateBackup: true,
			invalidKeyFn:        nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update backup failed")).To(Equal(true))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}

}

func newFakeBackupController() (*Controller, cache.Indexer, *FakeBackupControl) {
	fakeDeps := controller.NewFakeDependencies()
	bkc := NewController(fakeDeps)
	backupInformer := fakeDeps.InformerFactory.Pingcap().V1alpha1().Backups()
	backupControl := NewFakeBackupControl(backupInformer)
	bkc.control = backupControl
	return bkc, backupInformer.Informer().GetIndexer(), backupControl
}

func newBackup() *v1alpha1.Backup {
	return &v1alpha1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test-bk"),
		},
		Spec: v1alpha1.BackupSpec{
			From: &v1alpha1.TiDBAccessConfig{
				Host:       "10.1.1.2",
				Port:       v1alpha1.DefaultTiDBServerPort,
				User:       v1alpha1.DefaultTidbUser,
				SecretName: "demo1-tidb-secret",
			},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Provider:   v1alpha1.S3StorageProviderTypeCeph,
					Endpoint:   "http://10.0.0.1",
					Bucket:     "test1-demo1",
					SecretName: "demo",
				},
			},
			StorageClassName: pointer.StringPtr("local-storage"),
			StorageSize:      "1Gi",
			CleanPolicy:      v1alpha1.CleanPolicyTypeDelete,
		},
	}
}
