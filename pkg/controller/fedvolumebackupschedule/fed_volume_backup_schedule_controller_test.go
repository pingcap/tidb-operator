// Copyright 2023 PingCAP, Inc.
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

package fedvolumebackupschedule

import (
	"fmt"
	"testing"

	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	//"k8s.io/utils/pointer"
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
		name           string
		addBsToIndexer bool
		invalidKeyFn   func(bs *v1alpha1.VolumeBackupSchedule) string
		errExpectFn    func(*GomegaWithT, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		bs := newBackupSchedule()
		bsc, bsIndexer, _ := newFakeBackupScheduleController()

		if test.addBsToIndexer {
			err := bsIndexer.Add(bs)
			g.Expect(err).NotTo(HaveOccurred())
		}

		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(bs)
		if test.invalidKeyFn != nil {
			key = test.invalidKeyFn(bs)
		}

		err := bsc.sync(key)

		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}

	tests := []testcase{
		{
			name:           "normal",
			addBsToIndexer: true,
			invalidKeyFn:   nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:           "invalid backup key",
			addBsToIndexer: true,
			invalidKeyFn: func(bs *v1alpha1.VolumeBackupSchedule) string {
				return fmt.Sprintf("test/demo/%s", bs.GetName())
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
		},
		{
			name:           "can't found backup schedule",
			addBsToIndexer: false,
			invalidKeyFn:   nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}

}

func newFakeBackupScheduleController() (*Controller, cache.Indexer, *controller.FakeFedVolumeBackupControl) {
	fakeDeps := controller.NewFakeBrFedDependencies()
	bsc := NewController(fakeDeps)
	bsInformer := fakeDeps.InformerFactory.Federation().V1alpha1().VolumeBackups()
	backupScheduleControl := controller.NewFakeFedVolumeBackupControl(bsInformer)
	bsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: bsc.enqueueBackupSchedule,
		UpdateFunc: func(old, cur interface{}) {
			bsc.enqueueBackupSchedule(cur)
		},
		DeleteFunc: bsc.enqueueBackupSchedule,
	})
	//bsc.control = backupScheduleControl
	return bsc, bsInformer.Informer().GetIndexer(), backupScheduleControl
}

func newBackupSchedule() *v1alpha1.VolumeBackupSchedule {
	return &v1alpha1.VolumeBackupSchedule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BackupScheduler",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-bks",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test-bks"),
		},
		Spec: v1alpha1.VolumeBackupScheduleSpec{
			Schedule:       "1 */10 * * *",
			MaxBackups:     pointer.Int32Ptr(10),
			BackupTemplate: v1alpha1.VolumeBackupSpec{},
		},
	}
}
