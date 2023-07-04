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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup/backupschedule"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBackupScheduleControlUpdateBackupSchedule(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name             string
		update           func(bs *v1alpha1.VolumeBackupSchedule)
		syncBsManagerErr bool
		updateStatusErr  bool
		errExpectFn      func(*GomegaWithT, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		bs := newBackupSchedule()
		if test.update != nil {
			test.update(bs)
		}
		control, bsManager, bsStatusUpdater := newFakeBackupScheduleControl()

		if test.syncBsManagerErr {
			bsManager.SetSyncError(fmt.Errorf("backup schedule sync error"))
		}

		if test.updateStatusErr {
			bsStatusUpdater.SetUpdateBackupScheduleError(fmt.Errorf("update backupSchedule status error"), 0)
		}

		err := control.UpdateBackupSchedule(bs)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}
	tests := []testcase{
		{
			name:             "backup schedule sync error",
			update:           nil,
			syncBsManagerErr: true,
			updateStatusErr:  false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "backup schedule sync error")).To(Equal(true))
			},
		},
		{
			name:             "backup schedule status is not updated",
			update:           nil,
			syncBsManagerErr: false,
			updateStatusErr:  false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "normal",
			update: func(bs *v1alpha1.VolumeBackupSchedule) {
				bs.Status.LastBackupTime = &metav1.Time{Time: time.Now()}
			},
			syncBsManagerErr: false,
			updateStatusErr:  false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "backup schedule status update failed",
			update: func(bs *v1alpha1.VolumeBackupSchedule) {
				bs.Status.LastBackupTime = &metav1.Time{Time: time.Now()}
			},
			syncBsManagerErr: false,
			updateStatusErr:  true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update backupSchedule status error")).To(Equal(true))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeBackupScheduleControl() (ControlInterface, *backupschedule.FakeBackupScheduleManager, *controller.FakeVolumeBackupScheduleStatusUpdater) {
	cli := fake.NewSimpleClientset()
	bsInformer := informers.NewSharedInformerFactory(cli, 0).Federation().V1alpha1().VolumeBackupSchedules()
	statusUpdater := controller.NewFakeVolumeBackupScheduleStatusUpdater(bsInformer)
	bsManager := backupschedule.NewFakeBackupScheduleManager()
	control := NewDefaultVolumeBackupScheduleControl(statusUpdater, bsManager)

	return control, bsManager, statusUpdater
}
