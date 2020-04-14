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

	"github.com/pingcap/tidb-operator/pkg/controller"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/backup"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestBackupControlUpdateBackup(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                 string
		update               func(backup *v1alpha1.Backup)
		syncBackupManagerErr bool
		updateBackupErr      bool
		errExpectFn          func(*GomegaWithT, *v1alpha1.Backup, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		backup := newBackup()
		if test.update != nil {
			test.update(backup)
		}
		control, backupIndexer, backupManager, fakeCli := newFakeBackupControl()

		backupIndexer.Add(backup)

		if test.syncBackupManagerErr {
			backupManager.SetSyncError(fmt.Errorf("backup manager sync error"))
		}

		if test.updateBackupErr {
			fakeCli.AddReactor("update", "backups", func(action core.Action) (bool, runtime.Object, error) {
				update := action.(core.UpdateAction)
				return true, update.GetObject(), apierrors.NewInternalError(fmt.Errorf("API server down"))
			})
		}

		err := control.UpdateBackup(backup)
		if test.errExpectFn != nil {
			test.errExpectFn(g, backup, err)
		}
	}
	tests := []testcase{
		{
			name:                 "add protection finalizer failed",
			update:               nil,
			syncBackupManagerErr: false,
			updateBackupErr:      true,
			errExpectFn: func(g *GomegaWithT, _ *v1alpha1.Backup, err error) {
				g.Expect(err).To(HaveOccurred())
				fmt.Println(err.Error())
				g.Expect(strings.Contains(err.Error(), "API server down")).To(Equal(true))
			},
		},
		{
			name: "remove protection finalizer failed",
			update: func(backup *v1alpha1.Backup) {
				backup.Finalizers = append(backup.Finalizers, label.BackupProtectionFinalizer)
				backup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				backup.Status.Conditions = []v1alpha1.BackupCondition{
					{
						Type:   v1alpha1.BackupClean,
						Status: corev1.ConditionTrue,
					},
				}
			},
			syncBackupManagerErr: false,
			updateBackupErr:      true,
			errExpectFn: func(g *GomegaWithT, _ *v1alpha1.Backup, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server down")).To(Equal(true))
			},
		},
		{
			name: "backup manager sync failed",
			update: func(backup *v1alpha1.Backup) {
				backup.Finalizers = append(backup.Finalizers, label.BackupProtectionFinalizer)
			},
			syncBackupManagerErr: true,
			updateBackupErr:      false,
			errExpectFn: func(g *GomegaWithT, _ *v1alpha1.Backup, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "backup manager sync error")).To(Equal(true))
			},
		},
		{
			name:                 "update newly create backup normally",
			update:               nil,
			syncBackupManagerErr: false,
			updateBackupErr:      false,
			errExpectFn: func(g *GomegaWithT, backup *v1alpha1.Backup, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(backup.Finalizers)).To(Equal(1))
				g.Expect(backup.Finalizers[0]).To(Equal(label.BackupProtectionFinalizer))
			},
		},
		{
			name: "update a deleted backup normally",
			update: func(backup *v1alpha1.Backup) {
				backup.Finalizers = append(backup.Finalizers, label.BackupProtectionFinalizer)
				backup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				backup.Status.Conditions = []v1alpha1.BackupCondition{
					{
						Type:   v1alpha1.BackupClean,
						Status: corev1.ConditionTrue,
					},
				}
			},
			syncBackupManagerErr: false,
			updateBackupErr:      false,
			errExpectFn: func(g *GomegaWithT, backup *v1alpha1.Backup, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				g.Expect(len(backup.Finalizers)).To(Equal(0))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeBackupControl() (ControlInterface, cache.Indexer, *backup.FakeBackupManager, *fake.Clientset) {
	cli := &fake.Clientset{}

	backupInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().Backups()
	backupManager := backup.NewFakeBackupManager()
	control := NewDefaultBackupControl(cli, backupManager)

	return control, backupInformer.Informer().GetIndexer(), backupManager, cli
}
