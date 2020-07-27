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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
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
		name                   string
		backupHasBeenDeleted   bool
		backupIsInvalid        bool
		backupHasBeenCompleted bool
		backupHasBeenScheduled bool
		expectFn               func(*GomegaWithT, *Controller)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		backup := newBackup()
		bkc, _, _ := newFakeBackupController()

		if test.backupHasBeenDeleted {
			backup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		}

		if test.backupIsInvalid {
			backup.Status.Conditions = []v1alpha1.BackupCondition{
				{
					Type:   v1alpha1.BackupInvalid,
					Status: corev1.ConditionTrue,
				},
			}
		}

		if test.backupHasBeenCompleted {
			backup.Status.Conditions = []v1alpha1.BackupCondition{
				{
					Type:   v1alpha1.BackupComplete,
					Status: corev1.ConditionTrue,
				},
			}
		}

		if test.backupHasBeenScheduled {
			backup.Status.Conditions = []v1alpha1.BackupCondition{
				{
					Type:   v1alpha1.BackupScheduled,
					Status: corev1.ConditionTrue,
				},
			}
		}

		bkc.updateBackup(backup)
		if test.expectFn != nil {
			test.expectFn(g, bkc)
		}
	}

	tests := []testcase{
		{
			name:                   "backup has been deleted",
			backupHasBeenDeleted:   true,
			backupIsInvalid:        false,
			backupHasBeenCompleted: false,
			backupHasBeenScheduled: false,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(1))
			},
		},
		{
			name:                   "backup is invalid",
			backupHasBeenDeleted:   false,
			backupIsInvalid:        true,
			backupHasBeenCompleted: false,
			backupHasBeenScheduled: false,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                   "backup has been completed",
			backupHasBeenDeleted:   false,
			backupIsInvalid:        false,
			backupHasBeenCompleted: true,
			backupHasBeenScheduled: false,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                   "backup has been scheduled",
			backupHasBeenDeleted:   false,
			backupIsInvalid:        false,
			backupHasBeenCompleted: false,
			backupHasBeenScheduled: true,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                   "backup is newly created",
			backupHasBeenDeleted:   false,
			backupIsInvalid:        false,
			backupHasBeenCompleted: false,
			backupHasBeenScheduled: false,
			expectFn: func(g *GomegaWithT, bkc *Controller) {
				g.Expect(bkc.queue.Len()).To(Equal(1))
			},
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
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(cli, 0)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)

	backupInformer := informerFactory.Pingcap().V1alpha1().Backups()
	backupControl := NewFakeBackupControl(backupInformer)

	bkc := NewController(kubeCli, cli, informerFactory, kubeInformerFactory)
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
			From: v1alpha1.TiDBAccessConfig{
				Host:       "10.1.1.2",
				Port:       constants.DefaultTidbPort,
				User:       constants.DefaultTidbUser,
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
