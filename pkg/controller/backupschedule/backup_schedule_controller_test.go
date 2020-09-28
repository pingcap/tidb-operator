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
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestBackupScheduleControllerEnqueueBackupSchedule(t *testing.T) {
	g := NewGomegaWithT(t)
	bks := newBackupSchedule()
	bsc, _, _ := newFakeBackupScheduleController()
	bsc.enqueueBackupSchedule(bks)
	g.Expect(bsc.queue.Len()).To(Equal(1))
}

func TestBackupScheduleControllerEnqueueBackupScheduleFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	bsc, _, _ := newFakeBackupScheduleController()
	bsc.enqueueBackupSchedule(struct{}{})
	g.Expect(bsc.queue.Len()).To(Equal(0))
}

func TestBackupScheduleControllerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                        string
		addBsToIndexer              bool
		errWhenUpdateBackupSchedule bool
		invalidKeyFn                func(bs *v1alpha1.BackupSchedule) string
		errExpectFn                 func(*GomegaWithT, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		bs := newBackupSchedule()
		bsc, bsIndexer, bsControl := newFakeBackupScheduleController()

		if test.addBsToIndexer {
			err := bsIndexer.Add(bs)
			g.Expect(err).NotTo(HaveOccurred())
		}

		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(bs)
		if test.invalidKeyFn != nil {
			key = test.invalidKeyFn(bs)
		}

		if test.errWhenUpdateBackupSchedule {
			bsControl.SetUpdateBackupScheduleError(fmt.Errorf("update backup schedule failed"), 0)
		}

		err := bsc.sync(key)

		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}

	tests := []testcase{
		{
			name:                        "normal",
			addBsToIndexer:              true,
			errWhenUpdateBackupSchedule: false,
			invalidKeyFn:                nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                        "invalid backup key",
			addBsToIndexer:              true,
			errWhenUpdateBackupSchedule: false,
			invalidKeyFn: func(bs *v1alpha1.BackupSchedule) string {
				return fmt.Sprintf("test/demo/%s", bs.GetName())
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
		},
		{
			name:                        "can't found backup schedule",
			addBsToIndexer:              false,
			errWhenUpdateBackupSchedule: false,
			invalidKeyFn:                nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                        "update backup schedule failed",
			addBsToIndexer:              true,
			errWhenUpdateBackupSchedule: true,
			invalidKeyFn:                nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update backup schedule failed")).To(Equal(true))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}

}

func newFakeBackupScheduleController() (*Controller, cache.Indexer, *FakeBackupScheduleControl) {
	fakeDeps := controller.NewFakeDependencies()
	bsc := NewController(fakeDeps)
	bsInformer := fakeDeps.InformerFactory.Pingcap().V1alpha1().BackupSchedules()
	backupScheduleControl := NewFakeBackupScheduleControl(bsInformer)

	bsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: bsc.enqueueBackupSchedule,
		UpdateFunc: func(old, cur interface{}) {
			bsc.enqueueBackupSchedule(cur)
		},
		DeleteFunc: bsc.enqueueBackupSchedule,
	})

	return bsc, bsInformer.Informer().GetIndexer(), backupScheduleControl
}

func newBackupSchedule() *v1alpha1.BackupSchedule {
	return &v1alpha1.BackupSchedule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BackupScheduler",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-bks",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test-bks"),
		},
		Spec: v1alpha1.BackupScheduleSpec{
			Schedule:   "1 */10 * * *",
			MaxBackups: pointer.Int32Ptr(10),
			BackupTemplate: v1alpha1.BackupSpec{
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
						SecretName: "demo",
						Bucket:     "test1-demo1",
					},
				},
			},
		},
	}
}
